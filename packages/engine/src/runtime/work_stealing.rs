// packages/engine/src/runtime/work_stealing.rs
//! Work-stealing scheduler for distributing simulations across agent pool
//!
//! Implements a lock-free work-stealing algorithm to efficiently distribute
//! 10,000+ simulations across a small pool of agent processes.
//!
//! # Architecture
//!
//! ```text
//! Worker 1        Worker 2        Worker 3        Worker 4
//! [Tasks...]      [Tasks...]      [Tasks...]      [Tasks...]
//!    ↓               ↓               ↓               ↓
//!    └───────── Steal ←──────────────┘               │
//!                                   Steal ←──────────┘
//! ```

use crossbeam::deque::{Injector, Stealer, Worker};
use std::sync::Arc;
use tokio::sync::Notify;
use tracing::{debug, trace};

/// A simulation task to be executed
#[derive(Debug, Clone)]
pub struct Task {
    /// Unique task ID
    pub id: String,
    
    /// Agent code to execute
    pub code: String,
    
    /// Task priority (higher = more urgent)
    pub priority: u32,
    
    /// Created timestamp
    pub created_at: std::time::Instant,
}

impl Task {
    pub fn new(id: String, code: String) -> Self {
        Self {
            id,
            code,
            priority: 0,
            created_at: std::time::Instant::now(),
        }
    }
    
    pub fn with_priority(mut self, priority: u32) -> Self {
        self.priority = priority;
        self
    }
}

/// Work-stealing scheduler for distributing tasks
pub struct WorkStealingScheduler {
    /// Global task queue (injector)
    global_queue: Arc<Injector<Task>>,
    
    /// Per-worker local queues
    workers: Vec<Worker<Task>>,
    
    /// Stealers for each worker
    stealers: Vec<Stealer<Task>>,
    
    /// Notification for new tasks
    notify: Arc<Notify>,
    
    /// Number of workers
    num_workers: usize,
}

impl WorkStealingScheduler {
    /// Create a new work-stealing scheduler
    pub fn new(num_workers: usize) -> Self {
        let global_queue = Arc::new(Injector::new());
        let notify = Arc::new(Notify::new());
        
        // Create worker queues
        let mut workers = Vec::with_capacity(num_workers);
        let mut stealers = Vec::with_capacity(num_workers);
        
        for _ in 0..num_workers {
            let worker = Worker::new_fifo();
            stealers.push(worker.stealer());
            workers.push(worker);
        }
        
        debug!("Work-stealing scheduler initialized with {} workers", num_workers);
        
        Self {
            global_queue,
            workers,
            stealers,
            notify,
            num_workers,
        }
    }
    
    /// Submit a task to the global queue
    pub fn submit(&self, task: Task) {
        trace!("Submitting task {} to global queue", task.id);
        self.global_queue.push(task);
        self.notify.notify_one();
    }
    
    /// Submit multiple tasks in batch
    pub fn submit_batch(&self, tasks: Vec<Task>) {
        let count = tasks.len();
        trace!("Submitting batch of {} tasks", count);
        
        for task in tasks {
            self.global_queue.push(task);
        }
        
        // Notify multiple workers
        for _ in 0..count.min(self.num_workers) {
            self.notify.notify_one();
        }
    }
    
    /// Get the next task for a worker (with work stealing)
    pub async fn get_task(&self, worker_id: usize) -> Option<Task> {
        let worker = &self.workers[worker_id];
        
        loop {
            // Try local queue first
            if let Some(task) = worker.pop() {
                trace!("Worker {} got task from local queue", worker_id);
                return Some(task);
            }
            
            // Try stealing from global queue
            match self.global_queue.steal() {
                crossbeam::deque::Steal::Success(task) => {
                    trace!("Worker {} stole task from global queue", worker_id);
                    return Some(task);
                }
                crossbeam::deque::Steal::Empty => {
                    // Try stealing from other workers
                    if let Some(task) = self.steal_from_others(worker_id) {
                        trace!("Worker {} stole task from another worker", worker_id);
                        return Some(task);
                    }
                    
                    // No tasks available, wait for notification
                    trace!("Worker {} waiting for tasks", worker_id);
                    self.notify.notified().await;
                }
                crossbeam::deque::Steal::Retry => {
                    // Race condition, retry
                    continue;
                }
            }
        }
    }
    
    /// Try to steal a task from other workers
    fn steal_from_others(&self, worker_id: usize) -> Option<Task> {
        // Try stealing from each worker in random order
        use rand::seq::SliceRandom;
        let mut rng = rand::thread_rng();
        
        let mut indices: Vec<usize> = (0..self.num_workers)
            .filter(|&i| i != worker_id)
            .collect();
        indices.shuffle(&mut rng);
        
        for &other_id in &indices {
            match self.stealers[other_id].steal() {
                crossbeam::deque::Steal::Success(task) => {
                    return Some(task);
                }
                crossbeam::deque::Steal::Empty | crossbeam::deque::Steal::Retry => {
                    continue;
                }
            }
        }
        
        None
    }
    
    /// Get scheduler statistics
    pub fn stats(&self) -> SchedulerStats {
        let global_count = self.global_queue.len();
        
        let local_counts: Vec<usize> = self.workers
            .iter()
            .map(|w| w.len())
            .collect();
        
        let total_local: usize = local_counts.iter().sum();
        
        SchedulerStats {
            global_queue_size: global_count,
            local_queue_sizes: local_counts,
            total_tasks: global_count + total_local,
            num_workers: self.num_workers,
        }
    }
}

/// Scheduler statistics
#[derive(Debug, Clone)]
pub struct SchedulerStats {
    pub global_queue_size: usize,
    pub local_queue_sizes: Vec<usize>,
    pub total_tasks: usize,
    pub num_workers: usize,
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_scheduler_creation() {
        let scheduler = WorkStealingScheduler::new(4);
        let stats = scheduler.stats();
        assert_eq!(stats.num_workers, 4);
        assert_eq!(stats.total_tasks, 0);
    }
    
    #[test]
    fn test_task_submission() {
        let scheduler = WorkStealingScheduler::new(4);
        
        let task = Task::new("task1".to_string(), "print('hello')".to_string());
        scheduler.submit(task);
        
        let stats = scheduler.stats();
        assert_eq!(stats.total_tasks, 1);
    }
    
    #[test]
    fn test_batch_submission() {
        let scheduler = WorkStealingScheduler::new(4);
        
        let tasks: Vec<Task> = (0..10)
            .map(|i| Task::new(format!("task{}", i), "code".to_string()))
            .collect();
        
        scheduler.submit_batch(tasks);
        
        let stats = scheduler.stats();
        assert_eq!(stats.total_tasks, 10);
    }
    
    #[tokio::test]
    async fn test_get_task() {
        let scheduler = WorkStealingScheduler::new(4);
        
        let task = Task::new("task1".to_string(), "print('hello')".to_string());
        scheduler.submit(task.clone());
        
        let retrieved = scheduler.get_task(0).await;
        assert!(retrieved.is_some());
        assert_eq!(retrieved.unwrap().id, task.id);
    }
}
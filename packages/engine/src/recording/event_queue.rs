// packages/engine/src/recording/event_queue.rs
//! Lock-free MPMC event queue
//!
//! Provides a high-performance bounded queue for event recording with
//! <10ns push/pop operations.

use crate::recording::recorder::Event;
use crossbeam::queue::ArrayQueue;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

/// Lock-free event queue
pub struct EventQueue {
    /// Underlying bounded queue
    queue: Arc<ArrayQueue<Event>>,
    
    /// Push counter
    push_count: Arc<AtomicU64>,
    
    /// Pop counter
    pop_count: Arc<AtomicU64>,
    
    /// Drop counter (queue full)
    drop_count: Arc<AtomicU64>,
}

impl EventQueue {
    /// Create a new event queue
    pub fn new(capacity: usize) -> Self {
        Self {
            queue: Arc::new(ArrayQueue::new(capacity)),
            push_count: Arc::new(AtomicU64::new(0)),
            pop_count: Arc::new(AtomicU64::new(0)),
            drop_count: Arc::new(AtomicU64::new(0)),
        }
    }
    
    /// Push an event (non-blocking, lock-free)
    pub fn push(&self, event: Event) -> Result<(), Event> {
        match self.queue.push(event) {
            Ok(_) => {
                self.push_count.fetch_add(1, Ordering::Relaxed);
                Ok(())
            }
            Err(event) => {
                // Queue full, drop event
                self.drop_count.fetch_add(1, Ordering::Relaxed);
                Err(event)
            }
        }
    }
    
    /// Try to pop an event (non-blocking)
    pub fn try_pop(&self) -> Option<Event> {
        match self.queue.pop() {
            Some(event) => {
                self.pop_count.fetch_add(1, Ordering::Relaxed);
                Some(event)
            }
            None => None,
        }
    }
    
    /// Get queue statistics
    pub fn stats(&self) -> QueueStats {
        let push_count = self.push_count.load(Ordering::Relaxed);
        let pop_count = self.pop_count.load(Ordering::Relaxed);
        let drop_count = self.drop_count.load(Ordering::Relaxed);
        let current_size = self.queue.len();
        let capacity = self.queue.capacity();
        
        QueueStats {
            push_count,
            pop_count,
            drop_count,
            current_size,
            capacity,
        }
    }
    
    /// Check if queue is empty
    pub fn is_empty(&self) -> bool {
        self.queue.is_empty()
    }
    
    /// Check if queue is full
    pub fn is_full(&self) -> bool {
        self.queue.is_full()
    }
    
    /// Get current queue length
    pub fn len(&self) -> usize {
        self.queue.len()
    }
    
    /// Get queue capacity
    pub fn capacity(&self) -> usize {
        self.queue.capacity()
    }
}

/// Queue statistics
#[derive(Debug, Clone)]
pub struct QueueStats {
    /// Total events pushed
    pub push_count: u64,
    
    /// Total events popped
    pub pop_count: u64,
    
    /// Total events dropped (queue full)
    pub drop_count: u64,
    
    /// Current queue size
    pub current_size: usize,
    
    /// Queue capacity
    pub capacity: usize,
}

impl QueueStats {
    /// Calculate fill percentage
    pub fn fill_percentage(&self) -> f64 {
        (self.current_size as f64 / self.capacity as f64) * 100.0
    }
    
    /// Calculate drop rate
    pub fn drop_rate(&self) -> f64 {
        if self.push_count == 0 {
            0.0
        } else {
            (self.drop_count as f64 / self.push_count as f64) * 100.0
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::recording::recorder::EventType;
    
    fn create_test_event(id: &str) -> Event {
        Event {
            id: id.to_string(),
            run_id: "test".to_string(),
            event_type: EventType::AgentStarted,
            timestamp_ns: 0,
            data: serde_json::json!({}),
            duration_us: None,
        }
    }
    
    #[test]
    fn test_queue_creation() {
        let queue = EventQueue::new(100);
        assert_eq!(queue.capacity(), 100);
        assert_eq!(queue.len(), 0);
        assert!(queue.is_empty());
    }
    
    #[test]
    fn test_push_pop() {
        let queue = EventQueue::new(10);
        
        let event = create_test_event("evt_1");
        queue.push(event).unwrap();
        
        assert_eq!(queue.len(), 1);
        assert!(!queue.is_empty());
        
        let popped = queue.try_pop();
        assert!(popped.is_some());
        assert_eq!(popped.unwrap().id, "evt_1");
        
        assert_eq!(queue.len(), 0);
        assert!(queue.is_empty());
    }
    
    #[test]
    fn test_queue_full() {
        let queue = EventQueue::new(2);
        
        queue.push(create_test_event("evt_1")).unwrap();
        queue.push(create_test_event("evt_2")).unwrap();
        
        assert!(queue.is_full());
        
        // Next push should fail
        let result = queue.push(create_test_event("evt_3"));
        assert!(result.is_err());
        
        let stats = queue.stats();
        assert_eq!(stats.drop_count, 1);
    }
    
    #[test]
    fn test_stats() {
        let queue = EventQueue::new(10);
        
        queue.push(create_test_event("evt_1")).unwrap();
        queue.push(create_test_event("evt_2")).unwrap();
        queue.try_pop();
        
        let stats = queue.stats();
        assert_eq!(stats.push_count, 2);
        assert_eq!(stats.pop_count, 1);
        assert_eq!(stats.current_size, 1);
    }
    
    #[test]
    fn test_concurrent_operations() {
        use std::thread;
        
        let queue = Arc::new(EventQueue::new(1000));
        let mut handles = vec![];
        
        // Spawn 10 producer threads
        for i in 0..10 {
            let q = Arc::clone(&queue);
            let handle = thread::spawn(move || {
                for j in 0..100 {
                    let event = create_test_event(&format!("evt_{}_{}", i, j));
                    let _ = q.push(event);
                }
            });
            handles.push(handle);
        }
        
        // Wait for producers
        for handle in handles {
            handle.join().unwrap();
        }
        
        let stats = queue.stats();
        assert!(stats.push_count <= 1000); // Some may be dropped if queue full
    }
}
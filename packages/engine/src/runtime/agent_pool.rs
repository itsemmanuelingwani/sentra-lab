// packages/engine/src/runtime/agent_pool.rs
//! Agent process pool for efficient resource usage
//!
//! Instead of spawning 10,000 separate processes, we maintain a small pool
//! of reusable agent processes (32-64) and multiplex simulations across them.
//!
//! # Architecture
//!
//! ```text
//! AgentPool
//! ├─ Available: [Agent1, Agent2, ...]  (idle processes)
//! ├─ Busy: [Agent3, Agent4, ...]       (running simulations)
//! └─ Waiters: [Task1, Task2, ...]      (queued tasks)
//! ```
//!
//! # Performance
//!
//! - 64 processes × 50MB = 3.2GB (vs 500GB for 10K processes)
//! - Acquire latency: <1ms (from available pool)
//! - Blocking wait: when all processes busy (backpressure)

use crate::runtime::agent_runtime::{AgentRuntime, AgentRuntimeConfig};
use crate::runtime::process_manager::ProcessType;
use crate::utils::errors::{EngineError, Result};
use std::sync::Arc;
use tokio::sync::{Semaphore, Mutex};
use tracing::{debug, info, warn};

/// Configuration for the agent pool
#[derive(Debug, Clone)]
pub struct AgentPoolConfig {
    /// Number of agent processes in the pool (default: 64)
    pub pool_size: usize,
    
    /// Maximum concurrent simulations (default: 10,000)
    pub max_concurrent: usize,
    
    /// Per-agent timeout in seconds (default: 300)
    pub agent_timeout_secs: u64,
    
    /// Process types to support
    pub supported_types: Vec<ProcessType>,
}

impl Default for AgentPoolConfig {
    fn default() -> Self {
        Self {
            pool_size: 64,
            max_concurrent: 10_000,
            agent_timeout_secs: 300,
            supported_types: vec![
                ProcessType::Python,
                ProcessType::NodeJs,
                ProcessType::Go,
            ],
        }
    }
}

/// A pooled agent that can be reused across simulations
pub struct PooledAgent {
    /// Unique ID for this agent in the pool
    pub id: usize,
    
    /// The underlying agent runtime
    pub runtime: AgentRuntime,
    
    /// Process type (Python, Node.js, Go)
    pub process_type: ProcessType,
    
    /// Number of simulations executed by this agent
    pub execution_count: u64,
}

impl PooledAgent {
    /// Create a new pooled agent
    async fn new(id: usize, process_type: ProcessType, config: AgentRuntimeConfig) -> Result<Self> {
        let runtime = AgentRuntime::new(config).await?;
        
        Ok(Self {
            id,
            runtime,
            process_type,
            execution_count: 0,
        })
    }
    
    /// Execute a simulation on this agent
    pub async fn execute(&mut self, code: &str) -> Result<String> {
        self.execution_count += 1;
        self.runtime.execute(code).await
    }
    
    /// Reset agent state (between simulations)
    pub async fn reset(&mut self) -> Result<()> {
        self.runtime.reset().await
    }
}

/// Agent pool for efficient resource management
pub struct AgentPool {
    /// Configuration
    config: AgentPoolConfig,
    
    /// Available agents (idle pool)
    available: Arc<Mutex<Vec<PooledAgent>>>,
    
    /// Semaphore to limit concurrent acquisitions
    semaphore: Arc<Semaphore>,
    
    /// Total agents created
    total_agents: Arc<Mutex<usize>>,
}

impl AgentPool {
    /// Create a new agent pool
    pub async fn new(pool_size: usize) -> Result<Self> {
        let config = AgentPoolConfig {
            pool_size,
            ..Default::default()
        };
        
        Self::with_config(config).await
    }
    
    /// Create agent pool with custom configuration
    pub async fn with_config(config: AgentPoolConfig) -> Result<Self> {
        info!("Initializing agent pool with {} processes", config.pool_size);
        
        let available = Arc::new(Mutex::new(Vec::with_capacity(config.pool_size)));
        let semaphore = Arc::new(Semaphore::new(config.pool_size));
        let total_agents = Arc::new(Mutex::new(0));
        
        let pool = Self {
            config,
            available,
            semaphore,
            total_agents,
        };
        
        // Pre-spawn agent processes for each supported type
        pool.initialize_pool().await?;
        
        Ok(pool)
    }
    
    /// Pre-spawn agent processes
    async fn initialize_pool(&self) -> Result<()> {
        let agents_per_type = self.config.pool_size / self.config.supported_types.len();
        
        for process_type in &self.config.supported_types {
            for i in 0..agents_per_type {
                let agent_id = {
                    let mut total = self.total_agents.lock().await;
                    *total += 1;
                    *total
                };
                
                debug!("Spawning {:?} agent #{}", process_type, agent_id);
                
                let runtime_config = AgentRuntimeConfig {
                    process_type: *process_type,
                    timeout_secs: self.config.agent_timeout_secs,
                    ..Default::default()
                };
                
                match PooledAgent::new(agent_id, *process_type, runtime_config).await {
                    Ok(agent) => {
                        let mut available = self.available.lock().await;
                        available.push(agent);
                    }
                    Err(e) => {
                        warn!("Failed to spawn agent #{}: {}", agent_id, e);
                        return Err(e);
                    }
                }
            }
        }
        
        info!("Agent pool initialized with {} processes", self.config.pool_size);
        Ok(())
    }
    
    /// Acquire an agent from the pool (blocks if all busy)
    pub async fn acquire(&self) -> Result<PooledAgent> {
        // Wait for available slot (backpressure mechanism)
        let permit = self.semaphore.acquire().await
            .map_err(|_| EngineError::PoolExhausted)?;
        
        // Get agent from available pool
        let mut available = self.available.lock().await;
        
        if let Some(agent) = available.pop() {
            debug!("Acquired agent #{} from pool", agent.id);
            permit.forget(); // Keep semaphore acquired
            Ok(agent)
        } else {
            // Pool exhausted (shouldn't happen due to semaphore)
            warn!("Agent pool exhausted despite semaphore");
            Err(EngineError::PoolExhausted)
        }
    }
    
    /// Release an agent back to the pool
    pub async fn release(&self, mut agent: PooledAgent) -> Result<()> {
        // Reset agent state
        if let Err(e) = agent.reset().await {
            warn!("Failed to reset agent #{}: {}", agent.id, e);
            // Don't return agent to pool if reset failed
            self.semaphore.add_permits(1);
            return Err(e);
        }
        
        debug!("Releasing agent #{} back to pool", agent.id);
        
        // Return to available pool
        let mut available = self.available.lock().await;
        available.push(agent);
        
        // Release semaphore
        self.semaphore.add_permits(1);
        
        Ok(())
    }
    
    /// Get pool statistics
    pub async fn stats(&self) -> PoolStats {
        let available = self.available.lock().await;
        let available_count = available.len();
        
        PoolStats {
            total_agents: self.config.pool_size,
            available_agents: available_count,
            busy_agents: self.config.pool_size - available_count,
            max_concurrent: self.config.max_concurrent,
        }
    }
}

/// Pool statistics
#[derive(Debug, Clone)]
pub struct PoolStats {
    pub total_agents: usize,
    pub available_agents: usize,
    pub busy_agents: usize,
    pub max_concurrent: usize,
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_pool_creation() {
        let pool = AgentPool::new(8).await.unwrap();
        let stats = pool.stats().await;
        assert_eq!(stats.total_agents, 8);
        assert_eq!(stats.available_agents, 8);
    }
    
    #[tokio::test]
    async fn test_acquire_release() {
        let pool = AgentPool::new(4).await.unwrap();
        
        // Acquire agent
        let agent = pool.acquire().await.unwrap();
        let stats = pool.stats().await;
        assert_eq!(stats.available_agents, 3);
        assert_eq!(stats.busy_agents, 1);
        
        // Release agent
        pool.release(agent).await.unwrap();
        let stats = pool.stats().await;
        assert_eq!(stats.available_agents, 4);
        assert_eq!(stats.busy_agents, 0);
    }
    
    #[tokio::test]
    async fn test_concurrent_acquisitions() {
        let pool = Arc::new(AgentPool::new(4).await.unwrap());
        
        let mut handles = vec![];
        
        // Spawn 10 tasks trying to acquire agents
        for i in 0..10 {
            let pool_clone = Arc::clone(&pool);
            let handle = tokio::spawn(async move {
                let agent = pool_clone.acquire().await.unwrap();
                tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
                pool_clone.release(agent).await.unwrap();
                i
            });
            handles.push(handle);
        }
        
        // Wait for all tasks
        for handle in handles {
            handle.await.unwrap();
        }
        
        // All agents should be back in pool
        let stats = pool.stats().await;
        assert_eq!(stats.available_agents, 4);
    }
}
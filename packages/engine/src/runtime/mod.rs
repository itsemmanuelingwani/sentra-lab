// packages/engine/src/runtime/mod.rs
//! Agent execution runtime
//!
//! This module provides the core agent execution environment, including:
//!
//! - **Agent Pool**: Pooled agent processes for efficient resource usage
//! - **Agent Runtime**: Lifecycle management for individual agents
//! - **Process Manager**: Process spawning and management (Python, Node.js, Go)
//! - **Sandbox**: Isolated execution with resource limits
//! - **Resource Limiter**: CPU, memory, and network throttling
//! - **Work Stealing**: Efficient task scheduling across agent pool
//!
//! # Architecture
//!
//! ```text
//! ┌─────────────────────────────────────────────────────────┐
//! │                    Agent Pool (64)                      │
//! │  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
//! │  │ Python   │  │ Node.js  │  │ Go       │  ...        │
//! │  │ Process  │  │ Process  │  │ Process  │             │
//! │  └──────────┘  └──────────┘  └──────────┘             │
//! │         ▲            ▲            ▲                     │
//! │         │            │            │                     │
//! │         └────────────┴────────────┘                     │
//! │                      │                                  │
//! │           Work-Stealing Scheduler                       │
//! │                      │                                  │
//! │         ┌────────────┴────────────┐                     │
//! │         │                         │                     │
//! │    10,000 Async Tasks (Simulations)                     │
//! └─────────────────────────────────────────────────────────┘
//! ```
//!
//! # Performance
//!
//! - **Memory per simulation**: <2KB (vs 50MB per process)
//! - **Concurrent simulations**: 10,000+ on a laptop
//! - **Pool size**: 32-64 processes (configurable)
//! - **Task overhead**: <100ns per context switch

pub mod agent_pool;
pub mod agent_runtime;
pub mod process_manager;
pub mod resource_limiter;
pub mod sandbox;
pub mod work_stealing;

// Re-export commonly used types
pub use agent_pool::{AgentPool, AgentPoolConfig, PooledAgent};
pub use agent_runtime::{AgentRuntime, AgentRuntimeConfig, RuntimeHandle};
pub use process_manager::{ProcessManager, ProcessType, SpawnConfig};
pub use resource_limiter::{ResourceLimits, ResourceLimiter};
pub use sandbox::{Sandbox, SandboxConfig};
pub use work_stealing::{WorkStealingScheduler, Task};
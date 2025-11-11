// packages/engine/src/lib.rs
//! Sentra Lab Simulation Engine Library
//!
//! This library provides the core components for running high-performance
//! AI agent simulations with production parity.
//!
//! # Architecture
//!
//! The engine is structured into several key modules:
//!
//! - **runtime**: Agent execution, process management, sandboxing
//! - **executor**: Scenario orchestration and execution
//! - **interception**: HTTP/DNS/syscall interception layer
//! - **recording**: High-performance event capture and storage
//! - **replay**: Time-travel debugging and deterministic replay
//! - **state**: State management with copy-on-write semantics
//! - **cost**: Real-time cost estimation and tracking
//! - **grpc**: gRPC API server
//! - **observability**: Metrics, tracing, and logging
//! - **utils**: Common utilities and helpers

// Public module exports
pub mod cost;
pub mod executor;
pub mod grpc;
pub mod interception;
pub mod observability;
pub mod recording;
pub mod replay;
pub mod runtime;
pub mod state;
pub mod utils;

// Re-export commonly used types
pub use runtime::agent_pool::{AgentPool, AgentPoolConfig};
pub use runtime::agent_runtime::{AgentRuntime, AgentRuntimeConfig};
pub use utils::config::EngineConfig;
pub use utils::errors::{EngineError, Result};

// Version information
pub const VERSION: &str = env!("CARGO_PKG_VERSION");
pub const GIT_HASH: &str = env!("GIT_HASH");

/// Engine build information
pub struct BuildInfo {
    pub version: &'static str,
    pub git_hash: &'static str,
    pub build_timestamp: &'static str,
    pub rustc_version: &'static str,
}

impl BuildInfo {
    pub fn current() -> Self {
        Self {
            version: VERSION,
            git_hash: GIT_HASH,
            build_timestamp: env!("BUILD_TIMESTAMP"),
            rustc_version: env!("RUSTC_VERSION"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_version() {
        assert!(!VERSION.is_empty());
    }

    #[test]
    fn test_build_info() {
        let info = BuildInfo::current();
        assert!(!info.version.is_empty());
    }
}
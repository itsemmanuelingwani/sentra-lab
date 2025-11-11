// packages/engine/src/runtime/agent_runtime.rs
//! Agent runtime for lifecycle management
//!
//! Manages the lifecycle of individual agent processes including:
//! - Process spawning and initialization
//! - Execution of agent code
//! - State reset between simulations
//! - Graceful shutdown and cleanup

use crate::runtime::process_manager::{ProcessManager, ProcessType, SpawnConfig};
use crate::runtime::sandbox::{Sandbox, SandboxConfig};
use crate::utils::errors::{EngineError, Result};
use std::process::Child;
use std::time::Duration;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::process::ChildStdin;
use tokio::sync::Mutex;
use tracing::{debug, error, warn};

/// Configuration for agent runtime
#[derive(Debug, Clone)]
pub struct AgentRuntimeConfig {
    /// Type of process to spawn
    pub process_type: ProcessType,
    
    /// Timeout for execution in seconds
    pub timeout_secs: u64,
    
    /// Sandbox configuration
    pub sandbox: SandboxConfig,
    
    /// Working directory for agent
    pub work_dir: Option<String>,
    
    /// Environment variables
    pub env_vars: Vec<(String, String)>,
}

impl Default for AgentRuntimeConfig {
    fn default() -> Self {
        Self {
            process_type: ProcessType::Python,
            timeout_secs: 300,
            sandbox: SandboxConfig::default(),
            work_dir: None,
            env_vars: vec![],
        }
    }
}

/// Handle to a running agent runtime
pub struct RuntimeHandle {
    /// Process ID
    pub pid: u32,
    
    /// Process type
    pub process_type: ProcessType,
    
    /// Started timestamp
    pub started_at: std::time::Instant,
}

/// Agent runtime managing a single agent process
pub struct AgentRuntime {
    /// Configuration
    config: AgentRuntimeConfig,
    
    /// The spawned process
    process: Mutex<Option<Child>>,
    
    /// Standard input handle for sending commands
    stdin: Mutex<Option<ChildStdin>>,
    
    /// Process manager
    manager: ProcessManager,
    
    /// Sandbox (resource limits)
    sandbox: Sandbox,
    
    /// Runtime handle
    handle: Option<RuntimeHandle>,
}

impl AgentRuntime {
    /// Create and initialize a new agent runtime
    pub async fn new(config: AgentRuntimeConfig) -> Result<Self> {
        let manager = ProcessManager::new();
        let sandbox = Sandbox::new(config.sandbox.clone())?;
        
        let mut runtime = Self {
            config,
            process: Mutex::new(None),
            stdin: Mutex::new(None),
            manager,
            sandbox,
            handle: None,
        };
        
        // Spawn initial process
        runtime.spawn().await?;
        
        Ok(runtime)
    }
    
    /// Spawn the agent process
    async fn spawn(&mut self) -> Result<()> {
        debug!("Spawning {:?} agent process", self.config.process_type);
        
        // Configure process spawn
        let spawn_config = SpawnConfig {
            process_type: self.config.process_type,
            work_dir: self.config.work_dir.clone(),
            env_vars: self.config.env_vars.clone(),
            timeout: Duration::from_secs(self.config.timeout_secs),
        };
        
        // Spawn process
        let mut child = self.manager.spawn(spawn_config).await?;
        
        // Apply resource limits
        if let Some(pid) = child.id() {
            self.sandbox.apply_limits(pid)?;
            
            self.handle = Some(RuntimeHandle {
                pid,
                process_type: self.config.process_type,
                started_at: std::time::Instant::now(),
            });
        }
        
        // Take stdin for communication
        let stdin = child.stdin.take()
            .ok_or_else(|| EngineError::ProcessSpawnFailed("Failed to capture stdin".into()))?;
        
        // Store process and stdin
        *self.process.lock().await = Some(child);
        *self.stdin.lock().await = Some(stdin);
        
        debug!("Agent process spawned successfully");
        Ok(())
    }
    
    /// Execute code on this agent
    pub async fn execute(&mut self, code: &str) -> Result<String> {
        debug!("Executing code on agent");
        
        // Get stdin handle
        let mut stdin_guard = self.stdin.lock().await;
        let stdin = stdin_guard.as_mut()
            .ok_or_else(|| EngineError::RuntimeError("No stdin available".into()))?;
        
        // Send code to agent process
        stdin.write_all(code.as_bytes()).await
            .map_err(|e| EngineError::RuntimeError(format!("Failed to write to stdin: {}", e)))?;
        
        stdin.write_all(b"\n__END__\n").await
            .map_err(|e| EngineError::RuntimeError(format!("Failed to write delimiter: {}", e)))?;
        
        stdin.flush().await
            .map_err(|e| EngineError::RuntimeError(format!("Failed to flush stdin: {}", e)))?;
        
        drop(stdin_guard); // Release lock
        
        // Read response with timeout
        let timeout = Duration::from_secs(self.config.timeout_secs);
        let result = tokio::time::timeout(timeout, self.read_response()).await
            .map_err(|_| EngineError::ExecutionTimeout)?;
        
        result
    }
    
    /// Read response from agent process
    async fn read_response(&self) -> Result<String> {
        let mut process_guard = self.process.lock().await;
        let process = process_guard.as_mut()
            .ok_or_else(|| EngineError::RuntimeError("No process available".into()))?;
        
        let stdout = process.stdout.take()
            .ok_or_else(|| EngineError::RuntimeError("Failed to capture stdout".into()))?;
        
        let mut reader = BufReader::new(stdout);
        let mut output = String::new();
        let mut line = String::new();
        
        loop {
            line.clear();
            match reader.read_line(&mut line).await {
                Ok(0) => break, // EOF
                Ok(_) => {
                    if line.trim() == "__END__" {
                        break;
                    }
                    output.push_str(&line);
                }
                Err(e) => {
                    error!("Error reading from stdout: {}", e);
                    return Err(EngineError::RuntimeError(format!("Read error: {}", e)));
                }
            }
        }
        
        // Return stdout to process
        process.stdout = Some(reader.into_inner());
        
        Ok(output)
    }
    
    /// Reset agent state (between simulations)
    pub async fn reset(&mut self) -> Result<()> {
        debug!("Resetting agent state");
        
        // For now, we restart the process
        // TODO: Implement in-process state reset for faster resets
        self.shutdown().await?;
        self.spawn().await?;
        
        Ok(())
    }
    
    /// Check if agent is healthy
    pub async fn health_check(&self) -> Result<bool> {
        let process_guard = self.process.lock().await;
        
        if let Some(process) = process_guard.as_ref() {
            // Check if process is still running
            match process.id() {
                Some(pid) => {
                    // Simple health check: process exists
                    Ok(true)
                }
                None => Ok(false),
            }
        } else {
            Ok(false)
        }
    }
    
    /// Gracefully shutdown the agent
    pub async fn shutdown(&mut self) -> Result<()> {
        debug!("Shutting down agent");
        
        let mut process_guard = self.process.lock().await;
        
        if let Some(mut process) = process_guard.take() {
            // Try graceful shutdown first
            if let Err(e) = process.kill().await {
                warn!("Failed to kill process gracefully: {}", e);
            }
            
            // Wait for process to exit
            match tokio::time::timeout(
                Duration::from_secs(5),
                process.wait()
            ).await {
                Ok(Ok(status)) => {
                    debug!("Process exited with status: {}", status);
                }
                Ok(Err(e)) => {
                    error!("Error waiting for process: {}", e);
                }
                Err(_) => {
                    warn!("Process did not exit in time, forcing kill");
                }
            }
        }
        
        *self.stdin.lock().await = None;
        
        Ok(())
    }
    
    /// Get runtime handle
    pub fn handle(&self) -> Option<&RuntimeHandle> {
        self.handle.as_ref()
    }
}

impl Drop for AgentRuntime {
    fn drop(&mut self) {
        // Best-effort cleanup
        let _ = futures::executor::block_on(self.shutdown());
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_runtime_creation() {
        let config = AgentRuntimeConfig::default();
        let runtime = AgentRuntime::new(config).await;
        assert!(runtime.is_ok());
    }
    
    #[tokio::test]
    async fn test_health_check() {
        let config = AgentRuntimeConfig::default();
        let runtime = AgentRuntime::new(config).await.unwrap();
        let is_healthy = runtime.health_check().await.unwrap();
        assert!(is_healthy);
    }
}
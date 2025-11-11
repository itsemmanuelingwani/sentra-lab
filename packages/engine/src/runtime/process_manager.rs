// packages/engine/src/runtime/process_manager.rs
//! Process manager for spawning and managing agent processes
//!
//! Supports multiple process types:
//! - Python (python3)
//! - Node.js (node)
//! - Go (go run)

use crate::utils::errors::{EngineError, Result};
use std::path::PathBuf;
use std::process::Stdio;
use std::time::Duration;
use tokio::process::{Child, Command};
use tracing::{debug, info};

/// Supported process types
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProcessType {
    Python,
    NodeJs,
    Go,
}

impl ProcessType {
    /// Get the command name for this process type
    pub fn command(&self) -> &str {
        match self {
            ProcessType::Python => "python3",
            ProcessType::NodeJs => "node",
            ProcessType::Go => "go",
        }
    }
    
    /// Get default arguments for this process type
    pub fn default_args(&self) -> Vec<&str> {
        match self {
            ProcessType::Python => vec!["-u", "-i"], // Unbuffered, interactive
            ProcessType::NodeJs => vec!["-i"], // Interactive REPL
            ProcessType::Go => vec!["run"], // go run
        }
    }
    
    /// Get the file extension for this process type
    pub fn extension(&self) -> &str {
        match self {
            ProcessType::Python => "py",
            ProcessType::NodeJs => "js",
            ProcessType::Go => "go",
        }
    }
}

/// Configuration for spawning a process
#[derive(Debug, Clone)]
pub struct SpawnConfig {
    /// Type of process to spawn
    pub process_type: ProcessType,
    
    /// Working directory
    pub work_dir: Option<String>,
    
    /// Environment variables
    pub env_vars: Vec<(String, String)>,
    
    /// Execution timeout
    pub timeout: Duration,
}

impl Default for SpawnConfig {
    fn default() -> Self {
        Self {
            process_type: ProcessType::Python,
            work_dir: None,
            env_vars: vec![],
            timeout: Duration::from_secs(300),
        }
    }
}

/// Process manager for spawning agent processes
pub struct ProcessManager {
    /// Paths to executables (cached)
    executable_paths: std::collections::HashMap<ProcessType, PathBuf>,
}

impl ProcessManager {
    /// Create a new process manager
    pub fn new() -> Self {
        Self {
            executable_paths: std::collections::HashMap::new(),
        }
    }
    
    /// Find executable for a process type
    fn find_executable(&mut self, process_type: ProcessType) -> Result<PathBuf> {
        // Check cache first
        if let Some(path) = self.executable_paths.get(&process_type) {
            return Ok(path.clone());
        }
        
        let command = process_type.command();
        
        // Try to find executable in PATH
        match which::which(command) {
            Ok(path) => {
                info!("Found {} at {:?}", command, path);
                self.executable_paths.insert(process_type, path.clone());
                Ok(path)
            }
            Err(e) => {
                Err(EngineError::ProcessSpawnFailed(
                    format!("Executable '{}' not found in PATH: {}", command, e)
                ))
            }
        }
    }
    
    /// Spawn a new process
    pub async fn spawn(&mut self, config: SpawnConfig) -> Result<Child> {
        let executable = self.find_executable(config.process_type)?;
        
        debug!("Spawning {:?} process: {:?}", config.process_type, executable);
        
        // Build command
        let mut command = Command::new(executable);
        
        // Add default arguments
        for arg in config.process_type.default_args() {
            command.arg(arg);
        }
        
        // Set working directory
        if let Some(work_dir) = &config.work_dir {
            command.current_dir(work_dir);
        }
        
        // Set environment variables
        for (key, value) in &config.env_vars {
            command.env(key, value);
        }
        
        // Configure stdio (we need stdin, stdout, stderr)
        command
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        
        // Spawn process
        let child = command.spawn()
            .map_err(|e| EngineError::ProcessSpawnFailed(
                format!("Failed to spawn process: {}", e)
            ))?;
        
        debug!("Process spawned with PID: {:?}", child.id());
        
        Ok(child)
    }
    
    /// Kill a process by PID
    pub async fn kill(&self, pid: u32) -> Result<()> {
        use nix::sys::signal::{kill, Signal};
        use nix::unistd::Pid;
        
        let pid = Pid::from_raw(pid as i32);
        
        // Try SIGTERM first (graceful)
        debug!("Sending SIGTERM to PID {}", pid);
        kill(pid, Signal::SIGTERM)
            .map_err(|e| EngineError::RuntimeError(format!("Failed to send SIGTERM: {}", e)))?;
        
        // Wait a bit
        tokio::time::sleep(Duration::from_secs(2)).await;
        
        // Check if still alive, send SIGKILL
        if kill(pid, None).is_ok() {
            debug!("Process still alive, sending SIGKILL to PID {}", pid);
            kill(pid, Signal::SIGKILL)
                .map_err(|e| EngineError::RuntimeError(format!("Failed to send SIGKILL: {}", e)))?;
        }
        
        Ok(())
    }
    
    /// Check if a process is running
    pub fn is_running(&self, pid: u32) -> bool {
        use nix::sys::signal::kill;
        use nix::unistd::Pid;
        
        let pid = Pid::from_raw(pid as i32);
        kill(pid, None).is_ok()
    }
}

impl Default for ProcessManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_find_executable() {
        let mut manager = ProcessManager::new();
        
        // Python should be available in CI
        let result = manager.find_executable(ProcessType::Python);
        assert!(result.is_ok());
    }
    
    #[tokio::test]
    async fn test_spawn_python() {
        let mut manager = ProcessManager::new();
        let config = SpawnConfig {
            process_type: ProcessType::Python,
            ..Default::default()
        };
        
        let result = manager.spawn(config).await;
        assert!(result.is_ok());
        
        if let Ok(mut child) = result {
            let _ = child.kill().await;
        }
    }
    
    #[test]
    fn test_process_type_command() {
        assert_eq!(ProcessType::Python.command(), "python3");
        assert_eq!(ProcessType::NodeJs.command(), "node");
        assert_eq!(ProcessType::Go.command(), "go");
    }
    
    #[test]
    fn test_process_type_extension() {
        assert_eq!(ProcessType::Python.extension(), "py");
        assert_eq!(ProcessType::NodeJs.extension(), "js");
        assert_eq!(ProcessType::Go.extension(), "go");
    }
}
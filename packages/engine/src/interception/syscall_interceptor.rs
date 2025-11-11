// packages/engine/src/interception/syscall_interceptor.rs
//! System call interceptor using LD_PRELOAD (Linux only)
//!
//! Intercepts low-level system calls to control:
//! - Network operations (socket, connect, bind)
//! - File I/O (open, read, write)
//! - Time operations (gettimeofday, clock_gettime)
//!
//! This provides the deepest level of interception for maximum control.

use crate::utils::errors::{EngineError, Result};
use std::path::PathBuf;
use tracing::{debug, warn};

/// Syscall interceptor configuration
#[derive(Debug, Clone)]
pub struct SyscallConfig {
    /// Enable network syscall interception
    pub intercept_network: bool,
    
    /// Enable file I/O interception
    pub intercept_file_io: bool,
    
    /// Enable time syscall interception (for determinism)
    pub intercept_time: bool,
    
    /// Path to preload library
    pub preload_library_path: Option<PathBuf>,
}

impl Default for SyscallConfig {
    fn default() -> Self {
        Self {
            intercept_network: true,
            intercept_file_io: false, // Too invasive, disabled by default
            intercept_time: true,     // Important for determinism
            preload_library_path: None,
        }
    }
}

/// Syscall interceptor (Linux only)
pub struct SyscallInterceptor {
    config: SyscallConfig,
}

impl SyscallInterceptor {
    /// Create a new syscall interceptor
    pub fn new(config: SyscallConfig) -> Self {
        Self { config }
    }
    
    /// Generate LD_PRELOAD environment variable
    pub fn get_preload_env(&self) -> Result<Option<String>> {
        #[cfg(target_os = "linux")]
        {
            if let Some(lib_path) = &self.config.preload_library_path {
                if lib_path.exists() {
                    debug!("Using LD_PRELOAD library: {:?}", lib_path);
                    Ok(Some(lib_path.to_string_lossy().to_string()))
                } else {
                    Err(EngineError::InterceptionFailed(format!(
                        "Preload library not found: {:?}",
                        lib_path
                    )))
                }
            } else {
                // Try to find in standard locations
                let standard_paths = vec![
                    "/usr/lib/sentra-lab/libinterceptor.so",
                    "/usr/local/lib/sentra-lab/libinterceptor.so",
                    "./target/release/libinterceptor.so",
                ];
                
                for path in standard_paths {
                    let path_buf = PathBuf::from(path);
                    if path_buf.exists() {
                        debug!("Found preload library at: {}", path);
                        return Ok(Some(path.to_string()));
                    }
                }
                
                warn!("No preload library found, syscall interception disabled");
                Ok(None)
            }
        }
        
        #[cfg(not(target_os = "linux"))]
        {
            warn!("Syscall interception not supported on this platform");
            Ok(None)
        }
    }
    
    /// Check if syscall interception is available
    pub fn is_available(&self) -> bool {
        #[cfg(target_os = "linux")]
        {
            self.get_preload_env().ok().flatten().is_some()
        }
        
        #[cfg(not(target_os = "linux"))]
        {
            false
        }
    }
    
    /// Get environment variables for process spawn
    pub fn get_env_vars(&self) -> Vec<(String, String)> {
        let mut env_vars = Vec::new();
        
        if let Ok(Some(preload)) = self.get_preload_env() {
            env_vars.push(("LD_PRELOAD".to_string(), preload));
        }
        
        // Pass interceptor configuration via env vars
        if self.config.intercept_network {
            env_vars.push(("SENTRA_INTERCEPT_NETWORK".to_string(), "1".to_string()));
        }
        
        if self.config.intercept_file_io {
            env_vars.push(("SENTRA_INTERCEPT_FILE_IO".to_string(), "1".to_string()));
        }
        
        if self.config.intercept_time {
            env_vars.push(("SENTRA_INTERCEPT_TIME".to_string(), "1".to_string()));
        }
        
        env_vars
    }
}

impl Default for SyscallInterceptor {
    fn default() -> Self {
        Self::new(SyscallConfig::default())
    }
}

/// Intercepted syscall types
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SyscallType {
    /// Network syscalls (socket, connect, bind, etc.)
    Network,
    
    /// File I/O syscalls (open, read, write, etc.)
    FileIO,
    
    /// Time syscalls (gettimeofday, clock_gettime, etc.)
    Time,
}

/// Syscall event (for recording)
#[derive(Debug, Clone)]
pub struct SyscallEvent {
    /// Type of syscall
    pub syscall_type: SyscallType,
    
    /// Syscall name
    pub name: String,
    
    /// Arguments (serialized)
    pub args: Vec<String>,
    
    /// Return value
    pub return_value: i64,
    
    /// Timestamp
    pub timestamp: std::time::Instant,
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_config_default() {
        let config = SyscallConfig::default();
        assert!(config.intercept_network);
        assert!(!config.intercept_file_io);
        assert!(config.intercept_time);
    }
    
    #[test]
    fn test_interceptor_creation() {
        let config = SyscallConfig::default();
        let interceptor = SyscallInterceptor::new(config);
        assert!(interceptor.config.intercept_network);
    }
    
    #[test]
    fn test_env_vars() {
        let config = SyscallConfig::default();
        let interceptor = SyscallInterceptor::new(config);
        let env_vars = interceptor.get_env_vars();
        
        // Should have at least the interceptor flags
        assert!(env_vars.iter().any(|(k, _)| k == "SENTRA_INTERCEPT_NETWORK"));
        assert!(env_vars.iter().any(|(k, _)| k == "SENTRA_INTERCEPT_TIME"));
    }
}
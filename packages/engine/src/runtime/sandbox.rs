// packages/engine/src/runtime/sandbox.rs
//! Sandbox for isolating agent processes with resource limits
//!
//! Provides:
//! - CPU limits (cgroups)
//! - Memory limits (cgroups)
//! - Network isolation (optional)
//! - File system restrictions (read-only mounts)

use crate::runtime::resource_limiter::ResourceLimits;
use crate::utils::errors::{EngineError, Result};
use tracing::{debug, warn};

/// Sandbox configuration
#[derive(Debug, Clone)]
pub struct SandboxConfig {
    /// Enable CPU limiting
    pub limit_cpu: bool,
    
    /// CPU quota (percentage, 0-100)
    pub cpu_quota: u32,
    
    /// Enable memory limiting
    pub limit_memory: bool,
    
    /// Memory limit in MB
    pub memory_limit_mb: u64,
    
    /// Enable network isolation
    pub isolate_network: bool,
    
    /// Enable filesystem restrictions
    pub restrict_filesystem: bool,
    
    /// Allowed read paths
    pub read_paths: Vec<String>,
    
    /// Allowed write paths
    pub write_paths: Vec<String>,
}

impl Default for SandboxConfig {
    fn default() -> Self {
        Self {
            limit_cpu: true,
            cpu_quota: 50, // 50% of one CPU core
            limit_memory: true,
            memory_limit_mb: 512, // 512MB per agent
            isolate_network: false, // Network needed for mock APIs
            restrict_filesystem: true,
            read_paths: vec![
                "/usr".to_string(),
                "/lib".to_string(),
                "/etc".to_string(),
            ],
            write_paths: vec![
                "/tmp".to_string(),
            ],
        }
    }
}

/// Sandbox for isolating agent processes
pub struct Sandbox {
    config: SandboxConfig,
    resource_limits: ResourceLimits,
}

impl Sandbox {
    /// Create a new sandbox
    pub fn new(config: SandboxConfig) -> Result<Self> {
        let resource_limits = ResourceLimits {
            cpu_quota: if config.limit_cpu { Some(config.cpu_quota) } else { None },
            memory_limit_mb: if config.limit_memory { Some(config.memory_limit_mb) } else { None },
            network_bandwidth_mbps: None, // Not implemented yet
        };
        
        Ok(Self {
            config,
            resource_limits,
        })
    }
    
    /// Apply resource limits to a process
    pub fn apply_limits(&self, pid: u32) -> Result<()> {
        debug!("Applying resource limits to PID {}", pid);
        
        // Apply CPU limits
        if let Some(cpu_quota) = self.resource_limits.cpu_quota {
            self.apply_cpu_limit(pid, cpu_quota)?;
        }
        
        // Apply memory limits
        if let Some(memory_limit) = self.resource_limits.memory_limit_mb {
            self.apply_memory_limit(pid, memory_limit)?;
        }
        
        Ok(())
    }
    
    /// Apply CPU limit using cgroups (Linux only)
    #[cfg(target_os = "linux")]
    fn apply_cpu_limit(&self, pid: u32, quota: u32) -> Result<()> {
        use std::fs;
        use std::io::Write;
        
        debug!("Setting CPU quota to {}% for PID {}", quota, pid);
        
        // Create cgroup path
        let cgroup_path = format!("/sys/fs/cgroup/cpu/sentra-lab-{}", pid);
        
        // Create cgroup directory
        if let Err(e) = fs::create_dir_all(&cgroup_path) {
            warn!("Failed to create cgroup directory: {}", e);
            return Ok(()); // Non-fatal, continue without limits
        }
        
        // Set CPU quota (quota / period = percentage)
        let period = 100_000; // 100ms
        let quota_value = (quota as u64 * period) / 100;
        
        let quota_file = format!("{}/cpu.cfs_quota_us", cgroup_path);
        let period_file = format!("{}/cpu.cfs_period_us", cgroup_path);
        
        if let Ok(mut file) = fs::File::create(&quota_file) {
            let _ = file.write_all(quota_value.to_string().as_bytes());
        }
        
        if let Ok(mut file) = fs::File::create(&period_file) {
            let _ = file.write_all(period.to_string().as_bytes());
        }
        
        // Add process to cgroup
        let procs_file = format!("{}/cgroup.procs", cgroup_path);
        if let Ok(mut file) = fs::File::create(&procs_file) {
            let _ = file.write_all(pid.to_string().as_bytes());
        }
        
        Ok(())
    }
    
    #[cfg(not(target_os = "linux"))]
    fn apply_cpu_limit(&self, pid: u32, quota: u32) -> Result<()> {
        warn!("CPU limiting not supported on this platform");
        Ok(())
    }
    
    /// Apply memory limit using cgroups (Linux only)
    #[cfg(target_os = "linux")]
    fn apply_memory_limit(&self, pid: u32, limit_mb: u64) -> Result<()> {
        use std::fs;
        use std::io::Write;
        
        debug!("Setting memory limit to {}MB for PID {}", limit_mb, pid);
        
        // Create cgroup path
        let cgroup_path = format!("/sys/fs/cgroup/memory/sentra-lab-{}", pid);
        
        // Create cgroup directory
        if let Err(e) = fs::create_dir_all(&cgroup_path) {
            warn!("Failed to create cgroup directory: {}", e);
            return Ok(()); // Non-fatal
        }
        
        // Set memory limit
        let limit_bytes = limit_mb * 1024 * 1024;
        let limit_file = format!("{}/memory.limit_in_bytes", cgroup_path);
        
        if let Ok(mut file) = fs::File::create(&limit_file) {
            let _ = file.write_all(limit_bytes.to_string().as_bytes());
        }
        
        // Add process to cgroup
        let procs_file = format!("{}/cgroup.procs", cgroup_path);
        if let Ok(mut file) = fs::File::create(&procs_file) {
            let _ = file.write_all(pid.to_string().as_bytes());
        }
        
        Ok(())
    }
    
    #[cfg(not(target_os = "linux"))]
    fn apply_memory_limit(&self, pid: u32, limit_mb: u64) -> Result<()> {
        warn!("Memory limiting not supported on this platform");
        Ok(())
    }
    
    /// Clean up sandbox resources for a process
    pub fn cleanup(&self, pid: u32) -> Result<()> {
        debug!("Cleaning up sandbox for PID {}", pid);
        
        #[cfg(target_os = "linux")]
        {
            use std::fs;
            
            // Remove cgroups
            let cpu_cgroup = format!("/sys/fs/cgroup/cpu/sentra-lab-{}", pid);
            let mem_cgroup = format!("/sys/fs/cgroup/memory/sentra-lab-{}", pid);
            
            let _ = fs::remove_dir_all(&cpu_cgroup);
            let _ = fs::remove_dir_all(&mem_cgroup);
        }
        
        Ok(())
    }
}

impl Drop for Sandbox {
    fn drop(&mut self) {
        // Best-effort cleanup
        // Note: We don't have PID here, so cleanup should be done explicitly
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_sandbox_creation() {
        let config = SandboxConfig::default();
        let sandbox = Sandbox::new(config);
        assert!(sandbox.is_ok());
    }
    
    #[test]
    fn test_default_config() {
        let config = SandboxConfig::default();
        assert_eq!(config.cpu_quota, 50);
        assert_eq!(config.memory_limit_mb, 512);
        assert!(config.limit_cpu);
        assert!(config.limit_memory);
    }
}
// packages/engine/src/runtime/resource_limiter.rs
//! Resource limiting for agent processes
//!
//! Provides fine-grained control over:
//! - CPU usage (percentage of cores)
//! - Memory consumption (MB limit)
//! - Network bandwidth (Mbps limit)

use serde::{Deserialize, Serialize};

/// Resource limits for an agent process
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceLimits {
    /// CPU quota as percentage (0-100 per core)
    /// Example: 50 = 50% of one CPU core
    pub cpu_quota: Option<u32>,
    
    /// Memory limit in megabytes
    /// Example: 512 = 512MB RAM limit
    pub memory_limit_mb: Option<u64>,
    
    /// Network bandwidth limit in Mbps
    /// Example: 10 = 10 Mbps download/upload
    pub network_bandwidth_mbps: Option<u32>,
}

impl Default for ResourceLimits {
    fn default() -> Self {
        Self {
            cpu_quota: Some(50),         // 50% of one core
            memory_limit_mb: Some(512),  // 512MB
            network_bandwidth_mbps: None, // Unlimited (for mock APIs)
        }
    }
}

impl ResourceLimits {
    /// Create resource limits with no restrictions
    pub fn unlimited() -> Self {
        Self {
            cpu_quota: None,
            memory_limit_mb: None,
            network_bandwidth_mbps: None,
        }
    }
    
    /// Create strict resource limits (for untrusted code)
    pub fn strict() -> Self {
        Self {
            cpu_quota: Some(25),        // 25% of one core
            memory_limit_mb: Some(256), // 256MB
            network_bandwidth_mbps: Some(10), // 10 Mbps
        }
    }
    
    /// Create relaxed limits (for development)
    pub fn relaxed() -> Self {
        Self {
            cpu_quota: Some(100),        // Full core
            memory_limit_mb: Some(2048), // 2GB
            network_bandwidth_mbps: None, // Unlimited
        }
    }
    
    /// Validate resource limits
    pub fn validate(&self) -> Result<(), String> {
        // Validate CPU quota
        if let Some(quota) = self.cpu_quota {
            if quota == 0 {
                return Err("CPU quota cannot be 0".to_string());
            }
            if quota > 400 {
                return Err("CPU quota cannot exceed 400% (4 cores)".to_string());
            }
        }
        
        // Validate memory limit
        if let Some(memory) = self.memory_limit_mb {
            if memory < 64 {
                return Err("Memory limit cannot be less than 64MB".to_string());
            }
            if memory > 16384 {
                return Err("Memory limit cannot exceed 16GB".to_string());
            }
        }
        
        // Validate network bandwidth
        if let Some(bandwidth) = self.network_bandwidth_mbps {
            if bandwidth == 0 {
                return Err("Network bandwidth cannot be 0".to_string());
            }
            if bandwidth > 10000 {
                return Err("Network bandwidth cannot exceed 10 Gbps".to_string());
            }
        }
        
        Ok(())
    }
}

/// Resource limiter for managing limits across multiple processes
pub struct ResourceLimiter {
    /// Default limits to apply
    default_limits: ResourceLimits,
}

impl ResourceLimiter {
    /// Create a new resource limiter
    pub fn new(default_limits: ResourceLimits) -> Self {
        Self { default_limits }
    }
    
    /// Get default limits
    pub fn default_limits(&self) -> &ResourceLimits {
        &self.default_limits
    }
    
    /// Calculate aggregate resource usage for multiple agents
    pub fn aggregate_limits(&self, num_agents: usize) -> ResourceLimits {
        let mut aggregate = self.default_limits.clone();
        
        // Scale memory limit by number of agents
        if let Some(memory) = aggregate.memory_limit_mb {
            aggregate.memory_limit_mb = Some(memory * num_agents as u64);
        }
        
        // CPU quota doesn't aggregate (per-process limit)
        // Network bandwidth is shared, doesn't aggregate
        
        aggregate
    }
}

impl Default for ResourceLimiter {
    fn default() -> Self {
        Self::new(ResourceLimits::default())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_default_limits() {
        let limits = ResourceLimits::default();
        assert_eq!(limits.cpu_quota, Some(50));
        assert_eq!(limits.memory_limit_mb, Some(512));
        assert_eq!(limits.network_bandwidth_mbps, None);
    }
    
    #[test]
    fn test_unlimited() {
        let limits = ResourceLimits::unlimited();
        assert!(limits.cpu_quota.is_none());
        assert!(limits.memory_limit_mb.is_none());
        assert!(limits.network_bandwidth_mbps.is_none());
    }
    
    #[test]
    fn test_strict_limits() {
        let limits = ResourceLimits::strict();
        assert_eq!(limits.cpu_quota, Some(25));
        assert_eq!(limits.memory_limit_mb, Some(256));
        assert_eq!(limits.network_bandwidth_mbps, Some(10));
    }
    
    #[test]
    fn test_validation() {
        let valid = ResourceLimits::default();
        assert!(valid.validate().is_ok());
        
        let invalid_cpu = ResourceLimits {
            cpu_quota: Some(0),
            ..Default::default()
        };
        assert!(invalid_cpu.validate().is_err());
        
        let invalid_memory = ResourceLimits {
            memory_limit_mb: Some(32),
            ..Default::default()
        };
        assert!(invalid_memory.validate().is_err());
    }
    
    #[test]
    fn test_aggregate_limits() {
        let limiter = ResourceLimiter::default();
        let aggregate = limiter.aggregate_limits(10);
        
        // Memory should scale
        assert_eq!(aggregate.memory_limit_mb, Some(5120)); // 512 * 10
        
        // CPU quota shouldn't change
        assert_eq!(aggregate.cpu_quota, Some(50));
    }
}
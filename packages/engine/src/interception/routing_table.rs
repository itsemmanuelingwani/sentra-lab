// packages/engine/src/interception/routing_table.rs
//! Routing table for mapping domains to mock services
//!
//! Provides domain-based routing to redirect API calls to appropriate
//! mock services.

use crate::utils::errors::{EngineError, Result};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{debug, info};

/// Route definition
#[derive(Debug, Clone)]
pub struct Route {
    /// Source domain (e.g., "api.openai.com")
    pub domain: String,
    
    /// Target mock service URL (e.g., "http://localhost:8080")
    pub target: String,
    
    /// Optional path prefix to prepend
    pub path_prefix: Option<String>,
    
    /// Route priority (higher = checked first)
    pub priority: u32,
}

impl Route {
    pub fn new(domain: impl Into<String>, target: impl Into<String>) -> Self {
        Self {
            domain: domain.into(),
            target: target.into(),
            path_prefix: None,
            priority: 0,
        }
    }
    
    pub fn with_prefix(mut self, prefix: impl Into<String>) -> Self {
        self.path_prefix = Some(prefix.into());
        self
    }
    
    pub fn with_priority(mut self, priority: u32) -> Self {
        self.priority = priority;
        self
    }
}

/// Routing table
pub struct RoutingTable {
    /// Domain to route mapping
    routes: Arc<RwLock<HashMap<String, Route>>>,
}

impl RoutingTable {
    /// Create a new routing table
    pub fn new() -> Self {
        Self {
            routes: Arc::new(RwLock::new(HashMap::new())),
        }
    }
    
    /// Create routing table with default routes
    pub fn with_defaults() -> Self {
        let table = Self::new();
        
        // Add common API routes
        let routes = vec![
            Route::new("api.openai.com", "http://localhost:8080"),
            Route::new("api.anthropic.com", "http://localhost:8081"),
            Route::new("api.stripe.com", "http://localhost:8082"),
            Route::new("api.cohere.ai", "http://localhost:8083"),
            Route::new("generativelanguage.googleapis.com", "http://localhost:8084"),
        ];
        
        for route in routes {
            let _ = futures::executor::block_on(table.add_route(route));
        }
        
        table
    }
    
    /// Add a route
    pub async fn add_route(&self, route: Route) -> Result<()> {
        let domain = route.domain.clone();
        let mut routes = self.routes.write().await;
        
        info!("Adding route: {} -> {}", domain, route.target);
        
        routes.insert(domain, route);
        Ok(())
    }
    
    /// Remove a route
    pub async fn remove_route(&self, domain: &str) -> Result<()> {
        let mut routes = self.routes.write().await;
        
        if routes.remove(domain).is_some() {
            info!("Removed route for {}", domain);
            Ok(())
        } else {
            Err(EngineError::ConfigError(format!(
                "No route found for domain: {}",
                domain
            )))
        }
    }
    
    /// Lookup a route by domain
    pub fn lookup(&self, domain: &str) -> Option<Route> {
        let routes = futures::executor::block_on(self.routes.read());
        
        // Exact match first
        if let Some(route) = routes.get(domain) {
            debug!("Found exact route for {}", domain);
            return Some(route.clone());
        }
        
        // Wildcard match (e.g., *.openai.com)
        for (pattern, route) in routes.iter() {
            if pattern.starts_with("*.") {
                let suffix = &pattern[2..];
                if domain.ends_with(suffix) {
                    debug!("Found wildcard route for {} using {}", domain, pattern);
                    return Some(route.clone());
                }
            }
        }
        
        debug!("No route found for {}", domain);
        None
    }
    
    /// Get all routes
    pub async fn get_routes(&self) -> Vec<Route> {
        let routes = self.routes.read().await;
        routes.values().cloned().collect()
    }
    
    /// Clear all routes
    pub async fn clear_routes(&self) {
        let mut routes = self.routes.write().await;
        routes.clear();
        info!("Cleared all routes");
    }
    
    /// Export routes as configuration
    pub async fn export_config(&self) -> String {
        let routes = self.routes.read().await;
        
        let mut output = String::from("# Sentra Lab Routing Table\n\n");
        
        for route in routes.values() {
            output.push_str(&format!(
                "{} -> {}\n",
                route.domain, route.target
            ));
        }
        
        output
    }
}

impl Default for RoutingTable {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_add_route() {
        let table = RoutingTable::new();
        
        let route = Route::new("api.example.com", "http://localhost:8080");
        table.add_route(route).await.unwrap();
        
        let found = table.lookup("api.example.com");
        assert!(found.is_some());
        assert_eq!(found.unwrap().target, "http://localhost:8080");
    }
    
    #[tokio::test]
    async fn test_default_routes() {
        let table = RoutingTable::with_defaults();
        
        let openai = table.lookup("api.openai.com");
        assert!(openai.is_some());
        assert_eq!(openai.unwrap().target, "http://localhost:8080");
        
        let stripe = table.lookup("api.stripe.com");
        assert!(stripe.is_some());
        assert_eq!(stripe.unwrap().target, "http://localhost:8082");
    }
    
    #[tokio::test]
    async fn test_wildcard_match() {
        let table = RoutingTable::new();
        
        let route = Route::new("*.openai.com", "http://localhost:8080");
        table.add_route(route).await.unwrap();
        
        let found = table.lookup("api.openai.com");
        assert!(found.is_some());
        
        let found = table.lookup("chat.openai.com");
        assert!(found.is_some());
    }
    
    #[tokio::test]
    async fn test_remove_route() {
        let table = RoutingTable::new();
        
        let route = Route::new("api.example.com", "http://localhost:8080");
        table.add_route(route).await.unwrap();
        
        table.remove_route("api.example.com").await.unwrap();
        
        let found = table.lookup("api.example.com");
        assert!(found.is_none());
    }
    
    #[tokio::test]
    async fn test_export_config() {
        let table = RoutingTable::with_defaults();
        let config = table.export_config().await;
        
        assert!(config.contains("api.openai.com"));
        assert!(config.contains("api.stripe.com"));
    }
}
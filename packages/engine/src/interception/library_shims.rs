// packages/engine/src/interception/library_shims.rs
//! Library shims for SDK-specific interception
//!
//! Provides shim code that agents can import to redirect SDK calls
//! to mock services without modifying their code.
//!
//! Supported SDKs:
//! - OpenAI (Python, Node.js, Go)
//! - Anthropic (Python, TypeScript)
//! - Stripe (Python, Node.js, Go, Ruby)

use crate::utils::errors::Result;
use std::collections::HashMap;
use tracing::{debug, info};

/// Configuration for library shims
#[derive(Debug, Clone)]
pub struct ShimConfig {
    /// Mock base URLs by service
    pub mock_base_urls: HashMap<String, String>,
    
    /// Enable automatic shim injection
    pub auto_inject: bool,
    
    /// Custom shim scripts path
    pub custom_shims_path: Option<String>,
}

impl Default for ShimConfig {
    fn default() -> Self {
        let mut mock_base_urls = HashMap::new();
        mock_base_urls.insert("openai".to_string(), "http://localhost:8080".to_string());
        mock_base_urls.insert("anthropic".to_string(), "http://localhost:8081".to_string());
        mock_base_urls.insert("stripe".to_string(), "http://localhost:8082".to_string());
        
        Self {
            mock_base_urls,
            auto_inject: true,
            custom_shims_path: None,
        }
    }
}

/// Library shim manager
pub struct LibraryShim {
    config: ShimConfig,
}

impl LibraryShim {
    /// Create a new library shim manager
    pub fn new(config: ShimConfig) -> Self {
        info!("Initializing library shims with {} services", config.mock_base_urls.len());
        Self { config }
    }
    
    /// Generate Python shim code for OpenAI
    pub fn generate_openai_python_shim(&self) -> String {
        let base_url = self.config.mock_base_urls
            .get("openai")
            .unwrap_or(&"http://localhost:8080".to_string());
        
        debug!("Generating OpenAI Python shim for {}", base_url);
        
        format!(
            r#"
# Sentra Lab OpenAI Python Shim
# Automatically redirects OpenAI API calls to local mock

import os
os.environ['OPENAI_BASE_URL'] = '{base_url}'
os.environ['OPENAI_API_KEY'] = 'mock_key_sentra_lab'

# Monkey-patch OpenAI client if already imported
try:
    import openai
    openai.api_base = '{base_url}'
    openai.api_key = 'mock_key_sentra_lab'
except ImportError:
    pass
"#,
            base_url = base_url
        )
    }
    
    /// Generate Node.js shim code for OpenAI
    pub fn generate_openai_nodejs_shim(&self) -> String {
        let base_url = self.config.mock_base_urls
            .get("openai")
            .unwrap_or(&"http://localhost:8080".to_string());
        
        debug!("Generating OpenAI Node.js shim for {}", base_url);
        
        format!(
            r#"
// Sentra Lab OpenAI Node.js Shim
// Automatically redirects OpenAI API calls to local mock

process.env.OPENAI_BASE_URL = '{base_url}';
process.env.OPENAI_API_KEY = 'mock_key_sentra_lab';

// Monkey-patch OpenAI client
const Module = require('module');
const originalRequire = Module.prototype.require;

Module.prototype.require = function(id) {{
  const module = originalRequire.apply(this, arguments);
  
  if (id === 'openai') {{
    // Override baseURL in OpenAI client
    const originalConstructor = module.OpenAI;
    module.OpenAI = class extends originalConstructor {{
      constructor(config = {{}}) {{
        config.baseURL = '{base_url}';
        config.apiKey = 'mock_key_sentra_lab';
        super(config);
      }}
    }};
  }}
  
  return module;
}};
"#,
            base_url = base_url
        )
    }
    
    /// Generate Go shim code for OpenAI
    pub fn generate_openai_go_shim(&self) -> String {
        let base_url = self.config.mock_base_urls
            .get("openai")
            .unwrap_or(&"http://localhost:8080".to_string());
        
        debug!("Generating OpenAI Go shim for {}", base_url);
        
        format!(
            r#"
// Sentra Lab OpenAI Go Shim
// Automatically redirects OpenAI API calls to local mock

package main

import (
    "os"
)

func init() {{
    // Set environment variables
    os.Setenv("OPENAI_BASE_URL", "{base_url}")
    os.Setenv("OPENAI_API_KEY", "mock_key_sentra_lab")
}}
"#,
            base_url = base_url
        )
    }
    
    /// Generate Python shim code for Stripe
    pub fn generate_stripe_python_shim(&self) -> String {
        let base_url = self.config.mock_base_urls
            .get("stripe")
            .unwrap_or(&"http://localhost:8082".to_string());
        
        debug!("Generating Stripe Python shim for {}", base_url);
        
        format!(
            r#"
# Sentra Lab Stripe Python Shim
# Automatically redirects Stripe API calls to local mock

import os
os.environ['STRIPE_API_BASE'] = '{base_url}'
os.environ['STRIPE_API_KEY'] = 'sk_test_mock_sentra_lab'

# Monkey-patch Stripe if already imported
try:
    import stripe
    stripe.api_base = '{base_url}'
    stripe.api_key = 'sk_test_mock_sentra_lab'
except ImportError:
    pass
"#,
            base_url = base_url
        )
    }
    
    /// Generate shim for a specific language and service
    pub fn generate_shim(&self, language: &str, service: &str) -> Result<String> {
        let shim = match (language, service) {
            ("python", "openai") => self.generate_openai_python_shim(),
            ("nodejs", "openai") => self.generate_openai_nodejs_shim(),
            ("go", "openai") => self.generate_openai_go_shim(),
            ("python", "stripe") => self.generate_stripe_python_shim(),
            _ => {
                return Err(crate::utils::errors::EngineError::ConfigError(
                    format!("No shim available for {} / {}", language, service)
                ))
            }
        };
        
        Ok(shim)
    }
    
    /// Get environment variables for shimming
    pub fn get_env_vars(&self) -> Vec<(String, String)> {
        let mut env_vars = Vec::new();
        
        // Add base URLs as env vars
        for (service, url) in &self.config.mock_base_urls {
            let env_key = format!("{}_BASE_URL", service.to_uppercase());
            env_vars.push((env_key, url.clone()));
        }
        
        // Add mock API keys
        env_vars.push(("OPENAI_API_KEY".to_string(), "mock_key_sentra_lab".to_string()));
        env_vars.push(("ANTHROPIC_API_KEY".to_string(), "mock_key_sentra_lab".to_string()));
        env_vars.push(("STRIPE_API_KEY".to_string(), "sk_test_mock_sentra_lab".to_string()));
        
        env_vars
    }
}

impl Default for LibraryShim {
    fn default() -> Self {
        Self::new(ShimConfig::default())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_config_default() {
        let config = ShimConfig::default();
        assert!(config.mock_base_urls.contains_key("openai"));
        assert!(config.auto_inject);
    }
    
    #[test]
    fn test_shim_creation() {
        let config = ShimConfig::default();
        let shim = LibraryShim::new(config);
        assert!(shim.config.mock_base_urls.len() >= 3);
    }
    
    #[test]
    fn test_python_shim_generation() {
        let shim = LibraryShim::default();
        let code = shim.generate_openai_python_shim();
        assert!(code.contains("OPENAI_BASE_URL"));
        assert!(code.contains("localhost:8080"));
    }
    
    #[test]
    fn test_nodejs_shim_generation() {
        let shim = LibraryShim::default();
        let code = shim.generate_openai_nodejs_shim();
        assert!(code.contains("process.env.OPENAI_BASE_URL"));
        assert!(code.contains("localhost:8080"));
    }
    
    #[test]
    fn test_go_shim_generation() {
        let shim = LibraryShim::default();
        let code = shim.generate_openai_go_shim();
        assert!(code.contains("os.Setenv"));
        assert!(code.contains("OPENAI_BASE_URL"));
    }
    
    #[test]
    fn test_env_vars() {
        let shim = LibraryShim::default();
        let env_vars = shim.get_env_vars();
        
        assert!(env_vars.iter().any(|(k, _)| k == "OPENAI_API_KEY"));
        assert!(env_vars.iter().any(|(k, _)| k == "STRIPE_API_KEY"));
    }
}
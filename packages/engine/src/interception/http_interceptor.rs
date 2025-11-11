// packages/engine/src/interception/http_interceptor.rs
//! HTTP/HTTPS interceptor using MITM proxy
//!
//! Transparently intercepts all HTTP/HTTPS traffic from agents and routes
//! to mock services. Handles TLS termination and re-encryption.

use crate::interception::routing_table::RoutingTable;
use crate::interception::tls_handler::TlsHandler;
use crate::utils::errors::{EngineError, Result};
use bytes::Bytes;
use http_body_util::{BodyExt, Empty, Full};
use hyper::body::Incoming;
use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{Method, Request, Response, StatusCode};
use hyper_util::rt::TokioIo;
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::net::TcpListener;
use tracing::{debug, error, info, warn};

/// Configuration for HTTP interceptor
#[derive(Debug, Clone)]
pub struct InterceptorConfig {
    /// Proxy listen address
    pub listen_addr: SocketAddr,
    
    /// Enable HTTPS interception
    pub enable_https: bool,
    
    /// Enable request logging
    pub log_requests: bool,
    
    /// Enable response logging
    pub log_responses: bool,
    
    /// Maximum body size to log (bytes)
    pub max_log_body_size: usize,
}

impl Default for InterceptorConfig {
    fn default() -> Self {
        Self {
            listen_addr: "127.0.0.1:8888".parse().unwrap(),
            enable_https: true,
            log_requests: true,
            log_responses: true,
            max_log_body_size: 10_000, // 10KB
        }
    }
}

/// HTTP/HTTPS interceptor
pub struct HttpInterceptor {
    config: InterceptorConfig,
    routing_table: Arc<RoutingTable>,
    tls_handler: Arc<TlsHandler>,
    http_client: hyper_util::client::legacy::Client<
        hyper_util::client::legacy::connect::HttpConnector,
        Full<Bytes>,
    >,
}

impl HttpInterceptor {
    /// Create a new HTTP interceptor
    pub fn new(
        config: InterceptorConfig,
        routing_table: Arc<RoutingTable>,
        tls_handler: Arc<TlsHandler>,
    ) -> Self {
        let http_client = hyper_util::client::legacy::Client::builder(
            hyper_util::rt::TokioExecutor::new(),
        )
        .build_http();
        
        Self {
            config,
            routing_table,
            tls_handler,
            http_client,
        }
    }
    
    /// Start the interceptor proxy server
    pub async fn start(self: Arc<Self>) -> Result<()> {
        let listener = TcpListener::bind(self.config.listen_addr)
            .await
            .map_err(|e| {
                EngineError::InterceptionFailed(format!("Failed to bind proxy: {}", e))
            })?;
        
        info!("HTTP interceptor listening on {}", self.config.listen_addr);
        
        loop {
            match listener.accept().await {
                Ok((stream, addr)) => {
                    let interceptor = Arc::clone(&self);
                    
                    tokio::spawn(async move {
                        debug!("Accepted connection from {}", addr);
                        
                        let io = TokioIo::new(stream);
                        
                        let service = service_fn(move |req| {
                            let interceptor = Arc::clone(&interceptor);
                            async move { interceptor.handle_request(req).await }
                        });
                        
                        if let Err(e) = http1::Builder::new()
                            .serve_connection(io, service)
                            .await
                        {
                            error!("Connection error: {}", e);
                        }
                    });
                }
                Err(e) => {
                    error!("Failed to accept connection: {}", e);
                }
            }
        }
    }
    
    /// Handle incoming HTTP request
    async fn handle_request(
        &self,
        req: Request<Incoming>,
    ) -> Result<Response<Full<Bytes>>> {
        let method = req.method().clone();
        let uri = req.uri().clone();
        let headers = req.headers().clone();
        
        debug!("Intercepted request: {} {}", method, uri);
        
        // Extract host from request
        let host = uri
            .host()
            .or_else(|| {
                headers
                    .get("host")
                    .and_then(|h| h.to_str().ok())
                    .and_then(|h| h.split(':').next())
            })
            .unwrap_or("unknown");
        
        // Log request if enabled
        if self.config.log_requests {
            self.log_request(&method, &uri, &headers);
        }
        
        // Route to mock service
        if let Some(route) = self.routing_table.lookup(host) {
            debug!("Routing {} to mock service at {}", host, route.target);
            
            // Forward to mock service
            let result = self.forward_to_mock(req, &route.target).await;
            
            match result {
                Ok(response) => {
                    if self.config.log_responses {
                        self.log_response(&response);
                    }
                    Ok(response)
                }
                Err(e) => {
                    error!("Failed to forward request: {}", e);
                    Ok(self.error_response(
                        StatusCode::BAD_GATEWAY,
                        "Failed to reach mock service",
                    ))
                }
            }
        } else {
            warn!("No route found for host: {}", host);
            Ok(self.error_response(
                StatusCode::BAD_GATEWAY,
                "No mock service configured for this host",
            ))
        }
    }
    
    /// Forward request to mock service
    async fn forward_to_mock(
        &self,
        mut req: Request<Incoming>,
        target: &str,
    ) -> Result<Response<Full<Bytes>>> {
        // Rewrite URI to target mock service
        let path_and_query = req
            .uri()
            .path_and_query()
            .map(|pq| pq.as_str())
            .unwrap_or("/");
        
        let target_uri = format!("{}{}", target, path_and_query);
        
        // Read request body
        let body_bytes = req
            .into_body()
            .collect()
            .await
            .map_err(|e| EngineError::InterceptionFailed(format!("Body read error: {}", e)))?
            .to_bytes();
        
        // Create new request to mock
        let mock_req = Request::builder()
            .method(req.method())
            .uri(target_uri)
            .body(Full::new(body_bytes))
            .map_err(|e| {
                EngineError::InterceptionFailed(format!("Request build error: {}", e))
            })?;
        
        // Forward to mock service
        let response = self.http_client.request(mock_req).await.map_err(|e| {
            EngineError::InterceptionFailed(format!("Mock request failed: {}", e))
        })?;
        
        // Convert response
        let (parts, body) = response.into_parts();
        let body_bytes = body
            .collect()
            .await
            .map_err(|e| {
                EngineError::InterceptionFailed(format!("Response body error: {}", e))
            })?
            .to_bytes();
        
        let mut response = Response::from_parts(parts, Full::new(body_bytes));
        
        Ok(response)
    }
    
    /// Create error response
    fn error_response(&self, status: StatusCode, message: &str) -> Response<Full<Bytes>> {
        Response::builder()
            .status(status)
            .body(Full::new(Bytes::from(message.to_string())))
            .unwrap()
    }
    
    /// Log HTTP request
    fn log_request(&self, method: &Method, uri: &hyper::Uri, headers: &hyper::HeaderMap) {
        debug!("Request: {} {}", method, uri);
        for (name, value) in headers {
            if let Ok(val_str) = value.to_str() {
                debug!("  {}: {}", name, val_str);
            }
        }
    }
    
    /// Log HTTP response
    fn log_response(&self, response: &Response<Full<Bytes>>) {
        debug!("Response: {}", response.status());
        for (name, value) in response.headers() {
            if let Ok(val_str) = value.to_str() {
                debug!("  {}: {}", name, val_str);
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_config_default() {
        let config = InterceptorConfig::default();
        assert!(config.enable_https);
        assert!(config.log_requests);
    }
    
    #[tokio::test]
    async fn test_interceptor_creation() {
        let config = InterceptorConfig::default();
        let routing_table = Arc::new(RoutingTable::new());
        let tls_handler = Arc::new(TlsHandler::new());
        
        let interceptor = HttpInterceptor::new(config, routing_table, tls_handler);
        assert!(interceptor.config.enable_https);
    }
}
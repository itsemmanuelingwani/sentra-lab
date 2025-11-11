// packages/engine/src/main.rs
//! Sentra Lab Simulation Engine
//!
//! High-performance agent runtime for executing, recording, and replaying
//! AI agent simulations with production parity.

use anyhow::Result;
use sentra_lab_engine::grpc::server::SimulationServer;
use sentra_lab_engine::observability::{init_metrics, init_tracing};
use sentra_lab_engine::runtime::agent_pool::AgentPool;
use sentra_lab_engine::utils::config::EngineConfig;
use std::net::SocketAddr;
use std::sync::Arc;
use tracing::{error, info};

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize observability (tracing, metrics, logging)
    init_tracing()?;
    init_metrics()?;

    info!("Starting Sentra Lab Simulation Engine v{}", env!("CARGO_PKG_VERSION"));

    // Load configuration
    let config = EngineConfig::load()?;
    info!("Configuration loaded: {:?}", config);

    // Create agent pool (32-64 pooled processes)
    let pool_size = config.runtime.pool_size;
    info!("Initializing agent pool with {} processes", pool_size);
    let agent_pool = Arc::new(AgentPool::new(pool_size).await?);

    // Create and start gRPC server
    let addr: SocketAddr = format!("{}:{}", config.server.host, config.server.port)
        .parse()
        .expect("Invalid server address");

    info!("Starting gRPC server on {}", addr);
    let server = SimulationServer::new(agent_pool, config.clone());

    // Graceful shutdown handler
    let shutdown_signal = async {
        tokio::signal::ctrl_c()
            .await
            .expect("Failed to install CTRL+C signal handler");
        info!("Received shutdown signal, cleaning up...");
    };

    // Run server with graceful shutdown
    match tonic::transport::Server::builder()
        .add_service(server.into_service())
        .serve_with_shutdown(addr, shutdown_signal)
        .await
    {
        Ok(_) => {
            info!("Server stopped gracefully");
            Ok(())
        }
        Err(e) => {
            error!("Server error: {}", e);
            Err(e.into())
        }
    }
}
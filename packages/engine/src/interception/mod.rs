// packages/engine/src/interception/mod.rs
//! Request interception layer
//!
//! This module provides transparent interception of all external calls:
//!
//! - **HTTP Interceptor**: MITM proxy for HTTP/HTTPS traffic
//! - **DNS Interceptor**: Custom DNS resolver redirecting to mocks
//! - **Syscall Interceptor**: LD_PRELOAD hooks for system calls (Linux)
//! - **Library Shims**: SDK-specific interception (OpenAI, Stripe, etc.)
//! - **TLS Handler**: TLS termination and re-encryption
//! - **Routing Table**: Domain to mock service mapping
//!
//! # Architecture
//!
//! ```text
//! Agent Code (Unmodified)
//!     │
//!     ├─ HTTP Request → HTTP Interceptor → Mock APIs
//!     ├─ DNS Lookup → DNS Interceptor → 127.0.0.1
//!     ├─ Library Call → Library Shim → Mock Implementation
//!     └─ Syscall → Syscall Hook → Sandboxed Operation
//! ```

pub mod dns_interceptor;
pub mod http_interceptor;
pub mod library_shims;
pub mod routing_table;
pub mod syscall_interceptor;
pub mod tls_handler;

// Re-export commonly used types
pub use dns_interceptor::{DnsInterceptor, DnsMapping};
pub use http_interceptor::{HttpInterceptor, InterceptorConfig};
pub use library_shims::{LibraryShim, ShimConfig};
pub use routing_table::{Route, RoutingTable};
pub use syscall_interceptor::SyscallInterceptor;
pub use tls_handler::{TlsConfig, TlsHandler};
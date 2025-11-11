#!/bin/bash

# Script to create Engine structure
# Run this from the packages/engine directory

set -e

echo "Creating Engine structure..."

# Create directory structure
mkdir -p src/runtime
mkdir -p src/executor
mkdir -p src/interception
mkdir -p src/recording
mkdir -p src/replay
mkdir -p src/state
mkdir -p src/cost
mkdir -p src/grpc
mkdir -p src/observability
mkdir -p src/utils
mkdir -p proto
mkdir -p tests/fixtures/agents
mkdir -p tests/fixtures/scenarios
mkdir -p benches

# Create root src files
touch src/main.rs
touch src/lib.rs

# Create runtime files
touch src/runtime/mod.rs
touch src/runtime/agent_pool.rs
touch src/runtime/agent_runtime.rs
touch src/runtime/process_manager.rs
touch src/runtime/sandbox.rs
touch src/runtime/resource_limiter.rs
touch src/runtime/work_stealing.rs

# Create executor files
touch src/executor/mod.rs
touch src/executor/scenario_executor.rs
touch src/executor/step_executor.rs
touch src/executor/parallel_executor.rs
touch src/executor/scheduler.rs
touch src/executor/assertion_validator.rs

# Create interception files
touch src/interception/mod.rs
touch src/interception/http_interceptor.rs
touch src/interception/dns_interceptor.rs
touch src/interception/syscall_interceptor.rs
touch src/interception/library_shims.rs
touch src/interception/tls_handler.rs
touch src/interception/routing_table.rs

# Create recording files
touch src/recording/mod.rs
touch src/recording/recorder.rs
touch src/recording/event_queue.rs
touch src/recording/compressor.rs
touch src/recording/storage.rs
touch src/recording/exporter.rs
touch src/recording/mmap_writer.rs

# Create replay files
touch src/replay/mod.rs
touch src/replay/replay_engine.rs
touch src/replay/debugger.rs
touch src/replay/time_travel.rs
touch src/replay/diff_engine.rs
touch src/replay/determinism_checker.rs

# Create state files
touch src/state/mod.rs
touch src/state/state_manager.rs
touch src/state/snapshot.rs
touch src/state/delta_log.rs
touch src/state/transaction_log.rs
touch src/state/persistence.rs

# Create cost files
touch src/cost/mod.rs
touch src/cost/estimator.rs
touch src/cost/token_counter.rs
touch src/cost/pricing_db.rs
touch src/cost/report_generator.rs

# Create grpc files
touch src/grpc/mod.rs
touch src/grpc/server.rs
touch src/grpc/handlers.rs
touch src/grpc/streaming.rs

# Create observability files
touch src/observability/mod.rs
touch src/observability/metrics.rs
touch src/observability/tracing.rs
touch src/observability/logging.rs
touch src/observability/profiler.rs

# Create utils files
touch src/utils/mod.rs
touch src/utils/id_generator.rs
touch src/utils/time.rs
touch src/utils/errors.rs
touch src/utils/config.rs

# Create proto files
touch proto/engine.proto
touch proto/events.proto
touch proto/state.proto

# Create test files
touch tests/integration_test.rs
touch tests/performance_test.rs
touch tests/determinism_test.rs

# Create benchmark files
touch benches/recording_bench.rs
touch benches/replay_bench.rs
touch benches/interception_bench.rs

# Create root files
touch Cargo.toml
touch Cargo.lock
touch build.rs
touch rustfmt.toml
touch clippy.toml
touch README.md

echo "✅ Structure created successfully!"
echo ""
echo "Directory structure:"
tree -L 3 || echo "Install 'tree' command to see the structure visualization"
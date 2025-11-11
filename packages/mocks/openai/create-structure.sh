#!/bin/bash

# Script to create OpenAI mock server structure
# Run this from the packages/mocks/openai directory

set -e

echo "Creating OpenAI mock server structure..."

# Create directory structure
mkdir -p cmd/server
mkdir -p internal/server
mkdir -p internal/handlers
mkdir -p internal/models
mkdir -p internal/tokenizer
mkdir -p internal/ratelimit
mkdir -p internal/latency
mkdir -p internal/behavior
mkdir -p internal/generator
mkdir -p internal/fixtures
mkdir -p internal/pricing
mkdir -p internal/store
mkdir -p internal/metrics
mkdir -p pkg/client
mkdir -p fixtures/responses/chat
mkdir -p fixtures/responses/embeddings
mkdir -p fixtures/responses/images
mkdir -p fixtures/patterns
mkdir -p fixtures/errors
mkdir -p config
mkdir -p test
mkdir -p scripts

# Create entry point
touch cmd/server/main.go

# Create server files
touch internal/server/server.go
touch internal/server/middleware.go
touch internal/server/router.go
touch internal/server/context.go

# Create handler files
touch internal/handlers/chat_completions.go
touch internal/handlers/completions.go
touch internal/handlers/embeddings.go
touch internal/handlers/images.go
touch internal/handlers/models.go
touch internal/handlers/streaming.go
touch internal/handlers/errors.go

# Create model files
touch internal/models/request.go
touch internal/models/response.go
touch internal/models/model_config.go
touch internal/models/error.go

# Create tokenizer files
touch internal/tokenizer/tokenizer.go
touch internal/tokenizer/cache.go
touch internal/tokenizer/estimator.go
touch internal/tokenizer/counter.go

# Create rate limit files
touch internal/ratelimit/token_bucket.go
touch internal/ratelimit/limiter.go
touch internal/ratelimit/storage.go
touch internal/ratelimit/headers.go
touch internal/ratelimit/tier.go

# Create latency files
touch internal/latency/simulator.go
touch internal/latency/profiles.go
touch internal/latency/jitter.go
touch internal/latency/streaming.go

# Create behavior files
touch internal/behavior/error_injector.go
touch internal/behavior/cache_simulator.go
touch internal/behavior/load_simulator.go
touch internal/behavior/network_simulator.go

# Create generator files
touch internal/generator/chat.go
touch internal/generator/completion.go
touch internal/generator/embedding.go
touch internal/generator/image.go
touch internal/generator/streaming.go

# Create fixture files
touch internal/fixtures/loader.go
touch internal/fixtures/matcher.go
touch internal/fixtures/store.go
touch internal/fixtures/validator.go

# Create pricing files
touch internal/pricing/calculator.go
touch internal/pricing/pricing_db.go
touch internal/pricing/tracker.go
touch internal/pricing/headers.go

# Create store files
touch internal/store/memory.go
touch internal/store/redis.go
touch internal/store/interface.go

# Create metrics files
touch internal/metrics/prometheus.go
touch internal/metrics/logger.go
touch internal/metrics/tracing.go

# Create client file
touch pkg/client/client.go

# Create fixture data files
touch fixtures/responses/chat/generic.yaml
touch fixtures/responses/chat/code.yaml
touch fixtures/responses/chat/creative.yaml
touch fixtures/responses/chat/technical.yaml
touch fixtures/responses/embeddings/default.yaml
touch fixtures/responses/images/urls.yaml
touch fixtures/patterns/greetings.yaml
touch fixtures/patterns/questions.yaml
touch fixtures/patterns/code_requests.yaml
touch fixtures/errors/rate_limit.yaml
touch fixtures/errors/server_error.yaml
touch fixtures/errors/timeout.yaml

# Create config files
touch config/default.yaml
touch config/production.yaml
touch config/fast.yaml
touch config/models.yaml

# Create test files
touch test/integration_test.go
touch test/latency_test.go
touch test/ratelimit_test.go
touch test/fixtures_test.go

# Create script files
touch scripts/generate_fixtures.go
touch scripts/benchmark.sh
touch scripts/validate_parity.sh

# Make scripts executable
chmod +x scripts/benchmark.sh
chmod +x scripts/validate_parity.sh

# Create root files
touch go.mod
touch go.sum
touch Makefile
touch Dockerfile
touch README.md

echo "✅ Structure created successfully!"
echo ""
echo "Directory structure:"
tree -L 3 || echo "Install 'tree' command to see the structure visualization"
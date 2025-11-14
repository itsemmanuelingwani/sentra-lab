# SENTRA LAB - Makefile
# Production-grade build system for multi-language monorepo

.PHONY: help setup build test clean docker-up docker-down dev release lint fmt check install

# Default target
.DEFAULT_GOAL := help

# Colors for output
GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
NC     := \033[0m # No Color

# Project metadata
VERSION := $(shell cat VERSION 2>/dev/null || echo "1.0.0")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Directories
CLI_DIR         := packages/cli
ENGINE_DIR      := packages/engine
MOCKS_DIR       := packages/mocks
SDK_PYTHON_DIR  := packages/sdk-python
SDK_JS_DIR      := packages/sdk-javascript
SDK_GO_DIR      := packages/sdk-go
CLOUD_DIR       := cloud

# Build outputs
BUILD_DIR := build
DIST_DIR  := dist

# Docker Compose
DOCKER_COMPOSE := docker-compose.yml
DOCKER_COMPOSE_DEV := docker-compose.dev.yml

##@ General

help: ## Display this help message
	@echo "$(GREEN)╔═══════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(GREEN)║                    SENTRA LAB BUILD SYSTEM                    ║$(NC)"
	@echo "$(GREEN)╚═══════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "Version: $(VERSION) | Commit: $(COMMIT)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Setup

setup: ## First-time setup (install dependencies)
	@echo "$(GREEN)🔧 Setting up Sentra Lab development environment...$(NC)"
	@$(MAKE) check-deps
	@$(MAKE) install-rust
	@$(MAKE) install-go-deps
	@$(MAKE) install-node-deps
	@$(MAKE) install-python-deps
	@echo "$(GREEN)✅ Setup complete!$(NC)"

check-deps: ## Check if required tools are installed
	@echo "$(YELLOW)Checking dependencies...$(NC)"
	@command -v docker >/dev/null 2>&1 || { echo "$(RED)❌ Docker not found. Install from https://docker.com$(NC)"; exit 1; }
	@command -v docker-compose >/dev/null 2>&1 || { echo "$(RED)❌ Docker Compose not found$(NC)"; exit 1; }
	@command -v cargo >/dev/null 2>&1 || { echo "$(RED)❌ Rust/Cargo not found. Install from https://rustup.rs$(NC)"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo "$(RED)❌ Go not found. Install from https://go.dev$(NC)"; exit 1; }
	@command -v node >/dev/null 2>&1 || { echo "$(RED)❌ Node.js not found. Install from https://nodejs.org$(NC)"; exit 1; }
	@command -v python3 >/dev/null 2>&1 || { echo "$(RED)❌ Python3 not found$(NC)"; exit 1; }
	@echo "$(GREEN)✅ All dependencies found$(NC)"

install-rust: ## Install Rust dependencies
	@echo "$(YELLOW)Installing Rust dependencies...$(NC)"
	@cd $(ENGINE_DIR) && cargo fetch
	@echo "$(GREEN)✅ Rust dependencies installed$(NC)"

install-go-deps: ## Install Go dependencies
	@echo "$(YELLOW)Installing Go dependencies...$(NC)"
	@cd $(CLI_DIR) && go mod download
	@cd $(SDK_GO_DIR) && go mod download
	@cd $(MOCKS_DIR)/openai && go mod download
	@cd $(MOCKS_DIR)/coreledger && go mod download
	@cd $(MOCKS_DIR)/aws && go mod download
	@cd $(MOCKS_DIR)/databases && go mod download
	@echo "$(GREEN)✅ Go dependencies installed$(NC)"

install-node-deps: ## Install Node.js dependencies
	@echo "$(YELLOW)Installing Node.js dependencies...$(NC)"
	@cd $(SDK_JS_DIR) && npm install
	@cd $(MOCKS_DIR)/stripe && npm install
	@echo "$(GREEN)✅ Node.js dependencies installed$(NC)"

install-python-deps: ## Install Python dependencies
	@echo "$(YELLOW)Installing Python dependencies...$(NC)"
	@cd $(SDK_PYTHON_DIR) && pip install -e ".[dev]"
	@echo "$(GREEN)✅ Python dependencies installed$(NC)"

##@ Build

build: ## Build all packages
	@echo "$(GREEN)🔨 Building all packages...$(NC)"
	@$(MAKE) build-cli
	@$(MAKE) build-engine
	@$(MAKE) build-mocks
	@$(MAKE) build-sdks
	@echo "$(GREEN)✅ All packages built successfully!$(NC)"

build-cli: ## Build CLI (Go)
	@echo "$(YELLOW)Building CLI...$(NC)"
	@mkdir -p $(BUILD_DIR)/cli
	@cd $(CLI_DIR) && CGO_ENABLED=0 go build \
		-ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" \
		-o ../../$(BUILD_DIR)/cli/sentra-lab \
		./cmd/sentra-lab
	@echo "$(GREEN)✅ CLI built: $(BUILD_DIR)/cli/sentra-lab$(NC)"

build-engine: ## Build simulation engine (Rust)
	@echo "$(YELLOW)Building simulation engine...$(NC)"
	@mkdir -p $(BUILD_DIR)/engine
	@cd $(ENGINE_DIR) && cargo build --release
	@cp $(ENGINE_DIR)/target/release/sentra-engine $(BUILD_DIR)/engine/
	@echo "$(GREEN)✅ Engine built: $(BUILD_DIR)/engine/sentra-engine$(NC)"

build-mocks: ## Build mock services
	@echo "$(YELLOW)Building mock services...$(NC)"
	@$(MAKE) build-mock-openai
	@$(MAKE) build-mock-stripe
	@$(MAKE) build-mock-coreledger
	@$(MAKE) build-mock-aws
	@echo "$(GREEN)✅ All mocks built$(NC)"

build-mock-openai: ## Build OpenAI mock (Go)
	@mkdir -p $(BUILD_DIR)/mocks/openai
	@cd $(MOCKS_DIR)/openai && go build -o ../../../$(BUILD_DIR)/mocks/openai/mock-openai ./cmd/server

build-mock-stripe: ## Build Stripe mock (Node.js)
	@mkdir -p $(BUILD_DIR)/mocks/stripe
	@cd $(MOCKS_DIR)/stripe && npm run build
	@cp -r $(MOCKS_DIR)/stripe/dist $(BUILD_DIR)/mocks/stripe/

build-mock-coreledger: ## Build CoreLedger mock (Go)
	@mkdir -p $(BUILD_DIR)/mocks/coreledger
	@cd $(MOCKS_DIR)/coreledger && go build -o ../../../$(BUILD_DIR)/mocks/coreledger/mock-coreledger ./cmd/server

build-mock-aws: ## Build AWS mock (Go)
	@mkdir -p $(BUILD_DIR)/mocks/aws
	@cd $(MOCKS_DIR)/aws && go build -o ../../../$(BUILD_DIR)/mocks/aws/mock-aws ./cmd/server

build-sdks: ## Build all SDKs
	@echo "$(YELLOW)Building SDKs...$(NC)"
	@cd $(SDK_PYTHON_DIR) && python setup.py build
	@cd $(SDK_JS_DIR) && npm run build
	@cd $(SDK_GO_DIR) && go build ./...
	@echo "$(GREEN)✅ SDKs built$(NC)"

##@ Testing

test: ## Run all tests
	@echo "$(GREEN)🧪 Running all tests...$(NC)"
	@$(MAKE) test-cli
	@$(MAKE) test-engine
	@$(MAKE) test-mocks
	@$(MAKE) test-sdks
	@$(MAKE) test-integration
	@echo "$(GREEN)✅ All tests passed!$(NC)"

test-cli: ## Test CLI (Go)
	@echo "$(YELLOW)Testing CLI...$(NC)"
	@cd $(CLI_DIR) && go test -v -race -cover ./...

test-engine: ## Test simulation engine (Rust)
	@echo "$(YELLOW)Testing simulation engine...$(NC)"
	@cd $(ENGINE_DIR) && cargo test --release

test-mocks: ## Test mock services
	@echo "$(YELLOW)Testing mock services...$(NC)"
	@cd $(MOCKS_DIR)/openai && go test -v ./...
	@cd $(MOCKS_DIR)/stripe && npm test
	@cd $(MOCKS_DIR)/coreledger && go test -v ./...
	@cd $(MOCKS_DIR)/aws && go test -v ./...

test-sdks: ## Test all SDKs
	@echo "$(YELLOW)Testing SDKs...$(NC)"
	@cd $(SDK_PYTHON_DIR) && pytest -v
	@cd $(SDK_JS_DIR) && npm test
	@cd $(SDK_GO_DIR) && go test -v ./...

test-integration: ## Run integration tests
	@echo "$(YELLOW)Running integration tests...$(NC)"
	@cd tests/integration && go test -v ./...

test-e2e: ## Run end-to-end tests
	@echo "$(YELLOW)Running E2E tests...$(NC)"
	@cd tests/e2e && go test -v ./...

test-coverage: ## Generate test coverage report
	@echo "$(YELLOW)Generating coverage report...$(NC)"
	@mkdir -p coverage
	@cd $(CLI_DIR) && go test -coverprofile=../../coverage/cli.out ./...
	@cd $(ENGINE_DIR) && cargo tarpaulin --out Html --output-dir ../../coverage/engine
	@echo "$(GREEN)✅ Coverage reports in coverage/$(NC)"

##@ Docker

docker-build: ## Build all Docker images
	@echo "$(GREEN)🐳 Building Docker images...$(NC)"
	docker build -f infrastructure/docker/Dockerfile.engine -t sentra/lab-engine:$(VERSION) .
	docker build -f infrastructure/docker/Dockerfile.cli -t sentra/lab-cli:$(VERSION) .
	docker build -f infrastructure/docker/Dockerfile.mock-openai -t sentra/mock-openai:$(VERSION) .
	docker build -f infrastructure/docker/Dockerfile.mock-stripe -t sentra/mock-stripe:$(VERSION) .
	@echo "$(GREEN)✅ Docker images built$(NC)"

docker-up: ## Start all services with Docker Compose
	@echo "$(GREEN)🚀 Starting Sentra Lab services...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)✅ Services started!$(NC)"
	@echo ""
	@echo "$(YELLOW)Simulation Engine:$(NC) localhost:50051"
	@echo "$(YELLOW)Mock OpenAI:$(NC)       localhost:8080"
	@echo "$(YELLOW)Mock Stripe:$(NC)       localhost:8081"
	@echo "$(YELLOW)Mock CoreLedger:$(NC)   localhost:8082"

docker-down: ## Stop all services
	@echo "$(YELLOW)Stopping services...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE) down
	@echo "$(GREEN)✅ Services stopped$(NC)"

docker-logs: ## View Docker logs
	docker-compose -f $(DOCKER_COMPOSE) logs -f

docker-clean: ## Clean Docker resources (volumes, networks)
	@echo "$(YELLOW)Cleaning Docker resources...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE) down -v --remove-orphans
	@echo "$(GREEN)✅ Docker resources cleaned$(NC)"

##@ Development

dev: ## Start development environment
	@echo "$(GREEN)🚀 Starting development environment...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE_DEV) up -d
	@echo "$(GREEN)✅ Development environment running$(NC)"

dev-cli: ## Run CLI in development mode
	@cd $(CLI_DIR) && go run ./cmd/sentra-lab $(ARGS)

dev-engine: ## Run engine in development mode
	@cd $(ENGINE_DIR) && cargo run

watch-cli: ## Watch CLI for changes and rebuild
	@cd $(CLI_DIR) && go run github.com/cosmtrek/air@latest

watch-engine: ## Watch engine for changes and rebuild
	@cd $(ENGINE_DIR) && cargo watch -x run

##@ Code Quality

lint: ## Run linters
	@echo "$(YELLOW)Running linters...$(NC)"
	@$(MAKE) lint-go
	@$(MAKE) lint-rust
	@$(MAKE) lint-js
	@$(MAKE) lint-python
	@echo "$(GREEN)✅ Linting complete$(NC)"

lint-go: ## Lint Go code
	@cd $(CLI_DIR) && golangci-lint run ./...
	@cd $(SDK_GO_DIR) && golangci-lint run ./...

lint-rust: ## Lint Rust code
	@cd $(ENGINE_DIR) && cargo clippy -- -D warnings

lint-js: ## Lint JavaScript/TypeScript
	@cd $(SDK_JS_DIR) && npm run lint
	@cd $(MOCKS_DIR)/stripe && npm run lint

lint-python: ## Lint Python code
	@cd $(SDK_PYTHON_DIR) && flake8 . && mypy .

fmt: ## Format all code
	@echo "$(YELLOW)Formatting code...$(NC)"
	@cd $(CLI_DIR) && go fmt ./...
	@cd $(ENGINE_DIR) && cargo fmt
	@cd $(SDK_JS_DIR) && npm run format
	@cd $(SDK_PYTHON_DIR) && black . && isort .
	@echo "$(GREEN)✅ Code formatted$(NC)"

check: ## Run all checks (lint + test)
	@$(MAKE) lint
	@$(MAKE) test

##@ Installation

install: build ## Install CLI locally
	@echo "$(GREEN)📦 Installing sentra-lab CLI...$(NC)"
	@cp $(BUILD_DIR)/cli/sentra-lab /usr/local/bin/
	@chmod +x /usr/local/bin/sentra-lab
	@echo "$(GREEN)✅ Installed to /usr/local/bin/sentra-lab$(NC)"

uninstall: ## Uninstall CLI
	@echo "$(YELLOW)Uninstalling sentra-lab CLI...$(NC)"
	@rm -f /usr/local/bin/sentra-lab
	@echo "$(GREEN)✅ Uninstalled$(NC)"

##@ Release

release: ## Create release build
	@echo "$(GREEN)📦 Creating release v$(VERSION)...$(NC)"
	@$(MAKE) clean
	@$(MAKE) test
	@$(MAKE) build
	@$(MAKE) package
	@echo "$(GREEN)✅ Release v$(VERSION) created in $(DIST_DIR)/$(NC)"

package: ## Package release artifacts
	@echo "$(YELLOW)Packaging release...$(NC)"
	@mkdir -p $(DIST_DIR)
	@tar -czf $(DIST_DIR)/sentra-lab-$(VERSION)-linux-amd64.tar.gz -C $(BUILD_DIR)/cli sentra-lab
	@tar -czf $(DIST_DIR)/sentra-lab-$(VERSION)-darwin-amd64.tar.gz -C $(BUILD_DIR)/cli sentra-lab
	@cd $(DIST_DIR) && sha256sum *.tar.gz > checksums.txt
	@echo "$(GREEN)✅ Packages created$(NC)"

publish: ## Publish release (Docker Hub, npm, PyPI)
	@echo "$(GREEN)📦 Publishing release v$(VERSION)...$(NC)"
	@$(MAKE) publish-docker
	@$(MAKE) publish-npm
	@$(MAKE) publish-pypi
	@echo "$(GREEN)✅ Release published$(NC)"

publish-docker: ## Push Docker images to registry
	docker push sentra/lab-engine:$(VERSION)
	docker push sentra/lab-cli:$(VERSION)
	docker push sentra/mock-openai:$(VERSION)
	docker push sentra/mock-stripe:$(VERSION)

publish-npm: ## Publish JavaScript SDK to npm
	@cd $(SDK_JS_DIR) && npm publish

publish-pypi: ## Publish Python SDK to PyPI
	@cd $(SDK_PYTHON_DIR) && python setup.py sdist bdist_wheel && twine upload dist/*

##@ Cleanup

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@cd $(ENGINE_DIR) && cargo clean
	@cd $(SDK_JS_DIR) && rm -rf dist node_modules
	@cd $(SDK_PYTHON_DIR) && rm -rf build dist *.egg-info
	@echo "$(GREEN)✅ Clean complete$(NC)"

clean-all: clean docker-clean ## Clean everything (including Docker)
	@echo "$(GREEN)✅ All cleaned$(NC)"

##@ Documentation

docs: ## Generate documentation
	@echo "$(YELLOW)Generating documentation...$(NC)"
	@cd docs && make html
	@echo "$(GREEN)✅ Documentation generated in docs/_build/html$(NC)"

docs-serve: ## Serve documentation locally
	@cd docs && make serve

##@ Benchmarks

bench: ## Run benchmarks
	@echo "$(YELLOW)Running benchmarks...$(NC)"
	@cd $(ENGINE_DIR) && cargo bench
	@cd $(CLI_DIR) && go test -bench=. -benchmem ./...

##@ Database

db-migrate: ## Run database migrations
	@echo "$(YELLOW)Running migrations...$(NC)"
	@cd $(CLOUD_DIR)/api-server && go run ./cmd/migrate

db-seed: ## Seed database with test data
	@echo "$(YELLOW)Seeding database...$(NC)"
	@cd $(CLOUD_DIR)/api-server && go run ./cmd/seed

##@ Utilities

version: ## Show version
	@echo "Sentra Lab v$(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Built: $(DATE)"

deps-update: ## Update dependencies
	@echo "$(YELLOW)Updating dependencies...$(NC)"
	@cd $(CLI_DIR) && go get -u ./... && go mod tidy
	@cd $(ENGINE_DIR) && cargo update
	@cd $(SDK_JS_DIR) && npm update
	@cd $(SDK_PYTHON_DIR) && pip install --upgrade -r requirements.txt
	@echo "$(GREEN)✅ Dependencies updated$(NC)"

security-audit: ## Run security audits
	@echo "$(YELLOW)Running security audits...$(NC)"
	@cd $(CLI_DIR) && gosec ./...
	@cd $(ENGINE_DIR) && cargo audit
	@cd $(SDK_JS_DIR) && npm audit
	@cd $(SDK_PYTHON_DIR) && safety check
	@echo "$(GREEN)✅ Security audit complete$(NC)"

proto-gen: ## Generate protobuf code
	@echo "$(YELLOW)Generating protobuf code...$(NC)"
	@cd $(CLI_DIR)/api/proto && protoc --go_out=. --go-grpc_out=. *.proto
	@cd $(ENGINE_DIR)/proto && protoc --rust_out=. *.proto
	@echo "$(GREEN)✅ Protobuf code generated$(NC)"

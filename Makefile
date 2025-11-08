.PHONY: help build test clean install

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all packages
	@echo "Building all packages..."
	# Add build commands here

test: ## Run tests
	@echo "Running tests..."
	# Add test commands here

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	# Add clean commands here

install: ## Install dependencies
	@echo "Installing dependencies..."
	# Add install commands here

dev: ## Start development environment
	docker-compose up

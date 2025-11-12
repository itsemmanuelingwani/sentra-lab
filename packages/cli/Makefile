.PHONY: build install test clean fmt lint run-test help

VERSION := 1.0.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)"

BIN_DIR := bin
BIN_NAME := sentra

help:
	@echo "Sentra Lab CLI - Build Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build        Build the CLI binary"
	@echo "  make install      Install CLI to /usr/local/bin"
	@echo "  make test         Run all tests"
	@echo "  make clean        Remove build artifacts"
	@echo "  make fmt          Format code"
	@echo "  make lint         Run linters"
	@echo "  make run-test     Run CLI test command"

build:
	@echo "Building Sentra Lab CLI..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BIN_NAME) ./cmd/sentra-lab
	@echo "✓ Build complete: $(BIN_DIR)/$(BIN_NAME)"

build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BIN_NAME)-darwin-amd64 ./cmd/sentra-lab
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BIN_NAME)-darwin-arm64 ./cmd/sentra-lab
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BIN_NAME)-linux-amd64 ./cmd/sentra-lab
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BIN_NAME)-linux-arm64 ./cmd/sentra-lab
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BIN_NAME)-windows-amd64.exe ./cmd/sentra-lab
	@echo "✓ Multi-platform build complete"

install: build
	@echo "Installing Sentra Lab CLI..."
	sudo cp $(BIN_DIR)/$(BIN_NAME) /usr/local/bin/
	@echo "✓ Installed to /usr/local/bin/$(BIN_NAME)"

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests complete"

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Format complete"

lint:
	@echo "Running linters..."
	golangci-lint run ./...
	@echo "✓ Lint complete"

vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "✓ Vet complete"

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies updated"

run-test:
	@echo "Running Sentra Lab test..."
	$(BIN_DIR)/$(BIN_NAME) lab test

run-init:
	@echo "Running Sentra Lab init..."
	$(BIN_DIR)/$(BIN_NAME) lab init test-project

dev:
	@echo "Starting development mode..."
	go run ./cmd/sentra-lab lab start

docker-build:
	@echo "Building Docker image..."
	docker build -t sentra/lab-cli:$(VERSION) -t sentra/lab-cli:latest .
	@echo "✓ Docker image built"

docker-push:
	@echo "Pushing Docker image..."
	docker push sentra/lab-cli:$(VERSION)
	docker push sentra/lab-cli:latest
	@echo "✓ Docker image pushed"

release: clean build-all
	@echo "Creating release archive..."
	@mkdir -p release
	tar -czf release/sentra-lab-$(VERSION)-darwin-amd64.tar.gz -C $(BIN_DIR) $(BIN_NAME)-darwin-amd64
	tar -czf release/sentra-lab-$(VERSION)-darwin-arm64.tar.gz -C $(BIN_DIR) $(BIN_NAME)-darwin-arm64
	tar -czf release/sentra-lab-$(VERSION)-linux-amd64.tar.gz -C $(BIN_DIR) $(BIN_NAME)-linux-amd64
	tar -czf release/sentra-lab-$(VERSION)-linux-arm64.tar.gz -C $(BIN_DIR) $(BIN_NAME)-linux-arm64
	zip -j release/sentra-lab-$(VERSION)-windows-amd64.zip $(BIN_DIR)/$(BIN_NAME)-windows-amd64.exe
	@echo "✓ Release archives created in release/"

check: fmt vet lint test
	@echo "✓ All checks passed"
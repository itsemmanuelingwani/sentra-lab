#!/bin/bash

# Sentra Lab CLI Package Setup Script
# Run this script from the sentra-lab root directory

set -e  # Exit on error

echo "🚀 Setting up Sentra Lab CLI package structure..."

# Navigate to packages/cli
cd packages/cli

# Create cmd structure
echo "📦 Creating cmd structure..."
mkdir -p cmd/sentra-lab
mkdir -p cmd/init
mkdir -p cmd/start
mkdir -p cmd/test
mkdir -p cmd/replay
mkdir -p cmd/config
mkdir -p cmd/cloud

# Create internal structure
echo "🔧 Creating internal structure..."
mkdir -p internal/docker
mkdir -p internal/grpc
mkdir -p internal/config
mkdir -p internal/ui
mkdir -p internal/reporter
mkdir -p internal/utils

# Create pkg structure
echo "📚 Creating pkg structure..."
mkdir -p pkg/client

# Create api structure
echo "🌐 Creating api structure..."
mkdir -p api/proto

# Create templates directory
echo "📄 Creating templates directory..."
mkdir -p templates

echo "✍️  Creating empty files..."

# cmd/sentra-lab files
touch cmd/sentra-lab/main.go

# cmd/init files
touch cmd/init/init.go
touch cmd/init/templates.go
touch cmd/init/scaffolding.go

# cmd/start files
touch cmd/start/start.go
touch cmd/start/docker_manager.go
touch cmd/start/health_checker.go

# cmd/test files
touch cmd/test/test.go
touch cmd/test/runner.go
touch cmd/test/parallel.go
touch cmd/test/reporter.go

# cmd/replay files
touch cmd/replay/replay.go
touch cmd/replay/debugger.go
touch cmd/replay/exporter.go

# cmd/config files
touch cmd/config/config.go
touch cmd/config/validator.go
touch cmd/config/migrator.go

# cmd/cloud files
touch cmd/cloud/login.go
touch cmd/cloud/sync.go
touch cmd/cloud/push.go

# internal/docker files
touch internal/docker/client.go
touch internal/docker/compose.go
touch internal/docker/container.go
touch internal/docker/network.go

# internal/grpc files
touch internal/grpc/client.go
touch internal/grpc/engine_client.go

# internal/config files
touch internal/config/loader.go
touch internal/config/parser.go
touch internal/config/schema.go

# internal/ui files
touch internal/ui/tui.go
touch internal/ui/progress.go
touch internal/ui/table.go
touch internal/ui/spinner.go

# internal/reporter files
touch internal/reporter/console.go
touch internal/reporter/junit.go
touch internal/reporter/json.go
touch internal/reporter/markdown.go

# internal/utils files
touch internal/utils/logger.go
touch internal/utils/crypto.go
touch internal/utils/version.go

# pkg/client files
touch pkg/client/lab_client.go

# api/proto files
touch api/proto/engine.proto
touch api/proto/cloud.proto

# templates files
touch templates/lab.yaml.tmpl
touch templates/scenario.yaml.tmpl
touch templates/mocks.yaml.tmpl

# Root level files
touch go.mod
touch go.sum
touch Makefile
touch README.md

echo ""
echo "✅ CLI package structure created successfully!"
echo ""
echo "📁 Directory structure:"
tree -L 3 -F 2>/dev/null || find . -type d | head -20
echo ""
echo "📝 Files created:"
find . -type f -name "*.go" -o -name "*.proto" -o -name "*.tmpl" -o -name "Makefile" -o -name "README.md" | wc -l | xargs echo "Total files:"
echo ""
echo "🎯 Next steps:"
echo "   1. Initialize Go module: go mod init github.com/your-username/sentra-lab/packages/cli"
echo "   2. Add dependencies to go.mod"
echo "   3. Start implementing the CLI commands"
echo "   4. Add proto definitions for gRPC"
echo ""
echo "🚀 Happy coding!"
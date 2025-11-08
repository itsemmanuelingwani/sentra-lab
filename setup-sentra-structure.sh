#!/bin/bash

# Sentra Lab Project Structure Setup Script
# Run this script inside your sentra-lab directory

set -e  # Exit on error

echo "🚀 Setting up Sentra Lab project structure..."

# Create packages directory structure
echo "📦 Creating packages structure..."
mkdir -p packages/{cli,engine,mocks,sdk-python,sdk-javascript,sdk-go,shared}

# Create cloud directory structure
echo "☁️  Creating cloud platform structure..."
mkdir -p cloud/{api-server,web-dashboard,worker}

# Create root-level directories
echo "📂 Creating root directories..."
mkdir -p docs examples fixtures infrastructure scripts tests .github/workflows

# Create placeholder README files for main directories
echo "📝 Creating README files..."

cat > packages/README.md << 'EOF'
# Packages

This directory contains all the core packages for Sentra Lab.

## Structure
- **cli/** - Command-line interface (Go)
- **engine/** - Simulation engine (Rust)
- **mocks/** - Mock services for testing
- **sdk-python/** - Python SDK
- **sdk-javascript/** - JavaScript/TypeScript SDK
- **sdk-go/** - Go SDK
- **shared/** - Shared libraries and utilities
EOF

cat > cloud/README.md << 'EOF'
# Cloud Platform (Closed Source)

This directory contains the cloud platform components.

## Structure
- **api-server/** - Backend API server
- **web-dashboard/** - Web-based dashboard UI
- **worker/** - Background job workers
EOF

cat > docs/README.md << 'EOF'
# Documentation

Welcome to Sentra Lab documentation.

## Contents
- Getting Started
- API Reference
- Tutorials
- Architecture
EOF

cat > examples/README.md << 'EOF'
# Examples

Example agents and scenarios for Sentra Lab.

Browse through the examples to learn how to build with Sentra Lab.
EOF

cat > fixtures/README.md << 'EOF'
# Fixtures

Mock fixtures and test data for development and testing.
EOF

cat > infrastructure/README.md << 'EOF'
# Infrastructure

Docker, Kubernetes, and other infrastructure configuration files.
EOF

cat > scripts/README.md << 'EOF'
# Scripts

Build and utility scripts for the project.
EOF

cat > tests/README.md << 'EOF'
# Tests

Integration and end-to-end tests.
EOF

# Create docker-compose.yml
echo "🐳 Creating docker-compose.yml..."
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  # Add your services here
  # Example:
  # api:
  #   build: ./cloud/api-server
  #   ports:
  #     - "8080:8080"
  
  # mock-service:
  #   build: ./packages/mocks
  #   ports:
  #     - "9090:9090"

networks:
  sentra-network:
    driver: bridge
EOF

# Create Makefile
echo "⚙️  Creating Makefile..."
cat > Makefile << 'EOF'
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
EOF

# Create LICENSE (MIT as placeholder)
echo "📄 Creating LICENSE..."
cat > LICENSE << 'EOF'
MIT License

Copyright (c) 2025 Sentra Lab

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
EOF

# Create README.md
echo "📖 Creating README.md..."
cat > README.md << 'EOF'
# 🧪 Sentra Lab

> AI Agent Testing & Simulation Framework

## Overview

Sentra Lab is a comprehensive framework for testing and simulating AI agents in controlled environments.

## Project Structure

```
sentra-lab/
├── 📂 packages/          # Core packages (open-source)
├── 📂 cloud/             # Cloud platform (closed-source)
├── 📂 docs/              # Documentation
├── 📂 examples/          # Example agents & scenarios
├── 📂 fixtures/          # Test data
├── 📂 infrastructure/    # Infrastructure configs
├── 📂 scripts/           # Build scripts
└── 📂 tests/             # Integration tests
```

## Quick Start

```bash
# Install dependencies
make install

# Start development environment
make dev

# Run tests
make test
```

## Documentation

See the [docs](./docs) directory for detailed documentation.

## Contributing

Please read [CONTRIBUTING.md](./CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.
EOF

# Create CONTRIBUTING.md
echo "🤝 Creating CONTRIBUTING.md..."
cat > CONTRIBUTING.md << 'EOF'
# Contributing to Sentra Lab

Thank you for your interest in contributing to Sentra Lab!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/sentra-lab.git`
3. Create a branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `make test`
6. Commit your changes: `git commit -am 'Add some feature'`
7. Push to the branch: `git push origin feature/your-feature-name`
8. Submit a pull request

## Development Guidelines

- Write clear, descriptive commit messages
- Add tests for new features
- Update documentation as needed
- Follow the existing code style
- Keep pull requests focused and atomic

## Code of Conduct

Please note that this project is released with a [Code of Conduct](./CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms.

## Questions?

Feel free to open an issue for any questions or concerns.
EOF

# Create CODE_OF_CONDUCT.md
echo "📜 Creating CODE_OF_CONDUCT.md..."
cat > CODE_OF_CONDUCT.md << 'EOF'
# Code of Conduct

## Our Pledge

We pledge to make participation in our project a harassment-free experience for everyone, regardless of age, body size, disability, ethnicity, gender identity and expression, level of experience, nationality, personal appearance, race, religion, or sexual identity and orientation.

## Our Standards

Examples of behavior that contributes to creating a positive environment include:

- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

Examples of unacceptable behavior include:

- The use of sexualized language or imagery
- Trolling, insulting/derogatory comments, and personal or political attacks
- Public or private harassment
- Publishing others' private information without explicit permission
- Other conduct which could reasonably be considered inappropriate in a professional setting

## Enforcement

Instances of abusive, harassing, or otherwise unacceptable behavior may be reported by contacting the project team. All complaints will be reviewed and investigated promptly and fairly.

## Attribution

This Code of Conduct is adapted from the Contributor Covenant, version 2.0.
EOF

# Create GitHub Actions workflow
echo "⚡ Creating GitHub Actions workflow..."
mkdir -p .github/workflows
cat > .github/workflows/ci.yml << 'EOF'
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Run tests
      run: make test
    
    - name: Build
      run: make build
EOF

# Create .gitignore
echo "🙈 Creating .gitignore..."
cat > .gitignore << 'EOF'
# Dependencies
node_modules/
vendor/
target/
__pycache__/
*.pyc
*.pyo
*.egg-info/
dist/
build/

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local
*.local

# Logs
*.log
logs/

# Build artifacts
*.exe
*.dll
*.so
*.dylib
*.test
*.out

# Docker
docker-compose.override.yml
EOF

echo ""
echo "✅ Sentra Lab project structure created successfully!"
echo ""
echo "📁 Directory structure:"
tree -L 2 -F 2>/dev/null || find . -maxdepth 2 -type d | sed 's|[^/]*/| |g'
echo ""
echo "🎯 Next steps:"
echo "   1. Review the generated files"
echo "   2. Initialize git (if not already done): git init"
echo "   3. Make initial commit: git add . && git commit -m 'Initial project structure'"
echo "   4. Start building your packages!"
echo ""
echo "🚀 Happy coding!"
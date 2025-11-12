# Sentra Lab CLI

**Local-first simulation platform for AI agents**

Test AI agents without API costs, production risks, or infrastructure complexity.

## Features

- 🚀 **Local-First** - Runs entirely on your laptop, no internet required
- 💰 **Zero-Cost Testing** - Mock OpenAI, Stripe, AWS, and more
- 🔄 **Time-Travel Debugging** - Replay any execution step-by-step
- 📊 **Cost Estimation** - Predict production costs before deploying
- 🧪 **Scenario-Driven** - Define complex test scenarios in YAML
- 🔧 **CI/CD Ready** - Integrate with GitHub Actions, GitLab CI
- ☁️ **Cloud-Hybrid** - Optional team collaboration features

## Quick Start

### Installation

```bash
# macOS (Homebrew)
brew install sentra/tap/lab

# Linux/macOS (curl)
curl -fsSL https://lab.sentra.dev/install.sh | sh

# Windows (PowerShell)
iwr -useb https://lab.sentra.dev/install.ps1 | iex

# From source
git clone https://github.com/sentra-lab/cli
cd cli
make install
```

### Initialize Project

```bash
# Create a new project
sentra lab init my-agent

cd my-agent

# Start mock services
sentra lab start

# Run test scenarios
sentra lab test
```

## Usage

### Basic Commands

```bash
# Initialize new project
sentra lab init <name>

# Start mock services
sentra lab start

# Run test scenarios
sentra lab test

# Replay failed tests
sentra lab replay

# Stop services
sentra lab stop

# View logs
sentra lab logs -f

# Check status
sentra lab status
```

### Configuration

Edit `lab.yaml`:

```yaml
name: my-agent
version: "1.0"

agent:
  runtime: python
  entry_point: agent.py
  timeout: 30s

mocks:
  openai:
    enabled: true
    port: 8080
    latency_ms: 1000
    rate_limit: 3500
    error_rate: 0.01

simulation:
  record_full_trace: true
  enable_cost_tracking: true
  max_concurrent_scenarios: 10
```

### Writing Scenarios

Create `scenarios/test.yaml`:

```yaml
name: "Payment Flow Test"
description: "Test complete payment processing"

steps:
  - id: "create-intent"
    action: agent_request
    input: "Process payment for $99.99"
    expect:
      - calls: ["stripe.payment_intents.create"]
      - payment_status: "succeeded"
  
  - id: "verify-cost"
    action: verify_cost
    expect:
      - total_cost: <$0.10
```

### Replay Debugging

```bash
# Replay last failed run
sentra lab replay

# Replay specific run
sentra lab replay run-abc123

# Step-by-step mode
sentra lab replay run-abc123 --step

# Export to JSON
sentra lab replay run-abc123 --export report.json
```

## Project Structure

```
my-agent/
├── lab.yaml              # Main configuration
├── mocks.yaml            # Mock service configuration
├── scenarios/            # Test scenarios
│   ├── basic-test.yaml
│   └── payment-flow.yaml
├── fixtures/             # Mock response fixtures
│   ├── openai-responses.yaml
│   └── stripe-cards.yaml
├── agent.py              # Your agent code
└── .sentra-lab/          # Local storage
    ├── recordings/       # Test recordings
    └── sentra.db         # Simulation database
```

## Templates

```bash
# Python agent with OpenAI
sentra lab init my-agent --template=python

# Node.js agent with TypeScript
sentra lab init my-agent --template=nodejs

# Go agent
sentra lab init my-agent --template=go

# Full stack (all mocks)
sentra lab init my-agent --template=fullstack
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Test Agent
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install Sentra Lab
        run: curl -fsSL https://lab.sentra.dev/install.sh | sh
      
      - name: Run tests
        run: |
          sentra lab start
          sentra lab test --format junit --output results.xml
      
      - name: Upload results
        uses: actions/upload-artifact@v4
        with:
          name: test-results
          path: results.xml
```

## Cloud Features (Optional)

```bash
# Authenticate
sentra lab cloud login

# Upload test runs
sentra lab cloud push

# Download team runs
sentra lab cloud pull

# Sync data
sentra lab cloud sync
```

## Development

### Prerequisites

- Go 1.21+
- Docker 20.10+
- Make

### Building from Source

```bash
# Clone repository
git clone https://github.com/sentra-lab/cli
cd cli

# Install dependencies
make deps

# Build
make build

# Run tests
make test

# Install locally
make install
```

### Project Layout

```
sentra-lab/
├── cmd/                  # CLI commands
│   ├── sentra-lab/       # Main entry point
│   ├── init/             # Init command
│   ├── start/            # Start command
│   ├── test/             # Test command
│   ├── replay/           # Replay command
│   ├── config/           # Config command
│   └── cloud/            # Cloud command
├── internal/             # Internal packages
│   ├── docker/           # Docker client
│   ├── grpc/             # gRPC client
│   ├── config/           # Config loader
│   ├── ui/               # TUI components
│   ├── reporter/         # Test reporters
│   └── utils/            # Utilities
├── go.mod
├── Makefile
└── README.md
```

## Documentation

- [Getting Started](https://docs.sentra.dev/getting-started)
- [Writing Scenarios](https://docs.sentra.dev/scenarios)
- [Mock Services](https://docs.sentra.dev/mocks)
- [CI/CD Integration](https://docs.sentra.dev/ci-cd)
- [API Reference](https://docs.sentra.dev/api)

## Support

- **Documentation:** https://docs.sentra.dev
- **Discord:** https://discord.gg/sentra-lab
- **GitHub Issues:** https://github.com/sentra-lab/cli/issues
- **Email:** support@sentra.dev

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Docker](https://www.docker.com/) - Containerization
- [gRPC](https://grpc.io/) - RPC framework

---

**Made with ❤️ by the Sentra team**
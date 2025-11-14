# SENTRA LAB

```
███████╗███████╗███╗   ██╗████████╗██████╗  █████╗     ██╗      █████╗ ██████╗ 
██╔════╝██╔════╝████╗  ██║╚══██╔══╝██╔══██╗██╔══██╗    ██║     ██╔══██╗██╔══██╗
███████╗█████╗  ██╔██╗ ██║   ██║   ██████╔╝███████║    ██║     ███████║██████╔╝
╚════██║██╔══╝  ██║╚██╗██║   ██║   ██╔══██╗██╔══██║    ██║     ██╔══██║██╔══██╗
███████║███████╗██║ ╚████║   ██║   ██║  ██║██║  ██║    ███████╗██║  ██║██████╔╝
╚══════╝╚══════╝╚═╝  ╚═══╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝    ╚══════╝╚═╝  ╚═╝╚═════╝ 
```

**Test AI agents locally. Zero API costs. Production-perfect simulations.**

[![Build Status](https://github.com/sentra-dev/sentra-lab/workflows/CI/badge.svg)](https://github.com/sentra-dev/sentra-lab/actions)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-1.0.0-green.svg)](CHANGELOG.md)
[![Discord](https://img.shields.io/discord/YOUR_DISCORD_ID?color=7289da&label=discord)](https://discord.gg/sentra-lab)
[![Twitter](https://img.shields.io/twitter/follow/sentra_lab?style=social)](https://twitter.com/sentra_lab)

---

## 🚀 What is Sentra Lab?

**Stop burning money testing AI agents in production.** Sentra Lab is a local-first simulation platform that lets you build, test, and debug AI agents without API costs, production risks, or infrastructure complexity.

```bash
# Install CLI (macOS/Linux)
brew install sentra/tap/lab

# Initialize your project
sentra lab init my-agent

# Start local simulation environment
sentra lab start

# Run your agent (connects to mocks automatically)
python agent.py

# Test with scenarios
sentra lab test

# 🎉 Result: Zero API costs, production-perfect testing
```

**In 60 seconds, you go from zero to testing your AI agent against realistic OpenAI, Stripe, AWS mocks.**

---

## ⚡ Why Sentra Lab?

### The Problem

Building AI agents is expensive and risky:
- 💸 **$1000+/month in API costs** just for testing
- 🐛 **Production bugs** caught too late
- ⏱️ **Slow feedback loops** (deploy → test → debug → repeat)
- 🔮 **No time-travel debugging** (why did my agent fail 3 hours ago?)
- 🎲 **Non-deterministic tests** (works sometimes, fails randomly)

### The Solution

Sentra Lab gives you a **production-perfect local environment**:

```python
# Your agent code (unchanged)
import openai

client = openai.OpenAI()
response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)

# Behind the scenes:
# ✓ Calls intercepted → routed to local mock
# ✓ Mock behaves EXACTLY like production (latency, errors, rate limits)
# ✓ Full request/response recorded for replay
# ✓ Cost calculated (what you WOULD spend in production)
# ✓ Zero actual API calls made
```

---

## 🎯 Key Features

### 🏠 Local-First Development
- **No internet required** - Everything runs on localhost
- **No API keys needed** - Mocks bypass authentication
- **No costs incurred** - Unlimited testing for free
- **Instant setup** - From zero to testing in 60 seconds

### 🎭 Production-Perfect Mocks
Mock services that behave **exactly like production**:
- **OpenAI** - Real latency (1-3s), rate limits (3500 RPM), token counting, errors
- **Anthropic** - Claude API with identical behavior
- **Stripe** - Payment processing, webhooks, 3D Secure, card declines
- **CoreLedger** - Agent payment infrastructure
- **AWS** - S3, Lambda, DynamoDB
- **Databases** - PostgreSQL, MongoDB, Redis

### 🎬 Scenario-Driven Testing
Define complex test scenarios in YAML:

```yaml
name: "Rate Limit Recovery"
steps:
  - action: agent_request
    input: "What's the return policy?"
    expect:
      - calls: openai.chat.completions
      - status: success

  - action: inject_error
    service: openai
    error: rate_limit_exceeded

  - action: agent_request
    input: "Is this product in stock?"
    expect:
      - retry_count: 3
      - backoff_strategy: exponential
      - success: true
```

### ⏪ Time-Travel Debugging
Every agent execution is **fully recorded**. Replay step-by-step:

```bash
# Replay any past run
sentra lab replay run-abc123

# Timeline:
[00:00.000] Agent initialized
[00:00.124] Received user input
[00:00.250] → OpenAI API call
[00:01.340] ← Response (150 tokens, $0.0045)
[00:01.450] → Database query
[00:01.890] Agent decision: "Call payment API"
[00:02.100] ← Payment failed (card_declined)

# Step through decisions
sentra lab replay run-abc123 --step-by-step

# Set breakpoints
sentra lab replay run-abc123 --breakpoint 00:01.890
```

### 💰 Cost Estimation
Know **exactly what your agent will cost** before deploying:

```bash
$ sentra lab test

3/3 scenarios passed
Total cost: $0.00 (simulated)

Production cost estimate:
  OpenAI (GPT-4):        $12.45
  Anthropic (Claude):    $3.20
  Stripe (transactions): $2.90
  Total:                 $18.55

Per 1000 runs: $6,183.33
Per month (30K runs): ~$185,500
```

### 🔄 CI/CD Integration
Run tests in your pipeline:

```yaml
# .github/workflows/test.yml
- name: Test AI Agent
  run: |
    sentra lab start --ci
    sentra lab test --format junit --output results.xml
    
- name: Upload Results
  uses: actions/upload-artifact@v3
  with:
    name: test-results
    path: results.xml
```

### 🚀 Load Testing
Test at scale (10K+ concurrent agents):

```bash
# Local: 100 concurrent simulations
sentra lab load --agents 100 --duration 5m

# Cloud: 10K+ concurrent simulations
sentra lab load --agents 10000 --duration 1h --cloud
```

---

## 📦 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   DEVELOPER ENVIRONMENT                     │
│  IDE + Terminal + Version Control                           │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                  SENTRA LAB (Docker Container)              │
├─────────────────────────────────────────────────────────────┤
│  CLI Interface                                              │
│  ├─ sentra lab init                                         │
│  ├─ sentra lab start                                        │
│  ├─ sentra lab test                                         │
│  └─ sentra lab replay                                       │
├─────────────────────────────────────────────────────────────┤
│  Simulation Engine (Rust)                                   │
│  ├─ Agent Runtime (Python/Node.js/Go)                       │
│  ├─ Scenario Executor                                       │
│  ├─ Event Recorder (protobuf + zstd)                        │
│  └─ Time-Travel Replay                                      │
├─────────────────────────────────────────────────────────────┤
│  Mock Service Layer                                         │
│  ├─ OpenAI Mock (latency, rate limits, errors)             │
│  ├─ Stripe Mock (webhooks, 3DS, declines)                   │
│  ├─ AWS Mock (S3, Lambda, DynamoDB)                         │
│  └─ Database Mocks (PostgreSQL, MongoDB, Redis)             │
├─────────────────────────────────────────────────────────────┤
│  Local Storage                                              │
│  ├─ SQLite (runs, events, metrics)                          │
│  └─ File System (recordings, exports)                       │
└─────────────────────────────────────────────────────────────┘
```

**Read the full architecture:** [docs/architecture/overview.md](docs/architecture/overview.md)

---

## 🏁 Quick Start

### Prerequisites
- Docker Desktop (for local mocks)
- Python 3.8+, Node.js 16+, or Go 1.21+ (depending on your agent)

### Installation

**macOS/Linux:**
```bash
brew install sentra/tap/lab
```

**Manual install:**
```bash
curl -fsSL https://lab.sentra.dev/install.sh | sh
```

**Verify installation:**
```bash
sentra lab version
```

### Create Your First Simulation

**1. Initialize project:**
```bash
sentra lab init customer-support-agent
cd customer-support-agent
```

This creates:
```
customer-support-agent/
├── lab.yaml              # Configuration
├── scenarios/            # Test scenarios
│   ├── basic-flow.yaml
│   └── error-handling.yaml
├── mocks.yaml            # Mock service config
└── agent.py              # Your agent code (example)
```

**2. Start simulation environment:**
```bash
sentra lab start
```

This starts:
- Simulation engine (localhost:50051)
- OpenAI mock (localhost:8080)
- Stripe mock (localhost:8081)
- Database mocks (localhost:5432, 27017, 6379)

**3. Run your agent:**
```python
# agent.py
import openai

# Automatically connects to local mock when running in Sentra Lab
client = openai.OpenAI()

response = client.chat.completions.create(
    model="gpt-4",
    messages=[
        {"role": "system", "content": "You are a customer support agent."},
        {"role": "user", "content": "What's your return policy?"}
    ]
)

print(response.choices[0].message.content)
```

```bash
python agent.py
# ✓ Calls routed to mock
# ✓ Full trace recorded
# ✓ Cost calculated: $0.0045
```

**4. Test with scenarios:**
```bash
sentra lab test
```

Output:
```
Running scenarios from ./scenarios/...
  ✓ basic-flow.yaml (2.3s) - PASSED
  ✓ error-handling.yaml (3.2s) - PASSED

2/2 passed (100%)
Total cost: $0.00 (simulated: $8.45 in production)
```

**5. Replay for debugging:**
```bash
sentra lab replay run-abc123
```

---

## 📚 Documentation

### Getting Started
- [Installation Guide](docs/getting-started/installation.md)
- [Quick Start Tutorial](docs/getting-started/quick-start.md)
- [First Simulation](docs/getting-started/first-simulation.md)

### Guides
- [Writing Scenarios](docs/guides/writing-scenarios.md)
- [Using Mocks](docs/guides/using-mocks.md)
- [Recording & Replay](docs/guides/recording-replay.md)
- [Cost Estimation](docs/guides/cost-estimation.md)
- [CI/CD Integration](docs/guides/ci-cd-integration.md)

### Mock Services
- [OpenAI Mock](docs/mocks/openai.md)
- [Anthropic Mock](docs/mocks/anthropic.md)
- [Stripe Mock](docs/mocks/stripe.md)
- [AWS Mock](docs/mocks/aws.md)
- [Custom Mocks](docs/mocks/custom-mocks.md)

### API Reference
- [CLI Reference](docs/api/cli-reference.md)
- [Python SDK](docs/api/python-sdk.md)
- [JavaScript SDK](docs/api/javascript-sdk.md)
- [Go SDK](docs/api/go-sdk.md)

### Architecture
- [System Overview](docs/architecture/overview.md)
- [Simulation Engine](docs/architecture/simulation-engine.md)
- [Mock Layer](docs/architecture/mock-layer.md)
- [Recording System](docs/architecture/recording-system.md)

---

## 🌟 Real-World Examples

### Customer Support Agent
```bash
cd examples/agents/customer-support
sentra lab test
```

Tests:
- ✓ Handle multiple intents (refund, shipping, product info)
- ✓ Recover from OpenAI rate limits
- ✓ Handle database connection failures
- ✓ Process payment refunds correctly

### Sales Agent with Payments
```bash
cd examples/agents/sales-agent
sentra lab test
```

Tests:
- ✓ Product recommendations (OpenAI)
- ✓ Inventory checks (Database)
- ✓ Payment processing (Stripe)
- ✓ Failed payment handling
- ✓ Webhook delivery

### Multi-Step Data Analyst
```bash
cd examples/agents/data-analyst
sentra lab test
```

Tests:
- ✓ Natural language to SQL (OpenAI)
- ✓ Query execution (PostgreSQL)
- ✓ Results visualization
- ✓ Error handling (syntax errors, timeouts)

**Browse all examples:** [examples/](examples/)

---

## 🔧 SDKs

Sentra Lab provides SDKs for all major languages:

### Python
```bash
pip install sentra-lab
```

```python
from sentra_lab import LabClient, Scenario

client = LabClient()

scenario = Scenario.builder() \
    .name("Test payment flow") \
    .step(agent_request("Process payment for $99.99")) \
    .expect(calls=["stripe.payment_intents.create"]) \
    .build()

result = client.run(scenario)
print(f"Status: {result.status}")
print(f"Cost: ${result.cost_usd}")
```

### JavaScript/TypeScript
```bash
npm install @sentra-lab/sdk
```

```typescript
import { LabClient, Scenario } from '@sentra-lab/sdk';

const client = new LabClient();

const scenario = new Scenario()
  .name('Test payment flow')
  .step(agentRequest('Process payment for $99.99'))
  .expect({ calls: ['stripe.payment_intents.create'] });

const result = await client.run(scenario);
console.log(`Status: ${result.status}`);
console.log(`Cost: $${result.costUsd}`);
```

### Go
```bash
go get github.com/sentra-dev/sentra-lab/sdk-go
```

```go
import "github.com/sentra-dev/sentra-lab/sdk-go/lab"

client := lab.NewClient()

scenario := lab.NewScenario().
    Name("Test payment flow").
    Step(lab.AgentRequest("Process payment for $99.99")).
    Expect(lab.Calls("stripe.payment_intents.create"))

result, err := client.Run(context.Background(), scenario)
fmt.Printf("Status: %s\n", result.Status)
fmt.Printf("Cost: $%.2f\n", result.CostUSD)
```

---

## 🤝 Contributing

We welcome contributions! Sentra Lab is open source and built by the community.

**Ways to contribute:**
- 🐛 Report bugs
- 💡 Suggest features
- 📝 Improve documentation
- 🔌 Build new mocks
- 🧪 Add test scenarios
- 🎨 Improve UI/UX

**Getting started:**
1. Read [CONTRIBUTING.md](CONTRIBUTING.md)
2. Check [good first issues](https://github.com/sentra-dev/sentra-lab/labels/good%20first%20issue)
3. Join our [Discord](https://discord.gg/sentra-lab)

**Development setup:**
```bash
# Clone repo
git clone https://github.com/sentra-dev/sentra-lab.git
cd sentra-lab

# Setup environment
make setup

# Build all packages
make build

# Run tests
make test

# Start local development
make dev
```

---

## 📊 Roadmap

### ✅ Current (v1.0)
- [x] Local simulation engine
- [x] OpenAI, Anthropic, Stripe, AWS mocks
- [x] Scenario testing
- [x] Time-travel replay
- [x] Cost estimation
- [x] CLI tools
- [x] Python/JS/Go SDKs

### 🚧 In Progress (v1.1)
- [ ] Web dashboard (view results, analytics)
- [ ] Team collaboration (share scenarios)
- [ ] CI/CD plugins (GitHub Actions, GitLab CI)
- [ ] More mocks (Google Cloud, Azure, Anthropic)
- [ ] Load testing (cloud-based, 10K+ agents)

### 🔮 Future (v2.0)
- [ ] Visual scenario builder (no-code)
- [ ] AI-powered test generation
- [ ] Production monitoring integration
- [ ] Multi-region load testing
- [ ] Custom mock marketplace

**Vote on features:** [GitHub Discussions](https://github.com/sentra-dev/sentra-lab/discussions)

---

## 💬 Community

- **Discord:** [Join our community](https://discord.gg/sentra-lab)
- **Twitter:** [@sentra_lab](https://twitter.com/sentra_lab)
- **GitHub Discussions:** [Ask questions](https://github.com/sentra-dev/sentra-lab/discussions)
- **Blog:** [lab.sentra.dev/blog](https://lab.sentra.dev/blog)

---

## 📜 License

Sentra Lab is licensed under the [MIT License](LICENSE).

**Open source, free forever.** Cloud features (team collaboration, load testing) are optional paid add-ons.

---

## 🙏 Acknowledgments

Built with amazing open source tools:
- [Rust](https://www.rust-lang.org/) - Simulation engine
- [Go](https://go.dev/) - CLI and mocks
- [Docker](https://www.docker.com/) - Local development
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI
- [tiktoken](https://github.com/openai/tiktoken) - Token counting

Inspired by:
- [LocalStack](https://localstack.cloud/) - Local AWS testing
- [Stripe Mock](https://github.com/stripe/stripe-mock) - Mock payment API
- [VCR](https://github.com/vcr/vcr) - HTTP recording/replay

---

## 🚀 Get Started Now

```bash
# Install
brew install sentra/tap/lab

# Initialize
sentra lab init my-agent

# Start testing
sentra lab start
python agent.py
sentra lab test

# Zero API costs. Production-perfect testing. Time-travel debugging.
```

**Questions?** [Join our Discord](https://discord.gg/sentra-lab) or [open an issue](https://github.com/sentra-dev/sentra-lab/issues/new).

---

**Made with ❤️ by the Sentra team and contributors.**

⭐ Star us on GitHub if you find this useful!

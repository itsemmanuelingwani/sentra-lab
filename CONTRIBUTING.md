# Contributing to Sentra Lab

**First off, thank you!** 🎉 Sentra Lab is built by the community, for the community. Whether you're fixing a typo, adding a new mock service, or proposing a major feature, your contribution matters.

---

## 📋 Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [How Can I Contribute?](#how-can-i-contribute)
3. [Development Setup](#development-setup)
4. [Project Structure](#project-structure)
5. [Making Changes](#making-changes)
6. [Testing Guidelines](#testing-guidelines)
7. [Code Style](#code-style)
8. [Pull Request Process](#pull-request-process)
9. [Adding New Mocks](#adding-new-mocks)
10. [Release Process](#release-process)

---

## 📜 Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you're expected to uphold this code. Please report unacceptable behavior to conduct@sentra.dev.

---

## 🤝 How Can I Contribute?

### Reporting Bugs

**Before submitting:**
- Check if the bug is already reported in [Issues](https://github.com/sentra-dev/sentra-lab/issues)
- Try the latest `main` branch to see if it's fixed

**When submitting:**
- Use the bug report template
- Include reproduction steps
- Provide system info (OS, Docker version, Sentra Lab version)
- Share relevant logs

### Suggesting Features

**Before suggesting:**
- Check [Discussions](https://github.com/sentra-dev/sentra-lab/discussions) for existing ideas
- Consider if this fits Sentra Lab's scope (local-first AI agent testing)

**When suggesting:**
- Use the feature request template
- Explain the problem you're solving
- Provide use cases
- Consider implementation complexity

### Improving Documentation

Documentation improvements are ALWAYS welcome:
- Fix typos
- Clarify confusing sections
- Add examples
- Translate to other languages

### Contributing Code

Great! Read on for setup instructions.

---

## 🔧 Development Setup

### Prerequisites

1. **Required tools:**
   ```bash
   # macOS
   brew install go rust node python3 docker
   
   # Linux (Ubuntu/Debian)
   sudo apt-get install golang-go rustc nodejs npm python3 docker.io
   ```

2. **Verify installations:**
   ```bash
   go version       # >= 1.21
   cargo --version  # >= 1.70
   node --version   # >= 16
   python3 --version # >= 3.8
   docker --version
   ```

### Clone & Build

```bash
# Clone repository
git clone https://github.com/sentra-dev/sentra-lab.git
cd sentra-lab

# First-time setup (installs all dependencies)
make setup

# Build all packages
make build

# Run tests
make test

# Start development environment
make dev
```

**That's it!** You're ready to contribute.

---

## 📁 Project Structure

Sentra Lab is a monorepo with multiple packages:

```
sentra-lab/
├── packages/
│   ├── cli/                 # Go CLI
│   ├── engine/              # Rust simulation engine
│   ├── mocks/               # Mock services
│   │   ├── openai/          # Go
│   │   ├── anthropic/       # Go
│   │   ├── stripe/          # Node.js
│   │   ├── coreledger/      # Go
│   │   ├── aws/             # Go
│   │   └── databases/       # Go
│   ├── sdk-python/          # Python SDK
│   ├── sdk-javascript/      # JavaScript/TypeScript SDK
│   └── sdk-go/              # Go SDK
├── cloud/                   # Cloud platform (optional, closed-source)
├── docs/                    # Documentation
├── examples/                # Example agents & scenarios
├── fixtures/                # Mock response fixtures
└── tests/                   # Integration & E2E tests
```

**Key directories:**
- `packages/cli` - Command-line interface developers use
- `packages/engine` - Core simulation engine (Rust for performance)
- `packages/mocks` - Production-realistic mock services
- `packages/sdk-*` - Language-specific SDKs

---

## 🛠️ Making Changes

### Branching Strategy

```bash
# Create feature branch
git checkout -b feature/add-gemini-mock

# Or bug fix branch
git checkout -b fix/replay-crash-on-large-recordings
```

**Branch naming:**
- `feature/description` - New features
- `fix/description` - Bug fixes
- `docs/description` - Documentation
- `refactor/description` - Code refactoring
- `test/description` - Test improvements

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add Google Gemini mock service
fix: prevent replay crash on large recordings
docs: update scenario writing guide
refactor: simplify event recording logic
test: add integration tests for OpenAI mock
```

**Format:**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Examples:**
```
feat(mocks): add Cohere LLM mock service

- Implements chat and embed endpoints
- Realistic latency (500-2000ms)
- Rate limiting (1000 RPM)
- Cost tracking

Closes #123

---

fix(engine): prevent memory leak in event recorder

Events were not being flushed properly when simulations
ended abruptly. Now using defer to ensure cleanup.

Fixes #456
```

---

## 🧪 Testing Guidelines

**Every change MUST include tests.** No exceptions.

### Running Tests

```bash
# All tests
make test

# Specific package
cd packages/cli && go test ./...
cd packages/engine && cargo test

# With coverage
make test-coverage

# Integration tests
make test-integration

# E2E tests (requires Docker)
make test-e2e
```

### Writing Tests

**CLI (Go):**
```go
func TestScenarioExecution(t *testing.T) {
    // Arrange
    scenario := loadScenario("fixtures/basic-flow.yaml")
    
    // Act
    result, err := runScenario(scenario)
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, StatusPassed, result.Status)
    assert.Greater(t, result.CostUSD, 0.0)
}
```

**Engine (Rust):**
```rust
#[test]
fn test_event_recording() {
    // Arrange
    let recorder = EventRecorder::new();
    let event = create_test_event();
    
    // Act
    recorder.record(event).await.unwrap();
    
    // Assert
    let events = recorder.get_events().await;
    assert_eq!(events.len(), 1);
}
```

**Mocks (Go):**
```go
func TestOpenAIMock(t *testing.T) {
    // Arrange
    mock := NewOpenAIMock(config)
    req := createChatRequest()
    
    // Act
    resp, err := mock.ChatCompletion(req)
    
    // Assert
    require.NoError(t, err)
    assert.NotEmpty(t, resp.Choices)
    assert.Greater(t, resp.Usage.TotalTokens, 0)
}
```

### Test Coverage Requirements

- **Minimum:** 80% coverage
- **Critical paths:** 100% coverage (simulation engine, recording)
- **Mocks:** Test all endpoints, errors, rate limits

---

## 🎨 Code Style

### Go

Follow [Effective Go](https://golang.org/doc/effective_go.html):

```go
// ✅ GOOD
func (s *SimulationEngine) Run(ctx context.Context, scenario *Scenario) (*Result, error) {
    if scenario == nil {
        return nil, ErrNilScenario
    }
    
    result := &Result{
        StartedAt: time.Now(),
        Status:    StatusRunning,
    }
    
    // ... implementation
    
    return result, nil
}

// ❌ BAD - No context, no error handling
func Run(scenario *Scenario) *Result {
    return &Result{}
}
```

**Run linter:**
```bash
cd packages/cli
golangci-lint run ./...
```

### Rust

Follow [Rust Style Guide](https://rust-lang.github.io/api-guidelines/):

```rust
// ✅ GOOD
impl EventRecorder {
    pub async fn record(&self, event: Event) -> Result<(), RecordError> {
        let bytes = event.encode_to_vec();
        let compressed = compress(&bytes)?;
        
        self.buffer.push(compressed).await?;
        
        if self.should_flush() {
            self.flush().await?;
        }
        
        Ok(())
    }
}

// ❌ BAD - Synchronous, no error handling
impl EventRecorder {
    pub fn record(&self, event: Event) {
        // ... no error handling
    }
}
```

**Run linter:**
```bash
cd packages/engine
cargo clippy -- -D warnings
```

### Python

Follow [PEP 8](https://pep8.org/):

```python
# ✅ GOOD
def run_scenario(scenario: Scenario) -> Result:
    """Execute a scenario and return results.
    
    Args:
        scenario: The scenario to execute
        
    Returns:
        Result object with execution details
        
    Raises:
        ScenarioError: If scenario execution fails
    """
    if not scenario.validate():
        raise ScenarioError("Invalid scenario")
    
    return execute(scenario)

# ❌ BAD - No types, no docstring
def run_scenario(s):
    return execute(s)
```

**Run linter:**
```bash
cd packages/sdk-python
flake8 . && mypy .
```

### TypeScript

Follow [Google TypeScript Style](https://google.github.io/styleguide/tsguide.html):

```typescript
// ✅ GOOD
async function runScenario(scenario: Scenario): Promise<Result> {
  if (!scenario.validate()) {
    throw new ScenarioError('Invalid scenario');
  }
  
  const result = await execute(scenario);
  return result;
}

// ❌ BAD - No types
async function runScenario(scenario) {
  return execute(scenario);
}
```

**Run linter:**
```bash
cd packages/sdk-javascript
npm run lint
```

### Formatting

```bash
# Format all code
make fmt
```

---

## 🚀 Pull Request Process

### Before Submitting

- [ ] Code builds without errors
- [ ] All tests pass
- [ ] Linters pass
- [ ] Documentation updated (if needed)
- [ ] Changelog updated (if user-facing change)

### PR Description

Use the PR template:

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Tested locally with `make test`

## Screenshots (if applicable)

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-reviewed my code
- [ ] Commented complex logic
- [ ] Updated documentation
- [ ] No new warnings
- [ ] Added tests
- [ ] All tests pass
```

### Review Process

1. **Automated checks:** CI must pass (tests, linters, build)
2. **Code review:** At least 1 maintainer approval required
3. **Testing:** Reviewer tests changes locally
4. **Merge:** Maintainer merges using "Squash and merge"

### After Merge

- PR is automatically closed
- Changes included in next release
- You're added to contributors list! 🎉

---

## 🎭 Adding New Mocks

Want to add a mock for a new service? Awesome! Here's how:

### 1. Choose Language

- **Go** - Most mocks (OpenAI, AWS, CoreLedger)
- **Node.js** - Complex async flows (Stripe webhooks)
- **Rust** - Performance-critical (rarely needed)

### 2. Create Structure

```bash
mkdir -p packages/mocks/your-service
cd packages/mocks/your-service

# Go mock
mkdir -p cmd/server internal/{handlers,models,config}

# Node.js mock
npm init -y
mkdir -p src/{handlers,services,types}
```

### 3. Implement Mock

**Key requirements:**
- ✅ **Production-realistic latency**
- ✅ **Rate limiting**
- ✅ **Error injection**
- ✅ **Cost tracking** (if applicable)
- ✅ **Fixture-based responses**

**Example (Go):**
```go
// internal/handlers/chat.go
func (h *ChatHandler) Create(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    var req ChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, 400, "invalid_request", err.Error())
        return
    }
    
    // 2. Check rate limit
    if !h.rateLimiter.Allow(r.Context()) {
        writeError(w, 429, "rate_limit_exceeded", "Too many requests")
        return
    }
    
    // 3. Simulate latency
    latency := calculateLatency(req.Model)
    time.Sleep(latency)
    
    // 4. Inject errors (probabilistic)
    if rand.Float64() < h.errorRate {
        writeError(w, 503, "service_unavailable", "Server overloaded")
        return
    }
    
    // 5. Generate response
    resp := h.generateResponse(req)
    
    // 6. Track cost
    h.costTracker.Record(req.Model, resp.Usage.TotalTokens)
    
    json.NewEncoder(w).Encode(resp)
}
```

### 4. Add Fixtures

```yaml
# fixtures/your-service/responses.yaml
patterns:
  - match: "Hello"
    response:
      message: "Hi there!"
      tokens: 10
  
  - match: "What is the capital of France?"
    response:
      message: "The capital of France is Paris."
      tokens: 15
```

### 5. Add Tests

```go
func TestYourServiceMock(t *testing.T) {
    tests := []struct {
        name     string
        request  Request
        wantErr  bool
        wantCode int
    }{
        {
            name: "successful request",
            request: Request{...},
            wantErr: false,
            wantCode: 200,
        },
        {
            name: "rate limited",
            request: Request{...},
            wantErr: true,
            wantCode: 429,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ... test implementation
        })
    }
}
```

### 6. Add Documentation

Create `docs/mocks/your-service.md`:

```markdown
# Your Service Mock

## Overview
Brief description of the service

## Endpoints
| Method | Path | Description |
|--------|------|-------------|
| POST   | /v1/chat | Create chat completion |

## Configuration
```yaml
mocks:
  your-service:
    enabled: true
    port: 8090
    latency: 1000ms
```

## Examples
[Code examples]
```

### 7. Update Docker Compose

```yaml
# docker-compose.yml
mock-your-service:
  image: sentra/mock-your-service:latest
  ports:
    - "8090:8080"
  volumes:
    - ./fixtures/your-service:/fixtures:ro
  environment:
    - LOG_LEVEL=info
```

### 8. Submit PR

Title: `feat(mocks): add Your Service mock`

---

## 📦 Release Process

**For maintainers only.**

### Version Numbering

Follow [Semantic Versioning](https://semver.org/):
- **Major:** Breaking changes (v1.0.0 → v2.0.0)
- **Minor:** New features (v1.0.0 → v1.1.0)
- **Patch:** Bug fixes (v1.0.0 → v1.0.1)

### Release Steps

```bash
# 1. Update version
echo "1.2.0" > VERSION

# 2. Update CHANGELOG.md
vim CHANGELOG.md

# 3. Commit & tag
git add VERSION CHANGELOG.md
git commit -m "chore: release v1.2.0"
git tag -a v1.2.0 -m "Release v1.2.0"

# 4. Push
git push origin main --tags

# 5. GitHub Actions will:
#    - Build all packages
#    - Run tests
#    - Create Docker images
#    - Publish to npm/PyPI
#    - Create GitHub release
```

---

## 🏆 Recognition

**All contributors are recognized:**

- Added to `CONTRIBUTORS.md`
- Mentioned in release notes
- GitHub contributor badge

**Top contributors become maintainers** with merge access.

---

## 💬 Getting Help

**Stuck? Need guidance?**

- **Discord:** [Join our community](https://discord.gg/sentra-lab)
- **Discussions:** [Ask questions](https://github.com/sentra-dev/sentra-lab/discussions)
- **Email:** contribute@sentra.dev

**Remember:** There are no stupid questions. We were all beginners once. 🙂

---

## 🎉 Thank You!

Every contribution makes Sentra Lab better. Whether you fixed a typo, added a new mock, or improved documentation - **you're making a difference.**

**Happy coding!** 🚀

---

**PS:** Don't forget to ⭐ star the repo if you find it useful!

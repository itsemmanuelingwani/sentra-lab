# Changelog

All notable changes to Sentra Lab will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- Nothing yet

### Changed
- Nothing yet

### Deprecated
- Nothing yet

### Removed
- Nothing yet

### Fixed
- Nothing yet

### Security
- Nothing yet

---

## [1.0.0] - 2025-11-14

### Added

**Core Features:**
- 🎉 Initial release of Sentra Lab
- ⚡ Rust-based simulation engine for high-performance agent testing
- 🖥️ Go CLI with interactive terminal UI (Bubble Tea)
- 🎭 Production-realistic mock services:
  - OpenAI (GPT-4, GPT-3.5, embeddings)
  - Anthropic (Claude)
  - Stripe (payments, webhooks, 3D Secure)
  - CoreLedger (agent payments)
  - AWS (S3, Lambda, DynamoDB)
  - Databases (PostgreSQL, MongoDB, Redis)
- 📝 YAML-based scenario testing
- ⏪ Time-travel debugging (replay any simulation step-by-step)
- 💰 Production cost estimation (predict costs before deploying)
- 📊 Event recording and compression (protobuf + zstd)
- 🐳 Docker Compose for local development
- 📚 Comprehensive documentation

**SDKs:**
- 🐍 Python SDK with pytest integration
- 📦 JavaScript/TypeScript SDK
- 🐹 Go SDK

**CLI Commands:**
- `sentra lab init` - Initialize new project
- `sentra lab start` - Start simulation environment
- `sentra lab test` - Run test scenarios
- `sentra lab replay` - Replay simulations for debugging
- `sentra lab export` - Export results (JSON, JUnit, HAR)
- `sentra lab config` - Manage configuration

**Developer Experience:**
- Zero-configuration setup (sensible defaults)
- Works offline (local-first architecture)
- No API keys required for mocks
- Automatic code interception (no agent code changes needed)
- Hot reload for development
- Structured logging with tracing

**Performance:**
- Simulation start time: <2 seconds
- Mock API response: <10ms (local)
- Recording overhead: <5% latency impact
- Concurrent simulations: 100+ on laptop

**Documentation:**
- Getting started guides
- API reference for all SDKs
- Mock service documentation
- Architecture deep-dives
- Example agents and scenarios
- Contributing guidelines

### Changed
- N/A (initial release)

### Fixed
- N/A (initial release)

### Security
- Agent sandboxing with Docker containers
- Network isolation (only mock APIs accessible)
- No real API keys required
- Local-only data by default

---

## [0.9.0] - 2025-10-15 (Beta)

### Added
- Beta release for early adopters
- OpenAI mock (GPT-4, GPT-3.5)
- Stripe mock (basic payment flows)
- CLI with init, start, test commands
- Basic scenario execution
- Event recording (uncompressed)

### Known Issues
- Memory leak in event recorder (fixed in 1.0.0)
- Slow replay for large recordings (fixed in 1.0.0)
- Limited error injection (improved in 1.0.0)

---

## [0.8.0] - 2025-09-01 (Alpha)

### Added
- Proof of concept
- Basic OpenAI mock
- Simple CLI
- Docker-based architecture

### Known Issues
- Not production-ready
- Limited features
- No documentation

---

## Versioning Policy

### Major Versions (X.0.0)
Breaking changes that require migration:
- API changes
- Configuration format changes
- Data format changes

**Example:** Sentra Lab 2.0.0 may require regenerating recordings from 1.x

### Minor Versions (1.X.0)
New features, backwards-compatible:
- New mock services
- New CLI commands
- New SDK features
- Performance improvements

**Example:** Sentra Lab 1.1.0 adds Gemini mock, but works with 1.0.0 scenarios

### Patch Versions (1.0.X)
Bug fixes, no new features:
- Security patches
- Bug fixes
- Documentation updates

**Example:** Sentra Lab 1.0.1 fixes replay crash, no new features

---

## Deprecation Policy

**We follow a 3-release deprecation cycle:**

1. **Release N:** Feature announced as deprecated (warning in logs)
2. **Release N+1:** Warning becomes more prominent
3. **Release N+2:** Feature removed

**Example:**
- `v1.0.0`: `sentra lab test --legacy` deprecated
- `v1.1.0`: Warning: "Use `sentra lab test` instead"
- `v1.2.0`: `--legacy` flag removed

---

## Release Schedule

We aim for:
- **Major releases:** Yearly
- **Minor releases:** Quarterly
- **Patch releases:** As needed

**Security patches are released immediately.**

---

## Migration Guides

When upgrading between major versions, see:
- [v1.0 → v2.0 Migration Guide](docs/migrations/v1-to-v2.md) (when v2 is released)

---

## Stay Updated

- **GitHub Releases:** https://github.com/sentra-dev/sentra-lab/releases
- **Blog:** https://lab.sentra.dev/blog
- **Twitter:** [@sentra_lab](https://twitter.com/sentra_lab)
- **Discord:** https://discord.gg/sentra-lab

---

## Links

- [1.0.0]: https://github.com/sentra-dev/sentra-lab/releases/tag/v1.0.0
- [0.9.0]: https://github.com/sentra-dev/sentra-lab/releases/tag/v0.9.0
- [0.8.0]: https://github.com/sentra-dev/sentra-lab/releases/tag/v0.8.0
- [Unreleased]: https://github.com/sentra-dev/sentra-lab/compare/v1.0.0...HEAD

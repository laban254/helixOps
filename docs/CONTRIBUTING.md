# Contributing to HelixOps

Thank you for your interest in contributing to HelixOps! We welcome contributions from the community and are excited to work together to build the best AI SRE agent.

## 1. Getting Started

### Prerequisites
- [Go 1.21+](https://golang.org/dl/)
- Docker & Docker Compose (for local mock environment testing)

### Local Setup
1. Fork the repo and clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/helixops.git
   cd helixops
   ```
2. Set up the mock environment (PostgreSQL, Prometheus, Loki, AlertManager, Ollama):
   ```bash
   docker-compose up -d
   ```
3. Run the agent locally:
   ```bash
   go run ./cmd/mcp
   ```

## 2. Finding Something to Work On

Check out our [Issue Tracker](https://github.com/helixops/helixops/issues). If you're new to the codebase, look for issues labeled:
- `good first issue`
- `help wanted`
- `documentation`

## 3. Submitting Changes

1. Create a new feature branch:
   ```bash
   git checkout -b feature/my-awesome-feature
   ```
2. Make your changes and write tests if applicable. Run the test suite:
   ```bash
   make test
   # Or using go directly:
   go test ./... -race -cover
   ```
3. Commit your changes. Use descriptive commit messages.
   ```bash
   git commit -m "feat: added datadog client integration"
   ```
4. Push to your fork and open a Pull Request! A maintainer will review your PR shortly.

## 4. Code Standards
- **Go Format**: Ensure your code is formatted with `gofmt`.
- **Modularity**: We follow the `cmd/`, `internal/`, and `pkg/` structure. Put domain logic in `internal/` and reusable tools in `pkg/`.
- **Testing**: We strive for high coverage. Please include unit tests for new logic.
- **Documentation**: Update relevant docs when adding features or changing APIs.

## 5. Architecture Principles
- **Privacy First**: Never log raw metrics or logs to external services by default.
- **Zero Migration**: Changes must not require users to migrate their existing infrastructure.
- **Performance**: Keep the agent lightweight and responsive (Go single binary).
- **Extensibility**: Design interfaces for easy addition of new LLM providers and clients.

Thank you for contributing to HelixOps!

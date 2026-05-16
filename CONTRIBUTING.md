# Contributing to HelixOps

Thank you for your interest in contributing to HelixOps! We welcome contributions from the community.

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
3. Run the agent (HTTP webhook server):
   ```bash
   go run ./cmd/agent
   ```
   Or run the MCP stdio server:
   ```bash
   go run ./cmd/mcp
   ```

## 2. Finding Something to Work On

Check out our [Issue Tracker](https://github.com/helixops/helixops/issues). Look for issues labeled:
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
   go test ./... -race -cover
   ```
3. Commit with descriptive messages:
   ```bash
   git commit -m "feat: add datadog client integration"
   ```
4. Push to your fork and open a Pull Request.

## 4. Code Standards
- **Go Format**: Use `gofmt`.
- **Modularity**: `cmd/`, `internal/`, `pkg/` structure. Domain logic in `internal/`, reusable tools in `pkg/`.
- **Testing**: Include unit tests for new logic.
- **Documentation**: Update relevant docs when adding features.

## 5. Architecture Principles
- **Privacy First**: Never log raw metrics or logs to external services by default.
- **Zero Migration**: Changes must not require users to migrate existing infrastructure.
- **Performance**: Keep the agent lightweight and responsive (Go single binary).
- **Extensibility**: Design interfaces for easy addition of new LLM providers and clients.

Thank you for contributing to HelixOps!

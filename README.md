# HelixOps - AI SRE Agent

**The On-Call Copilot that lives in your cluster.**

HelixOps is an AI SRE Agent that connects to your existing infrastructure (Prometheus, Loki, GitHub) to automate Root Cause Analysis. NOT another observability platform - an agent that overlays on existing tools.

## Features

- 🚨 **Alert Enrichment**: Automatically correlates alerts with metrics and code changes
- 📊 **Golden Signals**: Latency, error rate, and traffic analysis
- 🐛 **Log Mining**: Error log correlation and analysis
- 🤖 **AI-Powered RCA**: LLM-based root cause identification
- 📢 **Multi-Channel Output**: Slack/Discord notifications + Markdown reports
- 🔒 **Privacy-First**: Local Ollama support for sensitive environments

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- (Optional) Prometheus, Loki, GitHub for production

### Development

```bash
# Clone the repository
git clone https://github.com/helixops/helixops.git
cd helixops

# Start mock environment
docker-compose up -d

# Build and run
go build -o helix-agent ./cmd/agent
./helix-agent

# Run tests
go test ./... -race -cover
```

### Configuration

Edit `config.yaml` to configure:

```yaml
app:
  host: "0.0.0.0"
  port: 8080

prometheus:
  url: "http://localhost:9090"

loki:
  url: "http://localhost:3100"

github:
  api_url: "https://api.github.com"
  # Set GITHUB_TOKEN environment variable

llm:
  provider: "openai"  # openai, anthropic, or ollama
  model: "gpt-4o"
  # Set OPENAI_API_KEY environment variable
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub API token |
| `OPENAI_API_KEY` | OpenAI API key |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `SLACK_WEBHOOK_URL` | Slack webhook URL |

## API Endpoints

### Webhook

`POST /webhook` - Receive alerts from AlertManager

```json
{
  "version": "4",
  "status": "firing",
  "alerts": [{
    "status": "firing",
    "labels": {
      "service_name": "cart-service",
      "alertname": "HighLatency",
      "severity": "warning"
    },
    "annotations": {
      "summary": "High latency detected on cart-service"
    },
    "startsAt": "2024-01-15T10:00:00Z"
  }]
}
```

### Health

`GET /health` - Health check endpoint

`GET /ready` - Readiness check endpoint

## Architecture

```
AlertManager → Webhook → Orchestrator → LLM → Output (Slack/Markdown)
                     ↓
              Prometheus (metrics)
                     ↓
              GitHub (commits)
                     ↓
              Loki (logs)
```

## Project Structure

```
helixops/
├── cmd/agent/main.go           # Entry point
├── internal/
│   ├── server/                 # HTTP handlers
│   ├── clients/                # API clients
│   │   ├── prometheus/         # PromQL client
│   │   ├── github/            # GitHub API client
│   │   └── loki/              # LogQL client
│   ├── orchestrator/           # Context preparation
│   ├── analyzer/               # RCA logic
│   ├── output/                 # Output channels
│   └── config/                 # Configuration
├── pkg/llm/                    # LLM providers
├── config.yaml                 # Configuration file
├── Dockerfile                   # Container image
└── docker-compose.yml           # Development environment
```

## Deployment

### Docker

```bash
docker build -t helixops:latest .
docker run -p 8080:8080 \
  -v $(pwd)/config.yaml:/etc/helixops/config.yaml \
  helixops:latest
```

### Kubernetes

See `k8s/` directory for Kubernetes manifests.

### Helm

```bash
helm install helixops ./helm/helixops
```

## Development

### Adding a New LLM Provider

1. Implement the `Provider` interface in `pkg/llm/provider.go`
2. Add provider type to `ProviderType` constants
3. Update `NewProvider()` factory function

### Adding a New Output Channel

1. Create new file in `internal/output/`
2. Implement `Send()` method
3. Add to `OutputConfig` in `config.yaml`

## Testing

```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -race -cover

# Run specific test
go test -v ./internal/server/...
```

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## License

MIT License - see LICENSE file for details.

## Support

- 📧 Email: support@helixops.io
- 💬 Discord: https://discord.gg/helixops
- 📖 Docs: https://docs.helixops.io

# HelixOps - AI SRE Agent

HelixOps is a Go-based incident analysis service that ingests Alertmanager webhooks, collects telemetry from Prometheus, Loki, GitHub, and optional Tempo, then produces RCA summaries, Markdown reports, Slack notifications, and postmortems.

## Scope

HelixOps currently provides two entry points:

- `cmd/agent` for the HTTP webhook service
- `cmd/mcp` for the Model Context Protocol server

## Implemented capabilities

- Alertmanager webhook ingestion
- Health and readiness endpoints
- Concurrent context collection from Prometheus, Loki, GitHub, and optional Tempo
- LLM-based RCA and postmortem generation
- Slack and Markdown outputs
- PostgreSQL incident storage

## Quick start

### HTTP agent

```bash
docker-compose up -d
go run ./cmd/agent
```

Send a sample alert:

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @test-alert.json
```

### MCP server

```bash
go run ./cmd/mcp
```

## Configuration summary

The agent loads `config.yaml` from the workspace root, `./config`, or `/etc/helixops`, then merges environment variables.

Required integrations depend on the mode you use:

- Prometheus and Loki for analysis
- GitHub token for commit lookup
- `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, or Ollama for LLM analysis
- `SLACK_WEBHOOK_URL` for notifications
- `HELIX_DB_PASSWORD` for PostgreSQL

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for the exact schema and environment names.

## Documentation

- [Architecture](docs/ARCHITECTURE.md)
- [Deployment](docs/DEPLOYMENT.md)
- [Configuration](docs/CONFIGURATION.md)
- [API reference](docs/API_REFERENCE.md)
- [Testing](docs/TESTING.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Security](SECURITY.md)
- [Changelog](CHANGELOG.md)

## Notes

- PostgreSQL is optional but enabled in the production configuration.
- Ollama is the supported local LLM path.
- The repository includes both HTTP and MCP entry points; they serve different runtime use cases.

# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Project Overview
**HelixOps** - AI SRE Agent that connects to existing stack (Prometheus, Loki, GitHub) to automate Root Cause Analysis. NOT an observability platform - an agent that overlays on existing tools.

## Technical Stack
- **Core**: Go (single binary, low footprint, K8s deployment ready)
- **LLM**: LangChain Go - supports OpenAI/Anthropic cloud and Ollama local
- **State**: SQLite (embedded, no external DB required)
- **Integrations**: Prometheus (PromQL), Loki (LogQL), GitHub (REST/GraphQL)

## Build Commands (Go Project)
```bash
# Build the agent
go build -o helix-agent main.go

# Run tests
go test ./...

# Run with hot reload (dev)
air
```

## Key Architectural Patterns
- **Webhook Listener**: Agent listens on `/webhook` for alerts from AlertManager
- **Context Window Orchestrator**: Prepares data for LLM (metrics + logs + commits)
- **Multi-Channel Output**: Slack/Discord notifications + Markdown incident reports
- **Privacy-First**: Raw logs stay in VPC; only summaries leave (configurable)

## Project Structure (Planned)
```
helixops/
├── cmd/
│   └── agent/
│       └── main.go           # Entry point
├── internal/
│   ├── server/               # HTTP handlers (webhook receiver)
│   ├── clients/              # Prometheus, Loki, GitHub clients
│   ├── orchestrator/          # Context preparation for LLM
│   └── analyzer/             # RCA logic
├── pkg/
│   ├── llm/                  # LangChain Go integration
│   └── models/               # Data models
├── config/
│   └── config.go             # Configuration loading
├── migrations/               # SQLite schema
└── docker-compose.yml        # Mock environment (app + prometheus + alertmanager)
```

## Critical Conventions
- **Zero Migration**: Never require users to change existing setup
- **Privacy First**: Local Ollama option for sensitive environments
- **Embedded DB**: Use SQLite, never external dependencies
- **Webhook Standard**: Support generic Prometheus AlertManager webhooks

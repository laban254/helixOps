# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Project Overview
**HelixOps** - AI SRE Agent that connects to existing stack (Prometheus, Loki, GitHub) to automate Root Cause Analysis. NOT an observability platform - an agent that overlays on existing tools.

## Technical Stack
- **Core**: Go (single binary, low footprint, K8s deployment ready)
- **LLM**: LangChain Go - supports OpenAI/Anthropic cloud and Ollama local
- **State**: PostgreSQL (Docker-compatible, production-ready)
- **Integrations**: Prometheus (PromQL), Loki (LogQL), GitHub (REST/GraphQL)

## Build Commands (Go Project)
```bash
# Build the agent
go build -o helix-agent ./cmd/mcp

# Run tests
go test ./...

# Run with hot reload (dev)
air

# Run directly
go run ./cmd/mcp
```

## Key Architectural Patterns
- **Webhook Listener**: Agent listens on `/webhook` for alerts from AlertManager
- **Context Window Orchestrator**: Prepares data for LLM (metrics + logs + commits)
- **Multi-Channel Output**: Slack/Discord notifications + Markdown incident reports
- **Privacy-First**: Raw logs stay in VPC; only summaries leave (configurable)

## Project Structure
```
helixops/
├── cmd/
│   └── mcp/
│       └── main.go           # Entry point
├── internal/
│   ├── server/               # HTTP handlers (webhook receiver)
│   ├── clients/              # Prometheus, Loki, GitHub, Tempo clients
│   ├── orchestrator/         # Context preparation for LLM
│   ├── analyzer/             # RCA logic
│   ├── output/               # Slack, Markdown output
│   ├── postmortem/           # Postmortem generation
│   ├── remediation/          # Remediation rules
│   ├── config/               # Configuration loading
│   ├── db/                   # PostgreSQL database layer
│   ├── models/               # Data models
│   └── mcp/                  # Model Context Protocol server
├── config/                   # Configuration files
│   ├── config.yaml           # Example configuration
│   ├── prometheus.yml        # Prometheus config
│   └── alertmanager.yml      # AlertManager config
├── docs/                     # Documentation
└── docker-compose.yml        # Mock environment (app + prometheus + alertmanager + ollama)
```

## Critical Conventions
- **Zero Migration**: Never require users to change existing setup
- **Privacy First**: Local Ollama option for sensitive environments
- **Production-Ready DB**: PostgreSQL for scalability and Docker compatibility
- **Webhook Standard**: Support generic Prometheus AlertManager webhooks

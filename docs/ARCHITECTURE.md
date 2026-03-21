# HelixOps Architecture

## System Overview

HelixOps is an AI-powered SRE Agent designed to run within your Kubernetes cluster. It acts as an intelligent incident investigation system that correlates metrics, logs, and code changes to automate Root Cause Analysis (RCA).

```
┌─────────────────────────────────────────────────────────────────┐
│                      Your Cluster (VPC)                         │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐    │
│  │ Prometheus   │  │ Loki/Logs    │  │ Your Microservices │    │
│  │ (Metrics)    │  │ (Aggregator) │  │ (with Tempo traces)│    │
│  └──────────────┘  └──────────────┘  └────────────────────┘    │
│         ▲                ▲                        ▲              │
│         │                │                        │              │
│         │ /api/v1/query  │ /loki/api/v1          │              │
│         │                │                        │              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │            HelixOps Agent (Go Binary)                   │   │
│  │  ┌────────────────────────────────────────────────────┐ │   │
│  │  │ HTTP Server (port 8080)                            │ │   │
│  │  │  - POST /webhook       (AlertManager webhook)      │ │   │
│  │  │  - GET  /health       (Liveness probe)            │ │   │
│  │  │  - GET  /ready        (Readiness probe)           │ │   │
│  │  │  - GET  /postmortems  (Postmortem list)           │ │   │
│  │  └────────────────────────────────────────────────────┘ │   │
│  │                          ▲                               │   │
│  │  ┌────────────────────────┴────────────────────────┐   │   │
│  │  │                                                  │   │   │
│  │  ▼                                                  ▼   │   │
│  │  ┌──────────────────┐      ┌──────────────────┐       │   │
│  │  │ Orchestrator     │      │ LLM Interface    │       │   │
│  │  │ - Concurrency    │      │ - OpenAI/Claude  │       │   │
│  │  │ - Context Build  │      │ - Ollama (local) │       │   │
│  │  └──────────────────┘      └──────────────────┘       │   │
│  │         ▲                           ▲                   │   │
│  │         │                           │                   │   │
│  │  ┌──────┴───────┬──────────────────┴──────┐            │   │
│  │  ▼              ▼                          ▼            │   │
│  │  Analyzer    Postmortem       Output Layer             │   │
│  │  (RCA)       Generator        - Slack/Discord          │   │
│  │              (Reports)        - Markdown Files         │   │
│  │                               - PostgreSQL DB              │   │
│  │  ┌────────────────────────────────────────┐            │   │
│  │  │ PostgreSQL Database             │            │   │
│  │  │ - API credentials                      │            │   │
│  │  │ - Service mappings (service→repo)      │            │   │
│  │  │ - Incident history                     │            │   │
│  │  └────────────────────────────────────────┘            │   │
│  └─────────────────────────────────────────────────────────┘   │
│                          ▼                                      │
│  ┌─────────────────────────────────────────┐                  │
│  │ AlertManager (Prometheus)               │                  │
│  └─────────────────────────────────────────┘                  │
└─────────────────────────────────────────────────────────────────┘
                         │ webhook
                         ▼
        ┌────────────────────────────────┐
        │ External Systems (Optional)    │
        ├────────────────────────────────┤
        │ - GitHub API (commit history)  │
        │ - Slack/Discord webhooks       │
        │ - OpenAI/Anthropic endpoints   │
        │ - Ollama (local LLM)           │
        └────────────────────────────────┘
```

---

## Component Architecture

### 1. HTTP Server (`internal/server/`)

**Responsibilities:**
- Receive Prometheus AlertManager webhooks
- Route requests to appropriate handlers
- Implement graceful shutdown
- Health and readiness probes for K8s

**Key Files:**
- `server.go` - Server initialization and dependency injection
- `router.go` - HTTP routing with chi/v5
- `handlers.go` - Request handlers

**Design Pattern:** Dependency injection allowing easy testing and modularity.

---

### 2. Orchestrator (`internal/orchestrator/`)

**Responsibilities:**
- Coordinate concurrent data collection from multiple sources
- Build a unified context window for LLM analysis
- Handle errors gracefully (partial data acceptable)
- Respect configured time windows and rate limits

**How It Works:**

```
Input: Alert + Service Name
  │
  ├─ Goroutine 1: Fetch Metrics from Prometheus
  │  └─ Query: latency, error rate, RPS (±15min window)
  │
  ├─ Goroutine 2: Fetch Logs from Loki
  │  └─ Query: error logs, stack traces (with timestamps)
  │
  ├─ Goroutine 3: Fetch Commits from GitHub
  │  └─ Query: recent commits to service repo (last 24h)
  │
  └─ Goroutine 4: Fetch Traces from Tempo (optional)
     └─ Query: slow spans, latency distribution

  │
  └─ Aggregate into AnalysisContext
     └─ Pass to LLM for analysis
```

**Key Data Structures:**

```go
type AnalysisContext struct {
    ServiceName string
    Alert AlertInfo
    Metrics MetricsSummary
    RecentCommits []CommitInfo
    Traces TraceContext
    TimeWindow TimeWindow
}
```

---

### 3. LLM Abstraction (`pkg/llm/`)

**Responsibilities:**
- Define provider interface
- Implement multiple backends
- Handle prompt engineering
- Manage token limits and costs

**Supported Providers:**
- **OpenAI**: GPT-4o via cloud API
- **Anthropic**: Claude 3.5 via cloud API
- **Ollama**: Local LLM for privacy-first deployments

**Provider Interface:**

```go
type Provider interface {
    Analyze(ctx context.Context, prompt string) (string, error)
    Name() string
}
```

**Why Abstract?**
- Swap providers without changing business logic
- Easy to add new providers (e.g., Bedrock, vLLM)
- Supports both cloud and local inference

---

### 4. Analyzer (`internal/analyzer/`)

**Responsibilities:**
- Execute RCA workflow
- Build context-aware prompts
- Invoke LLM for analysis
- Parse and structure responses

**RCA Workflow:**

```
Input: AlertItem + AnalysisContext
  │
  ├─ Build Prompt
  │  └─ Structure: problem + metrics + commits + instructions
  │
  ├─ Call LLM
  │  └─ Stream response for faster feedback
  │
  ├─ Parse Response
  │  └─ Extract: root cause, confidence, next steps
  │
  └─ Return AnalysisResult
     └─ Include confidence score and reasoning
```

---

### 5. Postmortem Generator (`internal/postmortem/`)

**Responsibilities:**
- Generate formal incident postmortems
- Apply remediation rules
- Format as Markdown
- Integrate with output layer

**Execution Path (on Alert Resolution):**

```
Alert Status: "resolved"
  │
  ├─ Prepare full context (start time → resolved time)
  │
  ├─ Invoke LLM for postmortem summary
  │
  ├─ Query Remediation Rules Engine
  │  └─ Match RCA pattern to known fixes
  │
  └─ Format as Markdown report
     └─ Include timeline, root cause, action items
```

---

### 6. Data Clients

#### 6.1 Prometheus Client (`internal/clients/prometheus/`)

**Responsibilities:**
- Execute PromQL queries
- Handle timeouts and retries
- Parse numeric results

**Query Examples:**

```promql
# Latency (p99)
histogram_quantile(0.99, 
  sum(rate(http_request_duration_seconds_bucket{service='cart-service'}[5m])) by (le)
)

# Error Rate
sum(rate(http_requests_total{service='cart-service',status=~'5..'}[5m])) / 
sum(rate(http_requests_total{service='cart-service'}[5m]))

# Requests Per Second
sum(rate(http_requests_total{service='cart-service'}[5m]))
```

#### 6.2 Loki Client (`internal/clients/loki/`)

**Responsibilities:**
- Execute LogQL queries
- Extract structured logs
- Handle large result sets

**Query Example:**

```logql
{job="cart-service"} 
| json 
| level="error"
| timestamp >= "${START_TIME}" and timestamp <= "${END_TIME}"
```

#### 6.3 GitHub Client (`internal/clients/github/`)

**Responsibilities:**
- Fetch commit history
- Extract PR information
- Correlate commits with alert time

**API Endpoints Used:**
- `GET /repos/{owner}/{repo}/commits` (REST)
- GraphQL for detailed author info (optional)

#### 6.4 Tempo Client (`internal/clients/tempo/`)

**Responsibilities:**
- Query distributed traces
- Identify slow spans
- Correlate with alerts

**Optional integration** for complete observability correlation.

---

### 7. Output Layer (`internal/output/`)

**Responsibilities:**
- Format incident reports
- Send to multiple channels
- Persist to filesystem

**Supported Channels:**
- **Slack**: Rich message blocks with buttons
- **Discord**: Similar formatted messages
- **Markdown Files**: Local storage for compliance
- **PostgreSQL**: Historical tracking

---

### 8. Database (`internal/db/`)

**Technology:** PostgreSQL (Docker-compatible)

**Stored Entities:**
- API credentials (encrypted)
- Service→Repository mappings
- Incident history
- Analysis results

**Benefits:**
- Docker Compose integration
- Standard SQL access
- Easy backups
- Connection pooling for high load

---

### 9. Configuration (`internal/config/`)

**Loading Strategy:**
1. Load default values
2. Merge from `config.yaml`
3. Override with environment variables

**Config Structure:**

```yaml
app:
  host: 0.0.0.0
  port: 8080
  log_level: info

prometheus:
  url: http://prometheus:9090
  timeout: 10s

loki:
  url: http://loki:3100
  timeout: 10s

github:
  api_url: https://api.github.com
  token_env: GITHUB_TOKEN

llm:
  provider: openai  # or "anthropic" or "ollama"
  model: gpt-4o
  temperature: 0.7
  max_tokens: 2000
  
  # For Ollama
  ollama_url: http://localhost:11434
  ollama_model: llama2

output:
  slack:
    enabled: true
    webhook_url_env: SLACK_WEBHOOK_URL
  markdown:
    enabled: true
    output_dir: /reports
```

---

### 10. MCP Server (`internal/mcp/`)

**Purpose:** Expose HelixOps as tools to other AI agents via Model Context Protocol

**Exposed Tools:**
- `analyze_alert` - Perform full RCA
- `get_service_metrics` - Query golden signals
- `search_logs` - Query Loki
- `get_recent_commits` - Fetch repo commits

**Integration:** Allows Claude/other models to call HelixOps as a client library

---

## Data Flow: Alert to Postmortem

### Scenario: Alert Fires

```
1. AlertManager Webhook Received
   POST /webhook {AlertManagerPayload}

2. Handler Parses Alert
   Extract: service_name, alert_name, severity, startsAt

3. Orchestrator Prepares Context
   Concurrent Goroutines:
   - Prometheus: Fetch metrics
   - Loki: Fetch logs
   - GitHub: Fetch commits
   - Tempo: Fetch traces (optional)

4. Analyzer Invokes LLM
   Send: context + prompt

5. LLM Returns Analysis
   Parse: root_cause, confidence, next_steps

6. Output Generated
   - Send to Slack/Discord
   - Save Markdown report
   - Store in PostgreSQL

7. Acknowledge Handler
   Return 200 OK to AlertManager
```

### Scenario: Alert Resolves

```
1. AlertManager Webhook Received
   POST /webhook {AlertManagerPayload with status: "resolved"}

2. Handler Detects Resolution
   Extract incident time window (startsAt → endsAt)

3. Postmortem Generator Invoked
   Prepare full context (duration of incident)

4. LLM Generates Postmortem
   Summary: What happened, why, how it was detected

5. Remediation Rules Applied
   Query: RCA pattern → known fixes

6. Report Generated
   Markdown file + Slack notification

7. Database Updated
   Mark incident as resolved + store postmortem
```

---

## Concurrency Model

**Design Principle:** Maximize parallelism within incident analysis phase.

### Goroutine Pool Strategy

```go
// Orchestrator uses buffered channel for concurrent collection
type result struct {
    metrics models.MetricsSummary
    commits []models.CommitInfo
    traces  tempo.TraceContext
    err     error
}

resultCh := make(chan result, 3)

// Launch 3 independent goroutines
go func() { resultCh <- result{metrics: fetchMetrics()} }()
go func() { resultCh <- result{commits: fetchCommits()} }()
go func() { resultCh <- result{traces: fetchTraces()} }()

// Wait for all to complete (with timeout)
for i := 0; i < 3; i++ {
    r := <-resultCh
    if r.err != nil && aggregatedErr == nil {
        aggregatedErr = r.err
    }
}
```

**Benefits:**
- Client queries parallelized (not sequential)
- Faster incident analysis (3x speedup vs serial)
- Graceful degradation (missing data acceptable)

---

## Error Handling Strategy

### Philosophy

**Robustness Over Perfection**: An incomplete analysis is better than no analysis.

### Patterns

1. **Client Errors** (Prometheus unavailable)
   - Log error, continue with available data
   - Return partial AnalysisContext

2. **LLM Errors** (Rate limited)
   - Retry with exponential backoff
   - Fall back to rule-based analysis

3. **Output Errors** (Slack webhook fails)
   - Log error, still persist to file
   - Alert operator via metrics

---

## Security Considerations

### Data Privacy

1. **Credential Storage**
   - API keys encrypted at rest in PostgreSQL
   - Never logged
   - Require explicit configuration

2. **Local LLM Support**
   - Option to use Ollama (on-premise)
   - Logs/metrics never leave VPC
   - Only anonymized summaries sent externally

3. **RBAC-Ready**
   - Prepare for future role-based access
   - Database schema supports user tracking

### Network Security

1. **TLS Support**
   - All external API calls over HTTPS
   - Support mutual TLS for future cloud plane

2. **Webhook Validation**
   - Validate AlertManager webhook source (optional IP whitelist)
   - Rate limiting on webhook endpoint

---

## Performance Characteristics

| Operation | Typical Duration | P95 | Notes |
|-----------|------------------|-----|-------|
| Webhook processing | 50ms | 100ms | Immediate ack |
| Context collection | 2-5s | 10s | 4 concurrent queries |
| LLM analysis | 3-8s | 15s | Depends on LLM latency |
| Total alert→Slack | 5-15s | 20s | E2E, async processing |

---

## Extensibility Points

### 1. New LLM Provider

```go
// Implement interface in pkg/llm/
type MyProvider struct { }
func (p *MyProvider) Analyze(ctx context.Context, prompt string) (string, error) { }
func (p *MyProvider) Name() string { return "myprovider" }

// Register in NewProvider() factory
```

### 2. New Data Client

```go
// Implement in internal/clients/
// Add to Orchestrator.PrepareContext()
```

### 3. New Output Channel

```go
// Implement in internal/output/
// Add to handler.processAlerts()
```

---

## Deployment Topology

### Single-Cluster Deployment

```
One Kubernetes Cluster:
├─ HelixOps Agent (1 replica)
├─ Prometheus
├─ Alertmanager
├─ Loki
└─ Your Microservices
```

### Multi-Cluster Future (Phase 3)

```
Multiple Kubernetes Clusters:
├─ Cluster 1: HelixOps Agent → api.helixops.com
├─ Cluster 2: HelixOps Agent → api.helixops.com
├─ Cluster 3: HelixOps Agent → api.helixops.com
├─ ...
└─ SaaS Control Plane: Dashboard + Knowledge Base
```

---

## Key Architectural Decisions

| Decision | Rationale |
|----------|-----------|
| **Go** | Single binary, K8s-native, fast, low footprint |
| **PostgreSQL** | Docker Compose, standard SQL, production-ready |
| **Concurrency** | Faster incident analysis, responsive system |
| **Abstracted LLM** | Provider flexibility, cloud + local support |
| **Webhook-driven** | Integrates with standard alerting (AlertManager) |
| **Async Processing** | Fast webhook ack, non-blocking analysis |

---

## Testing Architecture

- **Unit Tests**: Client mocks, orchestrator flow
- **Integration Tests**: Docker Compose environment
- **E2E Tests**: Alert→Webhook→Postmortem flow
- **Coverage**: `go test ./... -cover`

See [TESTING.md](TESTING.md) for detailed testing guide.

---

## References

- [Go Best Practices](https://golang.org/doc/)
- [Prometheus Querying](https://prometheus.io/docs/prometheus/latest/querying/)
- [Loki LogQL](https://grafana.com/docs/loki/latest/query/)
- [AlertManager Webhook Format](https://prometheus.io/docs/alerting/latest/clients/)

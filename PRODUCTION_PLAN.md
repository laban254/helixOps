# HelixOps — Production Plan (Implementation Specs)

> **Updated:** 2026-05-16

---

## Phase 11: Rate Limiting

### Config
Add to `internal/config/config.go`:
```go
type ServerConfig struct {
    RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}
type RateLimitConfig struct {
    Enabled            bool    `mapstructure:"enabled"`
    RequestsPerSecond  float64 `mapstructure:"requests_per_second"`
    Burst              int     `mapstructure:"burst"`
}
```
Defaults: `enabled: true`, `rps: 10`, `burst: 20`.

### Middleware
Add to `internal/server/middleware.go`:
- `RateLimitMiddleware(limiter *rate.Limiter)` — returns 429 with `Retry-After` header
- `NewRateLimiter(rps, burst)` — constructor

### Wiring
- `SetupRouter(handler, cfg)` — apply middleware when `cfg.Server.RateLimit.Enabled`
- `server.go` — pass `cfg` to `SetupRouter`
- `handlers_test.go` — set `cfg.Server.RateLimit.Enabled = false`

### Dependencies
- `go get golang.org/x/time/rate`

---

## Phase 12: Kubernetes Manifests

Create `k8s/` directory:
```
k8s/
├── deployment.yaml      # 2 replicas, port 8080, liveness/readiness probes
├── service.yaml         # ClusterIP
├── configmap.yaml       # config.yaml content
├── hpa.yaml             # CPU 70%, min 2, max 10
└── kustomization.yaml   # common labels, namespace
```

Probes: `GET /health` (liveness), `GET /ready` (readiness), `terminationGracePeriodSeconds: 45`.

---

## Phase 13: Production Validation

### Load Test
```bash
hey -n 6000 -c 100 -m POST -H "Content-Type: application/json" \
  -D ./test-alert.json http://localhost:8080/webhook
```
Pass: p50 < 200ms, p99 < 1s, 0 errors, 0 panics.

### Chaos Test
Kill each dependency (Prometheus, Loki, DB) independently. Verify:
- No crash/panic
- Alert acknowledged with 202
- Error logged per-source
- Auto-recovery when dependency returns

### Security
```bash
gosec ./...      # No HIGH/CRITICAL findings
trivy fs .       # No CRITICAL CVEs
```

---

## Phase 7–9: Unit Test Plans

### `internal/db` (0% → 80%)
- Use PostgreSQL test container or SQLite in-memory
- Test: New, Migrate, CreateIncident, GetIncident, ResolveIncident, ListIncidents, CreateAnalysisResult, Ping, Close

### `internal/output` (0% → 80%)
- MarkdownReporter: constructor, Report, SendPostmortem, formatting methods
- SlackSender: SendAnalysis (200/error/network), SendPostmortem, buildMessage

### `internal/orchestrator` (0% → 70%)
- Mock all clients. Test: PrepareContext (all succeed, partial fail, all fail), HealthCheck, fetch methods

### `internal/clients/github` (0% → 80%)
- httptest server. Test: FetchCommits, FetchCommitsByRepo, splitRepo, auth headers

### `internal/clients/loki` (0% → 70%)
- httptest server. Test: Query, QueryErrorLogs, Health

### `internal/config` (28% → 80%)
- Test: Load (file found/not found/env overrides), all GetDuration methods, Validate expanded, ProviderType

### `internal/server` (31% → 70%)
- Test: all webhook scenarios, health (component failures), ready, postmortem endpoints, processAlerts

### `internal/postmortem` (0% → 70%)
- Mock LLM. Test: Generate, buildPrompt, assembleMarkdown

### `internal/remediation` (0% → 70%)
- Test: NewEngine, AddRule, GetSuggestions (match/no match/multiple)

### `internal/clients/prometheus` (44% → 80%)
- Test: QueryLatencyP99, QueryErrorRate, QueryRPS, Health

### `internal/clients/tempo` (64% → 80%)
- Test: GetTracesByService, GetTraceByID, error cases

### `internal/analyzer` (57% → 80%)
- Test: Analyze, AnalyzeWithContext, parseLLMResponse, buildContextPrompt

### `internal/logging` (0% → 70%)
- Test: Init sets correct log level

### `internal/mcp` (0% → 70%)
- Test: all tool handlers with mock deps

### `pkg/llm` (41% → 80%)
- Test: NewProvider (all providers), Analyze, Name

---

## Phase 10: Integration Tests

Directory: `internal/integration/e2e_test.go` with `//go:build integration` tag.

Scenarios:
1. **Happy path** — POST alert → 202 → DB has incident → Markdown file created
2. **Prometheus down** — 202 received, analysis proceeds with data gaps
3. **Loki down** — 202 received, logs noted unavailable
4. **DB down** — 202 received, in-memory processing, Markdown file still created
5. **Invalid JSON** → 400
6. **Empty alerts** → 400
7. **Oversized body** → 413
8. **GET /health** — healthy vs degraded
9. **GET /postmortems** — list and single

Runner:
```bash
go test -v -tags=integration -timeout 120s ./internal/integration/
```

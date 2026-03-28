# HelixOps Production Readiness Plan

**Status**: Comprehensive audit + actionable plan  
**Last Updated**: 2026-03-14  
**Target**: Full end-to-end production deployment

---

## Executive Summary

HelixOps is an **~70% feature-complete** SRE agent. Core architecture is sound, but several critical areas need hardening before production:

| Area | Status | Priority | Impact |
|------|--------|----------|--------|
| Webhook/Alert Ingestion | ✅ 90% |🔴 Critical | Fully functional end-to-end |
| LLM Integration | ✅ 85% | 🔴 Critical | Uses lightweight model, tested |
| Error Handling | ⚠️ 40% | 🔴 Critical | Missing graceful fallbacks |
| Logging/Observability | ⚠️ 50% | 🟡 High | Basic log.Printf, needs structured logging |
| Database Persistence | ⚠️ 30% | 🟡 High | SQLite schema exists, not integrated |
| Configuration Validation | ⚠️ 60% | 🟡 High | Partial env var support |
| Test Coverage | ⚠️ 40% | 🟡 High | 12 packages at 0% |
| Graceful Shutdown | ✅ 90% | 🟢 Medium | Implemented, needs testing |
| Rate Limiting | ❌ 0% | 🟢 Medium | Not implemented |
| Metrics Export | ❌ 0% | 🟢 Low | Not implemented (can use Prom scrape) |

---

---

# 🚀 EXECUTION PHASES

## Phase 1: Error Handling & Resilience ⚠️
**Status**: NOT STARTED  
**Duration**: 1-2 days  
**Goal**: Make system work with partial data (graceful degradation)

### Phase 1 - What Gets Done
- ✅ Refactor orchestrator to handle failed data sources
- ✅ Update analyzer to accept incomplete context
- ✅ Update handlers to save partial analysis
- ✅ Add error tracking to models
- ✅ Test graceful degradation scenarios

### Phase 1 - Files to Modify
1. `internal/models/` - Add error fields to AnalysisContext
2. `internal/orchestrator/context.go` - Implement graceful collection
3. `internal/analyzer/rca.go` - Handle nil fields
4. `internal/server/handlers.go` - Save partial results
5. `internal/server/handlers_test.go` - Add degradation tests

### Phase 1 - Deliverable
```bash
✓ Alert webhook works even if Prometheus is down
✓ LLM still analyzes with just alert + commit data
✓ Error collection logged but doesn't fail request
✓ GET /health returns component-by-component status
```

### Phase 1 - Entry Point
```bash
cd /home/kibe/pro/engineeringhub/HelixOps
# Start here when ready
```

---

## Phase 2: Structured Logging 📊
**Status**: NOT STARTED  
**Duration**: 1 day  
**Goal**: Replace all log.Printf with slog (JSON structured logs)

### Phase 2 - What Gets Done
- ✅ Add slog to cmd/agent/main.go
- ✅ Replace all log.Printf with slog calls
- ✅ Add contextual fields (service, alert_id, request_id)
- ✅ Make log level configurable from config.yaml
- ✅ Verify JSON output to stdout

### Phase 2 - Files to Modify
1. `cmd/agent/main.go` - Initialize slog handler
2. `internal/server/handlers.go` - Replace log.Printf
3. `internal/orchestrator/context.go` - Replace log.Printf
4. `internal/analyzer/rca.go` - Replace log.Printf
5. `internal/config/config.go` - Add LogLevel loading

### Phase 2 - Deliverable
```json
{"time":"2026-03-14T12:30:45Z","level":"info","msg":"Webhook received","request_id":"abc-123","alerts":1}
{"time":"2026-03-14T12:30:46Z","level":"info","msg":"Analysis complete","service":"api","confidence":"high"}
```

---

## Phase 3: Configuration Validation & Startup ✅
**Status**: NOT STARTED  
**Duration**: 1 day  
**Goal**: Validate all config at startup, fail fast on bad config

### Phase 3 - What Gets Done
- ✅ Implement `Config.Validate()` method
- ✅ Check all required fields exist
- ✅ Test connectivity to Prometheus/Loki/Tempo/Ollama
- ✅ Add `--validate` flag for dry-run
- ✅ Return meaningful error messages

### Phase 3 - Files to Modify
1. `internal/config/config.go` - Add Validate() method
2. `cmd/agent/main.go` - Call Validate() before Start()
3. `internal/server/server.go` - Add connectivity tests

### Phase 3 - Deliverable
```bash
$ ./helix-agent
Error: Missing required configuration "prometheus.url"
Check config.yaml or set environment variable HELIX_PROMETHEUS_URL

$ ./helix-agent --validate
✓ Configuration valid
✓ Prometheus reachable (http://prometheus:9090)
✓ Loki reachable (http://loki:3100)
✓ Ollama reachable with model qwen2.5:0.5b loaded
```

---

## Phase 4: Database Integration 💾
**Status**: NOT STARTED  
**Duration**: 2 days  
**Goal**: Connect SQLite and save incidents

### Phase 4 - What Gets Done
- ✅ Initialize DB in server.New()
- ✅ Run migrations on startup
- ✅ Implement SaveIncident() method
- ✅ Implement GetIncident() and ListIncidents()
- ✅ Update handlers to save alerts to DB
- ✅ Make /postmortems endpoints work

### Phase 4 - Files to Modify
1. `internal/db/db.go` - Ensure migrations work
2. `internal/server/server.go` - Add DB initialization
3. `internal/server/handlers.go` - Call db.SaveIncident()
4. `internal/models/incident.go` - Add DB methods
5. Add `internal/db/methods.go` - Incident CRUD

### Phase 4 - Deliverable
```bash
$ curl http://localhost:8080/webhook -d @test-alert.json
{"status":"accepted","message":"Processing 1 alerts"}

$ curl http://localhost:8080/postmortems
[{"id":"abc-123","service":"api-service","alert":"HighLatency","status":"firing"}]

$ sqlite3 helix.db "SELECT * FROM incidents LIMIT 1"
abc-123|api-service|HighLatency|firing|2026-03-14...
```

---

## Phase 5: Request Tracing 🔗
**Status**: NOT STARTED  
**Duration**: 1 day  
**Goal**: Add request IDs for correlation across logs

### Phase 5 - What Gets Done
- ✅ Add RequestIDMiddleware
- ✅ Generate uuid for each request
- ✅ Pass through to async operations
- ✅ Include in all logs
- ✅ Return in response headers

### Phase 5 - Files to Modify
1. `internal/server/router.go` - Add middleware
2. `internal/server/handlers.go` - Use request_id in logs
3. `internal/orchestrator/context.go` - Pass request_id
4. `internal/analyzer/rca.go` - Include in logs
5. `internal/db/db.go` - Save request_id with incident

### Phase 5 - Deliverable
```
Request arrives: X-Request-ID: req-12345
├─ Logs: request_id=req-12345
├─ Analysis: request_id=req-12345  
├─ DB: incidents.request_id=req-12345
└─ Response: X-Request-ID: req-12345

All logs for one request can be traced! 🔍
```

---

## Phase 6: Enhanced Health Checks 🏥
**Status**: NOT STARTED  
**Duration**: 1 day  
**Goal**: Deep health checksfor each dependency

### Phase 6 - What Gets Done
- ✅ Check database connectivity
- ✅ Check Prometheus connectivity
- ✅ Check Loki connectivity
- ✅ Check LLM connectivity
- ✅ Return component-by-component status
- ✅ Return 503 if critical component down

### Phase 6 - Files to Modify
1. `internal/server/handlers.go` - Implement HandleHealth
2. `pkg/llm/provider.go` - Add Health() method
3. `internal/clients/prometheus/client.go` - Add Health()
4. `internal/clients/loki/client.go` - Add Health()
5. `internal/db/db.go` - Add Ping()

### Phase 6 - Deliverable
```json
{
  "status": "degraded",
  "timestamp": "2026-03-14T12:30:45Z",
  "checks": {
    "database": true,
    "prometheus": true,
    "loki": false,
    "llm": true
  }
}
HTTP 503 (Loki down, but can still work)
```

---

## Phase 7: Unit Tests - config Package 🧪
**Status**: NOT STARTED  
**Duration**: 1 day  
**Goal**: Add 40+ tests for config package (0% → 80%)

### Phase 7 - What Gets Done
- ✅ Create `internal/config/config_test.go`
- ✅ Test YAML loading
- ✅ Test environment variable overrides
- ✅ Test validation logic
- ✅ Test default values
- ✅ Test timeout parsing

### Phase 7 - Files to Modify
1. Create `internal/config/config_test.go`
2. Create `internal/config/testdata/` fixtures
3. Update `go.mod` if missing test dependencies

### Phase 7 - Deliverable
```bash
go test ./internal/config/... -v
=== RUN   TestLoadConfig
=== RUN   TestValidateRequired
=== RUN   TestEnvOverrides
=== RUN   TestTimeoutParsing
... 40+ tests ...
ok      helixops/internal/config    2.345s  coverage: 85%
```

---

## Phase 8: Unit Tests - analyzer Package 🧪
**Status**: NOT STARTED  
**Duration**: 1 day  
**Goal**: Add 20+ tests for analyzer (0% → 80%)

### Phase 8 - What Gets Done
- ✅ Create `internal/analyzer/rca_test.go`
- ✅ Test with mock LLM provider
- ✅ Test error handling
- ✅ Test prompt building
- ✅ Test response parsing

### Phase 8 - Files to Modify
1. Create `internal/analyzer/rca_test.go`
2. Create mock LLM provider in test file

### Phase 8 - Deliverable
```bash
go test ./internal/analyzer/... -v
=== RUN   TestAnalyzeAlert
=== RUN   TestAnalyzeWithMissingData
=== RUN   TestLLMTimeout
... 20+ tests ...
ok      helixops/internal/analyzer    1.234s  coverage: 82%
```

---

## Phase 9: Unit Tests - db Package 🧪
**Status**: NOT STARTED  
**Duration**: 1.5 days  
**Goal**: Add 30+ tests for db (0% → 80%)

### Phase 9 - What Gets Done
- ✅ Create `internal/db/db_test.go`
- ✅ Use SQLite in-memory for tests
- ✅ Test migrations
- ✅ Test CRUD operations
- ✅ Test transaction handling

### Phase 9 - Files to Modify
1. Create `internal/db/db_test.go`
2. Add test helper functions

### Phase 9 - Deliverable
```bash
go test ./internal/db/... -v
=== RUN   TestMigrations
=== RUN   TestSaveIncident
=== RUN   TestGetIncident
... 30+ tests ...
ok      helixops/internal/db    3.456s  coverage: 85%
```

---

## Phase 10: Integration Tests 🔗
**Status**: NOT STARTED  
**Duration**: 2 days  
**Goal**: End-to-end workflow tests

### Phase 10 - What Gets Done
- ✅ Create `tests/integration/` directory
- ✅ Test webhook → analysis → DB flow
- ✅ Test with real Ollama model
- ✅ Test error scenarios
- ✅ Test graceful shutdown

### Phase 10 - Files to Modify
1. Create `tests/integration/setup.go`
2. Create `tests/integration/webhook_test.go`
3. Create `tests/integration/llm_test.go`
4. Create `tests/integration/db_test.go`

### Phase 10 - Deliverable
```bash
go test ./tests/integration/... -v
=== RUN   TestEndToEndAlertFlow
  ✓ Alert received
  ✓ Context prepared
  ✓ LLM analyzed
  ✓ Incident saved to DB
  ✓ Response returned
=== RUN   TestGracefulDegradation
  ✓ Works with Prometheus down
  ✓ Works with Loki down
... 10+ scenarios ...
ok      helixops/tests/integration    45.123s
```

---

## Phase 11: Rate Limiting 🚦
**Status**: NOT STARTED  
**Duration**: 1 day  
**Goal**: Add rate limiting to webhook endpoint

### Phase 11 - What Gets Done
- ✅ Add golang.org/x/time/rate dependency
- ✅ Implement RateLimitMiddleware
- ✅ Add configurable rate limits
- ✅ Return 429 (Too Many Requests) when exceeded
- ✅ Add metrics for rate limit hits

### Phase 11 - Files to Modify
1. `go.mod` - Add x/time dependency
2. Create `internal/server/middleware.go`
3. `internal/server/router.go` - Use middleware
4. `internal/config/config.go` - Add rate limit config

### Phase 11 - Deliverable
```bash
# Normal load works fine
for i in {1..10}; do curl http://localhost:8080/webhook -d @test.json; done
✓ 10 requests successful

# Exceed rate limit
for i in {1..100}; do curl http://localhost:8080/webhook -d @test.json; done
✓ First 10 succeed
✗ 90 get 429 Too Many Requests
```

---

## Phase 12: Graceful Shutdown 🛑
**Status**: NOT STARTED  
**Duration**: 1 day  
**Goal**: Proper shutdown signal handling

### Phase 12 - What Gets Done
- ✅ Enhance signal handling (SIGTERM/SIGINT)
- ✅ Drain in-flight requests (30s timeout)
- ✅ Close database cleanly
- ✅ Log shutdown events
- ✅ Test with K8s preStop hooks

### Phase 12 - Files to Modify
1. `internal/server/server.go` - Enhance Shutdown()
2. `cmd/agent/main.go` - Test signal handling

### Phase 12 - Deliverable
```bash
# Start agent in background
./helix-agent &

# Send SIGTERM
kill -TERM $PID

# Logs show:
2026/03/14 12:30:45 Shutdown signal received, draining...
2026/03/14 12:30:50 Waiting for 3 in-flight requests...
2026/03/14 12:30:52 Closing database...
2026/03/14 12:30:52 Shutdown complete

# Exit code 0
```

---

## Phase 13: Kubernetes Manifests 🐳
**Status**: NOT STARTED  
**Duration**: 1.5 days  
**Goal**: Production-ready K8s deployment

### Phase 13 - What Gets Done
- ✅ Create `k8s/deployment.yaml`
- ✅ Create `k8s/service.yaml`
- ✅ Create `k8s/configmap.yaml`
- ✅ Create `k8s/pvc.yaml` (SQLite DB)
- ✅ Add health probes
- ✅ Add resource limits

### Phase 13 - Files to Create
1. `k8s/deployment.yaml` - Pod definition
2. `k8s/service.yaml` - LoadBalancer/ClusterIP
3. `k8s/configmap.yaml` - config.yaml
4. `k8s/pvc.yaml` - DB storage
5. `k8s/kustomization.yaml` - Overlay support

### Phase 13 - Deliverable
```bash
kubectl apply -f k8s/
deployment.apps/helixops created
service/helixops created
configmap/helixops-config created
persistentvolumeclaim/helixops-db created

kubectl get pod -o wide
helixops-xxxx  1/1  Running  0  2m  Ready

curl http://localhost:8080/health
{"status":"healthy","timestamp":"2026-03-14T12:30:45Z"}
```

---

## Phase 14: Production Validation Checklist ✅
**Status**: NOT STARTED  
**Duration**: 2 days  
**Goal**: Final pre-production validation

### Phase 14 - What Gets Done
- ✅ Run full test suite: `go test ./... -race -cover` → 70%+
- ✅ Load test: 100 RPS sustained
- ✅ Chaos test: Kill dependencies one-by-one
- ✅ Shutdown test: Drain in-flight requests
- ✅ Config validation: All required fields present
- ✅ Security: No credentials in logs
- ✅ Documentation: Runbooks complete

### Phase 14 - Deliverable
```
✓ Test Coverage: 75%
✓ Load Test: 100 RPS, p99 latency 500ms
✓ Chaos Test: Works with dependencies failing
✓ Shutdown: No errors, clean exit
✓ Security: No tokens in logs
✓ Docs: Complete operations guide
✓ Ready for production! 🚀
```

---



---

# ✅ HOW TO PROCEED

## Start Here: Phase 1
To begin Phase 1 (Error Handling):

```bash
cd /home/kibe/pro/engineeringhub/HelixOps
git checkout -b phase-1-error-handling
# Review this plan: cat PRODUCTION_PLAN.md | grep -A 50 "Phase 1: Error"
```

## Phase Execution Order
```
Phase 1  (1-2 days)  → Error Handling & Resilience
Phase 2  (1 day)     → Structured Logging  
Phase 3  (1 day)     → Configuration Validation
Phase 4  (2 days)    → Database Integration
Phase 5  (1 day)     → Request Tracing
Phase 6  (1 day)     → Enhanced Health Checks
Phase 7  (1 day)     → Unit Tests: config
Phase 8  (1 day)     → Unit Tests: analyzer
Phase 9  (1.5 days)  → Unit Tests: db
Phase 10 (2 days)    → Integration Tests
Phase 11 (1 day)     → Rate Limiting
Phase 12 (1 day)     → Graceful Shutdown
Phase 13 (1.5 days)  → Kubernetes Manifests
Phase 14 (2 days)    → Final Validation
```

**Total Effort**: ~17 days, achievable in ~3-4 weeks with concurrent work

---

# 📊 PROGRESS TRACKING

## Status Board

```
[  NOT_STARTED  ]  Phase 1:  Error Handling & Resilience
[  NOT_STARTED  ]  Phase 2:  Structured Logging
[  NOT_STARTED  ]  Phase 3:  Configuration Validation
[  NOT_STARTED  ]  Phase 4:  Database Integration
[  NOT_STARTED  ]  Phase 5:  Request Tracing
[  NOT_STARTED  ]  Phase 6:  Enhanced Health Checks
[  NOT_STARTED  ]  Phase 7:  Unit Tests - config
[  NOT_STARTED  ]  Phase 8:  Unit Tests - analyzer
[  NOT_STARTED  ]  Phase 9:  Unit Tests - db
[  NOT_STARTED  ]  Phase 10: Integration Tests
[  NOT_STARTED  ]  Phase 11: Rate Limiting
[  NOT_STARTED  ]  Phase 12: Graceful Shutdown
[  NOT_STARTED  ]  Phase 13: Kubernetes Manifests
[  NOT_STARTED  ]  Phase 14: Production Validation
```

---

# 🎯 Success Metrics

### By Phase Completion

| Phase | Metric | Target | Status |
|-------|--------|--------|--------|
| 1 | Error handling | Works with 1 data source down | ❌ |
| 2 | Logging | All JSON structured | ❌ |
| 3 | Config | Validates at startup | ❌ |
| 4 | Database | Incidents persist | ❌ |
| 5 | Tracing | All logs have request_id | ❌ |
| 6 | Health | Deep component checks | ❌ |
| 7-9 | Tests | Unit coverage > 70% | ❌ |
| 10 | Integration | E2E workflows pass | ❌ |
| 11 | Rate Limit | 10 RPS enforced | ❌ |
| 12 | Shutdown | Clean exit < 2s | ❌ |
| 13 | K8s | Replicas, PVC, probes | ❌ |
| 14 | Validation | Production ready | ❌ |

---

# 🚀 QUICK NEXT STEPS

## Immediate Actions

```bash
# 1. Review this plan in detail
cat PRODUCTION_PLAN.md

# 2. Create phase 1 branch
git checkout -b phase/1-error-handling

# 3. Start with Phase 1 files
# - internal/models/analysis_context.go (add Errors field)
# - internal/orchestrator/context.go (graceful degradation)
# - internal/analyzer/rca.go (handle nil fields)

# 4. Run tests to establish baseline
go test ./... -race -cover -v

# 5. Update this plan as you complete phases
# (Search and replace [NOT_STARTED] → [IN_PROGRESS] → [COMPLETED])
```

---

## Known Limitations & Future Work

### Current Known Issues
1. **Postmortem Report Generation** - Uses basic templates, not customizable
2. **Slack Integration** - Webhook URL must be set manually
3. **Rate Limiting** - Very basic, no per-user limits
4. **Metrics Export** - No Prometheus metrics yet (roadmap)
5. **Tracing** - No distributed tracing (OpenTelemetry future)

### Post-MVP Features
- [ ] Multi-tenant support with OIDC
- [ ] Custom postmortem templates
- [ ] Incident correlation (group related alerts)
- [ ] Auto-remediation hooks
- [ ] Slack/Teams interactive commands
- [ ] Cost analysis in postmortems
- [ ] Historical trend analysis

---

## 📚 Documentation References

- [ARCHITECTURE.md](docs/ARCHITECTURE.md) - System design
- [DEPLOYMENT.md](docs/DEPLOYMENT.md) - Deployment procedures
- [CONFIGURATION.md](docs/CONFIGURATION.md) - Configuration reference
- [TESTING.md](docs/TESTING.md) - Testing guide
- [API_REFERENCE.md](docs/API_REFERENCE.md) - API endpoints

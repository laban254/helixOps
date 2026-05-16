# HelixOps — Next Steps & Implementation Plan

> **Updated:** 2026-05-16
> See `PRODUCTION_PLAN.md` for detailed implementation specs.

---

## Phase 1: Production Hardening (Dev)

### P0 — Critical Reliability
- [x] Error handling & graceful degradation — Prometheus/Loki down doesn't crash
- [x] Structured logging (`slog`) with JSON output and `request_id`
- [x] Configuration validation at startup (`--validate` flag)
- [x] Graceful shutdown (SIGTERM drains requests)

### P1 — Data & Observability
- [x] PostgreSQL database with migrations and CRUD
- [x] Request tracing (UUID per alert, propagated through logs/DB)
- [x] Per-component health checks (`/health` returns DB/Prometheus/Loki/LLM status)

### P2 — Hardening
- [ ] Rate limiting — `golang.org/x/time/rate`, configurable, returns `429` (see PRODUCTION_PLAN.md Phase 11)
- [ ] Kubernetes manifests — `k8s/` with deployment, service, configmap, HPA
- [ ] Production validation — load test (100 RPS), chaos test, security scan

---

## Phase 2: Test Coverage

| Package | Current | Target | Priority |
|---------|---------|--------|----------|
| `internal/db` | 0% | 80% | High |
| `internal/output` | 0% | 80% | High |
| `internal/orchestrator` | 0% | 70% | High |
| `internal/clients/github` | 0% | 80% | High |
| `internal/clients/loki` | 0% | 70% | High |
| `internal/config` | 28% | 80% | Medium |
| `internal/server` | 31% | 70% | Medium |
| `internal/clients/prometheus` | 44% | 80% | Medium |
| `internal/clients/tempo` | 64% | 80% | Low |
| `internal/postmortem` | 0% | 70% | Medium |
| `internal/remediation` | 0% | 70% | Medium |
| `internal/logging` | 0% | 70% | Low |
| `internal/mcp` | 0% | 70% | Low |
| `pkg/llm` | 41% | 80% | Medium |
| `internal/analyzer` | 57% | 80% | Low |
| `internal/models` | 100% | — | Done |

Integration tests:
- [ ] Full E2E flow: alert → analysis → DB → Slack
- [ ] Graceful degradation when Prometheus/Loki/DB/LLM is down
- [ ] Invalid payload handling

---

## Phase 3: Growth & Community

- [ ] Demo recording (asciinema/Loom)
- [ ] Live sandbox environment
- [ ] README polish (demo GIF at top, comparison table)
- [ ] Hacker News "Show HN" post
- [ ] Social media distribution (Twitter/X, LinkedIn, Reddit)
- [ ] GitHub Sponsors + FUNDING.yml
- [ ] GitHub Discussions or Discord

---

## Phase 4: Enterprise & Extensions

- [ ] Multi-tenant with OIDC/SSO
- [ ] RBAC for incident data
- [ ] Extended integrations (Datadog, New Relic, CloudWatch)
- [ ] More output channels (Teams, PagerDuty, Opsgenie)
- [ ] Custom postmortem templates
- [ ] Interactive Slack commands
- [ ] Incident correlation and SLO breach prediction

---

*Last Updated: 2026-05-16*

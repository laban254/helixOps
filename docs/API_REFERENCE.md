# HelixOps API Reference

## Overview

HelixOps exposes HTTP endpoints for webhook ingestion, health checks, and postmortem retrieval. This document provides complete API specifications.

---

## Base URL

```
http://localhost:8080
```

In Kubernetes:
```
http://helixops.helixops.svc.cluster.local:8080
```

---

## Endpoints

### 1. Health Check

**Endpoint:** `GET /health`

**Purpose:** Kubernetes liveness probe. Indicates if HelixOps agent is running.

**Response:**

```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "healthy",
  "timestamp": "2025-03-04T10:30:45Z"
}
```

**Status Codes:**
- `200 OK` - Agent is running
- `503 Service Unavailable` - Agent crashed or unhealthy

---

### 2. Readiness Check

**Endpoint:** `GET /ready`

**Purpose:** Kubernetes readiness probe. Indicates if HelixOps is ready to process webhooks.

**Response:**

```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "ready",
  "checks": {
    "database": "ok",
    "config": "ok"
  }
}
```

**Status Codes:**
- `200 OK` - Ready to accept webhooks
- `503 Service Unavailable` - Dependencies not available

---

### 3. Alert Webhook Receiver

**Endpoint:** `POST /webhook`

**Purpose:** Receives Prometheus AlertManager webhook payloads. Acknowledges immediately and processes asynchronously.

**Content-Type:** `application/json`

**Request Body:**

```json
{
  "version": "4",
  "groupKey": "{}:{severity=~\"warning\"}",
  "status": "firing",
  "receiver": "helixops",
  "groupLabels": {
    "severity": "critical"
  },
  "commonLabels": {
    "alertname": "HighLatency",
    "service_name": "cart-service",
    "severity": "critical"
  },
  "commonAnnotations": {
    "summary": "High latency detected on cart-service",
    "description": "Latency has exceeded 500ms for the last 5 minutes"
  },
  "externalURL": "http://alertmanager:9093",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "HighLatency",
        "service_name": "cart-service",
        "severity": "critical"
      },
      "annotations": {
        "summary": "High latency detected",
        "description": "Latency: 523ms (threshold: 500ms)"
      },
      "startsAt": "2025-03-04T10:25:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://prometheus:9090/graph",
      "fingerprint": "a1b2c3d4e5f6"
    }
  ]
}
```

**Required Fields:**
- `alerts[].status` - Must be "firing" or "resolved"
- `alerts[].labels.service_name` - Service identifier for context collection
- `alerts[].labels.alertname` - Alert rule name
- `alerts[].startsAt` - Alert start time (RFC3339)

**Optional Fields:**
- `alerts[].labels.severity` - Alert severity (critical, warning, info)
- `alerts[].annotations.summary` - Human-readable alert description

**Response:**

```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "accepted",
  "message": "Processing 1 alerts",
  "alert_ids": ["a1b2c3d4e5f6"]
}
```

**Status Codes:**
- `200 OK` - Webhook accepted and queued for processing
- `400 Bad Request` - Invalid payload format
- `413 Payload Too Large` - Payload exceeds size limit
- `429 Too Many Requests` - Rate limit exceeded
- `503 Service Unavailable` - Agent not ready

**Processing Behavior:**

1. **Synchronous:** Handler parses and validates payload
2. **Immediate Response:** Returns 200 OK to AlertManager
3. **Asynchronous:** Background goroutine processes analysis
4. **Fired Alerts:** Triggers RCA analysis
5. **Resolved Alerts:** Triggers postmortem generation

**Error Handling:**

Invalid fields are logged but don't block processing:

```bash
# Example: missing service_name
curl -X POST http://localhost:8080/webhook \
  -d '{"alerts":[{"status":"firing","labels":{"alertname":"Test"}}]}'

# Response: 200 OK
# Log: "Skipping alert: missing service_name"
```

---

### 4. List Postmortems

**Endpoint:** `GET /postmortems`

**Purpose:** Retrieve list of resolved incidents (postmortems) from the database.

**Query Parameters:**
- `service_name` (optional) - Filter by service
- `start_time` (optional) - Filter by start time (RFC3339)
- `end_time` (optional) (optional) - Filter by end time (RFC3339)
- `limit` (optional) - Maximum results (default: 50)

**Response:**

```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "postmortems": [
    {
      "id": "pm_abc123",
      "incident_name": "High Latency on cart-service",
      "service_name": "cart-service",
      "date": "2025-03-04T10:30:00Z",
      "duration_minutes": 45,
      "root_cause": "Database query timeout due to missing index",
      "severity": "critical",
      "status": "resolved"
    }
  ],
  "count": 1,
  "total": 1
}
```

**Status Codes:**
- `200 OK` - Success
- `400 Bad Request` - Invalid query parameters
- `503 Service Unavailable` - Database error

---

### 5. Get Postmortem Details

**Endpoint:** `GET /postmortems/{id}`

**Purpose:** Retrieve full postmortem report with analysis details.

**Path Parameters:**
- `id` - Postmortem ID

**Response:**

```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "pm_abc123",
  "incident_name": "High Latency on cart-service",
  "service_name": "cart-service",
  "date": "2025-03-04T10:30:00Z",
  "started_at": "2025-03-04T09:45:00Z",
  "resolved_at": "2025-03-04T10:30:00Z",
  "duration_minutes": 45,
  "root_cause": "Database query timeout due to missing index",
  "impact": "5% of checkout requests failed",
  "detection_method": "Latency spike detected by Prometheus alert",
  "action_items": [
    "Add index on carts.user_id column",
    "Update runbook for database performance",
    "Review slow query logs"
  ],
  "remediation_rules": [
    {
      "title": "Check Database Query Performance",
      "category": "database",
      "suggestion": "Review slow query logs and add missing indexes"
    }
  ],
  "metrics": {
    "latency_p99_before": 145,
    "latency_p99_after": 520,
    "error_rate_before": 0.1,
    "error_rate_after": 5.2,
    "requests_per_second": 1200
  },
  "recent_commits": [
    {
      "sha": "abc1234",
      "author": "alice@example.com",
      "message": "Refactor cart queries",
      "timestamp": "2025-03-04T09:30:00Z",
      "url": "https://github.com/org/repo/commit/abc1234"
    }
  ],
  "markdown_report": "# Incident Postmortem\n..."
}
```

**Status Codes:**
- `200 OK` - Success
- `404 Not Found` - Postmortem ID not found
- `500 Internal Server Error` - Retrieval error

---

## Request/Response Format

### Common Headers

**Request:**
```
Content-Type: application/json
User-Agent: AlertManager/0.26.0
```

**Response:**
```
Content-Type: application/json
X-Request-ID: req_12345678
X-Processing-Time: 250ms
```

### Error Responses

All errors follow this format:

```json
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "invalid_payload",
  "message": "Missing required field: service_name",
  "request_id": "req_abc123",
  "timestamp": "2025-03-04T10:30:45Z"
}
```

---

## Alert Payload Format Details

### Alert Status Values

- `"firing"` - Alert is currently active
- `"resolved"` - Alert has been resolved

### Label Conventions

**Required Labels:**
- `service_name` - Unique identifier for the affected service
- `alertname` - Name of the alert rule

**Recommended Labels:**
- `severity` - Alert severity (critical, warning, info)
- `cluster` - Cluster identifier (for multi-cluster)
- `team` - On-call team responsible

**Example:**

```json
{
  "labels": {
    "alertname": "HighErrorRate",
    "service_name": "payment-service",
    "severity": "critical",
    "cluster": "us-east-1",
    "team": "platform"
  }
}
```

### Annotation Conventions

**Recommended Annotations:**
- `summary` - Short description
- `description` - Detailed description
- `runbook_url` - Link to runbook

**Example:**

```json
{
  "annotations": {
    "summary": "High error rate on payment-service",
    "description": "Error rate has exceeded 5% for 10 minutes",
    "runbook_url": "https://wiki.example.com/runbooks/payment-service"
  }
}
```

---

## Integration Examples

### AlertManager Configuration

```yaml
# alertmanager.yml
global:
  resolve_timeout: 5m

receivers:
- name: 'helixops'
  webhook_configs:
  - url: 'http://helixops:8080/webhook'
    send_resolved: true
    headers:
      X-Custom-Header: 'HelixOps'

route:
  receiver: 'helixops'
  repeat_interval: 1h
```

### Prometheus Alert Rule

```yaml
# prometheus.yml
groups:
- name: application
  rules:
  - alert: HighLatency
    expr: histogram_quantile(0.99, http_request_duration_seconds_bucket{service="cart-service"}) > 0.5
    for: 5m
    labels:
      severity: critical
      service_name: cart-service
    annotations:
      summary: "High latency on {{ $labels.service_name }}"
      description: "P99 latency is {{ $value }}s"
```

### curl Examples

**Test webhook:**

```bash
# Basic test
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "version": "4",
    "status": "firing",
    "alerts": [{
      "status": "firing",
      "labels": {
        "alertname": "TestAlert",
        "service_name": "test-service",
        "severity": "critical"
      },
      "annotations": {
        "summary": "Test alert for verification"
      },
      "startsAt": "2025-03-04T10:00:00Z"
    }]
  }'

# From file
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @test-alert.json

# With authentication (future)
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d @test-alert.json
```

**Get postmortems:**

```bash
# List all
curl http://localhost:8080/postmortems

# Filter by service
curl 'http://localhost:8080/postmortems?service_name=cart-service'

# Get specific postmortem
curl http://localhost:8080/postmortems/pm_abc123
```

---

## Rate Limiting

Current limits (subject to change):
- **Webhook endpoint**: 1000 requests/second per instance
- **Query endpoints**: 100 requests/second per IP

Response when rate limited:

```json
HTTP/1.1 429 Too Many Requests
Retry-After: 60

{
  "error": "rate_limit_exceeded",
  "message": "Too many requests",
  "retry_after_seconds": 60
}
```

---

## Timeouts

- **Webhook processing**: 30 seconds (acknowledged immediately)
- **Context collection**: 10 seconds per source (Prometheus, Loki, GitHub)
- **LLM analysis**: 30 seconds (depends on provider)
- **Total E2E**: ~60 seconds (background)

---

## Backward Compatibility

- Current API version: `v1` (implicit)
- Future API versioning: `POST /api/v2/webhook`
- Deprecation policy: 6-month notice before breaking changes

---

## Future API Enhancements (Phase 3+)

- `POST /api/v1/analyze` - On-demand analysis
- `PUT /api/v1/incidents/{id}/action` - Execute remediation actions
- `GET /api/v1/events` - Server-sent events stream
- Authentication/RBAC support
- GraphQL endpoint option

---

## Support

For API issues:
- Check [TESTING.md](TESTING.md) for test examples
- See [ARCHITECTURE.md](ARCHITECTURE.md) for design context
- Review logs: `kubectl logs -n helixops helix-agent`
- Open GitHub issue: https://github.com/helixops/helixops/issues

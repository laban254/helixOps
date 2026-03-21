# Real-World Examples

Practical examples of deploying and using HelixOps in various scenarios.

---

## Example 1: Kubernetes Production Deployment

### Prerequisites
- Kubernetes cluster (EKS, GKE, AKS, or self-hosted)
- Prometheus & AlertManager already configured
- GitHub repository with deployments
- Slack channel for alerts

### Configuration

```yaml
# config.yaml
app:
  host: "0.0.0.0"
  port: 8080

database:
  enabled: true
  host: "postgres.default.svc.cluster.local"
  port: 5432
  username: "helixops"
  password: "${HELIX_DB_PASSWORD}"  # From Kubernetes secret

llm:
  provider: "anthropic"
  anthropic_api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-3-5-sonnet-20241022"

github:
  token: "${GITHUB_TOKEN}"
  default_org: "mycompany"

output:
  slack:
    enabled: true
    webhook_url: "${SLACK_WEBHOOK_URL}"

analysis:
  metrics_window: 15m
  commits_lookback: 24h
```

### Kubernetes Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: helixops
  namespace: monitoring
spec:
  replicas: 2
  selector:
    matchLabels:
      app: helixops
  template:
    metadata:
      labels:
        app: helixops
    spec:
      containers:
      - name: helixops
        image: helixops/helixops:latest
        ports:
        - containerPort: 8080
        env:
        - name: HELIX_DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: helixops-secrets
              key: db-password
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: helixops-secrets
              key: anthropic-api-key
        - name: GITHUB_TOKEN
          valueFrom:
            secretKeyRef:
              name: helixops-secrets
              key: github-token
        - name: SLACK_WEBHOOK_URL
          valueFrom:
            secretKeyRef:
              name: helixops-secrets
              key: slack-webhook
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: helixops
  namespace: monitoring
spec:
  selector:
    app: helixops
  ports:
  - port: 8080
    targetPort: 8080
```

### AlertManager Configuration

```yaml
route:
  group_by: ['alertname', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'helixops'
  routes:
  - match:
      team: platform
    receiver: 'helixops'
    continue: true

receivers:
- name: 'helixops'
  webhook_configs:
  - url: 'http://helixops.monitoring.svc.cluster.local:8080/webhook'
    send_resolved: true
```

---

## Example 2: Privacy-First Local Deployment

### Use Case
- Sensitive infrastructure
- Cannot send data to external LLMs
- Want complete data isolation

### Configuration

```yaml
# config.yaml - Privacy-first config
app:
  host: "0.0.0.0"
  port: 8080

database:
  enabled: true
  host: "localhost"
  port: 5432
  username: "helixops"
  password: "local_dev_password"

llm:
  provider: "ollama"
  ollama_url: "http://localhost:11434"
  model: "llama3.2"  # Or "mistral", "codellama"

github:
  token: "${GITHUB_TOKEN}"
  default_org: "mycompany"

output:
  slack:
    enabled: false  # Disable external output
  markdown:
    enabled: true
    path: "/var/log/helixops/postmortems"

# No external LLM calls - all processing stays local
```

### Running with Ollama

```bash
# Start Ollama with a suitable model
ollama serve &
ollama pull llama3.2

# Start PostgreSQL
docker run -d \
  --name postgres \
  -e POSTGRES_PASSWORD=local_dev_password \
  -e POSTGRES_DB=helixops \
  -p 5432:5432 \
  postgres:15

# Start HelixOps
docker-compose up -d
go run ./cmd/mcp
```

---

## Example 3: Multi-Service GitHub Organization

### Use Case
- Multiple services across different GitHub repos
- Need to map services to correct repositories

### Configuration

```yaml
github:
  token: "${GITHUB_TOKEN}"
  default_org: "mycompany"
  
  # Map service names to GitHub repos
  service_mapping:
    api-gateway: "mycompany/api-gateway"
    user-service: "mycompany/user-service"
    payment-service: "mycompany/payments"
    notification-service: "mycompany/notifications"
    analytics: "mycompany/data-analytics"
```

### How It Works

When an alert fires for `payment-service`:
1. HelixOps receives the alert with labels: `service="payment-service"`
2. Looks up the mapping in config
3. Finds `mycompany/payments` repo
4. Fetches recent commits from that repository

---

## Example 4: High-Volume Alert Processing

### Use Case
- Large cluster with many alerts
- Need to handle high throughput
- Want to avoid LLM rate limits

### Configuration

```yaml
app:
  host: "0.0.0.0"
  port: 8080
  
  # Rate limiting
  rate_limit:
    requests_per_second: 100
    burst: 200

analysis:
  # Smaller windows for faster processing
  metrics_window: 5m
  commits_lookback: 4h
  
  # Concurrency settings
  max_concurrent_analyses: 10
  analysis_timeout: 30s

llm:
  provider: "openai"
  openai_api_key: "${OPENAI_API_KEY}"
  model: "gpt-4o-mini"  # Faster, cheaper model
  # Or use caching to reduce LLM calls
  cache_enabled: true

output:
  slack:
    enabled: true
    webhook_url: "${SLACK_WEBHOOK_URL}"
    # Rate limit Slack notifications
    rate_limit_per_minute: 20
```

### AlertManager Grouping

```yaml
route:
  # Group alerts by service to avoid spam
  group_by: ['service', 'alertname']
  group_wait: 30s
  group_interval: 1m
  repeat_interval: 1h
  receiver: 'helixops'
```

---

## Example 5: Custom Postmortem Storage

### Use Case
- Store postmortems in S3
- Integrate with existing document system

### Configuration

```yaml
output:
  markdown:
    enabled: true
    path: "/var/log/helixops/postmortems"
  
  # Custom webhook for postmortem storage
  custom_webhooks:
    - name: "s3-archive"
      url: "https://my-internal-api.example.com/archive"
      headers:
        Authorization: "Bearer ${ARCHIVE_API_TOKEN}"
      events:
        - incident_created
        - analysis_complete
```

### Custom Output Handler

You can also write a custom output handler:

```go
// internal/output/custom.go
package output

type CustomHandler struct {
    client *http.Client
    url    string
    token  string
}

func (h *CustomHandler) Send(ctx context.Context, result *analyzer.AnalysisResult) error {
    payload := map[string]interface{}{
        "incident_id": result.IncidentID,
        "summary":     result.Summary,
        "root_cause":  result.RootCause,
        "timestamp":  result.Timestamp,
    }
    
    req, _ := http.NewRequestWithContext(ctx, "POST", h.url, nil)
    req.Header.Set("Authorization", "Bearer "+h.token)
    req.Header.Set("Content-Type", "application/json")
    
    return h.client.Do(req)
}
```

---

## Example 6: Development & Testing

### Local Development Config

```yaml
# config.dev.yaml
app:
  host: "0.0.0.0"
  port: 8080
  debug: true

database:
  enabled: true
  host: "localhost"
  port: 5432
  username: "helixops"
  password: "dev"

llm:
  provider: "ollama"
  ollama_url: "http://localhost:11434"
  model: "llama3.2"

github:
  token: "${GITHUB_TOKEN}"
  default_org: "testorg"

# Mock clients for testing
clients:
  prometheus:
    url: "http://localhost:9090"
  loki:
    url: "http://localhost:3100"

output:
  slack:
    enabled: false  # Disable in dev
  markdown:
    enabled: true
    path: "./postmortems-dev"
```

### Running Tests

```bash
# Run unit tests
go test ./... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Integration tests with Docker
docker-compose up -d
go test ./... -tags=integration
docker-compose down
```

---

## Debugging Tips

### Enable Debug Logging

```yaml
app:
  debug: true
  log_level: "debug"
```

### Check Component Status

```bash
# Health check
curl http://localhost:8080/health

# Ready check
curl http://localhost:8080/ready

# List recent incidents
curl http://localhost:8080/api/v1/incidents

# Get specific incident
curl http://localhost:8080/api/v1/incidents/<id>
```

### Common Issues

**No GitHub commits found:**
- Verify `service_mapping` in config
- Check GitHub token has repo access

**Missing metrics:**
- Verify Prometheus is scraping the service
- Check metric labels match service name

**LLM errors:**
- Verify API key is set
- Check rate limits
- Try a different model

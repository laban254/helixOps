# HelixOps Configuration Guide

Complete reference for configuring HelixOps agent settings.

---

## Configuration Sources (Priority)

HelixOps loads configuration in this order (later overrides earlier):

1. **Defaults** (hardcoded)
2. **config.yaml** file
3. **Environment variables**

---

## Full Configuration Reference

### Root Configuration

```yaml
# config.yaml

# Application settings
app:
  host: 0.0.0.0              # Bind address for HTTP server
  port: 8080                 # HTTP server port
  log_level: info            # Log verbosity: debug, info, warn, error

# Prometheus integration
prometheus:
  url: http://prometheus:9090
  timeout: 10s               # Query timeout

# Loki (Logs) integration
loki:
  url: http://loki:3100
  timeout: 10s               # Query timeout

# Grafana Tempo (Traces) integration - optional
tempo:
  url: http://tempo:3200
  timeout: 10s
  enabled: false             # Set to true if using Tempo
  slow_span_threshold_ms: 500
  search_limit: 100

# GitHub integration
github:
  api_url: https://api.github.com
  token_env: GITHUB_TOKEN    # Environment variable to read token from

# LLM Provider configuration
llm:
  provider: openai           # Options: openai, anthropic, ollama
  model: gpt-4o              # Model name
  temperature: 0.7           # Creativity (0.0 = deterministic, 1.0 = creative)
  max_tokens: 2000           # Max response length
  
  # For Ollama (local LLM)
  ollama_url: http://ollama:11434
  ollama_model: llama2

# Output channels
output:
  slack:
    enabled: true
    webhook_url_env: SLACK_WEBHOOK_URL
  
  discord:
    enabled: false
    webhook_url_env: DISCORD_WEBHOOK_URL
  
  markdown:
    enabled: true
    output_dir: /data/reports

# Analysis parameters
analysis:
  metrics_window: 15m        # Time window for metric queries (±15min around alert)
  commits_lookback: 24h      # How far back to look for commits
  logs_lookback: 1h         # How far back to look for error logs

# Database (PostgreSQL) - for incident history
database:
  enabled: true             # Enable incident history storage
  host: postgres          # PostgreSQL host
  port: 5432              # PostgreSQL port
  user: helixops          # Database user
  dbname: helixops        # Database name
  sslmode: disable        # SSL mode (disable/require)
```

---

## Configuration Sections

### Application Settings

```yaml
app:
  # Bind address (use 0.0.0.0 for all interfaces)
  host: 0.0.0.0
  
  # HTTP port (Kubernetes default: 8080)
  port: 8080
  
  # Log level
  log_level: info
  # Options:
  # - debug  : Verbose, includes all LLM prompts
  # - info   : Standard operational logs
  # - warn   : Warnings and errors only
  # - error  : Errors only
```

**Environment Override:**
```bash
export HELIX_APP_PORT=9090
export HELIX_APP_LOG_LEVEL=debug
```

---

### Prometheus Configuration

```yaml
prometheus:
  # Prometheus HTTP endpoint
  url: http://prometheus:9090
  
  # Query timeout (HelixOps gives up if Prometheus takes longer)
  timeout: 10s
  
  # Default golden signals queried:
  # - Latency (p99)
  # - Error Rate
  # - Requests Per Second
```

**Environment Override:**
```bash
export HELIX_PROMETHEUS_URL=http://prometheus.monitoring:9090
export HELIX_PROMETHEUS_TIMEOUT=15s
```

**Examples:**

```bash
# Test connectivity
curl http://prometheus:9090/api/v1/query?query=up

# Test with auth (future)
export HELIX_PROMETHEUS_URL=http://user:pass@prometheus:9090
```

---

### Loki Configuration

```yaml
loki:
  # Loki HTTP endpoint
  url: http://loki:3100
  
  # Query timeout
  timeout: 10s
```

**Environment Override:**
```bash
export HELIX_LOKI_URL=http://loki.logging:3100
export HELIX_LOKI_TIMEOUT=15s
```

**Loki Setup:**

Ensure Loki is configured to scrape logs from your services:

```yaml
# loki-config.yaml
scrape_configs:
- job_name: kubernetes-pods
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: pod
  - source_labels: [__meta_kubernetes_namespace]
    target_label: namespace
  - source_labels: [__meta_kubernetes_pod_label_app]
    target_label: job
```

---

### Tempo Configuration

```yaml
tempo:
  # Tempo HTTP endpoint
  url: http://tempo:3200
  
  # Query timeout
  timeout: 10s
  
  # Enable trace collection (optional)
  enabled: false
  
  # Span duration threshold (ms) to consider "slow"
  slow_span_threshold_ms: 500
  
  # Max traces to return in search
  search_limit: 100
```

**Environment Override:**
```bash
export HELIX_TEMPO_ENABLED=true
export HELIX_TEMPO_URL=http://tempo:3200
```

**When to Enable:**
- You're already using Tempo for tracing
- You want correlation between alerts and slow spans
- Optional but recommended for complete observability

---

### GitHub Configuration

```yaml
github:
  # GitHub API endpoint
  api_url: https://api.github.com
  # For GitHub Enterprise: https://your-domain/api/v3
  
  # Environment variable containing GitHub token
  token_env: GITHUB_TOKEN
```

**Environment Override:**
```bash
export HELIX_GITHUB_API_URL=https://github.company.com/api/v3
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxx  # GitHub Personal Access Token
```

**GitHub Token Setup:**

1. Create Personal Access Token:
   - Settings → Developer Settings → Personal Access Tokens
   - Scopes needed: `repo` (for commit history)

2. Store securely:
   ```bash
   # Local development
   export GITHUB_TOKEN=ghp_your_token
   
   # Kubernetes
   kubectl create secret generic github-secret \
     --from-literal=token=$GITHUB_TOKEN \
     -n helixops
   ```

**Service → Repository Mapping:**

Configure service-to-repo mappings in config:

```yaml
github:
  api_url: https://api.github.com
  token_env: GITHUB_TOKEN
  default_org: myorg              # Default org for repos
  service_mapping:                # Optional: explicit mappings
    cart-service: myorg/cart
    payment-service: myorg/payment
    order-service: myorg/order
```

---

### LLM Provider Configuration

#### OpenAI

```yaml
llm:
  provider: openai
  model: gpt-4o              # Latest: gpt-4o, gpt-4, gpt-3.5-turbo
  temperature: 0.7           # 0.0-1.0
  max_tokens: 2000           # Limit response length
  # API key loaded from environment: OPENAI_API_KEY
```

**Environment:**
```bash
export OPENAI_API_KEY=sk_live_xxxxxxxxxxxxxxxxxxxx
```

**Cost Estimate:**
- GPT-4o: ~$0.015 per 1000 tokens input
- Typical analysis: 2000 tokens = $0.03 per incident
- 100 incidents/day = ~$3/day

#### Anthropic (Claude)

```yaml
llm:
  provider: anthropic
  model: claude-3-5-sonnet   # Options: opus, sonnet, haiku
  temperature: 0.7
  max_tokens: 2000
  # API key loaded from environment: ANTHROPIC_API_KEY
```

**Environment:**
```bash
export ANTHROPIC_API_KEY=sk-ant-xxxxxxxxxxxxxxxxxxxx
```

**Cost Estimate:**
- Claude 3.5 Sonnet: ~$0.003 per 1000 tokens input
- Typical analysis: 2000 tokens = $0.006 per incident
- 100 incidents/day = ~$0.60/day

#### Ollama (Private/Local)

```yaml
llm:
  provider: ollama
  ollama_url: http://ollama:11434
  ollama_model: llama2        # Installed model name
  temperature: 0.5            # Lower for consistency
  max_tokens: 1500
  # No API key required (local)
```

**Setup:**

```bash
# Install Ollama (macOS, Linux, Windows)
# https://ollama.ai

# Run Ollama server
ollama serve

# In another terminal, pull model
ollama pull llama2             # 7B, ~4GB
# Or
ollama pull mistral            # 7B, ~4GB, faster
```

**Environment:**
```bash
export HELIX_LLM_PROVIDER=ollama
export HELIX_LLM_OLLAMA_URL=http://localhost:11434
export HELIX_LLM_OLLAMA_MODEL=llama2
```

**Advantages:**
- ✅ Private (no external API calls)
- ✅ No costs
- ✅ Works offline
- ❌ Slower inference (depends on hardware)
- ❌ Lower quality vs cloud LLMs

---

### Output Configuration

#### Slack

```yaml
output:
  slack:
    enabled: true
    webhook_url_env: SLACK_WEBHOOK_URL
```

**Setup:**

1. Create Slack app:
   - https://api.slack.com/apps
   - Click "Create New App"

2. Enable Incoming Webhooks:
   - Features → Incoming Webhooks
   - Click "Add New Webhook to Workspace"
   - Select target channel

3. Copy webhook URL:
   ```bash
   export SLACK_WEBHOOK_URL=https://hooks.slack.com/services/<your-workspace-id>/<your-app-id>/<your-token>
   ```

**Environment:**
```bash
export SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
```

#### Discord

```yaml
output:
  discord:
    enabled: true
    webhook_url_env: DISCORD_WEBHOOK_URL
```

**Setup:**

1. Create Discord webhook:
   - Server Settings → Integrations → Webhooks
   - Click "New Webhook"
   - Choose channel

2. Copy webhook URL:
   ```bash
   export DISCORD_WEBHOOK_URL=https://discordapp.com/api/webhooks/...
   ```

#### Markdown Reports

```yaml
output:
  markdown:
    enabled: true
    output_dir: /data/reports  # Where to save .md files
```

**Features:**
- ✅ Compliance (audit trail)
- ✅ Works offline
- ✅ Git-friendly
- ✅ GDPR (data stays local)

**Kubernetes:**
```bash
# Reports stored in PersistentVolume
kubectl logs -n helixops -l app=helixops  # See file paths
kubectl exec -it -n helixops helix-agent -- ls /data/reports
```

---

### Analysis Parameters

```yaml
analysis:
  # Time window around alert for metric queries
  # Example: If alert at 10:00, query metrics for [09:45, 10:15]
  metrics_window: 15m
  
  # How far back to look for code changes
  commits_lookback: 24h
```

**Options:**

```yaml
# Aggressive (more context, slower)
analysis:
  metrics_window: 30m
  commits_lookback: 7d

# Conservative (faster, less context)
analysis:
  metrics_window: 5m
  commits_lookback: 4h
```

**Environment:**
```bash
export HELIX_ANALYSIS_METRICS_WINDOW=20m
export HELIX_ANALYSIS_COMMITS_LOOKBACK=48h
```

---

### Database Configuration (PostgreSQL)

```yaml
database:
  # Enable PostgreSQL database for incident history
  enabled: true
  
  # PostgreSQL connection settings
  host: postgres        # or your PostgreSQL host
  port: 5432
  user: helixops
  dbname: helixops
  sslmode: disable     # disable for local, require for production
```

**Features:**
- ✅ Stores all incidents (open and resolved)
- ✅ Tracks root cause analysis results
- ✅ Query past incidents via API
- ✅ Scales for high alert volume
- ✅ Works with existing PostgreSQL infrastructure

**Environment:**
```bash
export HELIX_DATABASE_ENABLED=true
export HELIX_DB_HOST=postgres
export HELIX_DB_PASSWORD=your_password
```

**Kubernetes:**
```bash
# Use a PostgreSQL operator or external service
database:
  enabled: true
  host: postgres.namespace.svc.cluster.local
  port: 5432
  user: helixops
  dbname: helixops
  sslmode: require
```

---

## Example Configurations

### Development (Local with Ollama)

```yaml
app:
  host: 0.0.0.0
  port: 8080
  log_level: debug

prometheus:
  url: http://prometheus:9090
  timeout: 10s

loki:
  url: http://loki:3100
  timeout: 10s

github:
  api_url: https://api.github.com
  token_env: GITHUB_TOKEN
  default_org: myorg

llm:
  provider: ollama
  ollama_url: http://ollama:11434
  ollama_model: llama2
  temperature: 0.5

output:
  slack:
    enabled: false
  markdown:
    enabled: true
    output_dir: ./reports

database:
  enabled: true
  host: postgres
  port: 5432
  user: helixops
  dbname: helixops
  sslmode: disable
```

### Production (Cloud LLM)

```yaml
app:
  host: 0.0.0.0
  port: 8080
  log_level: warn

prometheus:
  url: http://prometheus.monitoring.svc.cluster.local:9090
  timeout: 10s

loki:
  url: http://loki.logging.svc.cluster.local:3100
  timeout: 10s

tempo:
  url: http://tempo.tracing.svc.cluster.local:3200
  enabled: true
  timeout: 10s

github:
  api_url: https://api.github.com
  token_env: GITHUB_TOKEN

llm:
  provider: openai
  model: gpt-4o
  temperature: 0.7
  max_tokens: 2000

output:
  slack:
    enabled: true
    webhook_url_env: SLACK_WEBHOOK_URL
  markdown:
    enabled: true
    output_dir: /data/reports
```

### Enterprise (On-Premise)

```yaml
app:
  host: 0.0.0.0
  port: 8080
  log_level: info

prometheus:
  url: http://prometheus.internal:9090
  timeout: 15s

loki:
  url: http://loki.internal:3100
  timeout: 15s

tempo:
  url: http://tempo.internal:3200
  enabled: true
  timeout: 15s

github:
  api_url: https://github.company.internal/api/v3
  token_env: GITHUB_TOKEN

llm:
  provider: ollama           # Private LLM
  ollama_url: http://llm.internal:11434
  ollama_model: llama2
  temperature: 0.5

output:
  slack:
    enabled: true
    webhook_url_env: SLACK_WEBHOOK_URL
  markdown:
    enabled: true
    output_dir: /data/reports
```

---

## Environment Variables Reference

| Variable | Description | Example |
|----------|-------------|---------|
| `HELIX_APP_HOST` | Bind address | `0.0.0.0` |
| `HELIX_APP_PORT` | HTTP port | `8080` |
| `HELIX_APP_LOG_LEVEL` | Log level | `info`, `debug` |
| `HELIX_PROMETHEUS_URL` | Prometheus endpoint | `http://prometheus:9090` |
| `HELIX_PROMETHEUS_TIMEOUT` | Prometheus timeout | `10s` |
| `HELIX_LOKI_URL` | Loki endpoint | `http://loki:3100` |
| `HELIX_LOKI_TIMEOUT` | Loki timeout | `10s` |
| `HELIX_TEMPO_ENABLED` | Enable Tempo | `true`, `false` |
| `HELIX_TEMPO_URL` | Tempo endpoint | `http://tempo:3200` |
| `HELIX_TEMPO_TIMEOUT` | Tempo timeout | `10s` |
| `HELIX_GITHUB_API_URL` | GitHub endpoint | `https://api.github.com` |
| `GITHUB_TOKEN` | GitHub token | `ghp_xxxx` |
| `HELIX_LLM_PROVIDER` | LLM provider | `openai`, `anthropic`, `ollama` |
| `HELIX_LLM_MODEL` | LLM model name | `gpt-4o`, `claude-3-5-sonnet` |
| `HELIX_LLM_TEMPERATURE` | LLM temperature | `0.7` |
| `HELIX_LLM_MAX_TOKENS` | LLM max tokens | `2000` |
| `OPENAI_API_KEY` | OpenAI API key | `sk_live_xxxx` |
| `ANTHROPIC_API_KEY` | Anthropic API key | `sk-ant-xxxx` |
| `HELIX_LLM_OLLAMA_URL` | Ollama endpoint | `http://ollama:11434` |
| `HELIX_LLM_OLLAMA_MODEL` | Ollama model | `llama2` |
| `SLACK_WEBHOOK_URL` | Slack webhook | `https://hooks.slack.com/...` |
| `DISCORD_WEBHOOK_URL` | Discord webhook | `https://discordapp.com/api/...` |
| `HELIX_OUTPUT_MARKDOWN_DIR` | Report directory | `/data/reports` |
| `HELIX_ANALYSIS_METRICS_WINDOW` | Metrics time window | `15m` |
| `HELIX_ANALYSIS_COMMITS_LOOKBACK` | Commits lookback | `24h` |

---

## Validation

Check your configuration:

```bash
# Verify file syntax
go run cmd/mcp/main.go --config=config.yaml --validate

# Test connections (future)
helix-agent --test-connections

# Output current config (sanitized)
helix-agent --show-config
```

---

## Troubleshooting Configuration

### "Config file not found"

```bash
# Check path
ls -la config.yaml

# Set explicit path
export CONFIG_FILE=/etc/helixops/config.yaml
helix-agent
```

### "Invalid provider: xyz"

Check supported providers:
- `openai`
- `anthropic`
- `ollama`

### "Prometheus unreachable"

```bash
# Test connectivity
curl -v http://prometheus:9090/api/v1/query?query=up

# Check configuration
echo "prometheus.url=$HELIX_PROMETHEUS_URL"
```

### "Out of memory"

Reduce `max_tokens` or use smaller Ollama model:

```yaml
llm:
  max_tokens: 1000         # Reduce from 2000
```

---

## Next Steps

1. Copy [config.yaml](../config.yaml) to your deployment
2. Update with your infrastructure details
3. Set environment variables for secrets
4. Test with [TESTING.md](TESTING.md)
5. Deploy using [DEPLOYMENT.md](DEPLOYMENT.md)

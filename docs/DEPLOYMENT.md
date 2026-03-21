# HelixOps Deployment Guide

This guide covers how to deploy HelixOps in different environments.

---

## Prerequisites

- **Kubernetes 1.20+** (for production)
- **Prometheus** with AlertManager configured
- **Loki** or other log aggregator (optional, but recommended)
- **GitHub account** with token (for commit history)
- **LLM API key** (OpenAI/Anthropic) or **Ollama** running locally

---

## Quick Start: Docker Compose (Development)

### 1. Clone and Navigate

```bash
git clone https://github.com/helixops/helixops.git
cd helixops
```

### 2. Configure Environment

```bash
# Copy example config
cp config.yaml config.yaml.local

# Edit with your settings
nano config.yaml.local
```

**Minimal config:**

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
  token_env: GITHUB_TOKEN  # Set via environment

llm:
  provider: ollama  # Use local for dev
  ollama_url: http://ollama:11434
  ollama_model: llama2
  temperature: 0.7
```

### 3. Start Stack

```bash
# Set required environment variables
export GITHUB_TOKEN=your_github_token_here

# Start all services
docker-compose up -d

# Check status
docker-compose ps

# View logs (Omit service name to see all)
docker-compose logs -f helix-agent
```

### 4. Verify

```bash
# Health check
curl http://localhost:8080/health

# Send test alert
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @test-alert.json

# Expected response
# {"status":"accepted","message":"Processing 1 alerts"}
```

### 5. Cleanup

```bash
docker-compose down
```

---

## Production Deployment: Kubernetes

### 1. Prerequisites

- Helm 3+ installed
- kubectl configured to your cluster
- Prometheus + Alertmanager + Loki already deployed

### 2. Create Namespace

```bash
kubectl create namespace helixops
```

### 3. Create Secrets

```bash
# GitHub token
kubectl create secret generic github-secret \
  --from-literal=token=${GITHUB_TOKEN} \
  -n helixops

# LLM API key (if using cloud)
kubectl create secret generic llm-secret \
  --from-literal=api-key=${OPENAI_API_KEY} \
  -n helixops

# Slack webhook (for output)
kubectl create secret generic output-secret \
  --from-literal=slack-webhook-url=${SLACK_WEBHOOK_URL} \
  -n helixops
```

### 4. Create ConfigMap

```bash
# Create config.yaml file
cat > config.yaml <<EOF
app:
  host: 0.0.0.0
  port: 8080
  log_level: info

prometheus:
  url: http://prometheus.shared-services.svc.cluster.local:9090
  timeout: 10s

loki:
  url: http://loki.shared-services.svc.cluster.local:3100
  timeout: 10s

github:
  api_url: https://api.github.com
  token_env: GITHUB_TOKEN

llm:
  provider: openai  # Use cloud LLM for production
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
EOF

# Create ConfigMap
kubectl create configmap helixops-config \
  --from-file=config.yaml \
  -n helixops
```

### 5. Deploy Using Helm

**Option A: Use existing Helm chart** (if available)

```bash
helm repo add helixops https://charts.helixops.io
helm repo update

helm install helixops helixops/helixops \
  -n helixops \
  --set image.tag=v1.0.0 \
  --set prometheus.url=http://prometheus:9090 \
  --set loki.url=http://loki:3100 \
  --set llm.provider=openai \
  --set secrets.github.enabled=true \
  --set secrets.llm.enabled=true
```

**Option B: Manual deployment with Kubernetes manifests**

Create `helixops-deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: helixops-agent
  namespace: helixops
  labels:
    app: helixops
spec:
  replicas: 1
  selector:
    matchLabels:
      app: helixops
  template:
    metadata:
      labels:
        app: helixops
    spec:
      serviceAccountName: helixops
      containers:
      - name: agent
        image: helixops:latest  # Use your registry
        imagePullPolicy: IfNotPresent
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP
        
        env:
        - name: GITHUB_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-secret
              key: token
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: llm-secret
              key: api-key
        - name: SLACK_WEBHOOK_URL
          valueFrom:
            secretKeyRef:
              name: output-secret
              key: slack-webhook-url
        
        volumeMounts:
        - name: config
          mountPath: /etc/helixops
          readOnly: true
        - name: data
          mountPath: /data
        
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 2
        
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      
      volumes:
      - name: config
        configMap:
          name: helixops-config
      - name: data
        emptyDir: {}

---
apiVersion: v1
kind: Service
metadata:
  name: helixops
  namespace: helixops
  labels:
    app: helixops
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 8080
    targetPort: http
    protocol: TCP
  selector:
    app: helixops

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: helixops
  namespace: helixops

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: helixops
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: helixops
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: helixops
subjects:
- kind: ServiceAccount
  name: helixops
  namespace: helixops
```

Deploy:

```bash
kubectl apply -f helixops-deployment.yaml
```

### 6. Configure AlertManager to Send Webhooks

Edit AlertManager ConfigMap:

```yaml
global:
  resolve_timeout: 5m

route:
  receiver: 'helixops'
  group_by: ['alertname', 'cluster', 'service']

receivers:
- name: 'helixops'
  webhook_configs:
  - url: 'http://helixops.helixops.svc.cluster.local:8080/webhook'
    send_resolved: true
```

Apply:

```bash
kubectl apply -f alertmanager-configmap.yaml
# Or, if using Prometheus Operator:
kubectl edit alertmanagerconfig -n monitoring
```

### 7. Verify Deployment

```bash
# Check pod status
kubectl get pods -n helixops

# Check logs
kubectl logs -n helixops -l app=helixops -f

# Port-forward to test
kubectl port-forward -n helixops svc/helixops 8080:8080

# Test health
curl localhost:8080/health

# Test webhook
curl -X POST localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @test-alert.json
```

---

## Production Deployment: Binary (VMs/Servers)

### 1. Build Binary

```bash
git clone https://github.com/helixops/helixops.git
cd helixops
go build -o helix-agent ./cmd/agent
```

### 2. Create systemd Service

Create `/etc/systemd/system/helixops.service`:

```ini
[Unit]
Description=HelixOps Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
User=helixops
WorkingDirectory=/opt/helixops
Environment="CONFIG_FILE=/etc/helixops/config.yaml"
Environment="GITHUB_TOKEN=your_token"
Environment="OPENAI_API_KEY=your_key"
Environment="SLACK_WEBHOOK_URL=your_webhook"

ExecStart=/opt/helixops/helix-agent
Restart=always
RestartSec=10s

# Resource limits
MemoryLimit=512M
CPUAccounting=true
MemoryAccounting=true

[Install]
WantedBy=multi-user.target
```

### 3. Start Service

```bash
sudo mkdir -p /opt/helixops /etc/helixops /var/log/helixops
sudo cp helix-agent /opt/helixops/
sudo cp config.yaml /etc/helixops/
sudo chown -R helixops:helixops /opt/helixops /etc/helixops /var/log/helixops

sudo systemctl daemon-reload
sudo systemctl enable helixops
sudo systemctl start helixops

# Check status
sudo systemctl status helixops

# View logs
sudo journalctl -u helixops -f
```

---

## Local Deployment with Ollama (Privacy-First)

### 1. Install Ollama

```bash
# Docker
docker run -d -p 11434:11434 -v ollama:/root/.ollama ollama/ollama

# Or native installation
# See: https://ollama.ai
```

### 2. Download Model

```bash
# Download Llama 2 (7B, ~4GB)
docker exec ollama ollama pull llama2

# Or Mistral (smaller, ~3GB)
docker exec ollama ollama pull mistral
```

### 3. Configure HelixOps

```yaml
llm:
  provider: ollama
  ollama_url: http://localhost:11434  # Adjust for your setup
  ollama_model: llama2
  temperature: 0.5
  max_tokens: 1500
```

### 4. Verify

```bash
# Test Ollama endpoint
curl http://localhost:11434/api/tags

# Test HelixOps
helix-agent  # Starts with local LLM
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @test-alert.json
```

---

## Monitoring HelixOps

### Prometheus Metrics (Future)

Planned for Phase 3+

```promql
# Alerts processed
helixops_alerts_processed_total

# Analysis latency
helixops_analysis_duration_seconds

# LLM API errors
helixops_llm_errors_total
```

### Application Logs

Configure log aggregation to centralize logs:

```bash
# JSON structured logs (recommended)
kubectl logs -n helixops helix-agent --timestamps

# View specific level
kubectl logs -n helixops helix-agent | grep ERROR
```

### Health Checks

```bash
# Liveness (is agent running?)
curl http://localhost:8080/health

# Readiness (is agent ready to accept webhooks?)
curl http://localhost:8080/ready

# Dependencies
GET /api/v1/status/prometheus
GET /api/v1/status/loki
GET /api/v1/status/github
GET /api/v1/status/llm
```

---

## Troubleshooting

### Pod won't start

```bash
kubectl describe pod -n helixops helix-agent
kubectl logs -n helixops helix-agent --previous
```

### Webhooks not received

```bash
# Check AlertManager configuration
kubectl get cm -n monitoring alertmanager-config -o yaml

# Verify connectivity
kubectl exec -it -n helixops <pod> -- bash
curl -i http://helixops:8080/health
```

### LLM errors

```bash
# Check API key
echo $OPENAI_API_KEY | head -c 20

# Test LLM API directly
curl -X POST https://api.openai.com/v1/chat/completions \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"test"}]}'
```

### Database locked errors

```bash
# Check if process is running
ps aux | grep helix-agent

# Restart pod
kubectl rollout restart deployment/helixops-agent -n helixops
```

---

## Backup and Recovery

### Backup PostgreSQL Database

```bash
# Local backup
cp /var/lib/helixops/helixops.db /backup/helixops-$(date +%Y%m%d).db

# Kubernetes backup
kubectl exec -n helixops helix-agent -- cp /data/helixops.db /tmp/backup.db
kubectl cp helixops/helix-agent:/tmp/backup.db ./helixops-backup.db
```

### Restore

```bash
# Copy database back
kubectl cp ./helixops-backup.db helixops/helix-agent:/data/helixops.db

# Restart pod
kubectl rollout restart deployment/helixops-agent -n helixops
```

---

## Scaling Considerations

### Single Instance (Recommended for MVP)

- Suitable for single cluster
- Webhook processing: ~100 alerts/sec
- Database: PostgreSQL

### Future Multi-Instance (Phase 3)

When deploying multiple agents:

```yaml
replicas: 3  # Multiple instances
```

Requirements:
- Shared database (PostgreSQL for cloud plane)
- Load balancer for webhook distribution
- Distributed tracing coordination

---

## Security Best Practices

1. **Network Policies**
   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: helixops-network-policy
   spec:
     podSelector:
       matchLabels:
         app: helixops
     ingress:
     - from:
       - podSelector:
           matchLabels:
             app: alertmanager
   ```

2. **Pod Security Policy**
   ```yaml
   runAsNonRoot: true
   runAsUser: 65534
   readOnlyRootFilesystem: true
   ```

3. **Secrets Management**
   - Use Sealed Secrets for GitOps
   - Or use environment-specific deployment platforms

4. **RBAC**
   - Minimal permissions (see deployment manifest)
   - No cluster-admin needed

---

## Uninstall

### Kubernetes

```bash
kubectl delete deployment helixops-agent -n helixops
kubectl delete service helixops -n helixops
kubectl delete configmap helixops-config -n helixops
kubectl delete secret github-secret llm-secret output-secret -n helixops
kubectl delete namespace helixops
```

### Systemd

```bash
sudo systemctl stop helixops
sudo systemctl disable helixops
sudo rm /etc/systemd/system/helixops.service
sudo systemctl daemon-reload
sudo rm -rf /opt/helixops /etc/helixops
```

### Docker Compose

```bash
docker-compose down -v  # -v removes volumes
```

---

## Next Steps

- Configure AlertManager routing rules
- Set up Slack channel for notifications
- Test with sample alerts
- Monitor logs and metrics
- Scale based on alert volume

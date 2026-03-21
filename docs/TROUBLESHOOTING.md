# Troubleshooting Guide

Common issues and solutions for HelixOps deployment and operation.

---

## Deployment Issues

### "Database connection failed" Error

**Symptom:** Server fails to start with database connection error.

**Solutions:**
1. Check PostgreSQL is running:
   ```bash
   docker-compose ps  # Should show postgres as "healthy"
   ```

2. Verify connection settings in `config.yaml`:
   ```yaml
   database:
     enabled: true
     host: "postgres"  # Use "localhost" for local, "postgres" for docker-compose
     port: 5432
   ```

3. Check environment variables:
   ```bash
   export HELIX_DB_PASSWORD=your_password
   ```

4. Check logs:
   ```bash
   docker-compose logs helixops
   ```

---

### Webhook Not Reaching HelixOps

**Symptom:** Alerts fire but no analysis occurs.

**Solutions:**
1. Verify AlertManager config points to HelixOps:
   ```yaml
   # alertmanager.yml
   receivers:
   - name: 'helixops'
     webhook_configs:
     - url: 'http://helixops:8080/webhook'
   ```

2. Check network connectivity:
   ```bash
   docker-compose exec alertmanager wget -qO- http://helixops:8080/health
   ```

3. Verify firewall allows traffic on port 8080

---

## LLM Integration Issues

### "LLM provider not configured" Error

**Symptom:** Analysis fails with LLM error.

**Solutions:**
1. Verify provider configuration:
   ```yaml
   llm:
     provider: openai  # or anthropic, ollama
   ```

2. Check API key is set:
   ```bash
   export OPENAI_API_KEY=sk-...
   # or
   export ANTHROPIC_API_KEY=sk-ant-...
   ```

3. For Ollama, verify service:
   ```bash
   curl http://ollama:11434/api/tags
   ```

### LLM Rate Limiting

**Symptom:** "Rate limit exceeded" errors during high alert volume.

**Solutions:**
1. Enable caching or reduce alert frequency
2. Add delays between analyses
3. Use a higher tier LLM plan with higher limits

---

## Alert Processing Issues

### "No recent commits found"

**Symptom:** Analysis shows no code changes.

**Solutions:**
1. Configure GitHub org/mapping:
   ```yaml
   github:
     default_org: "myorg"
     service_mapping:
       my-service: "myorg/my-service"
   ```

2. Verify token has repo access:
   ```bash
   export GITHUB_TOKEN=ghp_...
   curl -H "Authorization: Bearer $GITHUB_TOKEN" https://api.github.com/user
   ```

### Missing Metrics Data

**Symptom:** Analysis shows zero latency/error rate.

**Solutions:**
1. Verify Prometheus is scraping your service:
   ```bash
   curl 'http://prometheus:9090/api/v1/query?query=up'
   ```

2. Check metric labels match service name:
   ```yaml
   # Prometheus should scrape with service="your-service"
   ```

---

## Output Channel Issues

### Slack Notifications Not Sending

**Symptom:** Analysis completes but no Slack message.

**Solutions:**
1. Verify webhook URL:
   ```bash
   export SLACK_WEBHOOK_URL=https://hooks.slack.com/...
   ```

2. Test manually:
   ```bash
   curl -X POST -H 'Content-type: application/json' \
     --data '{"text":"Test"}' \
     $SLACK_WEBHOOK_URL
   ```

3. Check HelixOps logs for errors

---

## Performance Issues

### Slow Analysis Response

**Symptom:** High latency between alert and RCA result.

**Solutions:**
1. Reduce time windows:
   ```yaml
   analysis:
     metrics_window: 5m   # instead of 15m
     commits_lookback: 4h  # instead of 24h
   ```

2. Disable unused integrations (Tempo, Loki)
3. Use local LLM (Ollama) for faster response
4. Increase parallelism in orchestrator

---

## Health Check Failures

### /ready Endpoint Returns 503

**Symptom:** Kubernetes shows pod not ready.

**Solutions:**
1. Wait for dependencies to initialize
2. Check all required services are running:
   ```bash
   docker-compose ps
   ```

3. Review readiness probe logs:
   ```bash
   kubectl describe pod helixops | grep -A 10 "Conditions"
   ```

---

## Data & Privacy

### Verify No Data Leaves Cluster

**Solutions:**
1. Use Ollama for local inference:
   ```yaml
   llm:
     provider: ollama
     ollama_url: http://ollama:11434
   ```

2. Disable external webhooks:
   ```yaml
   output:
     slack:
       enabled: false
   ```

3. Use network policies to restrict egress

---

## Getting Help

- **GitHub Issues:** https://github.com/helixops/helixops/issues
- **Discussions:** https://github.com/helixops/helixops/discussions
- **Email:** support@helixops.io

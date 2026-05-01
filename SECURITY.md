# Security Policy

## Reporting Security Vulnerabilities

We take security seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

1. **Do NOT** create a public GitHub issue for security vulnerabilities
2. Email security concerns to: security@helixops.io
3. Include as much detail as possible:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Any suggested fixes (optional)

### Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 7 days
- **Fix Timeline**: Based on severity (critical: 24-72h, high: 1 week, medium: 2 weeks)

---

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

---

## Security Features

### Data Privacy

- **Credential Storage**: API keys are read from environment variables; database secret handling depends on your deployment
- **Local LLM Option**: Use Ollama to keep all model traffic inside your VPC
- **No External Data Leaks**: Raw telemetry stays in your environment, but output channels should still be reviewed before production use

### Network Security

- **TLS Support**: External API calls use HTTPS where the upstream service supports it
- **Webhook Validation**: AlertManager source validation is recommended, but not yet enforced in code
- **Rate Limiting**: Rate limiting is recommended for production, but not yet implemented in code

### Access Control

- **Environment Variables**: Sensitive config via environment, not config files
- **Database Isolation**: PostgreSQL credentials isolated per deployment
- **RBAC-Ready**: Schema prepared for future role-based access control

---

## Best Practices

### For Production Deployment

1. **Use Secrets Management**
   ```yaml
   # Don't hardcode credentials
   # Use Kubernetes secrets or HashiCorp Vault
   ```

2. **Network Policies**
   ```yaml
   # Restrict egress traffic in Kubernetes
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   spec:
     egress:
     - to:
       - namespaceSelector:
           matchLabels:
             name: monitoring
   ```

3. **TLS Encryption**
   - Enable TLS for all external connections where possible
   - Use mutual TLS for cloud-plane communication if your deployment requires it

4. **Regular Updates**
   - Stay on supported versions
   - Review security advisories

---

## Security Advisories

When security issues are discovered, they will be announced via:

- GitHub Security Advisories
- Release notes in CHANGELOG.md
- Community announcements

---

## Scope

The following are in scope for security reviews:

- HelixOps server (Go binary)
- Configuration loading
- Database interactions
- LLM API integrations
- Webhook handling
- Output channel integrations

The following are OUT of scope:

- Your existing monitoring infrastructure (Prometheus, Loki, etc.)
- Third-party LLM providers (OpenAI, Anthropic)
- Network infrastructure between services

---

**Thank you for helping keep HelixOps secure!**

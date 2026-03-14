# HelixOps Documentation

Complete documentation for HelixOps - The On-Call Copilot for your cluster.

---

## Getting Started

Start here to understand HelixOps and get it running.

### 📖 Core Documentation

| Document | Purpose | Audience |
|----------|---------|----------|
| [README.md](../README.md) | Project overview and quick start | Everyone |
| [ARCHITECTURE.md](ARCHITECTURE.md) | System design and components | Developers, Operators |
| [MVP.md](MVP.md) | Vision and core value proposition | Product, Leadership |

---

## Installation & Operations

### 🚀 Deployment

| Document | Purpose | Audience |
|----------|---------|----------|
| [DEPLOYMENT.md](DEPLOYMENT.md) | Installation in Kubernetes, Docker, VMs | DevOps Engineers, Operators |
| [CONFIGURATION.md](CONFIGURATION.md) | Configuration reference and examples | Operators |
| [API_REFERENCE.md](API_REFERENCE.md) | Webhook format and HTTP endpoints | Integration Engineers |

### ✅ Quality Assurance

| Document | Purpose | Audience |
|----------|---------|----------|
| [TESTING.md](TESTING.md) | Unit, integration, and E2E testing | QA Engineers, Developers |

---

## Development

### 👨‍💻 Contributing

| Document | Purpose | Audience |
|----------|---------|----------|
| [CONTRIBUTING.md](CONTRIBUTING.md) | Development setup and contribution guidelines | Contributors, Maintainers |
| [ROADMAP.md](ROADMAP.md) | Future phases and planned features | Product, Contributors |

---

## Quick Navigation

### "I want to..."

#### Deploy HelixOps

→ [DEPLOYMENT.md](DEPLOYMENT.md)

Start with: [Docker Compose Quick Start](DEPLOYMENT.md#quick-start-docker-compose-development)

#### Configure HelixOps

→ [CONFIGURATION.md](CONFIGURATION.md)

Start with: [Example Configurations](CONFIGURATION.md#example-configurations)

#### Integrate with AlertManager

→ [API_REFERENCE.md](API_REFERENCE.md#alert-webhook-receiver)

Start with: [AlertManager Configuration](API_REFERENCE.md#alertmanager-configuration)

#### Set up Slack notifications

→ [CONFIGURATION.md - Slack Setup](CONFIGURATION.md#slack)

#### Run tests

→ [TESTING.md](TESTING.md)

Start with: [Unit Tests](TESTING.md#step-1-run-unit-tests)

#### Understand the architecture

→ [ARCHITECTURE.md](ARCHITECTURE.md)

Start with: [System Overview](ARCHITECTURE.md#system-overview)

#### Contribute code

→ [CONTRIBUTING.md](CONTRIBUTING.md)

Start with: [Local Setup](CONTRIBUTING.md#local-setup)

#### Use local LLM (Ollama)

→ [DEPLOYMENT.md - Local Deployment](DEPLOYMENT.md#local-deployment-with-ollama-privacy-first)

Or [CONFIGURATION.md - Ollama](CONFIGURATION.md#ollama-private-local)

---

## Documentation Map

### By Document Type

**Product/Strategic:**
- [README.md](README.md) - Product messaging
- [MVP.md](MVP.md) - Core vision
- [ROADMAP.md](ROADMAP.md) - Future roadmap

**Technical Architecture:**
- [ARCHITECTURE.md](ARCHITECTURE.md) - System design
- [API_REFERENCE.md](API_REFERENCE.md) - API specs

**Operations:**
- [DEPLOYMENT.md](DEPLOYMENT.md) - Installation & deployment
- [CONFIGURATION.md](CONFIGURATION.md) - Configuration reference

**Development:**
- [CONTRIBUTING.md](CONTRIBUTING.md) - Development guidelines
- [TESTING.md](TESTING.md) - Testing procedures

---

## Common Workflows

### Workflow 1: First-Time Deployment (Development)

**Goal:** Get HelixOps running locally with Docker Compose

1. Read [README.md](README.md) - Understand what HelixOps is (5 min)
2. Follow [DEPLOYMENT.md - Docker Compose](DEPLOYMENT.md#quick-start-docker-compose-development) (20 min)
3. Follow [API_REFERENCE.md - Examples](API_REFERENCE.md#curl-examples) to send test alerts (10 min)
4. Check Slack notifications

**Total:** ~35 minutes

### Workflow 2: Production Deployment (Kubernetes)

**Goal:** Deploy HelixOps to Kubernetes cluster

1. Read [ARCHITECTURE.md - Overview](ARCHITECTURE.md#system-overview) (10 min)
2. Review [CONFIGURATION.md](CONFIGURATION.md) - determine your config (15 min)
3. Follow [DEPLOYMENT.md - Kubernetes](DEPLOYMENT.md#production-deployment-kubernetes) (30 min)
4. Configure AlertManager routing (10 min)
5. Verify with [TESTING.md](TESTING.md) (15 min)

**Total:** ~80 minutes

### Workflow 3: Contributing Code

**Goal:** Contribute a feature or fix

1. Review [CONTRIBUTING.md](CONTRIBUTING.md#local-setup) (10 min)
2. Setup local environment (20 min)
3. Make changes
4. Run [TESTING.md - Unit Tests](TESTING.md#step-1-run-unit-tests) (10 min)
5. Submit PR

### Workflow 4: Integration with Custom System

**Goal:** Integrate HelixOps with your monitoring stack

1. Review [API_REFERENCE.md](API_REFERENCE.md) - understand webhook format (15 min)
2. Review [CONFIGURATION.md](CONFIGURATION.md) - find relevant settings (10 min)
3. Configure your system (varies)
4. Test with [API_REFERENCE.md - Examples](API_REFERENCE.md#curl-examples) (10 min)

---

## Finding Answers

### Where to find...

**"How do I...?"**
→ Check the Quick Navigation section above

**"What is...?"**
→ Start with [README.md](README.md) or [ARCHITECTURE.md](ARCHITECTURE.md)

**"How do I configure...?"**
→ [CONFIGURATION.md](CONFIGURATION.md#configuration-sections)

**"What's the API format for...?"**
→ [API_REFERENCE.md](API_REFERENCE.md)

**"How do I deploy to...?"**
→ [DEPLOYMENT.md](DEPLOYMENT.md#production-deployment-kubernetes)

**"How do I test...?"**
→ [TESTING.md](TESTING.md)

**"How do I contribute...?"**
→ [CONTRIBUTING.md](CONTRIBUTING.md)

**"What's planned for the future?"**
→ [ROADMAP.md](ROADMAP.md)

---

## Document Conventions

### File Icons

| Icon | Meaning |
|------|---------|
| 📖 | Core documentation |
| 🚀 | Deployment/installation |
| ✅ | Testing/QA |
| 👨‍💻 | Development |
| 🔧 | Configuration |

### Code Examples

All code examples are tested and runnable:

```bash
# Shell commands in markdown code blocks
docker-compose up -d
```

```yaml
# YAML configuration examples
app:
  host: 0.0.0.0
  port: 8080
```

```go
// Go code examples
type Server struct {
    cfg *config.Config
}
```

### Terminal Sessions

```
$ command
output here
```

### Key Concepts

**Bold terms** indicate important concepts defined in the current or referenced documentation.

---

## Status & Versions

**Current Version:** MVP (v1.0.0 planned)

**Documentation Status:**
- ✅ Complete: ARCHITECTURE.md, DEPLOYMENT.md, API_REFERENCE.md, CONFIGURATION.md
- ✅ Updated: README.md, MVP.md, CONTRIBUTING.md, TESTING.md
- 🔜 Planned: Operations Guide, Troubleshooting Guide, FAQ

**Last Updated:** March 4, 2026

---

## Feedback & Updates

- **Report Issues:** [GitHub Issues](https://github.com/helixops/helixops/issues)
- **Suggest Improvements:** [GitHub Discussions](https://github.com/helixops/helixops/discussions)
- **Update Docs:** See [CONTRIBUTING.md](CONTRIBUTING.md#submitting-changes)

---

## Document Relationships

```
README.md (Entry Point)
    ├── MVP.md (Vision)
    ├── ARCHITECTURE.md (Design)
    ├── DEPLOYMENT.md (Ops)
    │   └── CONFIGURATION.md (Config Details)
    ├── API_REFERENCE.md (Integration)
    ├── TESTING.md (Quality)
    ├── CONTRIBUTING.md (Development)
    └── ROADMAP.md (Future)
```

---

## Summary

HelixOps documentation is organized around **user journeys**:

1. **Learn** → README.md + MVP.md
2. **Understand** → ARCHITECTURE.md
3. **Deploy** → DEPLOYMENT.md + CONFIGURATION.md
4. **Integrate** → API_REFERENCE.md
5. **Test** → TESTING.md
6. **Contribute** → CONTRIBUTING.md
7. **Extend** → ROADMAP.md

Each document is **self-contained** but **cross-referenced** for easy navigation.

---

## Quick Links

- 🏠 [Website](https://helixops.io/) (planned)
- 📊 [GitHub Repository](https://github.com/helixops/helixops)
- 💬 [Discord Community](https://discord.gg/helixops) (planned)
- 📧 [Contact](contact@helixops.io)

---

**Happy incident investigating with HelixOps! 🚀**

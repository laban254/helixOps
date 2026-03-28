# Changelog

All notable changes to HelixOps will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial MVP release
- AI-powered RCA for Prometheus alerts
- Integration with Prometheus, Loki, GitHub
- Multi-channel output (Slack, Markdown)
- Support for cloud LLMs (OpenAI, Anthropic) and local LLMs (Ollama)

### Changed
- **Documentation audit and comprehensive improvements**: All 16 documentation issues resolved
- **README.md enhancements**: Added competitive positioning, enhanced quick start guide, expanded troubleshooting section
- **Architecture documentation**: Converted ASCII diagram to Mermaid format for better readability
- **Configuration examples**: Standardized LLM provider configuration and database field names
- **Project structure**: Updated all references from `cmd/agent` to `cmd/mcp`

### Fixed
- **Database technology consistency**: Updated all documentation to reflect PostgreSQL (not SQLite)
- **Build command corrections**: Standardized to `go run ./cmd/mcp` across all documentation
- **Missing files**: Created LICENSE (MIT) and CHANGELOG.md files
- **Placeholder content**: Removed placeholder YouTube demo link and GIF from README.md
- **Date inconsistencies**: Updated all 2025-03-04 dates to 2024-01-01
- **Version numbers**: Updated SECURITY.md version table to match 0.1.x release
- **Health check endpoints**: Removed references to non-existent `/api/v1/status/*` endpoints
- **Service mapping documentation**: Added explanation of `github.service_mapping` configuration
- **Postmortem automation clarity**: Added explicit "no human input required" clarification

## [0.1.0] - 2024-01-01

### Added
- Initial project setup
- Basic webhook receiver
- Prometheus client
- LLM integration prototype

---

**Note**: This is a pre-release version. For production use, please refer to the latest stable release.
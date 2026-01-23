# Changelog

All notable changes to Waterflow will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub Actions CI/CD workflows with parallel jobs
- Integration and acceptance testing frameworks
- Docker image security scanning with Trivy
- Multi-platform binary releases (Linux/macOS/Windows × amd64/arm64)
- SHA256 checksums for release verification
- CLI tool (`waterflow`) for workflow management
- Go SDK for programmatic workflow control
- Workflow templates (single-server, multi-server, distributed-stack)

### Changed
- Enhanced CI workflow with lint, security, test, and build stages
- Improved Docker build with QEMU multi-platform support
- Updated README with CI/CD status badges

### Fixed
- Various test coverage improvements
- Documentation updates for consistency

## [0.1.0] - 2026-01-23

### Added
- Initial release of Waterflow workflow orchestration engine
- **Core Features:**
  - YAML DSL for declarative workflow definitions
  - Expression engine with 14 built-in functions
  - Conditional execution and control flow
  - Matrix parallel execution strategy
  - Timeout and retry policies
  - Temporal SDK integration for reliable execution

- **Distributed Agent System:**
  - Agent worker framework with task queue mapping
  - Multi-server task distribution
  - Docker agent image (51.6MB)

- **Node Plugin System:**
  - Shell command execution (`exec/shell`)
  - Script file execution (`exec/script`)
  - Sleep/delay node (`flow/sleep`)
  - HTTP request node (`http/request`)
  - File transfer via SCP (`file/transfer`)
  - Docker exec node (`docker/exec`)
  - Docker Compose node (`docker/compose`)
  - Custom plugin development support

- **CLI Tool:**
  - `waterflow validate` - Validate workflow YAML
  - `waterflow submit` - Submit workflow for execution
  - `waterflow status` - Check workflow status
  - `waterflow logs` - View workflow logs
  - `waterflow nodes` - List available nodes

- **Go SDK:**
  - Client library for API integration
  - Workflow submission and management
  - Status polling and log retrieval

- **Workflow Templates:**
  - Single-server deployment template
  - Multi-server health check template
  - Distributed stack deployment template

- **Production Features:**
  - Typed error handling with RFC 7807
  - Structured logging with sensitive data redaction
  - Prometheus metrics export
  - Event handler interface
  - Log handler interface
  - HTTPS/TLS support
  - Secret provider interface (Env, Vault)
  - Audit logging

- **Deployment:**
  - Docker Compose complete stack
  - Configuration management with validation
  - Health check and readiness probes
  - Comprehensive deployment documentation

- **Documentation:**
  - Quick start guide
  - REST API specification (OpenAPI 3.0)
  - YAML DSL syntax reference
  - Troubleshooting guide
  - Workflow examples library
  - Architecture concepts documentation

### Security
- TLS/HTTPS support for server and agent communication
- Secret provider interface for secure credential management
- Audit logging for compliance requirements
- Security best practices documentation

---

## Version History Summary

| Version | Date | Highlights |
|---------|------|------------|
| 0.1.0 | 2026-01-23 | Initial release with core workflow engine |

## Upgrade Notes

### Upgrading to 0.1.0

This is the initial release. No upgrade path required.

## Links

- [GitHub Releases](https://github.com/Websoft9/waterflow/releases)
- [Docker Hub](https://hub.docker.com/r/websoft9/waterflow-server)
- [Documentation](https://github.com/Websoft9/waterflow/tree/main/docs)

[Unreleased]: https://github.com/Websoft9/waterflow/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Websoft9/waterflow/releases/tag/v0.1.0

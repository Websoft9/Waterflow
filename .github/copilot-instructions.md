# Waterflow AI Agent Instructions

## 🏗️ Architecture Overview

Waterflow is a **YAML-driven DevOps workflow orchestration platform** that transforms declarative configurations into executable production workflows. The core architecture follows a hierarchical execution model:

**Workflow → Stages → Jobs** with container-based isolation and dependency management.

### Key Structural Patterns

**YAML Configuration Schema** (see `docs/concepts.md`):
```yaml
apiVersion: waterflow.io/v1
kind: Workflow
metadata:
  name: workflow-name
  description: "Purpose description"
spec:
  stages:
    - name: stage-name
      dependsOn: [previous-stage]  # Optional dependencies
      jobs:
        - name: job-name
          container: image:tag     # Required container image
          commands:                # Required command list
            - command1
            - command2
          env:                     # Optional environment variables
            - name: VAR_NAME
              value: "value"
```

**Execution Flow**:
- **Stages**: Execute sequentially with optional `dependsOn` relationships
- **Jobs**: Execute in parallel within each stage, each in isolated containers
- **Dependencies**: Stage-level only; jobs within stages are independent

## 🔄 Development Workflow

### BMAD Method Integration

Follow the [BMAD Method](https://github.com/bmad-code-org/BMAD-METHOD) structured approach:
- **Baseline**: Current state assessment before changes
- **Milestones**: Phased development with clear deliverables
- **Actions**: Week-by-week implementation tasks
- **Decisions**: Architecture decisions documented as ADRs

### Branch Naming & Commits

**Branch Patterns** (see `CONTRIBUTING.md`):
- Features: `feature/descriptive-name`
- Bug fixes: `fix/issue-number-description`
- Documentation: `docs/update-description`

**Commit Messages**: Clear, imperative mood descriptions referencing issues when applicable.

### Multi-Language Build System

**Conditional Execution** (see `.github/workflows/ci.yml`):
- Go: When `.go` files or `go.mod` present
- Node.js: When `package.json` present
- Python: When `requirements.txt` or `pyproject.toml` present

**Build Commands**:
```bash
# Development setup (varies by language)
make dev-setup          # When Makefile present
go mod download         # Go projects
npm install             # Node.js projects
pip install -r requirements.txt  # Python projects

# Testing (language-specific)
make test              # Universal make target
go test ./...          # Go testing
npm test               # Node.js testing
python -m pytest       # Python testing

# Linting (language-specific)
make lint             # Universal make target
golangci-lint run     # Go linting
npm run lint          # ESLint for Node.js
flake8 .              # Python linting
```

## 📝 Code Patterns & Conventions

### YAML Configuration Standards

**Indentation**: 2 spaces (never tabs)
**Comments**: Required for complex configurations
**Anchors**: Use `&anchor` and `*alias` for reusable sections
**Validation**: All YAML validated against JSON Schema

### Documentation Requirements

**PR Documentation Checklist** (see `.github/PULL_REQUEST_TEMPLATE/PULL_REQUEST_TEMPLATE.md`):
- [ ] README.md updated for user-facing changes
- [ ] API documentation updated
- [ ] Code comments added/updated
- [ ] CHANGELOG.md updated

**Documentation Structure** (see `docs/`):
- `docs/concepts.md`: Core terminology and architecture
- `docs/getting-started.md`: First workflow examples
- `docs/installation.md`: Setup instructions
- `docs/README.md`: Documentation navigation

### Testing Standards

**Coverage Requirements**: >80% for new code
**Test Categories**: Unit, integration, end-to-end
**Test Naming**: Descriptive, behavior-focused names
**Mock Usage**: External dependencies must be mocked

## 🔌 Integration Patterns

### Container Execution Model

**Job Isolation**: Each job runs in separate container
**Image Selection**: Specific tagged versions preferred over `latest`
**Volume Mounting**: Use named volumes for data persistence
**Resource Limits**: CPU/memory limits specified in production configs

### Platform Integrations

**Kubernetes**: Custom resources and operators (future)
**Cloud Providers**: AWS ECS, GCP Cloud Run, Azure Container Instances
**CI/CD**: GitHub Actions, GitLab CI, Jenkins integration
**Monitoring**: Prometheus metrics, distributed tracing

### Plugin Architecture

**Extension Points**: Custom workflow steps via plugins
**Plugin Discovery**: Automatic loading from configured directories
**Security**: Plugin sandboxing and permission models

## 🚨 Critical Conventions

### Security-First Approach

**Never commit secrets**: Use environment variables or secret management
**Input validation**: All user inputs validated and sanitized
**Container security**: Non-root execution, minimal base images
**Audit logging**: Security events automatically logged

### Error Handling

**Structured errors**: Custom error types with context
**Graceful degradation**: Failures don't crash entire workflows
**Retry logic**: Configurable retry policies for transient failures
**User feedback**: Clear error messages with actionable guidance

### Performance Considerations

**Parallel execution**: Maximize concurrent job execution within stages
**Resource efficiency**: Right-size containers and resource limits
**Caching strategy**: Cache dependencies and build artifacts
**Monitoring**: Built-in metrics and performance tracking

## 📚 Key Reference Files

- **`docs/concepts.md`**: Core architecture and terminology
- **`docs/getting-started.md`**: Workflow examples and patterns
- **`.github/workflows/ci.yml`**: Build and test automation
- **`CONTRIBUTING.md`**: Development processes and standards
- **`README.md`**: Project overview and BMAD method structure

## 🤖 AI-Assisted Development

**BMAD Integration**: Leverage AI for code generation but validate against established patterns
**Review Process**: AI-generated code requires human review for security and correctness
**Documentation**: AI assistance welcome for documentation but must follow established structure
**Testing**: AI can help generate tests but must meet coverage and quality standards

---

*These instructions are specific to Waterflow's architecture and development practices. Follow the BMAD Method for structured AI-assisted development.*
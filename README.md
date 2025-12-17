# Waterflow

[![CI](https://github.com/Websoft9/Waterflow/workflows/CI/badge.svg)](https://github.com/Websoft9/Waterflow/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Planning-blue)]()

[中文文档](README_zh.md) | English

**Declarative Workflow Orchestration Engine - YAML-Driven Enterprise Distributed Task Execution**

Waterflow is a declarative workflow orchestration engine that provides enterprise-grade distributed task execution capabilities. Define workflows with simple YAML DSL, leverage production-ready execution engine, and achieve reliable cross-server task orchestration with built-in fault tolerance, auto-retry, and complete state persistence.

```yaml
# Example: Distributed Application Deployment Workflow
name: deploy-app
jobs:
  deploy-web:
    runs-on: web-servers
    steps:
      - name: Pull Image
        uses: docker/exec
        with:
          command: docker pull myapp:latest
      
      - name: Deploy Container
        uses: docker/compose-up
        with:
          file: docker-compose.yml

  deploy-db:
    runs-on: db-servers
    steps:
      - name: Init Database
        uses: shell
        with:
          run: mysql -e "CREATE DATABASE IF NOT EXISTS myapp"
```

---

## ✨ Key Features

### 🎯 Declarative YAML DSL
- **Easy to Use** - GitHub Actions-style syntax with gentle learning curve
- **Version Control** - YAML files naturally support Git management
- **Type Safety** - Schema validation catches errors before runtime

### 🔄 Persistent Execution
- **Fault Tolerance** - Automatic recovery after Server/Agent crashes, workflows continue execution
- **Auto Retry** - Node-level retry policies with exponential backoff
- **Long Running** - Support workflows running for hours/days with complete state persistence
- **Process Resilience** - Workflow state survives process restarts with zero data loss

### 🌐 Distributed Agent Architecture
- **Cross-Server Orchestration** - Route tasks to specific server groups via `runs-on`
- **Natural Isolation** - Task Queue mechanism ensures complete server group isolation
- **Elastic Scaling** - Dynamically add/remove Agents without configuration changes

### 🔌 Extensible Node System
- **10 Built-in Nodes** - Control flow (condition/loop/sleep) + Actions (shell/http/file) + Docker management
- **Custom Nodes** - Simple interface for quick business logic extension
- **Plugin Architecture** - Node registry with hot-swap support

### 📊 Enterprise-Grade Observability
- **Event Sourcing** - Complete event history, all operations traceable
- **Real-time Log Streaming** - Support `tail -f` style execution log viewing
- **Time Travel Debugging** - View workflow state at any point in time

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────┐
│ Waterflow Server (Stateless REST API)   │
│ • YAML Parser (Server-side parsing)     │
│ • Temporal Client                       │
└─────────────────────────────────────────┘
              ↓ gRPC
┌─────────────────────────────────────────┐
│ Temporal Server (Event Sourcing)        │
│ • WaterflowWorkflow (Interpreter)       │
│ • Task Queue Routing                    │
│ • Event History Persistence             │
└─────────────────────────────────────────┘
              ↓ Long Polling
┌─────────────────────────────────────────┐
│ Waterflow Agent (Target Servers)        │
│ • Temporal Worker                       │
│ • Node Executors (10 built-in)          │
└─────────────────────────────────────────┘
```

**Key Design Principles:**
- ✅ **Event Sourcing** - Complete execution history tracking, support time-travel debugging
- ✅ **Single-Node Execution** - Each step runs as independent unit with precise timeout/retry control
- ✅ **Plugin Architecture** - Hot-swappable node system, extend functionality without restart
- ✅ **Stateless Server** - All workflow state persisted externally, enabling horizontal scaling

Details: [Architecture Documentation](docs/architecture.md) | [Architecture Decision Records](docs/adr/README.md)

---

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for development)

### 1. Deploy with Docker Compose

```bash
# Clone repository
git clone https://github.com/Websoft9/Waterflow.git
cd Waterflow

# Start Waterflow Server + Temporal + PostgreSQL
docker-compose up -d

# Verify service
curl http://localhost:8080/health
```

### 2. Deploy Agent to Target Servers

```bash
# Run Agent on target servers
docker run -d \
  -e TEMPORAL_HOST=waterflow-server:7233 \
  -e SERVER_GROUP=web-servers \
  -v /var/run/docker.sock:/var/run/docker.sock \
  waterflow/agent:latest
```

### 3. Submit Your First Workflow

```bash
# Create workflow file
cat > hello-world.yaml <<EOF
name: hello-world
jobs:
  greet:
    runs-on: web-servers
    steps:
      - name: Say Hello
        uses: shell
        with:
          run: echo "Hello from Waterflow!"
EOF

# Submit workflow
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/yaml" \
  --data-binary @hello-world.yaml

# Query status
curl http://localhost:8080/v1/workflows/{workflow-id}

# View logs
curl http://localhost:8080/v1/workflows/{workflow-id}/logs
```

---

## 📚 Documentation

### Core Documentation
- [Product Requirements (PRD)](docs/prd.md) - Product positioning, features, MVP scope
- [Architecture Documentation](docs/architecture.md) - Architecture decisions, tech stack, cross-cutting concerns
- [Epics & Stories](docs/epics.md) - 12 Epics, 110+ User Stories
- [Architecture Decision Records (ADRs)](docs/adr/README.md) - 6 core design decisions

### Analysis & Planning
- [Temporal Architecture Analysis](docs/analysis/temporal-architecture-analysis.md) - Temporal core capabilities, Workflow/Activity patterns, design validation
- [Architecture Optimization Summary](docs/analysis/architecture-optimization-summary.md) - 5 key optimizations, performance comparison, implementation recommendations
- [Epic Coverage Analysis](docs/epic-coverage-analysis.md) - Epic-to-ADR traceability matrix
- [Agent Architecture](docs/analysis/agent-architecture.md) - AI agent development methodology

### Implementation
- [Implementation Readiness Report](docs/implementation-readiness-report-2025-12-16.md) - Readiness assessment (98/100), Sprint 1 plan, 12-week roadmap
- [Sprint Artifacts](docs/sprint-artifacts/) - Detailed planning for all 10 Sprint 1 tasks

### Architecture Diagrams
- [Detailed Architecture](docs/diagrams/waterflow-detailed-architecture-20251215.excalidraw) - Complete 3-tier architecture design
- [Data Flow Diagram](docs/diagrams/waterflow-dataflow-simple-20251216.excalidraw) - Simplified data flow visualization

> Install [Excalidraw Extension](https://marketplace.visualstudio.com/items?itemName=pomdtr.excalidraw-editor) in VS Code to view diagrams

---

## 🎯 Use Cases

### 1. Distributed Application Deployment
```yaml
jobs:
  deploy-frontend:
    runs-on: web-servers
    steps:
      - uses: docker/compose-up
        with:
          file: frontend.yml
  
  deploy-backend:
    runs-on: app-servers
    needs: [deploy-database]
    steps:
      - uses: docker/compose-up
        with:
          file: backend.yml
  
  deploy-database:
    runs-on: db-servers
    steps:
      - uses: shell
        with:
          run: docker exec mysql mysql -e "CREATE DATABASE app"
```

### 2. Batch Operations & Health Checks
```yaml
jobs:
  health-check:
    runs-on: all-servers
    steps:
      - uses: shell
        with:
          run: |
            df -h
            free -m
            docker ps
      
      - uses: http/request
        with:
          url: http://localhost/health
          method: GET
```

### 3. Scheduled Backup Tasks
```yaml
jobs:
  backup:
    runs-on: db-servers
    steps:
      - uses: shell
        with:
          run: mysqldump -u root myapp > /backup/myapp.sql
      
      - uses: file/transfer
        with:
          source: /backup/myapp.sql
          destination: s3://backups/myapp-{date}.sql
```

---

## 🌟 Why Waterflow?

**Problem:**
- Traditional workflow engines require extensive coding with steep learning curves
- Cross-server task orchestration lacks simple, reliable solutions
- Script-based automation lacks durability, retry mechanisms, and state management
- Existing tools (Airflow/Jenkins) are too heavy or not suitable for general workflows

**Solution:**
- **Declarative YAML DSL** - GitHub Actions-style syntax, productive in 10 minutes
- **Enterprise-Grade Execution** - Built-in fault tolerance, auto-retry, and state persistence
- **Agent Architecture** - Natural distributed execution, no SSH configuration required
- **Lightweight Deployment** - Single binary + Docker Compose, running in 5 minutes

**Target Users:**
- DevOps engineers needing cross-server automation
- Platform teams building internal workflow platforms
- Developers wanting Temporal simplicity without complexity
- Teams requiring reliable long-running task orchestration

---

## 📋 Development Method

This project uses **BMAD (Brownfield/Modern Agentic Development) Method** for development workflow.

**What is BMAD?**
- AI-assisted agile development methodology
- Works with GitHub Copilot agents
- Provides structured workflows for entire SDLC (Analysis → Planning → Architecture → Implementation)

**For Developers:**
- All workflow configurations are in `.bmad/` directory
- 10+ specialized AI agents available (use `@` to invoke in GitHub Copilot Chat)
- See [.bmad/bmm/docs/quick-start.md](.bmad/bmm/docs/quick-start.md) for usage guide

**Key Agents Used:**
- `@architect` - Architecture design and optimization
- `@prd` - Product requirements collaboration
- `@epic` - Epic breakdown and story writing
- `@implementation` - Implementation readiness assessment

---

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on:

- Branch Strategy (Git Flow)
- Commit Message Convention (Conventional Commits)
- Pull Request Process
- Code Standards

---

## 🔒 Security

See [SECURITY.md](SECURITY.md) for reporting security vulnerabilities.

---

## 📄 License

This project is licensed under the MIT License - see [LICENSE](LICENSE) for details.

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/Websoft9">Websoft9</a>
</p>

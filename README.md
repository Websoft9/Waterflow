# Waterflow

[![CI](https://github.com/Websoft9/Waterflow/workflows/CI/badge.svg)](https://github.com/Websoft9/Waterflow/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Planning-blue)]()

**声明式工作流编排引擎 - 让 YAML 驱动企业级分布式任务执行**

Waterflow 是基于 Temporal 构建的声明式工作流编排服务，通过 YAML DSL 和分布式 Agent 模式，让您以简单的方式实现跨服务器的可靠任务编排，无需了解底层 Temporal 的复杂性。

```yaml
# 示例: 分布式应用部署工作流
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

## ✨ 核心特性

### 🎯 声明式 YAML DSL
- **简单易用** - GitHub Actions 风格语法，学习曲线平缓
- **版本控制** - YAML 文件天然支持 Git 管理
- **类型安全** - Schema 验证，运行前捕获错误

### 🔄 持久化执行 (基于 Temporal)
- **进程容错** - Server/Agent 崩溃后自动恢复，工作流继续执行
- **自动重试** - 节点级重试策略，指数退避
- **长时运行** - 支持数小时/数天的工作流，状态完整持久化

### 🌐 分布式 Agent 架构
- **跨服务器编排** - 通过 `runs-on` 将任务路由到特定服务器组
- **天然隔离** - Task Queue 机制确保服务器组完全隔离
- **弹性扩展** - 动态增减 Agent，无需配置变更

### 🔌 可扩展节点系统
- **10 个内置节点** - 控制流 (condition/loop/sleep) + 操作 (shell/http/file) + Docker 管理
- **自定义节点** - 简单接口，快速扩展业务逻辑
- **插件化** - 节点注册表，热插拔支持

### 📊 企业级可观测性
- **Event Sourcing** - 完整事件历史，所有操作可追溯
- **实时日志流** - 支持 `tail -f` 模式查看执行日志
- **时间旅行调试** - 查看任意时间点的工作流状态

---

## 🏗️ 架构设计

```
┌─────────────────────────────────────────┐
│ Waterflow Server (无状态 REST API)      │
│ • YAML Parser (Server 端解析)           │
│ • Temporal Client                       │
└─────────────────────────────────────────┘
              ↓ gRPC
┌─────────────────────────────────────────┐
│ Temporal Server (Event Sourcing)        │
│ • WaterflowWorkflow (解释器模式)        │
│ • Task Queue 路由                       │
│ • Event History 持久化                  │
└─────────────────────────────────────────┘
              ↓ Long Polling
┌─────────────────────────────────────────┐
│ Waterflow Agent (目标服务器)            │
│ • Temporal Worker                       │
│ • Node Executors (10个内置)             │
└─────────────────────────────────────────┘
```

**关键设计优化:**
- ✅ **解释器模式** - DSL 解析在 Server 端，确定性保证
- ✅ **批量执行** - 一个 job 所有 steps 打包成一个 Activity，Event 减少 100 倍
- ✅ **Task Queue 路由** - `runs-on` 直接映射 Temporal 队列，零开发成本
- ✅ **无状态 Server** - 状态存储在 Temporal，水平扩展，高可用

详见: [架构优化总结](docs/analysis/architecture-optimization-summary.md) | [Temporal 深度分析](docs/analysis/temporal-architecture-analysis.md)

---

## 🚀 快速开始

### 前置要求
- Docker & Docker Compose
- Go 1.21+ (开发)

### 1. 一键部署 (Docker Compose)

```bash
# 克隆仓库
git clone https://github.com/Websoft9/Waterflow.git
cd Waterflow

# 启动 Waterflow Server + Temporal + PostgreSQL
docker-compose up -d

# 验证服务
curl http://localhost:8080/health
```

### 2. 部署 Agent 到目标服务器

```bash
# 在目标服务器上运行 Agent
docker run -d \
  -e TEMPORAL_HOST=waterflow-server:7233 \
  -e SERVER_GROUP=web-servers \
  -v /var/run/docker.sock:/var/run/docker.sock \
  waterflow/agent:latest
```

### 3. 提交第一个工作流

```bash
# 创建工作流文件
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

# 提交工作流
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/yaml" \
  --data-binary @hello-world.yaml

# 查询状态
curl http://localhost:8080/v1/workflows/{workflow-id}

# 查看日志
curl http://localhost:8080/v1/workflows/{workflow-id}/logs
```

---

## 📚 文档

### 核心文档
- [产品需求文档 (PRD)](docs/prd.md) - 产品定位、功能需求、MVP 范围
- [技术架构文档](docs/architecture.md) - 架构决策、技术栈、横切关注点
- [Epic 和 Story 拆解](docs/epics.md) - 12 个 Epic，110+ User Stories

### 架构分析
- [Temporal 架构深度分析](docs/analysis/temporal-architecture-analysis.md) - Temporal 核心能力、Workflow/Activity 模式、设计验证
- [架构优化总结](docs/analysis/architecture-optimization-summary.md) - 5 个关键优化、性能对比、实施建议

### 实施计划
- [实施准备报告](docs/implementation-readiness-report-2025-12-15.md) - 准备度评估 (98/100)，Sprint 1 计划，12 周路线图

### 架构图
- [系统架构图](docs/diagrams/waterflow-detailed-architecture-20251215.excalidraw) - 完整的 3 层架构设计

> 在 VS Code 中安装 [Excalidraw 扩展](https://marketplace.visualstudio.com/items?itemName=pomdtr.excalidraw-editor) 查看架构图

---

## 🛣️ 项目状态

**当前阶段:** 📋 **规划与设计完成** (2025-12-15)

✅ **已完成:**
- [x] PRD 编写 (产品定位、功能需求、成功标准)
- [x] 技术架构设计 (技术栈选型、架构决策)
- [x] Epic 拆解 (12 个 Epic，110+ Stories)
- [x] Temporal 深度分析 (核心能力验证)
- [x] 架构优化 (5 个关键优化点)
- [x] 实施准备评估 (98/100 分，READY 状态)
- [x] 3 张架构图 (系统/详细/优化)

🔄 **下一步:**
- [ ] Sprint 1 实施 (Epic 1: 项目基础设施，10 Stories，2 周)
- [ ] MVP 开发 (3-4 个月)
- [ ] 生产就绪 (4-6 个月)

**实施路线图:**
- **第 1-3 月:** MVP (Server + Agent + 10 节点 + Docker Compose)
- **第 4-6 月:** 生产就绪 (多语言 SDK + Web UI + 监控集成)
- **第 7-12 月:** 生态增长 (节点市场 + 社区模板)

详见: [实施准备报告](docs/implementation-readiness-report-2025-12-15.md)

---

## 🎯 典型使用场景

### 1. 分布式应用部署
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

### 2. 批量运维巡检
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

### 3. 定时备份任务
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

## 🔌 REST API

### 核心端点

```bash
# 提交工作流
POST /v1/workflows
Content-Type: application/yaml
Body: <YAML 工作流定义>

# 查询状态
GET /v1/workflows/{id}
Response: { "status": "running", "progress": "50%", ... }

# 获取日志
GET /v1/workflows/{id}/logs
Response: <日志流>

# 取消工作流
POST /v1/workflows/{id}/cancel

# 验证 YAML
POST /v1/validate
Body: <YAML 内容>

# 列出可用节点
GET /v1/nodes
Response: [{ "name": "shell", "version": "1.0", ... }]

# 列出 Agent
GET /v1/agents
Response: [{ "id": "agent-001", "group": "web-servers", ... }]
```

完整 API 文档: OpenAPI 3.0 规范 (开发中)

---

## 🧩 内置节点 (10 个)

### 控制流 (3 个)
- `condition` - 条件判断 (if/else)
- `loop` - 循环迭代 (for-each)
- `sleep` - 延时等待

### 操作 (4 个)
- `shell` - Shell 命令执行
- `script` - 脚本执行 (Bash/Python)
- `file/transfer` - 文件传输
- `http/request` - HTTP 请求

### Docker (3 个)
- `docker/exec` - Docker 命令执行
- `docker/compose-up` - Docker Compose 启动
- `docker/compose-down` - Docker Compose 停止

### 自定义节点
```go
type NodeExecutor interface {
    Execute(ctx context.Context, params map[string]interface{}) (Result, error)
}

// 注册自定义节点
nodeRegistry.Register("my-custom-node", &MyExecutor{})
```

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

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on:

- Branch Strategy (Git Flow)
- Commit Message Convention (Conventional Commits)
- Pull Request Process
- Code Standards

## 🔒 Security

See [SECURITY.md](SECURITY.md) for reporting security vulnerabilities.

## 📄 License

This project is licensed under the MIT License - see [LICENSE](LICENSE) for details.

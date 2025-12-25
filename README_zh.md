# Waterflow

[![CI](https://github.com/Websoft9/Waterflow/workflows/CI/badge.svg)](https://github.com/Websoft9/Waterflow/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Planning-blue)]()

[English](README.md) | 中文文档

**声明式工作流编排引擎 - 让 YAML 驱动企业级分布式任务执行**

Waterflow 是一个声明式工作流编排引擎,提供企业级分布式任务执行能力。通过简单的 YAML DSL 定义工作流,结合生产级执行引擎,实现跨服务器的可靠任务编排,内置容错、自动重试和完整状态持久化。

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
- **简单易用** - GitHub Actions 风格语法,学习曲线平缓
- **版本控制** - YAML 文件天然支持 Git 管理
- **类型安全** - Schema 验证,运行前捕获错误

### 🔄 持久化执行
- **进程容错** - Server/Agent 崩溃后自动恢复,工作流继续执行
- **自动重试** - 节点级重试策略,指数退避
- **长时运行** - 支持数小时/数天的工作流,状态完整持久化
- **进程弹性** - 工作流状态在进程重启后零数据丢失恢复

---

## 🚀 快速开始

### 使用 Docker Compose 一键部署

```bash
# 克隆仓库
git clone https://github.com/websoft9/waterflow.git
cd waterflow

# 启动所有服务
cd deployments
docker compose up -d

# 验证部署
curl http://localhost:8080/health
```

### 提交你的第一个工作流

```bash
# 提交 hello-world 示例
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d "{\"yaml\": \"$(cat ../examples/hello-world.yaml | sed 's/"/\\"/g' | tr '\n' ' ')\"}"

# 查看工作流状态
curl http://localhost:8080/v1/workflows
```

### 访问 Temporal UI

访问 http://localhost:8088 在 Temporal Web UI 中查看工作流可视化执行过程。

详细部署说明请参考 [快速开始指南](docs/quick-start.md) 或 [部署文档](docs/deployment.md)。

---

### 🌐 分布式 Agent 架构
- **跨服务器编排** - 通过 `runs-on` 将任务路由到特定服务器组
- **天然隔离** - Task Queue 机制确保服务器组完全隔离
- **弹性扩展** - 动态增减 Agent,无需配置变更

### 🔌 可扩展节点系统
- **10 个内置节点** - 控制流 (condition/loop/sleep) + 操作 (shell/http/file) + Docker 管理
- **自定义节点** - 简单接口,快速扩展业务逻辑
- **插件化** - 节点注册表,热插拔支持

### 📊 企业级可观测性
- **Event Sourcing** - 完整事件历史,所有操作可追溯
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

**关键设计原则:**
- ✅ **Event Sourcing** - 完整执行历史追踪,支持时间旅行调试
- ✅ **单节点执行** - 每个步骤作为独立单元运行,精确超时/重试控制
- ✅ **插件架构** - 热插拔节点系统,无需重启即可扩展功能
- ✅ **无状态 Server** - 所有工作流状态外部持久化,支持水平扩展

详见: [架构文档](docs/architecture.md) | [架构决策记录](docs/adr/README.md)

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
cd deployments
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
- [Epic 和 Story 拆解](docs/epics.md) - 12 个 Epic,110+ User Stories
- [架构决策记录 (ADRs)](docs/adr/README.md) - 6 个核心设计决策

### 架构分析与规划
- [Temporal 架构深度分析](docs/analysis/temporal-architecture-analysis.md) - Temporal 核心能力、Workflow/Activity 模式、设计验证
- [架构优化总结](docs/analysis/architecture-optimization-summary.md) - 5 个关键优化、性能对比、实施建议
- [Epic 覆盖分析](docs/epic-coverage-analysis.md) - Epic 到 ADR 的可追溯矩阵
- [Agent 架构](docs/analysis/agent-architecture.md) - AI 智能体开发方法论

### 实施计划
- [实施准备报告](docs/implementation-readiness-report-2025-12-16.md) - 准备度评估 (98/100),Sprint 1 计划,12 周路线图
- [Sprint 工件](docs/sprint-artifacts/) - 全部 10 个 Sprint 1 任务的详细规划

### 架构图
- [详细架构图](docs/diagrams/waterflow-detailed-architecture-20251215.excalidraw) - 完整的 3 层架构设计
- [数据流图](docs/diagrams/waterflow-dataflow-simple-20251216.excalidraw) - 简化的数据流可视化

> 在 VS Code 中安装 [Excalidraw 扩展](https://marketplace.visualstudio.com/items?itemName=pomdtr.excalidraw-editor) 查看架构图

---

## 🎯 使用场景

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

## 🌟 为什么选择 Waterflow?

**问题:**
- 传统工作流引擎需要大量编码,学习曲线陡峭
- 跨服务器任务编排缺少简单可靠的解决方案
- 脚本自动化缺乏持久性、重试机制和状态管理
- 现有工具 (Airflow/Jenkins) 太重或不适合通用工作流

**解决方案:**
- **声明式 YAML DSL** - GitHub Actions 风格,10 分钟上手
- **企业级执行** - 内置容错、自动重试和状态持久化
- **Agent 架构** - 天然分布式执行,无需 SSH 配置
- **轻量部署** - 单一二进制 + Docker Compose,5 分钟运行

**目标用户:**
- 需要跨服务器自动化的 DevOps 工程师
- 构建内部工作流平台的平台团队
- 想要简单可靠工作流编排的开发者
- 需要可靠长时运行任务编排的团队

---

## 📋 开发方法

本项目使用 **BMAD (Brownfield/Modern Agentic Development) 方法** 进行开发工作流。

**什么是 BMAD?**
- AI 辅助的敏捷开发方法论
- 与 GitHub Copilot 智能体配合使用
- 为整个 SDLC 提供结构化工作流 (分析 → 规划 → 架构 → 实施)

**对于开发者:**
- 所有工作流配置在 `.bmad/` 目录中
- 提供 10+ 个专业 AI 智能体 (在 GitHub Copilot Chat 中使用 `@` 调用)
- 参见 [.bmad/bmm/docs/quick-start.md](.bmad/bmm/docs/quick-start.md) 了解使用指南

**使用的关键智能体:**
- `@architect` - 架构设计和优化
- `@prd` - 产品需求协作
- `@epic` - Epic 拆解和 Story 编写
- `@implementation` - 实施准备评估

---

## 🌟 为什么选择 Waterflow?

**问题:**
- 传统工作流引擎需要大量编码,学习曲线陡峭
- 跨服务器任务编排缺少简单可靠的解决方案
- 脚本自动化缺乏持久性、重试机制和状态管理
- 现有工具 (Airflow/Jenkins) 太重或不适合通用工作流

**解决方案:**
- **声明式 YAML DSL** - GitHub Actions 风格,10 分钟上手
- **企业级执行** - 内置容错、自动重试和状态持久化
- **Agent 架构** - 天然分布式执行,无需 SSH 配置
- **轻量部署** - 单一二进制 + Docker Compose,5 分钟运行

**目标用户:**
- 需要跨服务器自动化的 DevOps 工程师
- 构建内部工作流平台的平台团队
- 想要简单可靠工作流编排的开发者
- 需要可靠长时运行任务编排的团队

---

## 🤝 贡献

欢迎贡献! 请阅读 [CONTRIBUTING.md](CONTRIBUTING.md) 了解详情:

- 分支策略 (Git Flow)
- 提交信息约定 (Conventional Commits)
- Pull Request 流程
- 代码标准

---

## 🔒 安全

参见 [SECURITY.md](SECURITY.md) 了解如何报告安全漏洞。

---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

---

<p align="center">
  由 <a href="https://github.com/Websoft9">Websoft9</a> 用 ❤️ 制作
</p>

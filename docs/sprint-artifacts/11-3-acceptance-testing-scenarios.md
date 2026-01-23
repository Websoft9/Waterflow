# Story 11.3: 验收测试场景

**Status:** code-complete

## Story

As a **产品经理**,
I want **验收测试覆盖关键场景**,
So that **确保 MVP 目标达成**。

## Acceptance Criteria

### AC1: 场景1 - 多服务器健康检查工作流通过

**Given** PRD 定义的验收场景 1  
**When** 执行多服务器健康检查工作流  
**Then** 在 3 台服务器上并发执行脚本  
**And** 实时显示执行进度  
**And** 收集 CPU、内存、磁盘使用率  
**And** 结果聚合到单一健康报告  
**And** 测试在 5 分钟内完成

### AC2: 场景2 - 分布式应用部署工作流通过

**Given** PRD 定义的验收场景 2  
**When** 执行分布式应用部署工作流  
**Then** 在不同服务器组部署 Web 应用和数据库  
**And** 按依赖顺序执行 (Database → Application)  
**And** 执行健康检查验证部署成功  
**And** 失败时自动重试  
**And** 支持失败回滚机制  
**And** 测试在 10 分钟内完成

### AC3: 每个场景有自动化测试脚本

**Given** 验收测试场景定义  
**When** 查看测试脚本  
**Then** 每个场景有独立的测试脚本 (Shell 或 Go)  
**And** 脚本可无人值守运行  
**And** 脚本参数化支持不同环境  
**And** 脚本包含预置条件检查  
**And** 脚本包含清理步骤

### AC4: 测试结果生成报告

**Given** 验收测试执行完成  
**When** 检查测试报告  
**Then** 生成结构化测试报告 (Markdown 或 HTML)  
**And** 报告包含每个场景的通过/失败状态  
**And** 报告包含执行时间统计  
**And** 失败场景包含详细错误信息和日志  
**And** 报告可作为发布 artifact 保存

### AC5: 所有验收测试通过才能发布

**Given** CI/CD 发布流程  
**When** 创建新版本发布  
**Then** 验收测试作为发布门禁运行  
**And** 任一验收测试失败阻止发布  
**And** 发布流程记录验收测试结果  
**And** 发布说明包含验收测试摘要

## Tasks / Subtasks

### Task 1: 验收测试基础设施 (AC: #3, #4)

- [x] 1.1 创建 test/acceptance/ 目录结构
- [x] 1.2 创建 docker-compose.acceptance.yaml (模拟多服务器环境)
- [x] 1.3 创建验收测试运行脚本 scripts/run-acceptance-tests.sh
- [x] 1.4 创建测试报告生成器 (Markdown 格式)
- [x] 1.5 添加 Makefile target: `make acceptance-test`

### Task 2: 多服务器环境模拟 (AC: #1, #2)

- [x] 2.1 创建 Docker 网络模拟多服务器拓扑
- [x] 2.2 配置 3 个 Agent 容器模拟不同服务器
- [x] 2.3 配置服务器组 Task Queue 映射
- [x] 2.4 验证 Agent 注册和任务分发

### Task 3: 场景1 - 多服务器健康检查 (AC: #1)

- [x] 3.1 创建 test/acceptance/scenario_health_check_test.go
- [x] 3.2 使用 examples/workflows/multi-server-health-check.yaml
- [x] 3.3 测试并发执行 (Matrix 策略)
- [x] 3.4 测试进度实时显示
- [x] 3.5 测试 CPU/内存/磁盘指标收集
- [x] 3.6 测试健康报告生成
- [x] 3.7 测试阈值告警触发
- [x] 3.8 验证执行时间 < 5 分钟

### Task 4: 场景2 - 分布式应用部署 (AC: #2)

- [x] 4.1 创建 test/acceptance/scenario_distributed_deploy_test.go
- [x] 4.2 使用 examples/workflows/distributed-stack-deployment.yaml
- [x] 4.3 测试数据库部署 (PostgreSQL)
- [x] 4.4 测试应用部署 (依赖数据库)
- [x] 4.5 测试 Job 依赖顺序执行
- [x] 4.6 测试健康检查验证
- [x] 4.7 测试失败重试机制
- [x] 4.8 测试回滚机制
- [x] 4.9 验证执行时间 < 10 分钟

### Task 5: 测试工作流 Fixtures (AC: #1, #2)

- [x] 5.1 简化健康检查工作流用于测试
- [x] 5.2 简化分布式部署工作流用于测试
- [x] 5.3 创建模拟应用 Docker 镜像
- [x] 5.4 创建测试数据库初始化脚本
- [x] 5.5 文档化 fixture 使用方法

### Task 6: 测试报告生成 (AC: #4)

- [x] 6.1 创建报告模板 (Markdown)
- [x] 6.2 实现场景执行结果收集
- [x] 6.3 实现执行时间统计
- [x] 6.4 实现失败详情和日志附加
- [x] 6.5 实现报告输出 (文件 + stdout)
- [ ] 6.6 可选: HTML 报告生成

### Task 7: CI/CD 集成 (AC: #5)

- [x] 7.1 更新 .github/workflows/ 添加验收测试 job
- [x] 7.2 配置验收测试作为 release 门禁
- [x] 7.3 配置测试报告作为 artifact 保存
- [x] 7.4 配置失败时阻止发布
- [x] 7.5 添加验收测试状态到 PR 检查

### Task 8: 附加验收场景 (可选扩展)

- [ ] 8.1 场景3: 单服务器应用部署
- [ ] 8.2 场景4: 工作流取消和恢复
- [ ] 8.3 场景5: 长时运行工作流 (超时测试)
- [ ] 8.4 场景6: 密钥注入 (SecretProvider)
- [ ] 8.5 场景7: 事件通知 (EventHandler)

### Task 9: 文档和指南 (AC: #3)

- [x] 9.1 创建 test/acceptance/README.md
- [x] 9.2 文档化验收测试场景
- [x] 9.3 文档化环境要求
- [x] 9.4 文档化如何本地运行
- [x] 9.5 文档化如何添加新场景

## Dev Notes

### PRD 定义的验收测试场景

**来源:** [docs/prd.md - 验收测试场景](../prd.md)

**场景 1: 多服务器健康检查**
> 在 3 台服务器上并发执行脚本，实时显示进度，结果聚合到单一报告。

**场景 2: 分布式应用部署**
> 在不同服务器组部署 Web 应用和数据库，按依赖顺序执行，健康检查验证，失败自动重试和回滚

### 现有工作流模板

**场景 1 对应模板:**
- [examples/workflows/multi-server-health-check.yaml](../../examples/workflows/multi-server-health-check.yaml)
- 428 行完整实现
- 支持 Matrix 并行执行
- 包含 CPU/内存/磁盘检查
- 生成 Markdown 健康报告

**场景 2 对应模板:**
- [examples/workflows/distributed-stack-deployment.yaml](../../examples/workflows/distributed-stack-deployment.yaml)
- 427 行完整实现
- 支持 Database → Application 依赖部署
- 支持健康检查和回滚
- 参数化配置

### 验收测试环境架构

```
┌─────────────────────────────────────────────────────────────────┐
│                   Docker Compose Acceptance Environment         │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────────────────┐ │
│  │PostgreSQL│  │ Temporal │  │ Waterflow Server :8080        │ │
│  │  :5432   │←─│  :7233   │←─│                               │ │
│  └──────────┘  └──────────┘  └───────────────────────────────┘ │
│                      ↑                                          │
│                      │ Task Queue                               │
│  ┌───────────────────┴───────────────────────────────────────┐ │
│  │                   Waterflow Agents                         │ │
│  │                                                            │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │ │
│  │  │  agent-1    │  │  agent-2    │  │  agent-3    │       │ │
│  │  │  (web-1)    │  │  (web-2)    │  │  (db-1)     │       │ │
│  │  │ linux-amd64 │  │ linux-amd64 │  │ db-server   │       │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘       │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
           ↑
           │ HTTP API
    ┌──────────────┐
    │ Acceptance   │
    │ Test Runner  │
    └──────────────┘
```

### 验收测试目录结构

```
test/acceptance/
├── docker-compose.acceptance.yaml    # 多 Agent 测试环境
├── scenario_health_check_test.go     # 场景1 测试
├── scenario_distributed_deploy_test.go # 场景2 测试
├── helpers.go                         # 测试辅助函数
├── report.go                          # 报告生成器
├── testdata/
│   ├── health-check-simplified.yaml   # 简化版健康检查
│   ├── distributed-deploy-simplified.yaml # 简化版部署
│   └── mock-app/                      # 模拟应用
│       ├── Dockerfile
│       └── main.go
└── README.md                          # 验收测试文档
```

### 测试执行命令

```bash
# 本地运行验收测试
make acceptance-test

# 运行特定场景
go test -v ./test/acceptance -run TestAcceptance_HealthCheck
go test -v ./test/acceptance -run TestAcceptance_DistributedDeploy

# 手动启动测试环境
docker compose -f test/acceptance/docker-compose.acceptance.yaml up -d
# 运行测试
go test -v ./test/acceptance
# 清理
docker compose -f test/acceptance/docker-compose.acceptance.yaml down -v
```

### 测试报告格式

```markdown
# Waterflow Acceptance Test Report

**Date:** 2026-01-23 15:30:00 UTC
**Version:** v1.0.0
**Environment:** Docker Compose

## Summary

| Scenario | Status | Duration |
|----------|--------|----------|
| Multi-Server Health Check | ✅ PASSED | 3m 42s |
| Distributed Stack Deployment | ✅ PASSED | 7m 15s |

**Total:** 2/2 passed (100%)
**Total Duration:** 10m 57s

## Scenario Details

### 1. Multi-Server Health Check

- **Status:** ✅ PASSED
- **Duration:** 3m 42s
- **Servers Checked:** 3
- **Health Report Generated:** /tmp/health_report_20260123.md

### 2. Distributed Stack Deployment

- **Status:** ✅ PASSED
- **Duration:** 7m 15s
- **Database:** PostgreSQL deployed successfully
- **Application:** myapp deployed successfully
- **Health Check:** API /health returned 200 OK

## Logs

[Attached: acceptance-test-logs.txt]
```

### Project Structure Notes

- 验收测试使用 `// +build acceptance` build tag
- 验收测试需要完整的 Docker Compose 环境
- 验收测试执行时间较长 (5-15 分钟)
- 验收测试是发布的必要门禁

### References

- [Source: docs/prd.md#验收测试场景] - PRD 定义的验收场景
- [Source: docs/epics.md#story-11.3] - 验收测试验收标准
- [Source: examples/workflows/multi-server-health-check.yaml] - 健康检查模板
- [Source: examples/workflows/distributed-stack-deployment.yaml] - 分布式部署模板

---

## Dev Agent Record

### Context Reference

<!-- Path(s) to story context XML will be added here by context workflow -->

### Agent Model Used

Claude Opus 4.5 (via GitHub Copilot)

### Debug Log References

### Completion Notes List

- **Task 1 完成**: 验收测试基础设施
  - `test/acceptance/docker-compose.acceptance.yaml` - 多 Agent 环境
  - `scripts/run-acceptance-tests.sh` - 测试运行脚本
  - `Makefile` 添加 acceptance-test targets

- **Task 2 完成**: 多服务器环境模拟
  - 3 个 Agent 容器 (web-1, web-2, db-1)
  - Task Queue 映射: linux-amd64, db-server
  - Docker 网络隔离

- **Task 3 完成**: 场景1 - 多服务器健康检查
  - `scenario_health_check_test.go` - Matrix 并发测试
  - 验证 CPU/内存/磁盘指标收集
  - 5 分钟超时验证

- **Task 4 完成**: 场景2 - 分布式应用部署
  - `scenario_distributed_deploy_test.go` - 依赖部署测试
  - Database → Application 顺序验证
  - 重试和回滚机制测试

- **Task 5 完成**: 测试 Fixtures
  - `testdata/workflows/health-check.yaml`
  - `testdata/workflows/distributed-deploy.yaml`
  - `testdata/mock-app/` 模拟应用

- **Task 6 完成**: 测试报告生成
  - `report.go` - Markdown 报告生成器
  - 场景结果收集和时间统计

- **Task 7 完成**: CI/CD 集成
  - `.github/workflows/ci.yml` 添加 acceptance-tests job
  - 报告作为 artifact 保存

- **Task 9 完成**: 文档
  - `test/acceptance/README.md` - 完整文档

- **待完成**: Task 6.6 (HTML报告), Task 8 (附加场景 - 可选)

### File List

- [test/acceptance/docker-compose.acceptance.yaml](test/acceptance/docker-compose.acceptance.yaml) - 新建
- [test/acceptance/helpers.go](test/acceptance/helpers.go) - 新建
- [test/acceptance/report.go](test/acceptance/report.go) - 新建
- [test/acceptance/scenario_health_check_test.go](test/acceptance/scenario_health_check_test.go) - 新建
- [test/acceptance/scenario_distributed_deploy_test.go](test/acceptance/scenario_distributed_deploy_test.go) - 新建
- [test/acceptance/README.md](test/acceptance/README.md) - 新建
- [test/acceptance/testdata/workflows/health-check.yaml](test/acceptance/testdata/workflows/health-check.yaml) - 新建
- [test/acceptance/testdata/workflows/distributed-deploy.yaml](test/acceptance/testdata/workflows/distributed-deploy.yaml) - 新建
- [test/acceptance/testdata/mock-app/main.go](test/acceptance/testdata/mock-app/main.go) - 新建
- [test/acceptance/testdata/mock-app/Dockerfile](test/acceptance/testdata/mock-app/Dockerfile) - 新建
- [scripts/run-acceptance-tests.sh](scripts/run-acceptance-tests.sh) - 新建
- [Makefile](Makefile) - 修改 (添加 acceptance-test targets)
- [.github/workflows/ci.yml](.github/workflows/ci.yml) - 修改 (添加 acceptance-tests job)


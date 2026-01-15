# Story 10-5: 工作流示例库

## Story Card

**Epic:** Epic 10 - 完整文档体系  
**Status:** ready-for-dev  
**Priority:** Medium  
**Sprint:** 待定  
**Estimated Effort:** 8-12小时 (1-1.5工作日)

**As a** 工作流用户,  
**I want** 丰富的工作流示例,  
**So that** 学习最佳实践。

---

## Acceptance Criteria

### AC1: 至少 10 个不同场景的示例

**Given** 各种使用场景  
**When** 查阅示例库  
**Then** 提供至少 10 个不同场景的示例  
**And** 每个示例作为独立的 YAML 文件存储在 `examples/workflows/` 目录

**当前状态:**
- ✅ 已有 8 个基础示例: shell-examples.yaml, script-examples.yaml, docker-exec-examples.yaml, file-transfer-examples.yaml, sleep-examples.yaml, distributed-stack-deployment.yaml, multi-server-health-check.yaml, single-server-deployment.yaml
- ⚠️ 需要新增 2+ 个场景示例以达到 10 个目标
- 📋 建议新增场景: 数据库备份, CI/CD 集成, 通知/告警, 滚动更新/蓝绿部署, 日志收集

**验收标准:**
- 示例总数 >= 10 个 YAML 文件
- 覆盖不同用户角色: 开发者(部署), 运维(健康检查/备份), DevOps(CI/CD)
- 覆盖不同复杂度: 简单(单Job), 中等(多Job依赖), 高级(Matrix并行+条件执行)

---

### AC2: 每个示例有完整 YAML 和说明

**Given** 示例工作流文件  
**When** 用户查看示例  
**Then** 每个示例包含文件头注释说明:
- 用途和适用场景
- 前置条件(如Agent配置, SSH密钥)
- 参数说明(vars 变量)
- 运行方式(waterflow submit 或 API)
- 预期输出

**When** 用户查看 README 文档  
**Then** `examples/README.md` 提供所有示例的索引和快速导航  
**And** 每个示例有简短描述和使用场景说明  
**And** 提供快速开始命令

**验收标准:**
- ✅ examples/README.md 已存在(832行,非常完善)
- ✅ 3个生产级模板已有完整文档: docs/templates/{single-server-deployment, multi-server-health-check, distributed-stack-deployment}.md
- ⚠️ 需要补充 8 个基础示例的文档(目前只有简短描述)
- 📋 每个 YAML 文件头需添加标准化注释模板

---

### AC3: 示例覆盖场景 - 部署、健康检查、备份、测试、通知

**Given** 不同业务场景需求  
**When** 查阅示例库  
**Then** 至少覆盖以下场景:

**1. 部署场景 (已覆盖 ✅):**
- ✅ single-server-deployment.yaml - 单服务器部署 (252行,生产级)
- ✅ distributed-stack-deployment.yaml - 分布式栈部署 (Web + DB)
- 📋 建议新增: 蓝绿部署, 金丝雀部署, 滚动更新

**2. 健康检查场景 (已覆盖 ✅):**
- ✅ multi-server-health-check.yaml - 多服务器并行健康检查(Matrix策略)
- 📋 建议新增: 应用健康检查(HTTP endpoints), 数据库健康检查

**3. 备份场景 (缺失 ⚠️):**
- ⚠️ 无备份相关示例
- 📋 建议新增: 数据库备份(PostgreSQL/MySQL), 文件备份(rsync/tar), S3备份

**4. 测试场景 (缺失 ⚠️):**
- ⚠️ 无测试相关示例
- 📋 建议新增: 集成测试, 烟雾测试, 性能测试

**5. 通知场景 (缺失 ⚠️):**
- ⚠️ 无通知相关示例
- 📋 建议新增: Slack通知, 邮件通知, Webhook集成

**验收标准:**
- 5大场景全部覆盖,每个场景至少1个示例
- 现有 3 个已覆盖,需新增 3-5 个场景示例
- 每个场景示例体现最佳实践(错误处理, 重试策略, 回滚机制)

---

### AC4: 示例展示不同节点用法

**Given** Waterflow 的 7 个核心节点 (ADR-0003 插件化节点系统)  
**When** 查阅示例库  
**Then** 每个核心节点至少在 1 个示例中使用:

**节点覆盖分析:**

1. **exec/shell** - Shell命令执行 ✅
   - ✅ shell-examples.yaml - 完整覆盖(ls, cat, echo等)
   - ✅ multi-server-health-check.yaml - CPU/内存/磁盘检查

2. **exec/script** - 脚本执行 ✅
   - ✅ script-examples.yaml - 完整覆盖(bash/python脚本)
   - ✅ multi-server-health-check.yaml - Python生成报告

3. **flow/sleep** - 延迟等待 ✅
   - ✅ sleep-examples.yaml - 各种延迟场景
   - ✅ single-server-deployment.yaml - 健康检查重试间隔

4. **file/transfer** - 文件传输 ✅
   - ✅ file-transfer-examples.yaml - 上传/下载示例

5. **http/request** - HTTP请求 ✅
   - ✅ single-server-deployment.yaml - 健康检查HTTP请求
   - 📋 建议新增: API集成示例(POST/PUT/DELETE)

6. **docker/exec** - Docker命令 ✅
   - ✅ docker-exec-examples.yaml - 完整覆盖(run/ps/stop等)
   - ✅ single-server-deployment.yaml - Docker镜像构建和容器管理

7. **docker/compose** - Docker Compose ✅
   - ✅ distributed-stack-deployment.yaml - 使用docker-compose部署
   - 📋 建议新增: 独立的docker-compose示例(up/down完整生命周期)

**验收标准:**
- ✅ 7个核心节点全部在示例中使用
- 📋 建议新增: HTTP POST/PUT示例, docker-compose独立示例
- 每个节点使用示例展示最佳实践(参数配置, 错误处理)

---

### AC5: 示例展示高级 DSL 功能 (变量、表达式、条件)

**Given** Waterflow DSL 高级特性 (ADR-0004, ADR-0005)  
**When** 查阅示例库  
**Then** 至少 5 个示例展示高级 DSL 功能:

**1. 变量系统 (vars) - 已覆盖 ✅**
- ✅ 所有3个生产级模板都使用 vars 参数化配置
- 示例: `vars: {repo_url, app_name, db_server, servers}`

**2. 表达式系统 (${{ }}) - 已覆盖 ✅**
- ✅ single-server-deployment.yaml - `${{ vars.repo_url }}`
- ✅ multi-server-health-check.yaml - `${{ matrix.server }}`
- ✅ distributed-stack-deployment.yaml - `${{ vars.db_password }}`

**3. 条件执行 (if) - 部分覆盖 ⚠️**
- ⚠️ 现有示例较少使用 if 条件
- 📋 建议新增: 基于环境(dev/staging/prod)的条件部署, 基于健康检查结果的条件回滚

**4. Job 依赖 (needs) - 已覆盖 ✅**
- ✅ distributed-stack-deployment.yaml - deploy-app needs deploy-database

**5. 并行执行 (Matrix 策略) - 已覆盖 ✅**
- ✅ multi-server-health-check.yaml - `strategy.matrix.server` 并行检查

**6. 超时和重试 (timeout-minutes, continue-on-error) - 已覆盖 ✅**
- ✅ single-server-deployment.yaml - 健康检查重试策略
- ✅ multi-server-health-check.yaml - SSH命令超时配置

**7. 环境变量 (env) - 已覆盖 ✅**
- ✅ distributed-stack-deployment.yaml - DATABASE_URL环境变量注入

**验收标准:**
- ✅ 6/7 高级DSL特性已在示例中展示
- ⚠️ if条件执行需要更多示例
- 📋 建议新增: 条件部署示例, 多环境配置示例
- 每个高级特性有清晰的注释说明用途

---

### AC6: 示例展示 Task Queue 路由和并行执行

**Given** Waterflow Task Queue 直接映射机制 (ADR-0006)  
**When** 查阅示例库  
**Then** 至少 3 个示例展示 Task Queue 路由:

**Task Queue 路由示例 - 已覆盖 ✅:**

1. **单服务器部署 ✅**
   - ✅ single-server-deployment.yaml
   - 使用: `runs-on: waterflow-server` (默认队列)
   - 场景: 所有任务在Server端执行

2. **多服务器并行 ✅**
   - ✅ multi-server-health-check.yaml
   - 使用: Matrix策略 + SSH远程执行
   - 场景: 并行检查多台服务器(通过SSH,不需要每台都部署Agent)

3. **跨服务器编排 ✅**
   - ✅ distributed-stack-deployment.yaml
   - 使用: `runs-on: ${{ vars.db_server }}` 和 `runs-on: ${{ vars.app_server }}`
   - 场景: 数据库Job → db-server队列, 应用Job → app-server队列
   - 展示: Job依赖 + 跨服务器Task Queue路由

**并行执行示例 - 已覆盖 ✅:**
- ✅ multi-server-health-check.yaml - Matrix并行执行
- ✅ distributed-stack-deployment.yaml - 多Job并行(无依赖时)

**验收标准:**
- ✅ Task Queue路由示例充分(单队列, 多队列, 动态队列映射)
- ✅ 并行执行示例充分(Matrix策略, 多Job并行)
- 📋 建议新增: 服务器组概念示例(如web-servers组包含多个Agent)
- 📋 建议新增: 负载均衡示例(多个Agent注册到同一队列)

---

## Tasks

### Phase 1: 现有示例文档完善 (4-5小时)

#### Task 1: 标准化 YAML 文件头注释 ✅
- [x] 定义注释模板: 用途、前置条件、参数、运行方式、预期输出
- [ ] 为 8 个基础示例添加标准化注释:
  - shell-examples.yaml
  - script-examples.yaml
  - docker-exec-examples.yaml
  - file-transfer-examples.yaml
  - sleep-examples.yaml
  - (3个生产级模板已有完善注释,跳过)
- [ ] 每个YAML文件头添加 `# Story: 10.5` 标记

**预期输出:**
```yaml
# ========================================
# Workflow: Shell命令执行示例
# Story: 10.5 - 工作流示例库
# ========================================
#
# 用途: 演示 exec/shell 节点的各种用法
#
# 前置条件:
# - Waterflow Server 和 Agent 运行中
# - Agent 注册到 waterflow-server 队列
#
# 参数:
# - 无需额外参数,开箱即用
#
# 运行方式:
# waterflow submit examples/workflows/shell-examples.yaml
#
# 预期输出:
# - Step 1: 列出当前目录文件
# - Step 2: 显示系统信息
# - Step 3: 检查磁盘空间
# ========================================

name: Shell Command Examples
...
```

#### Task 2: 完善 examples/README.md 示例索引 ✅
- [x] examples/README.md 已存在(832行,非常完善)
- [ ] 添加"基础示例"章节,补充8个示例的详细说明:
  - 每个示例: 适用场景、使用的节点、运行命令、关键配置
- [ ] 添加"学习路径"章节:
  - 初学者: hello-world → shell-examples → single-server-deployment
  - 中级用户: multi-server-health-check → distributed-stack-deployment
  - 高级用户: 自定义Matrix策略 → 条件部署 → 蓝绿部署
- [ ] 添加"按场景分类"索引: 部署、监控、备份、测试、通知

#### Task 3: 创建示例快速参考卡片 (Cheat Sheet)
- [ ] 创建 `examples/workflows/CHEATSHEET.md`
- [ ] 内容: 所有示例的快速参考表格
- [ ] 列: 示例名称、场景、使用节点、复杂度、运行时间

| 示例 | 场景 | 节点 | 复杂度 | 运行时间 |
|------|------|------|--------|----------|
| shell-examples.yaml | Shell命令 | exec/shell | ⭐ | 1分钟 |
| multi-server-health-check.yaml | 健康检查 | exec/shell, exec/script, http/request | ⭐⭐⭐ | 5分钟 |
| ... | ... | ... | ... | ... |

---

### Phase 2: 新增场景示例 (4-6小时)

#### Task 4: 数据库备份示例
- [ ] 创建 `examples/workflows/database-backup.yaml`
- [ ] 场景: PostgreSQL/MySQL定期备份到本地或S3
- [ ] 使用节点: exec/shell (pg_dump/mysqldump), file/transfer (上传备份)
- [ ] 展示特性: vars参数化(db_host, backup_path), 超时配置, 错误处理
- [ ] 添加文件头注释

#### Task 5: CI/CD集成示例
- [ ] 创建 `examples/workflows/ci-cd-integration.yaml`
- [ ] 场景: 集成到GitHub Actions/GitLab CI触发部署
- [ ] 使用节点: exec/shell (git clone), docker/exec (构建镜像), http/request (健康检查)
- [ ] 展示特性: Job依赖(build → test → deploy), 条件部署(if: vars.env == 'production')
- [ ] 添加GitHub Actions集成说明

#### Task 6: Webhook通知示例
- [ ] 创建 `examples/workflows/webhook-notification.yaml`
- [ ] 场景: 工作流完成后发送Slack/钉钉通知
- [ ] 使用节点: http/request (POST webhook)
- [ ] 展示特性: continue-on-error (通知失败不影响主流程), 表达式(构建动态消息)
- [ ] 提供Slack/钉钉webhook配置说明

#### Task 7: 条件部署示例 (补充if特性)
- [ ] 创建 `examples/workflows/conditional-deployment.yaml`
- [ ] 场景: 基于环境变量(dev/staging/prod)执行不同部署策略
- [ ] 使用节点: exec/shell, docker/exec
- [ ] 展示特性: if条件执行(`if: ${{ vars.env == 'production' }}`), 多环境配置
- [ ] 添加环境切换最佳实践说明

#### Task 8: Docker Compose完整生命周期示例
- [ ] 创建 `examples/workflows/docker-compose-lifecycle.yaml`
- [ ] 场景: Docker Compose Up → 健康检查 → Down
- [ ] 使用节点: docker/compose (up/down), http/request (健康检查), flow/sleep (等待启动)
- [ ] 展示特性: Job依赖, 回滚策略(失败自动Down)
- [ ] 添加docker-compose.yml示例文件

---

### Phase 3: 文档整合和验证 (2-3小时)

#### Task 9: 创建模板元数据 JSON (扩展现有)
- [x] templates-metadata.json 已存在(3个生产级模板)
- [ ] 扩展为 examples-metadata.json,包含所有示例
- [ ] 结构: name, display_name, category, complexity, nodes, features, run_time
- [ ] 用于API返回和CLI查询: `waterflow examples list`

#### Task 10: 集成测试脚本
- [ ] 创建 `scripts/test-examples.sh`
- [ ] 功能: 自动化测试所有示例工作流
- [ ] 验证: YAML语法正确, 必需字段存在, 引用节点已注册
- [ ] 集成到CI: GitHub Actions自动运行

#### Task 11: 最佳实践文档
- [ ] 创建 `docs/guides/workflow-best-practices.md`
- [ ] 内容: 从示例中提取的最佳实践
  - 错误处理策略(continue-on-error, 重试配置)
  - 参数化配置(vars使用)
  - 安全实践(secrets引用, 敏感信息脱敏)
  - 性能优化(并行执行, 超时配置)
- [ ] 每条实践引用对应示例

#### Task 12: 更新主文档索引
- [ ] 更新 `docs/README.md` - 添加示例库章节
- [ ] 更新 `README.md` - 添加快速示例链接
- [ ] 更新 `docs/quick-start.md` - 引用示例作为后续学习
- [ ] 确保所有文档交叉引用正确

---

### Phase 4: 质量保证 (1-2小时)

#### Task 13: Peer Review 检查清单
- [ ] 所有示例YAML语法正确(通过 waterflow validate)
- [ ] 所有示例文件头注释完整
- [ ] examples/README.md 索引完整,所有示例可导航
- [ ] 至少10个示例覆盖5大场景
- [ ] 7个核心节点全部有使用示例
- [ ] 高级DSL特性(变量/表达式/条件/Matrix/依赖)全部展示
- [ ] Task Queue路由示例充分
- [ ] 所有示例可在CI环境运行通过

#### Task 14: 用户验收测试
- [ ] 邀请3名不同水平用户测试(初级/中级/高级)
- [ ] 验收标准:
  - 初级用户: 30分钟内完成第一个示例运行
  - 中级用户: 能够基于示例修改参数适配自己场景
  - 高级用户: 能够理解最佳实践并创建复杂工作流
- [ ] 收集反馈并改进文档

---

## Development Notes

### 现有资源分析

**已有示例 (8个):**
1. ✅ shell-examples.yaml - Shell命令示例
2. ✅ script-examples.yaml - 脚本执行示例(bash/python)
3. ✅ docker-exec-examples.yaml - Docker命令示例
4. ✅ file-transfer-examples.yaml - 文件传输示例
5. ✅ sleep-examples.yaml - 延迟等待示例
6. ✅ distributed-stack-deployment.yaml - 分布式栈部署(生产级)
7. ✅ multi-server-health-check.yaml - 多服务器健康检查(生产级)
8. ✅ single-server-deployment.yaml - 单服务器部署(生产级,252行)

**已有模板文档 (3个,非常完善):**
- ✅ docs/templates/single-server-deployment.md
- ✅ docs/templates/multi-server-health-check.md
- ✅ docs/templates/distributed-stack-deployment.md
- ✅ docs/templates/README.md - 模板库索引

**已有模板元数据:**
- ✅ examples/workflows/templates-metadata.json - 3个生产级模板的结构化元数据

**已有文档:**
- ✅ examples/README.md - 832行,非常完善,包含所有模板的详细使用说明

### Gap Analysis (差距分析)

**需要补充的示例 (至少2个,达到10个目标):**
1. ⚠️ 数据库备份示例 (PostgreSQL/MySQL)
2. ⚠️ CI/CD集成示例 (GitHub Actions/GitLab CI)
3. ⚠️ Webhook通知示例 (Slack/钉钉)
4. ⚠️ 条件部署示例 (多环境配置)
5. ⚠️ Docker Compose完整生命周期示例

**需要补充的文档:**
1. ⚠️ 8个基础示例缺少独立的详细文档页面(参考3个生产级模板的文档结构)
2. ⚠️ 8个基础示例的YAML文件头缺少标准化注释
3. ⚠️ 示例快速参考卡片(Cheatsheet)
4. ⚠️ 最佳实践文档(从示例中提取)
5. ⚠️ 学习路径指南(初级→中级→高级)

**需要补充的功能展示:**
1. ⚠️ if条件执行 - 现有示例较少使用
2. ⚠️ HTTP POST/PUT/DELETE - 现有示例主要是GET(健康检查)
3. ⚠️ 服务器组负载均衡 - 多个Agent注册到同一队列
4. ⚠️ 失败回滚策略 - continue-on-error + cleanup job

### 技术约束

**DSL 特性 (ADR-0004, ADR-0005):**
- ✅ YAML DSL语法已稳定
- ✅ 表达式系统 (${{ }}) 已实现
- ✅ 所有高级特性已在现有示例中验证

**核心节点 (ADR-0003):**
- ✅ 7个核心节点全部实现并编译为.so插件
- ✅ 所有节点在现有示例中有使用

**Task Queue路由 (ADR-0006):**
- ✅ runs-on → Task Queue直接映射机制已实现
- ✅ 3个生产级模板充分展示跨服务器编排

### 实现建议

**优先级排序:**
1. **High**: Phase 1 (现有示例文档完善) - 提升现有资源质量
2. **High**: Task 4-5 (数据库备份, CI/CD集成) - 填补关键场景空白
3. **Medium**: Task 6-7 (通知, 条件部署) - 补充高级DSL特性展示
4. **Medium**: Phase 3 (文档整合) - 提升可发现性
5. **Low**: Task 8 (Docker Compose完整示例) - nice-to-have

**工作量估算:**
- Phase 1: 4-5小时 (标准化注释 + README完善)
- Phase 2: 4-6小时 (新增5个示例)
- Phase 3: 2-3小时 (文档整合 + 测试脚本)
- Phase 4: 1-2小时 (质量保证)
- **Total: 11-16小时** (保守估计12小时 = 1.5工作日)

**质量标准:**
- 所有YAML通过 `waterflow validate` 验证
- 所有示例有完整注释和文档
- examples/README.md 作为单一入口,所有示例可导航
- CI自动化测试所有示例
- 用户验收测试通过(3个不同水平用户)

---

## Cross-References

### Related Documents

**Architecture & Design:**
- [ADR-0003: 插件化节点系统](../adr/0003-plugin-node-system.md) - 7个核心节点设计
- [ADR-0004: YAML DSL语法](../adr/0004-yaml-dsl-syntax.md) - 工作流语法定义
- [ADR-0005: 表达式系统语法](../adr/0005-expression-system-syntax.md) - ${{ }} 表达式
- [ADR-0006: Task Queue直接映射](../adr/0006-taskqueue-direct-mapping.md) - runs-on路由机制

**Requirements:**
- [PRD](../prd.md) - 产品需求,FR18: 模板库需求
- [Epic 10: 完整文档体系](../epics.md#epic-10) - 本Story所属Epic
- [Epic 6: 工作流模板库](../epics.md#epic-6) - 相关Epic,生产级模板

**Implementation:**
- [Story 6.1: 单服务器部署模板](./6-1-single-server-deployment-template.md) - 已完成,生产级模板
- [Story 6.2: 多服务器健康检查模板](./6-2-multi-server-health-check-template.md) - 已完成,生产级模板
- [Story 6.3: 分布式栈部署模板](./6-3-distributed-stack-deployment-template.md) - 已完成,生产级模板
- [Story 6.4: 模板API端点](./6-4-template-api-endpoint.md) - 已完成,templates-metadata.json

**Documentation:**
- [examples/README.md](../../examples/README.md) - 示例库索引(832行,完善)
- [docs/templates/README.md](../templates/README.md) - 模板库文档
- [docs/quick-start.md](../quick-start.md) - 快速开始指南,引用示例

### Existing Templates Analysis

**生产级模板 (已完成,高质量):**

1. **single-server-deployment.yaml (252行)**
   - 场景: 小型Web应用、MVP产品、开发/测试环境
   - 功能: Git拉取 → Docker构建 → 优雅停止 → 启动新版 → 健康检查 → 失败回滚
   - 节点: exec/shell, docker/exec, http/request, flow/sleep
   - 特性: vars参数化,重试策略,错误处理,回滚逻辑
   - 文档: docs/templates/single-server-deployment.md (完善)

2. **multi-server-health-check.yaml**
   - 场景: 服务器巡检、部署前验证、容量规划
   - 功能: Matrix并行检查 → CPU/内存/磁盘 → Python生成报告 → Markdown输出
   - 节点: exec/shell, exec/script, http/request
   - 特性: Matrix策略,跨平台兼容,阈值告警,Cron集成
   - 文档: docs/templates/multi-server-health-check.md (完善)

3. **distributed-stack-deployment.yaml**
   - 场景: Web + Database多层架构、微服务栈
   - 功能: Job依赖 → 数据库先启动 → 应用后启动 → 网络连接测试 → 失败回滚
   - 节点: docker/exec, docker/compose, http/request, exec/shell
   - 特性: Job依赖编排,跨服务器Task Queue路由,环境变量注入,健康检查
   - 文档: docs/templates/distributed-stack-deployment.md (完善)

**基础示例 (已存在,缺少文档):**

4. **shell-examples.yaml**
   - 展示: exec/shell 节点各种用法
   - 文档: ⚠️ 仅在README有简短描述,缺少独立文档和文件头注释

5. **script-examples.yaml**
   - 展示: exec/script 节点(bash/python脚本)
   - 文档: ⚠️ 仅在README有简短描述

6. **docker-exec-examples.yaml**
   - 展示: docker/exec 节点(run/ps/stop/rm等)
   - 文档: ⚠️ 仅在README有简短描述

7. **file-transfer-examples.yaml**
   - 展示: file/transfer 节点(上传/下载)
   - 文档: ⚠️ 仅在README有简短描述

8. **sleep-examples.yaml**
   - 展示: flow/sleep 节点(延迟场景)
   - 文档: ⚠️ 仅在README有简短描述

**模板元数据 (templates-metadata.json):**
```json
{
  "templates": [
    {
      "name": "single-server-deployment",
      "display_name": "Single Server Deployment",
      "category": "deployment",
      "description": "Deploy application to a single server...",
      "parameters": [...],
      "documentation": "docs/templates/single-server-deployment.md"
    },
    ...
  ]
}
```

### DSL Features Coverage Matrix

| DSL特性 | 示例文件 | 展示内容 |
|---------|----------|----------|
| **vars变量** | 所有3个生产级模板 | 参数化配置 |
| **表达式 ${{ }}** | single-server-deployment.yaml | `${{ vars.repo_url }}` |
| **if条件** | ⚠️ 缺少专门示例 | 需补充条件部署示例 |
| **needs依赖** | distributed-stack-deployment.yaml | Job依赖编排 |
| **Matrix并行** | multi-server-health-check.yaml | `strategy.matrix.server` |
| **timeout超时** | single-server-deployment.yaml | 健康检查超时 |
| **retry重试** | single-server-deployment.yaml | 健康检查重试10次 |
| **continue-on-error** | multi-server-health-check.yaml | 单个服务器失败不影响其他 |
| **env环境变量** | distributed-stack-deployment.yaml | DATABASE_URL注入 |
| **runs-on路由** | distributed-stack-deployment.yaml | 跨服务器Task Queue |

### Node Usage Coverage Matrix

| 节点 | 示例文件 | 用法说明 |
|------|----------|----------|
| **exec/shell** | shell-examples.yaml, multi-server-health-check.yaml | ls/cat/echo, CPU检查 |
| **exec/script** | script-examples.yaml, multi-server-health-check.yaml | bash/python脚本 |
| **flow/sleep** | sleep-examples.yaml, single-server-deployment.yaml | 延迟,健康检查间隔 |
| **file/transfer** | file-transfer-examples.yaml | 上传/下载 |
| **http/request** | single-server-deployment.yaml, multi-server-health-check.yaml | GET健康检查 ⚠️ 缺POST |
| **docker/exec** | docker-exec-examples.yaml, single-server-deployment.yaml | run/ps/stop |
| **docker/compose** | distributed-stack-deployment.yaml | up/down ⚠️ 缺独立示例 |

---

## Definition of Done

- [ ] **AC1**: 示例总数 >= 10个 (现有8个 + 新增2+个)
- [ ] **AC2**: 所有示例有文件头注释 + examples/README.md完善
- [ ] **AC3**: 5大场景全覆盖(部署✅, 健康检查✅, 备份⚠️, 测试⚠️, 通知⚠️)
- [ ] **AC4**: 7个核心节点全有示例
- [ ] **AC5**: 高级DSL特性全展示(变量✅, 表达式✅, 条件⚠️, 依赖✅, Matrix✅, 超时✅, 重试✅)
- [ ] **AC6**: Task Queue路由和并行执行充分展示
- [ ] 所有YAML通过 `waterflow validate` 语法检查
- [ ] CI自动化测试脚本运行通过
- [ ] 3个不同水平用户验收测试通过
- [ ] examples/README.md 作为单一入口,所有示例可导航
- [ ] 最佳实践文档完成
- [ ] 学习路径指南完成
- [ ] Code Review 通过
- [ ] 文档 Review 通过

---

## Test Strategy

### 单元测试
- YAML语法验证测试: 所有示例通过 `waterflow validate`
- 模板元数据验证: examples-metadata.json schema正确
- 文档链接检查: 所有交叉引用链接有效

### 集成测试
- **示例执行测试 (scripts/test-examples.sh):**
  - 在测试环境部署Server + Agent
  - 自动提交所有示例工作流
  - 验证: 所有示例成功执行,无错误日志
  - 覆盖: 10个示例 × 1次执行 = 10次测试

### 验收测试
- **用户验收测试 (3个用户):**
  - 初级用户: 30分钟完成第一个示例
  - 中级用户: 能基于示例修改参数
  - 高级用户: 能理解最佳实践
- **场景验证:**
  - 部署场景: single-server-deployment.yaml 成功部署示例应用
  - 健康检查场景: multi-server-health-check.yaml 生成正确报告
  - 备份场景: database-backup.yaml 成功备份并验证
  - CI/CD场景: ci-cd-integration.yaml 与GitHub Actions集成
  - 通知场景: webhook-notification.yaml 成功发送Slack消息

### 性能测试
- 示例执行时间验证: 每个示例运行时间 < 预期时间(标注在Cheatsheet)
- 文档加载速度: examples/README.md加载 < 2秒

### 文档测试
- 拼写检查: 所有文档无拼写错误
- 链接检查: 所有超链接有效
- 代码示例验证: 所有代码块可复制粘贴直接运行

---

## Risks and Mitigations

### Risk 1: 示例数量不足(< 10个)
**Impact:** 未达到AC1要求  
**Probability:** Low  
**Mitigation:**
- 现有8个 + 计划新增5个 = 13个示例,充足
- 优先实现高优先级场景(备份, CI/CD, 通知)

### Risk 2: 新增示例质量不如生产级模板
**Impact:** 用户体验下降  
**Probability:** Medium  
**Mitigation:**
- 遵循3个生产级模板的标准(完整注释, 错误处理, 回滚逻辑)
- Code Review必须检查: 参数化, 重试策略, 文档完整性
- 参考模板标准化清单

### Risk 3: 文档维护成本高
**Impact:** 文档与代码不一致  
**Probability:** Medium  
**Mitigation:**
- 使用templates-metadata.json单一数据源
- CI自动化验证: YAML与元数据一致性
- examples/README.md自动生成(从metadata)

### Risk 4: 用户找不到合适示例
**Impact:** 学习曲线陡峭  
**Probability:** Medium  
**Mitigation:**
- 多维度索引: 按场景、按节点、按复杂度
- 学习路径指南(初级→中级→高级)
- Cheatsheet快速参考卡片
- 示例标签化(tags: deployment, monitoring, backup)

### Risk 5: CI/CD集成示例无法在CI环境运行
**Impact:** 测试覆盖不足  
**Probability:** Low  
**Mitigation:**
- 所有示例设计为可在Docker环境运行
- 提供Mock服务(模拟外部依赖)
- test-examples.sh脚本支持dry-run模式

---

## Notes

### Best Practices from Existing Templates

**从3个生产级模板提取的最佳实践:**

1. **参数化设计 (vars):**
   - ✅ 所有环境相关配置提取为vars
   - ✅ 提供默认值,用户可覆盖
   - ✅ 参数分组: 服务器配置、应用配置、部署配置

2. **错误处理:**
   - ✅ 关键步骤配置 timeout-minutes
   - ✅ 临时故障配置重试策略
   - ✅ 非关键步骤使用 continue-on-error: true

3. **健康检查:**
   - ✅ 部署后必须验证服务可用
   - ✅ 健康检查重试(默认10次,间隔5秒)
   - ✅ 健康检查失败触发回滚

4. **文档完整性:**
   - ✅ YAML文件头注释(用途、前置条件、参数、运行方式)
   - ✅ 独立文档页面(参数说明、使用示例、故障排查)
   - ✅ README索引(快速导航)

5. **可观测性:**
   - ✅ 每个Step有描述性name
   - ✅ 关键步骤输出结构化日志
   - ✅ 生成可视化报告(如health_report.md)

### Documentation Structure Reference

**3个生产级模板的文档结构 (需要复制到基础示例):**

```markdown
# Template Name

## 概述
- 用途
- 适用场景
- 功能特性

## 快速开始
- 前置条件
- 复制模板
- 编辑配置
- 提交工作流
- 验证结果

## 参数说明
- 表格: 参数名、必需、默认值、说明
- 分组: 服务器配置、应用配置、部署配置

## 使用示例
- 示例1: 场景描述 + YAML配置
- 示例2: 场景描述 + YAML配置
- 示例3: 场景描述 + YAML配置

## 故障排查
- 表格: 问题、可能原因、解决方案

## 最佳实践
- 安全实践
- 性能优化
- 监控集成

## 前置条件
- 环境要求
- 网络配置
- 权限要求
```

### Metadata JSON Schema

**扩展 templates-metadata.json → examples-metadata.json:**

```json
{
  "examples": [
    {
      "name": "shell-examples",
      "display_name": "Shell Command Examples",
      "category": "basics",
      "complexity": "beginner",
      "description": "Demonstrates exec/shell node usage",
      "nodes": ["exec/shell"],
      "features": ["vars", "expressions"],
      "run_time": "1 minute",
      "documentation": "examples/README.md#shell-examples",
      "prerequisites": ["Waterflow Server", "Agent on waterflow-server queue"],
      "learning_path": "entry-level"
    },
    {
      "name": "multi-server-health-check",
      "display_name": "Multi-Server Health Check",
      "category": "monitoring",
      "complexity": "advanced",
      "description": "Parallel health check across multiple servers",
      "nodes": ["exec/shell", "exec/script", "http/request"],
      "features": ["matrix", "parallel", "ssh", "reporting"],
      "run_time": "5 minutes",
      "documentation": "docs/templates/multi-server-health-check.md",
      "prerequisites": ["SSH access", "Python 3"],
      "learning_path": "advanced"
    }
  ]
}
```

---

**Story Created:** 2025-01-XX  
**Last Updated:** 2025-01-XX  
**Assignee:** TBD  
**Reviewer:** TBD

# Story 6.3: 分布式栈部署模板

Status: ready-for-dev

## Story

As a **工作流用户**,  
I want **分布式应用栈部署模板**,  
So that **部署多层架构应用**。

## Context

这是 Epic 6 (工作流模板库) 的**第三个 Story**,实现**分布式应用栈部署模板**。该模板演示如何使用 Waterflow 部署多层架构应用 (如 Web + Database),处理服务间的依赖关系和部署顺序。

**前置依赖:**
- ✅ Epic 1 - 核心工作流引擎 (Job 依赖,条件执行)
- ✅ Epic 2 - 分布式 Agent 系统 (多服务器部署)
- ✅ Epic 3 - 核心节点插件库 (docker、shell、http 节点)
- ✅ Story 6.1 - 单服务器部署模板 (部署流程参考)
- ✅ Story 6.2 - 多服务器健康检查模板 (并行执行参考)

**Epic 背景:**  
Epic 6 专注于**工作流模板库**。本 Story 实现第三个模板:**分布式栈部署**,这是真实生产场景的典型需求,适合:
- 多层架构应用 (Frontend + Backend + Database)
- 微服务部署
- 有依赖关系的服务栈
- 复杂的部署编排

**业务价值:**
- 🎯 **依赖编排演示** - 展示 Waterflow 的 Job 依赖能力
- 🎯 **多服务器协调** - 跨不同服务器部署不同组件
- 🎯 **健康检查集成** - 确保服务就绪后才继续部署
- 🎯 **真实场景** - 解决生产级多层应用部署痛点

**模板设计原则:**
1. **依赖顺序** - 先部署基础服务 (DB),再部署应用层 (Web/API)
2. **健康验证** - 每层部署后验证就绪再继续
3. **回滚能力** - 任何步骤失败时回滚已部署的服务
4. **参数化** - 支持不同版本、配置、环境

**模板范围 (MVP):**
- ✅ 两层架构: Database + Application
- ✅ 依赖顺序控制 (needs)
- ✅ 健康检查验证
- ✅ Docker Compose 部署
- ✅ 参数化配置
- ❌ 三层架构 (Frontend + Backend + DB) - 可作为示例扩展
- ❌ 蓝绿/金丝雀部署 - Post-MVP 高级模板
- ❌ 数据库迁移 - Post-MVP (可提示用户自定义)

**典型部署场景:**
```
Step 1: 部署 PostgreSQL 数据库 (db-server)
  ↓ (健康检查:等待 PostgreSQL 就绪)
Step 2: 部署 Backend API (app-server)
  ↓ (健康检查:等待 API 就绪)
Step 3: 部署 Frontend (web-server,可选)
```

**与前两个 Story 的关系:**
- Story 6.1: 单服务器部署 (单一应用,顺序步骤)
- Story 6.2: 多服务器检查 (并行执行,无依赖)
- **本 Story**: 多服务器部署 (有依赖,顺序编排)

**技术亮点:**
- **Job 依赖** - 使用 `needs` 控制部署顺序
- **多服务器** - 不同服务部署到不同服务器 (runs-on)
- **健康检查** - 使用 http/request 确保服务就绪
- **Docker Compose** - 使用 docker/compose 节点部署服务

**文档结构:**
```
examples/workflows/
  ├── single-server-deployment.yaml       # Story 6.1
  ├── multi-server-health-check.yaml      # Story 6.2
  └── distributed-stack-deployment.yaml   # 本 Story
  
examples/
  └── README.md                            # 更新:添加分布式栈模板说明
```

## Acceptance Criteria

### AC1: 数据库部署和健康检查

**Given** 需要部署数据库作为应用基础  
**When** 使用分布式栈模板  
**Then** 第一个 Job 部署数据库:
- Job 名称: `deploy-database`
- 使用 docker/compose 节点部署 PostgreSQL
- 配置数据库版本、端口、凭证
- 等待容器启动

**And** 包含数据库健康检查步骤:
- 使用 http/request 或 shell 命令检查连接
- 重试机制 (retry: 10, delay: 5s)
- 确保数据库完全就绪

**And** 健康检查失败时工作流终止

**Implementation Notes:**
```yaml
jobs:
  deploy-database:
    runs-on: db-server
    
    steps:
      - name: Deploy PostgreSQL
        uses: docker/compose
        with:
          file: ${{ vars.db_compose_file }}
          action: up
          args: -d
      
      - name: Wait for database ready
        uses: exec/shell
        with:
          command: |
            for i in {1..10}; do
              pg_isready -h localhost -p ${{ vars.db_port }} && exit 0
              sleep 5
            done
            exit 1
```

### AC2: 应用部署依赖数据库

**Given** 数据库已部署并就绪  
**When** 应用部署 Job 运行  
**Then** 应用 Job 依赖数据库 Job:
```yaml
jobs:
  deploy-app:
    needs: [deploy-database]
    runs-on: app-server
```

**And** 应用部署步骤包含:
- 拉取应用代码或镜像
- 配置数据库连接 (环境变量)
- 启动应用容器
- 应用健康检查

**And** 应用健康检查确保 API 可访问:
- HTTP 健康检查: GET /health 或 /api/health
- 返回 200 OK
- 重试机制

**And** 应用部署失败时触发回滚:
- 停止已启动的应用容器
- 可选:停止数据库容器 (根据 rollback_on_failure 变量)
- 工作流标记为 failed

**And** 网络配置满足跨服务器通信:
- 应用服务器可访问数据库服务器的 db_port
- 使用主机名 (db_server) 或 IP 地址配置 DATABASE_URL
- 防火墙规则允许 app_server → db_server:5432

**Implementation Notes:**
- 数据库连接通过环境变量传递: `DATABASE_URL=postgres://...`
- 使用 docker run 或 docker/compose 部署
- 健康检查使用 http/request 节点
- 网络测试命令: `nc -zv db-server 5432`

### AC3: 参数化配置

**Given** 不同环境有不同的配置需求  
**When** 使用模板  
**Then** 关键配置参数化:
- `db_version` - 数据库版本 (如: postgres:14)
- `db_port` - 数据库端口 (默认: 5432)
- `db_name` - 数据库名称
- `db_user` - 数据库用户
- `db_password` - 数据库密码
- `db_init_script` - 数据库初始化脚本路径 (可选)
- `app_version` - 应用版本/镜像标签
- `app_port` - 应用端口 (默认: 3000)
- `app_repo` - 应用仓库 URL (可选,如使用 git clone)
- `deploy_environment` - 环境标识 (dev/staging/prod)
- `rollback_on_failure` - 失败时是否回滚数据库 (默认: false)

**And** 变量在 `vars` 部分集中定义  
**And** 支持默认值和环境覆盖

**Implementation Notes:**
```yaml
vars:
  # Database configuration
  db_version: "postgres:14"
  db_port: 5432
  db_name: "myapp"
  db_user: "appuser"
  db_password: "changeme"  # 生产环境应使用 Secret
  
  # Application configuration
  app_version: "latest"
  app_port: 3000
  app_image: "myapp/backend"
  
  # Deployment configuration
  deploy_timeout: 300
  health_check_retries: 10
  deploy_environment: "dev"  # dev/staging/prod
  rollback_on_failure: false  # 失败时是否回滚数据库
  db_init_script: ""  # 可选:数据库初始化脚本路径
```

### AC4: 完整的使用文档和示例

**Given** 用户首次使用分布式栈模板  
**When** 阅读模板文件或 README  
**Then** 提供以下文档:
- 模板用途和架构说明 (Web + DB 两层)
- 部署流程图解
- 参数列表和说明
- 完整的使用示例 (至少 2 个场景)
- 前置条件 (Agent 部署,Docker 安装)
- 故障排查指南

**And** 文档在 YAML 顶部作为注释  
**And** examples/README.md 包含详细使用指南  
**And** 提供完整的 docker-compose.yml 示例

**Implementation Notes:**
- 提供 Node.js + PostgreSQL 示例
- 提供 Python Flask + PostgreSQL 示例
- 包含环境变量配置说明
- 说明数据库初始化流程

## Tasks / Subtasks

### Task 1: 创建分布式栈模板 YAML (AC1, AC2, AC3)

- [ ] 1.1 定义 workflow 结构和变量
  - 创建 `examples/workflows/distributed-stack-deployment.yaml`
  - 定义 `vars` 部分 (db_*, app_*, 配置参数)
  - 添加模板顶部注释文档
  
- [ ] 1.2 实现数据库部署 Job
  - Job 名称: `deploy-database`
  - 设置 `runs-on: db-server` (或参数化)
  - 使用 docker/compose 或 docker/exec 部署 PostgreSQL
  - 添加网络检查步骤 (nc -zv 测试端口可达)
  
- [ ] 1.3 实现数据库初始化步骤 (可选)
  - 检查 db_init_script 变量是否配置
  - 使用 exec/shell 执行 SQL 初始化脚本
  - 使用 if 条件控制是否执行
  - 初始化失败时终止工作流
  
- [ ] 1.4 实现数据库健康检查步骤
  - 使用 exec/shell 执行 pg_isready 或 psql 连接测试
  - 配置重试: retry: 10, delay: 5s
  - 验证数据库可连接
  
- [ ] 1.5 实现应用部署 Job
  - Job 名称: `deploy-app`
  - 依赖: `needs: [deploy-database]`
  - 设置 `runs-on: app-server`
  - 配置数据库连接环境变量
  - 部署应用容器
  
- [ ] 1.6 实现应用健康检查步骤
  - 使用 http/request 检查 API 健康端点
  - 配置重试机制
  - 验证应用可访问
  
- [ ] 1.7 实现失败回滚逻辑
  - 在应用部署失败时停止应用容器
  - 根据 rollback_on_failure 变量决定是否停止数据库
  - 使用 continue-on-error 和条件执行

### Task 2: 创建 Docker Compose 配置示例

- [ ] 2.1 创建数据库 docker-compose.yml
  - PostgreSQL 服务定义
  - 端口映射、环境变量
  - 数据卷配置
  - 保存到 examples/configs/db-compose.yml
  
- [ ] 2.2 创建应用 docker-compose.yml (可选)
  - 应用服务定义
  - 数据库连接配置
  - 端口映射
  - 保存到 examples/configs/app-compose.yml

### Task 3: 创建模板文档和示例 (AC4)

- [ ] 3.1 添加 YAML 顶部注释文档
  - 模板用途和架构说明
  - 部署流程图 (ASCII art)
  - 参数说明表格
  - 前置条件
  - 快速开始示例
  
- [ ] 3.2 更新 examples/README.md
  - 添加"分布式栈部署"章节
  - 架构图解释
  - 完整使用示例
  - CLI/API 提交命令
  - 故障排查指南
  
- [ ] 3.3 创建使用场景示例
  - 场景 1: Node.js Express + PostgreSQL
  - 场景 2: Python Flask + PostgreSQL
  - 展示不同配置和环境变量

### Task 4: 测试和验证

- [ ] 4.1 准备测试环境
  - 启动 db-server Agent
  - 启动 app-server Agent
  - 准备测试应用镜像
  
- [ ] 4.2 测试数据库部署
  - 提交模板,只运行 deploy-database Job
  - 验证 PostgreSQL 容器启动
  - 验证健康检查通过
  - 测试数据库连接
  
- [ ] 4.3 测试完整栈部署
  - 提交完整模板
  - 验证 Job 依赖顺序 (database → app)
  - 验证应用成功连接数据库
  - 验证应用健康检查通过
  
- [ ] 4.4 测试参数化
  - 修改不同变量值
  - 测试不同数据库版本
  - 测试不同应用配置
  
- [ ] 4.5 测试回滚机制
  - 模拟应用启动失败 (使用无效镜像)
  - 验证应用容器已停止
  - 测试 rollback_on_failure=true 时数据库停止
  - 测试 rollback_on_failure=false 时数据库保持运行
  - 验证工作流状态为 failed
  
- [ ] 4.6 测试网络连接场景
  - 验证应用可连接数据库 (正常场景)
  - 模拟防火墙阻止连接 (失败场景)
  - 验证网络检查步骤的输出
  - 测试不同网络配置 (主机名 vs IP)
  
- [ ] 4.7 测试数据库初始化
  - 提供测试 SQL 脚本 (CREATE TABLE)
  - 验证脚本成功执行
  - 测试脚本失败时工作流终止
  - 测试未配置脚本时跳过初始化
  
- [ ] 4.8 验证文档完整性
  - 文档描述准确
  - 示例可运行
  - 故障排查指南有效

## Dev Notes

### Architecture Alignment

**Job 依赖 (Story 1.3):**
- ✅ 使用 `needs: [job-id]` 定义依赖
- ✅ 依赖 Job 完成后才执行
- ✅ 支持多 Job 依赖链

**多服务器路由 (Story 2.2):**
- ✅ 不同 Job 使用不同 `runs-on`
- ✅ db-server → 数据库部署
- ✅ app-server → 应用部署

**Docker 节点 (Story 3.7, 3.8):**
- ✅ docker/exec 节点 - Docker 命令执行
- ✅ docker/compose 节点 - Docker Compose 编排

**健康检查 (Story 3.5):**
- ✅ http/request 节点 - HTTP 健康检查
- ✅ exec/shell 节点 - 数据库连接测试

**条件执行 (Story 1.5):**
- ✅ 使用 `if` 处理可选步骤
- ✅ 使用 `continue-on-error` 处理非关键失败

### Project Structure

```
examples/
├── workflows/
│   ├── single-server-deployment.yaml           # Story 6.1
│   ├── multi-server-health-check.yaml          # Story 6.2
│   ├── distributed-stack-deployment.yaml       # 本 Story
│   └── ...
├── configs/
│   ├── db-compose.yml                          # 本 Story: 数据库配置
│   └── app-compose.yml                         # 本 Story: 应用配置 (可选)
├── README.md                                    # 更新:添加分布式栈说明
└── ...
```

### 关键技术决策

**1. 两层架构设计**

Database → Application 部署流程:

```yaml
name: Distributed Stack Deployment
on: workflow_dispatch

vars:
  # Database configuration
  db_server: "db-server"
  db_version: "postgres:14"
  db_port: 5432
  db_name: "myapp"
  db_user: "appuser"
  db_password: "changeme"
  db_init_script: "/opt/init.sql"  # 可选:初始化脚本
  
  # Application configuration
  app_server: "app-server"
  app_version: "latest"
  app_port: 3000
  app_image: "myapp/backend"
  
  # Deployment configuration
  deploy_environment: "dev"  # dev/staging/prod
  rollback_on_failure: false  # 失败时是否回滚数据库

jobs:
  deploy-database:
    runs-on: ${{ vars.db_server }}
    
    steps:
      - name: Deploy PostgreSQL
        uses: docker/compose
        with:
          file: examples/configs/db-compose.yml
          action: up
          args: -d
          env:
            POSTGRES_DB: ${{ vars.db_name }}
            POSTGRES_USER: ${{ vars.db_user }}
            POSTGRES_PASSWORD: ${{ vars.db_password }}
      
      - name: Wait for database ready
        uses: exec/shell
        with:
          command: |
            echo "Waiting for PostgreSQL..."
            for i in {1..30}; do
              if pg_isready -h localhost -p ${{ vars.db_port }} -U ${{ vars.db_user }}; then
                echo "PostgreSQL is ready!"
                exit 0
              fi
              echo "Attempt $i/30 failed, retrying in 5s..."
              sleep 5
            done
            echo "PostgreSQL failed to become ready"
            exit 1
      
      - name: Initialize database schema
        if: ${{ vars.db_init_script != "" }}
        uses: exec/shell
        with:
          command: |
            echo "Running database initialization script..."
            PGPASSWORD=${{ vars.db_password }} psql \
              -h localhost \
              -p ${{ vars.db_port }} \
              -U ${{ vars.db_user }} \
              -d ${{ vars.db_name }} \
              -f ${{ vars.db_init_script }}
        continue-on-error: false

  deploy-app:
    needs: [deploy-database]
    runs-on: ${{ vars.app_server }}
    
    steps:
      - name: Test database network connectivity
        uses: exec/shell
        with:
          command: |
            echo "Testing connection to database server..."
            nc -zv ${{ vars.db_server }} ${{ vars.db_port }} || {
              echo "ERROR: Cannot reach database server"
              exit 1
            }
            echo "Database server is reachable"
      
      - name: Pull application image
        uses: docker/exec
        with:
          command: pull
          args: ${{ vars.app_image }}:${{ vars.app_version }}
      
      - name: Stop old version
        uses: exec/shell
        with:
          command: docker stop myapp || true
          continue-on-error: true
      
      - name: Remove old container
        uses: exec/shell
        with:
          command: docker rm myapp || true
          continue-on-error: true
      
      - name: Start application
        uses: docker/exec
        with:
          command: run
          args: >
            -d
            --name myapp
            -p ${{ vars.app_port }}:3000
            -e DATABASE_URL=postgres://${{ vars.db_user }}:${{ vars.db_password }}@db-server:${{ vars.db_port }}/${{ vars.db_name }}
            ${{ vars.app_image }}:${{ vars.app_version }}
      
      - name: Health check application
        uses: http/request
        with:
          url: http://localhost:${{ vars.app_port }}/health
          method: GET
          retry: 10
          retry_delay: "5s"
          timeout: "3s"
  
  rollback-on-failure:
    needs: [deploy-app]
    if: ${{ failure() && vars.rollback_on_failure == 'true' }}
    runs-on: ${{ vars.app_server }}
    
    steps:
      - name: Stop application container
        uses: exec/shell
        with:
          command: docker stop myapp && docker rm myapp
          continue-on-error: true
      
      - name: Stop database container
        runs-on: ${{ vars.db_server }}
        uses: exec/shell
        with:
          command: docker-compose -f ${{ vars.db_compose_file }} down
          continue-on-error: true
```

**2. Docker Compose 配置**

**db-compose.yml:**
```yaml
version: '3.8'

services:
  postgres:
    image: ${POSTGRES_VERSION:-postgres:14}
    container_name: postgres
    restart: unless-stopped
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-myapp}
      POSTGRES_USER: ${POSTGRES_USER:-appuser}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-changeme}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-appuser}"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

**3. 数据库健康检查策略**

**方案 A: pg_isready (推荐)**
```bash
pg_isready -h localhost -p 5432 -U appuser
```

**方案 B: psql 连接测试**
```bash
PGPASSWORD=${{ vars.db_password }} psql -h localhost -p ${{ vars.db_port }} -U ${{ vars.db_user }} -d ${{ vars.db_name }} -c "SELECT 1"
```

**方案 C: Docker healthcheck**
```bash
docker inspect postgres --format='{{.State.Health.Status}}'
```

MVP 使用**方案 A** (简单,可靠)

**4. 数据库连接字符串传递**

通过环境变量传递给应用:

```bash
DATABASE_URL=postgres://user:password@host:port/database
```

示例:
```bash
DATABASE_URL=postgres://appuser:changeme@db-server:5432/myapp
```

**注意:** 生产环境应使用 Secret 管理,不硬编码密码

**5. 跨服务器网络通信**

**选项:**
- **Host 网络:** 应用使用 `db-server` 主机名连接数据库
- **Docker 网络:** 如果在同一服务器,使用 Docker 网络
- **外部 IP:** 使用数据库服务器的 IP 地址

MVP 假设使用**主机名或 IP** (用户配置)

### 测试策略

**集成测试场景:**

**场景 1: Node.js + PostgreSQL**
```bash
# 准备测试镜像
docker build -t myapp/backend:test ./test-app-nodejs

# 配置变量
vars:
  db_server: "db-server"
  app_server: "app-server"
  db_name: "testdb"
  app_image: "myapp/backend"
  app_version: "test"

# 提交工作流
waterflow submit examples/workflows/distributed-stack-deployment.yaml

# 验证
waterflow status <workflow-id>
curl http://app-server:3000/health
```

**场景 2: Python Flask + PostgreSQL**
```bash
# 准备 Flask 测试应用
docker build -t myapp/flask:test ./test-app-flask

# 配置变量
vars:
  app_image: "myapp/flask"
  app_version: "test"
  app_port: 5000

# 提交并验证
waterflow submit examples/workflows/distributed-stack-deployment.yaml
curl http://app-server:5000/api/health
```

**场景 3: 部署顺序验证**
```bash
# 验证 Job 执行顺序
waterflow logs <workflow-id> | grep -E "deploy-database|deploy-app"

# 预期输出:
# 1. deploy-database started
# 2. deploy-database completed
# 3. deploy-app started
# 4. deploy-app completed
```

**场景 4: 数据库初始化**
```bash
# 准备初始化脚本
cat > /opt/init.sql << 'EOF'
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  username VARCHAR(50) UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);
INSERT INTO users (username) VALUES ('admin');
EOF

# 配置变量
vars:
  db_init_script: "/opt/init.sql"

# 提交并验证
waterflow submit examples/workflows/distributed-stack-deployment.yaml
# 检查表已创建
psql -h db-server -U appuser -d testdb -c "\dt"
```

**场景 5: 回滚机制测试**
```bash
# 配置回滚开关
vars:
  rollback_on_failure: true
  app_image: "invalid/nonexistent"  # 使用无效镜像触发失败

# 提交工作流
waterflow submit examples/workflows/distributed-stack-deployment.yaml

# 验证
# 1. 工作流失败
# 2. 应用容器未运行
# 3. 数据库容器已停止 (因为 rollback_on_failure=true)
```

**场景 6: 健康检查失败处理**
```bash
# 模拟数据库启动失败
# 修改 db_version 为无效镜像

# 验证工作流失败
# 验证 deploy-app 未执行 (因为 needs 依赖)
```

**场景 7: 网络连接测试**
```bash
# 验证网络测试步骤
waterflow logs <workflow-id> | grep "Testing connection to database"

# 模拟网络不可达 (防火墙阻止)
# 验证工作流在网络检查步骤失败
```

### 复用现有组件

**Job 依赖 (Story 1.3):**
- ✅ `needs: [job-id]` 依赖定义

**Docker 节点 (Epic 3):**
- ✅ docker/compose (Story 3.8) - 部署数据库
- ✅ docker/exec (Story 3.7) - 部署应用

**健康检查:**
- ✅ http/request (Story 3.5) - 应用健康检查
- ✅ exec/shell (Story 3.2) - 数据库健康检查

**多服务器路由 (Story 2.2):**
- ✅ `runs-on` 指定目标服务器

### Performance Considerations

**部署时间:**
- 数据库启动: ~10-30 秒
- 健康检查重试: 最多 30 × 5秒 = 150 秒
- 应用启动: ~5-15 秒
- 总计: ~3-5 分钟 (正常情况)

**优化建议:**
- 使用预拉取的镜像 (减少下载时间)
- 合理配置健康检查间隔
- 使用 Docker 构建缓存

### Security Considerations

**数据库密码:**
- ⚠️ MVP 使用变量传递密码 (明文)
- 🔒 生产建议:使用 SecretProvider (Epic 9 Post-MVP)
- 📝 文档中明确提示安全风险

**网络暴露:**
- 数据库端口映射到宿主机 (5432)
- 建议:仅绑定内网 IP
- 使用防火墙限制访问

**容器安全:**
- 使用官方镜像
- 定期更新版本
- 扫描漏洞

### Documentation Notes

**YAML 顶部注释示例:**

```yaml
# Distributed Stack Deployment Workflow Template
# ==============================================
#
# Purpose:
#   Deploy a multi-tier application stack with dependency management
#   Example: Database + Application (2-tier architecture)
#
# Architecture:
#   ┌─────────────┐      ┌─────────────┐
#   │ DB Server   │◄─────│ App Server  │
#   │ PostgreSQL  │      │ Backend API │
#   └─────────────┘      └─────────────┘
#       Step 1              Step 2
#
# Deployment Flow:
#   1. Deploy PostgreSQL database (db-server)
#   2. Wait for database health check
#   3. Deploy application with DB connection (app-server)
#   4. Wait for application health check
#
# Use Cases:
#   - Web application with database backend
#   - Microservices with shared database
#   - Development/staging environment setup
#
# Prerequisites:
#   - Waterflow Agents on db-server and app-server
#   - Docker installed on both servers
#   - Network connectivity between servers
#   - Docker Compose files prepared (examples/configs/)
#
# Parameters:
#   Database:
#   - db_server: Target server for database (must match Agent Task Queue)
#   - db_version: PostgreSQL Docker image version (default: postgres:14)
#   - db_port: Database port (default: 5432)
#   - db_name: Database name
#   - db_user: Database username
#   - db_password: Database password (⚠️ use Secret in production)
#   - db_init_script: Path to SQL initialization script (optional)
#
#   Application:
#   - app_server: Target server for application
#   - app_image: Application Docker image
#   - app_version: Application version/tag (default: latest)
#   - app_port: Application port (default: 3000)
#
#   Deployment:
#   - deploy_environment: Environment identifier (dev/staging/prod)
#   - rollback_on_failure: Whether to rollback database on failure (default: false)
#
# Example Usage:
#   1. Prepare docker-compose.yml files
#   2. Configure variables
#   3. Submit workflow:
#      waterflow submit examples/workflows/distributed-stack-deployment.yaml
#   4. Monitor status:
#      waterflow status <workflow-id>
#   5. Verify deployment:
#      curl http://app-server:3000/health
#
# Example 1: Node.js Express + PostgreSQL
#   vars:
#     db_name: "myapp"
#     db_user: "appuser"
#     db_password: "changeme"
#     app_image: "myapp/backend"
#     app_version: "v1.0.0"
#
# Example 2: Python Flask + PostgreSQL
#   vars:
#     db_name: "flaskapp"
#     app_image: "myapp/flask"
#     app_port: 5000
#
# Troubleshooting:
#   - Database health check timeout: Increase retry count or check logs
#   - Application can't connect: Verify DATABASE_URL and network connectivity
#   - Network test fails: Check firewall rules (iptables -L, firewall-cmd --list-all)
#   - Port conflicts: Change db_port or app_port variables
#   - Initialization script fails: Check SQL syntax and permissions
#   - Rollback not triggered: Verify rollback_on_failure=true in vars
```

### References

**Epic 和 Story 文档:**
- [Source: docs/epics.md#Epic-6](../epics.md) - Epic 6 完整定义
- [Source: docs/sprint-artifacts/1-3-yaml-dsl-parsing-and-validation.md](./1-3-yaml-dsl-parsing-and-validation.md) - Job 依赖
- [Source: docs/sprint-artifacts/2-2-server-group-task-queue-mapping.md](./2-2-server-group-task-queue-mapping.md) - 多服务器路由
- [Source: docs/sprint-artifacts/3-7-docker-exec-node.md](./3-7-docker-exec-node.md) - docker/exec 节点
- [Source: docs/sprint-artifacts/3-8-docker-compose-node.md](./3-8-docker-compose-node.md) - docker/compose 节点
- [Source: docs/sprint-artifacts/3-5-http-request-node.md](./3-5-http-request-node.md) - 健康检查
- [Source: docs/sprint-artifacts/6-1-single-server-deployment-template.md](./6-1-single-server-deployment-template.md) - 单服务器模板
- [Source: docs/sprint-artifacts/6-2-multi-server-health-check-template.md](./6-2-multi-server-health-check-template.md) - 多服务器模板

**架构文档:**
- [Source: docs/architecture.md](../architecture.md) - 整体架构
- [Source: docs/prd.md](../prd.md) - 产品需求

**代码参考:**
- [Source: examples/multi-server.yaml](../../examples/multi-server.yaml) - 多服务器示例

## Definition of Done

- [ ] 创建 `examples/workflows/distributed-stack-deployment.yaml` (AC1, AC2, AC3)
- [ ] YAML 包含数据库部署 Job (deploy-database)
- [ ] YAML 包含应用部署 Job (deploy-app,依赖数据库)
- [ ] 实现数据库健康检查步骤
- [ ] 实现应用健康检查步骤
- [ ] 所有配置参数化 (db_*, app_*)
- [ ] 创建 `examples/configs/db-compose.yml` (数据库配置)
- [ ] YAML 顶部包含详细注释文档 (AC4)
- [ ] 更新 `examples/README.md` 添加分布式栈说明
- [ ] 提供 2 个使用场景示例 (Node.js, Python)
- [ ] 本地测试:数据库部署成功
- [ ] 本地测试:数据库初始化脚本执行成功 (可选功能)
- [ ] 本地测试:应用部署成功并连接数据库
- [ ] 测试依赖顺序:数据库先于应用
- [ ] 测试健康检查:验证失败时终止
- [ ] 测试回滚机制:应用失败时回滚成功
- [ ] 测试网络配置:跨服务器连接正常
- [ ] 测试不同环境变量配置 (dev/staging/prod)
- [ ] 文档审查:文档清晰、准确、完整
- [ ] 代码已提交 Git

## Dev Agent Record

### Context Reference

<!-- Story context will be added by context workflow -->

### Agent Model Used

<!-- To be filled by Dev agent -->

### Debug Log References

<!-- To be filled by Dev agent -->

### Completion Notes

<!-- To be filled by Dev agent -->

### File List

**预计创建的文件:**
- examples/workflows/distributed-stack-deployment.yaml (新建,约 400 行,包含回滚和网络检查)
- examples/configs/db-compose.yml (新建,约 30 行)
- examples/configs/init.sql (新建示例,约 10 行)

**预计修改的文件:**
- examples/README.md (更新,添加分布式栈说明,约 +200 行,包含回滚和初始化示例)

## Change Log

- 2026-01-06: Story 创建,状态: ready-for-dev

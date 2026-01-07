# 分布式栈部署模板 (Distributed Stack Deployment)

## 概述

分布式栈部署模板用于编排多层应用的部署,通过 Job 依赖管理确保组件按正确顺序启动。典型场景是先部署数据库,再部署连接数据库的应用程序,并在每个阶段进行健康检查。

**模板特点:**
- ✅ **服务依赖编排** - 数据库先启动,应用后启动,确保依赖关系
- ✅ **跨服务器部署** - 数据库和应用可部署在不同服务器
- ✅ **数据持久化** - 使用 Docker Volume 持久化数据库数据
- ✅ **健康检查** - 每个服务启动后验证可用性
- ✅ **连接验证** - 应用启动前测试到数据库的网络连接
- ✅ **可选回滚** - 失败时自动清理已部署的服务

**工作流程:**
1. **部署数据库 (Job 1)**
   - 拉取 PostgreSQL 镜像
   - 停止旧版本容器
   - 启动新容器并挂载数据卷
   - 等待数据库就绪 (`pg_isready`)
   - (可选) 执行初始化 SQL 脚本

2. **部署应用 (Job 2,依赖 Job 1)**
   - 测试到数据库的网络连接
   - 拉取应用镜像
   - 启动应用容器并注入数据库连接信息
   - 健康检查应用 API 端点

3. **回滚 (Job 3,仅失败时)**
   - 停止应用容器
   - (可选) 停止数据库容器

---

## 前置条件

### 1. Waterflow 环境

- ✅ Waterflow Server 已部署
- ✅ 至少两个 Agent 分别部署在数据库服务器和应用服务器
- ✅ Agent 注册 Task Queue (例如: `db-server`, `app-server`)

**验证 Agent 连接:**
```bash
waterflow node-list
# 预期输出:
# NODE ID    TASK QUEUE    STATUS
# node-1     db-server     ONLINE
# node-2     app-server    ONLINE
```

### 2. 服务器依赖

**数据库服务器 (db-server):**

| 工具 | 用途 | 安装命令 |
|------|------|----------|
| **Docker** | 运行 PostgreSQL 容器 | [Docker 安装指南](https://docs.docker.com/engine/install/) |
| **pg_isready** | 检查 PostgreSQL 就绪状态 | 通常随 PostgreSQL 客户端工具安装 |
| **(可选) psql** | 执行初始化 SQL 脚本 | `sudo apt install postgresql-client` |

**应用服务器 (app-server):**

| 工具 | 用途 | 安装命令 |
|------|------|----------|
| **Docker** | 运行应用容器 | [Docker 安装指南](https://docs.docker.com/engine/install/) |
| **curl 或 nc** | 测试数据库网络连接 | `sudo apt install curl netcat` |

**验证依赖:**
```bash
# 在数据库服务器
docker --version
pg_isready --version  # 可选

# 在应用服务器
docker --version
nc -zv <db-server-ip> 5432  # 测试到数据库的网络连接
```

### 3. 网络配置

- ✅ 应用服务器可通过 IP/主机名访问数据库服务器
- ✅ 防火墙允许应用服务器到数据库服务器的 5432 端口

**配置防火墙 (数据库服务器):**

```bash
# Ubuntu (ufw)
sudo ufw allow from <app-server-ip> to any port 5432

# CentOS (firewalld)
sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" source address="<app-server-ip>" port protocol="tcp" port="5432" accept'
sudo firewall-cmd --reload

# iptables
sudo iptables -A INPUT -p tcp -s <app-server-ip> --dport 5432 -j ACCEPT
```

**测试网络连接:**

```bash
# 从应用服务器测试
nc -zv <db-server-ip> 5432
# 预期输出: Connection to <db-server-ip> 5432 port [tcp/postgresql] succeeded!

# 或使用 telnet
telnet <db-server-ip> 5432
```

### 4. 应用要求

- ✅ 应用支持通过环境变量配置数据库连接
- ✅ 应用提供健康检查端点 (推荐 `/health` 或 `/api/health`)
- ✅ 应用 Docker 镜像已推送到 Docker Hub 或私有 Registry

**示例环境变量 (应用需支持):**
```bash
DATABASE_URL="postgresql://appuser:password@db-server:5432/myapp"
# 或分别配置:
DB_HOST="db-server"
DB_PORT="5432"
DB_NAME="myapp"
DB_USER="appuser"
DB_PASSWORD="password"
```

**示例健康检查端点 (Node.js Express):**
```javascript
app.get('/health', async (req, res) => {
  try {
    // 检查数据库连接
    await db.query('SELECT 1');
    res.status(200).json({ status: 'healthy', database: 'connected' });
  } catch (error) {
    res.status(500).json({ status: 'unhealthy', error: error.message });
  }
});
```

---

## 参数说明

### 数据库配置参数

| 参数 | 默认值 | 说明 | 示例 |
|------|--------|------|------|
| `db_server` | `db-server` | 数据库 Agent 的 Task Queue 名称 | `prod-db-primary` |
| `db_version` | `postgres:14` | PostgreSQL 版本/镜像 | `postgres:15-alpine` |
| `db_port` | `5432` | PostgreSQL 监听端口 | `5433` |
| `db_name` | `myapp` | 数据库名称 | `production_db` |
| `db_user` | `appuser` | 数据库用户名 | `myapp_user` |
| `db_password` | `changeme` | 数据库密码 (⚠️ 生产环境使用 Secret!) | - |
| `db_init_script` | `""` | 初始化 SQL 脚本路径 (可选) | `/opt/schema/init.sql` |
| `db_data_volume` | `postgres_data` | 数据持久化 Volume 名称 | `myapp_db_data` |
| `db_container_name` | `postgres` | 数据库容器名称 | `myapp-postgres` |

### 应用配置参数

| 参数 | 默认值 | 说明 | 示例 |
|------|--------|------|------|
| `app_server` | `app-server` | 应用 Agent 的 Task Queue 名称 | `prod-web-1` |
| `app_image` | `myapp/backend` | 应用 Docker 镜像名称 | `myorg/api-server` |
| `app_version` | `latest` | 应用镜像版本/标签 | `v2.1.0` |
| `app_port` | `3000` | 应用 HTTP 端口 | `8080` |
| `app_container_name` | `myapp` | 应用容器名称 | `api-server` |
| `app_health_endpoint` | `/health` | 健康检查 API 路径 | `/api/healthz` |

### 部署配置参数

| 参数 | 默认值 | 说明 | 示例 |
|------|--------|------|------|
| `deploy_environment` | `dev` | 环境标识 (dev/staging/prod) | `production` |
| `deploy_timeout` | `300` | 部署最大超时时间 (秒) | `600` |
| `health_check_retries` | `10` | 健康检查最大重试次数 | `20` |
| `health_check_delay` | `5` | 健康检查重试延迟 (秒) | `10` |
| `rollback_on_failure` | `false` | 失败时是否回滚数据库 | `true` |

---

## 使用示例

### 示例 1: Node.js Express + PostgreSQL (开发环境)

**场景:** 部署 Express API 应用和 PostgreSQL 数据库到开发环境

**配置文件: `express-postgres-dev.yaml`**

```yaml
name: Deploy Express API + PostgreSQL (Dev)
on: workflow_dispatch

vars:
  # 数据库配置
  db_server: "dev-db"
  db_version: "postgres:14"
  db_name: "expressapp_dev"
  db_user: "devuser"
  db_password: "dev_password_123"
  db_data_volume: "express_dev_db"
  
  # 应用配置
  app_server: "dev-app"
  app_image: "myorg/express-api"
  app_version: "develop"
  app_port: "3000"
  app_container_name: "express-api-dev"
  app_health_endpoint: "/health"
  
  # 部署配置
  deploy_environment: "dev"
  health_check_retries: 10

jobs:
  deploy-database:
    runs-on: ${{ vars.db_server }}
    steps:
      # ... (模板步骤)

  deploy-application:
    needs: deploy-database
    runs-on: ${{ vars.app_server }}
    steps:
      # ... (模板步骤)
```

**应用 Dockerfile 示例:**

```dockerfile
FROM node:18-alpine
WORKDIR /app

# 安装依赖
COPY package*.json ./
RUN npm ci --only=production

# 复制应用代码
COPY . .

# 暴露端口
EXPOSE 3000

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD node healthcheck.js

# 启动应用
CMD ["node", "server.js"]
```

**应用代码 (server.js):**

```javascript
const express = require('express');
const { Pool } = require('pg');

const app = express();

// 从环境变量读取数据库配置
const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
  // 或:
  // host: process.env.DB_HOST,
  // port: process.env.DB_PORT,
  // database: process.env.DB_NAME,
  // user: process.env.DB_USER,
  // password: process.env.DB_PASSWORD,
});

// 健康检查端点
app.get('/health', async (req, res) => {
  try {
    await pool.query('SELECT 1');
    res.status(200).json({
      status: 'healthy',
      database: 'connected',
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      status: 'unhealthy',
      error: error.message
    });
  }
});

// 示例 API 端点
app.get('/users', async (req, res) => {
  const result = await pool.query('SELECT * FROM users');
  res.json(result.rows);
});

app.listen(3000, () => {
  console.log('Server running on port 3000');
});
```

**提交工作流:**
```bash
waterflow submit express-postgres-dev.yaml
```

**验证部署:**
```bash
# 查看工作流状态
waterflow status <workflow-id>

# 测试健康检查
curl http://dev-app:3000/health
# 预期: {"status":"healthy","database":"connected","timestamp":"..."}
```

---

### 示例 2: Python Django + PostgreSQL (生产环境,含初始化脚本)

**场景:** 部署 Django 应用到生产环境,自动执行数据库迁移

**配置文件: `django-postgres-prod.yaml`**

```yaml
name: Deploy Django App + PostgreSQL (Production)
on: workflow_dispatch

vars:
  # 数据库配置
  db_server: "prod-db-primary"
  db_version: "postgres:15-alpine"
  db_name: "django_prod"
  db_user: "django_user"
  db_password: "${DB_PASSWORD}"  # 从 Secret Provider 读取
  db_init_script: "/opt/schema/init.sql"
  db_data_volume: "django_prod_db"
  
  # 应用配置
  app_server: "prod-web-1"
  app_image: "myorg/django-api"
  app_version: "v2.1.0"  # 使用特定版本,而非 latest
  app_port: "8000"
  app_container_name: "django-api-prod"
  app_health_endpoint: "/api/healthz"
  
  # 部署配置
  deploy_environment: "production"
  health_check_retries: 20
  health_check_delay: "10s"
  rollback_on_failure: true  # 生产环境启用回滚

jobs:
  deploy-database:
    runs-on: ${{ vars.db_server }}
    steps:
      # ... (模板步骤)
      
      # 额外步骤: 执行数据库初始化脚本
      - name: Initialize database schema
        if: ${{ vars.db_init_script != '' }}
        uses: exec/shell@v1
        with:
          command: |
            docker exec postgres psql -U ${{ vars.db_user }} -d ${{ vars.db_name }} -f /docker-entrypoint-initdb.d/init.sql

  deploy-application:
    needs: deploy-database
    runs-on: ${{ vars.app_server }}
    steps:
      # ... (模板步骤)
      
      # 额外步骤: 执行 Django 迁移
      - name: Run Django migrations
        uses: exec/shell@v1
        with:
          command: |
            docker exec ${{ vars.app_container_name }} python manage.py migrate --noinput
```

**初始化 SQL 脚本 (init.sql):**

```sql
-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 创建基础表
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- 插入初始数据 (可选)
INSERT INTO users (username, email) VALUES ('admin', 'admin@example.com')
ON CONFLICT (username) DO NOTHING;
```

**Django 健康检查视图 (views.py):**

```python
from django.http import JsonResponse
from django.db import connection

def health_check(request):
    try:
        # 检查数据库连接
        with connection.cursor() as cursor:
            cursor.execute("SELECT 1")
        
        return JsonResponse({
            'status': 'healthy',
            'database': 'connected',
            'version': '2.1.0'
        })
    except Exception as e:
        return JsonResponse({
            'status': 'unhealthy',
            'error': str(e)
        }, status=500)
```

---

### 示例 3: 多服务编排 (Web + API + Database)

**场景:** 部署三层架构 - 前端 Nginx、后端 API、PostgreSQL 数据库

**配置文件: `three-tier-stack.yaml`**

```yaml
name: Three-Tier Stack Deployment
on: workflow_dispatch

vars:
  # 数据库配置
  db_server: "prod-db"
  db_name: "backend_db"
  
  # API 配置
  api_server: "prod-api"
  api_image: "myorg/api-server"
  api_version: "v1.5.0"
  api_port: "4000"
  
  # Web 前端配置
  web_server: "prod-web"
  web_image: "myorg/frontend"
  web_version: "v1.5.0"
  web_port: "80"

jobs:
  # Job 1: 部署数据库
  deploy-database:
    runs-on: ${{ vars.db_server }}
    steps:
      # ... (PostgreSQL 部署步骤)

  # Job 2: 部署 API (依赖数据库)
  deploy-api:
    needs: deploy-database
    runs-on: ${{ vars.api_server }}
    steps:
      - name: Start API server
        uses: exec/shell@v1
        with:
          command: |
            docker run -d --name api-server \
              -p ${{ vars.api_port }}:4000 \
              -e DATABASE_URL="postgresql://appuser:password@${{ vars.db_server }}:5432/backend_db" \
              ${{ vars.api_image }}:${{ vars.api_version }}
      
      - name: Health check API
        uses: http/request@v1
        with:
          url: "http://localhost:${{ vars.api_port }}/health"
          method: GET
        retry-strategy:
          max-attempts: 10

  # Job 3: 部署前端 (依赖 API)
  deploy-frontend:
    needs: deploy-api
    runs-on: ${{ vars.web_server }}
    steps:
      - name: Start Nginx frontend
        uses: exec/shell@v1
        with:
          command: |
            docker run -d --name frontend \
              -p ${{ vars.web_port }}:80 \
              -e API_URL="http://${{ vars.api_server }}:${{ vars.api_port }}" \
              ${{ vars.web_image }}:${{ vars.web_version }}
      
      - name: Test frontend
        uses: http/request@v1
        with:
          url: "http://localhost:${{ vars.web_port }}"
          method: GET
          expected-status: 200
```

---

## 定制指南

### 1. 切换数据库类型

模板默认使用 PostgreSQL,可轻松切换到 MySQL/MariaDB:

**MySQL 配置:**

```yaml
vars:
  db_version: "mysql:8.0"
  db_port: 3306

jobs:
  deploy-database:
    steps:
      - name: Start MySQL container
        uses: exec/shell@v1
        with:
          command: |
            docker run -d --name mysql \
              -p 3306:3306 \
              -e MYSQL_DATABASE=${{ vars.db_name }} \
              -e MYSQL_USER=${{ vars.db_user }} \
              -e MYSQL_PASSWORD=${{ vars.db_password }} \
              -e MYSQL_ROOT_PASSWORD=root_password \
              -v ${{ vars.db_data_volume }}:/var/lib/mysql \
              ${{ vars.db_version }}
      
      - name: Wait for MySQL ready
        uses: exec/shell@v1
        with:
          command: |
            for i in $(seq 1 30); do
              if docker exec mysql mysqladmin ping -h localhost -u root -proot_password >/dev/null 2>&1; then
                echo "MySQL is ready!"
                exit 0
              fi
              sleep 2
            done
            exit 1
```

**Redis 配置:**

```yaml
vars:
  cache_server: "cache-server"
  cache_version: "redis:7-alpine"

jobs:
  deploy-cache:
    runs-on: ${{ vars.cache_server }}
    steps:
      - name: Start Redis
        uses: exec/shell@v1
        with:
          command: |
            docker run -d --name redis \
              -p 6379:6379 \
              -v redis_data:/data \
              redis:7-alpine redis-server --appendonly yes
```

### 2. 添加数据库备份步骤

在数据库部署后自动备份:

```yaml
- name: Backup existing database
  uses: exec/shell@v1
  with:
    command: |
      BACKUP_DIR="/backups/$(date +%Y%m%d)"
      mkdir -p $BACKUP_DIR
      
      docker exec postgres pg_dump -U ${{ vars.db_user }} ${{ vars.db_name }} > \
        $BACKUP_DIR/${{ vars.db_name }}_backup.sql
      
      echo "Backup saved to: $BACKUP_DIR/${{ vars.db_name }}_backup.sql"
```

### 3. 使用 Docker Compose

对于复杂栈,使用 Docker Compose 简化配置:

**docker-compose.yml:**

```yaml
version: '3.8'

services:
  database:
    image: postgres:14
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "${DB_USER}"]
      interval: 5s
      timeout: 3s
      retries: 10

  application:
    image: ${APP_IMAGE}:${APP_VERSION}
    depends_on:
      database:
        condition: service_healthy
    environment:
      DATABASE_URL: postgresql://${DB_USER}:${DB_PASSWORD}@database:5432/${DB_NAME}
    ports:
      - "${APP_PORT}:3000"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
      interval: 10s
      timeout: 3s
      retries: 5

volumes:
  postgres_data:
```

**工作流步骤:**

```yaml
- name: Deploy stack with Docker Compose
  uses: exec/shell@v1
  with:
    command: |
      cd /opt/myapp
      
      # 导出环境变量
      export DB_NAME="${{ vars.db_name }}"
      export DB_USER="${{ vars.db_user }}"
      export DB_PASSWORD="${{ vars.db_password }}"
      export APP_IMAGE="${{ vars.app_image }}"
      export APP_VERSION="${{ vars.app_version }}"
      export APP_PORT="${{ vars.app_port }}"
      
      # 启动 Compose 栈
      docker compose up -d
      
      # 等待健康检查
      docker compose ps
```

### 4. 添加 SSL/TLS 证书

为应用配置 HTTPS:

```yaml
- name: Start application with SSL
  uses: exec/shell@v1
  with:
    command: |
      docker run -d --name ${{ vars.app_container_name }} \
        -p 443:443 \
        -p 80:80 \
        -v /etc/letsencrypt:/etc/letsencrypt:ro \
        -e SSL_CERT=/etc/letsencrypt/live/example.com/fullchain.pem \
        -e SSL_KEY=/etc/letsencrypt/live/example.com/privkey.pem \
        -e DATABASE_URL="..." \
        ${{ vars.app_image }}:${{ vars.app_version }}
```

### 5. 零停机滚动更新

使用蓝绿部署策略:

```yaml
- name: Blue-Green Deployment
  uses: exec/shell@v1
  with:
    command: |
      # 启动新版本 (绿)
      docker run -d --name myapp-green \
        -p 3001:3000 \
        -e DATABASE_URL="..." \
        ${{ vars.app_image }}:${{ vars.app_version }}
      
      # 健康检查新版本
      for i in $(seq 1 10); do
        if curl -f http://localhost:3001/health; then
          echo "Green version healthy"
          break
        fi
        sleep 5
      done
      
      # 切换流量 (更新 Nginx upstream 或 Load Balancer)
      # 这里假设使用 Nginx 反向代理
      sed -i 's/3000/3001/g' /etc/nginx/conf.d/myapp.conf
      nginx -s reload
      
      # 停止旧版本 (蓝)
      docker stop myapp-blue
      docker rm myapp-blue
      
      # 重命名新版本为蓝
      docker rename myapp-green myapp-blue
```

---

## 故障排查

### 常见问题

#### 1. 数据库启动但未就绪

**错误信息:**
```
PostgreSQL is not ready (attempt 30/30)
Workflow failed
```

**原因:** 数据库初始化时间超过预期

**解决方案:**

**增加等待时间:**
```yaml
- name: Wait for PostgreSQL
  uses: exec/shell@v1
  with:
    command: |
      for i in $(seq 1 60); do  # 增加到 60 次 (2 分钟)
        if pg_isready -h localhost -p ${{ vars.db_port }}; then
          exit 0
        fi
        sleep 2
      done
      exit 1
```

**检查数据库日志:**
```bash
docker logs postgres
# 查找错误信息,如权限问题、配置错误等
```

---

#### 2. 应用无法连接数据库

**错误信息:**
```
Error: connect ECONNREFUSED db-server:5432
```

**原因:** 网络配置问题或防火墙阻塞

**解决方案:**

**1. 验证网络连接:**
```bash
# 从应用服务器测试
nc -zv db-server 5432
telnet db-server 5432
ping db-server
```

**2. 检查防火墙:**
```bash
# 在数据库服务器
sudo ufw status
sudo iptables -L -n | grep 5432

# 允许应用服务器 IP
sudo ufw allow from <app-server-ip> to any port 5432
```

**3. 验证数据库监听地址:**
```bash
# 确保 PostgreSQL 监听所有接口,而非仅 localhost
docker exec postgres grep listen_addresses /var/lib/postgresql/data/postgresql.conf
# 应为: listen_addresses = '*'

# 如需修改,编辑 PostgreSQL 配置并重启
docker exec postgres psql -U postgres -c "ALTER SYSTEM SET listen_addresses TO '*';"
docker restart postgres
```

**4. 检查 `pg_hba.conf` 允许远程连接:**
```bash
docker exec postgres cat /var/lib/postgresql/data/pg_hba.conf
# 应包含:
# host  all  all  0.0.0.0/0  md5
```

---

#### 3. 健康检查失败但应用实际正常

**错误信息:**
```
Health check failed: HTTP 404 Not Found
```

**原因:** 健康检查端点路径错误或应用未实现

**解决方案:**

**1. 验证健康检查端点:**
```bash
# 手动测试
docker exec -it myapp curl http://localhost:3000/health

# 或从宿主机测试
curl http://localhost:3000/health
```

**2. 检查应用日志:**
```bash
docker logs myapp
# 查找启动错误或路由配置问题
```

**3. 修改健康检查路径:**
```yaml
vars:
  app_health_endpoint: "/api/healthz"  # 匹配应用实际路径
```

**4. 临时禁用健康检查 (调试用):**
```yaml
- name: Health check
  uses: http/request@v1
  with:
    url: "http://localhost:${{ vars.app_port }}${{ vars.app_health_endpoint }}"
  continue-on-error: true  # 失败也继续
```

---

#### 4. 回滚未触发

**现象:** 部署失败但未执行回滚 Job

**原因:** `rollback_on_failure` 设为 `false` 或回滚条件不满足

**解决方案:**

**1. 启用回滚:**
```yaml
vars:
  rollback_on_failure: true
```

**2. 检查回滚条件:**
```yaml
jobs:
  rollback:
    if: ${{ failure() && vars.rollback_on_failure == 'true' }}
    needs: [deploy-database, deploy-application]
```

**3. 手动回滚:**
```bash
# SSH 到数据库服务器
ssh db-server

# 停止数据库容器
docker stop postgres
docker rm postgres

# 恢复备份 (如有)
docker run -d --name postgres \
  -v old_postgres_data:/var/lib/postgresql/data \
  postgres:14
```

---

#### 5. 数据库数据丢失

**原因:** Volume 未正确挂载或容器被删除时同时删除了 Volume

**解决方案:**

**1. 验证 Volume 存在:**
```bash
docker volume ls | grep postgres_data
# 应显示: local  postgres_data

# 检查 Volume 详情
docker volume inspect postgres_data
```

**2. 确保容器使用 Named Volume:**
```yaml
- name: Start PostgreSQL
  uses: exec/shell@v1
  with:
    command: |
      docker run -d --name postgres \
        -v postgres_data:/var/lib/postgresql/data \  # Named Volume
        postgres:14
```

**3. 备份 Volume 数据:**
```bash
# 备份 Volume
docker run --rm \
  -v postgres_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/postgres_backup.tar.gz /data

# 恢复 Volume
docker volume create postgres_data_new
docker run --rm \
  -v postgres_data_new:/data \
  -v $(pwd):/backup \
  alpine sh -c "cd /data && tar xzf /backup/postgres_backup.tar.gz --strip 1"
```

---

## 最佳实践

### 1. 使用 Secret Provider 管理密码

**避免硬编码密码:**

```yaml
vars:
  # ❌ 不安全
  db_password: "plain_text_password"
  
  # ✅ 使用 Secret Provider
  db_password: "${{ secrets.DB_PASSWORD }}"
```

**配置 Secret Provider:**
```bash
# 添加 Secret
waterflow secret create DB_PASSWORD "your_secure_password"

# 或使用环境变量
export DB_PASSWORD="your_secure_password"
```

### 2. 使用健康检查而非固定延迟

**避免:**
```yaml
- name: Wait for app to start
  uses: exec/shell@v1
  with:
    command: sleep 30  # ❌ 不可靠
```

**推荐:**
```yaml
- name: Health check with retry
  uses: http/request@v1
  with:
    url: "http://localhost:3000/health"
  retry-strategy:
    max-attempts: 10
    initial-interval: 5s
```

### 3. 实现幂等性

确保工作流可重复执行:

```yaml
- name: Deploy database (idempotent)
  uses: exec/shell@v1
  with:
    command: |
      # 检查容器是否已存在
      if docker ps -a | grep -q postgres; then
        echo "Container exists, updating..."
        docker stop postgres || true
        docker rm postgres || true
      fi
      
      # 启动新容器
      docker run -d --name postgres ...
```

### 4. 监控和日志

添加日志收集和监控:

```yaml
- name: Configure logging
  uses: exec/shell@v1
  with:
    command: |
      docker run -d --name myapp \
        --log-driver=json-file \
        --log-opt max-size=10m \
        --log-opt max-file=3 \
        -e ENABLE_METRICS=true \
        myapp:latest
```

### 5. 蓝绿部署避免停机

实现零停机部署:

1. 部署新版本到不同端口
2. 健康检查新版本
3. 更新负载均衡器指向新版本
4. 停止旧版本

### 6. 数据库版本控制

使用数据库迁移工具:

```yaml
- name: Run database migrations
  uses: exec/shell@v1
  with:
    command: |
      # Node.js (Sequelize)
      docker exec myapp npx sequelize-cli db:migrate
      
      # Python (Alembic)
      docker exec myapp alembic upgrade head
      
      # Go (golang-migrate)
      docker exec myapp migrate -path ./migrations -database "$DB_URL" up
```

---

## 参考资源

- **Waterflow 文档:**
  - [快速开始](../quick-start.md)
  - [Job 依赖管理](../job-dependencies.md)
  - [条件执行](../conditional-execution.md)

- **节点文档:**
  - [docker/exec 节点](../nodes/docker-exec.md)
  - [http/request 节点](../nodes/http-request.md)

- **Docker 文档:**
  - [Docker Volume 管理](https://docs.docker.com/storage/volumes/)
  - [Docker Compose](https://docs.docker.com/compose/)
  - [Docker 网络](https://docs.docker.com/network/)

- **数据库文档:**
  - [PostgreSQL Docker 镜像](https://hub.docker.com/_/postgres)
  - [MySQL Docker 镜像](https://hub.docker.com/_/mysql)

- **相关模板:**
  - [单服务器部署](./single-server-deployment.md)
  - [多服务器健康检查](./multi-server-health-check.md)

---

**上一页:** [多服务器健康检查模板](./multi-server-health-check.md)  
**下一页:** [模板库首页](./README.md)

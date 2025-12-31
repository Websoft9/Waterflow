# docker/compose@v1

**分类**: docker (容器编排)  
**版本**: v1  
**状态**: stable

## 概述

docker/compose 节点用于管理 Docker Compose 栈的完整生命周期。支持 up/down 操作，管理多容器应用部署和清理。

**设计决策**: Story 3.8 和 3.9 合并为单个节点，通过 `action` 参数区分 up/down 操作。

**前置条件**: Agent 服务器必须安装 Docker Compose（`docker compose` 或 `docker-compose`）。

## 使用场景

- **应用栈部署**: 部署包含数据库、缓存、应用的完整栈
- **开发环境**: 快速启动开发依赖（数据库、消息队列）
- **测试环境**: 创建隔离的测试环境
- **蓝绿部署**: 管理多个应用版本
- **清理资源**: 彻底清理开发/测试环境

## 参数

| 参数名 | 类型 | 必需 | 默认值 | 描述 |
|--------|------|------|--------|------|
| action | string | ✅ | - | 操作: up/down |
| file | string | ❌ | docker-compose.yml | Compose 文件路径 |
| project_name | string | ❌ | - | 项目名称 (-p) |
| workdir | string | ❌ | - | 工作目录 |
| env | object | ❌ | {} | 环境变量 |
| timeout | string | ❌ | 10m | 操作超时 |
| **Up 专用参数** |  |  |  |  |
| detach | bool | ❌ | true | 后台运行 (-d) |
| build | bool | ❌ | false | 启动前构建 (--build) |
| force_recreate | bool | ❌ | false | 强制重建 (--force-recreate) |
| **Down 专用参数** |  |  |  |  |
| volumes | bool | ❌ | false | 删除 volumes (-v) |
| rmi | string | ❌ | - | 删除镜像: none/local/all |
| remove_orphans | bool | ❌ | true | 删除孤儿容器 |

### 参数详细说明

**action** (string, required)
- 操作类型: "up" 或 "down"
- up: 启动服务栈
- down: 停止并清理服务栈

**file** (string, optional, default: docker-compose.yml)
- Compose 文件路径
- 相对于 workdir 或绝对路径
- 示例: `"docker-compose.prod.yml"`

**project_name** (string, optional)
- 项目名称，用于隔离不同环境
- 默认使用目录名
- 示例: `"myapp-prod"`

**workdir** (string, optional)
- 工作目录
- Compose 文件的相对路径基准
- 示例: `"/app/deploy"`

**env** (object, optional)
- 环境变量，注入到 Compose 环境
- 示例: `{"VERSION": "1.2.3"}`

**timeout** (string, optional, default: 10m)
- 操作超时时间
- 大型栈建议增加
- 示例: `"20m"`

**detach** (bool, optional, default: true) [Up 专用]
- 后台运行服务
- true: 启动后立即返回
- false: 前台运行（阻塞）

**build** (bool, optional, default: false) [Up 专用]
- 启动前构建镜像
- true: 执行 docker-compose up --build
- false: 使用现有镜像

**force_recreate** (bool, optional, default: false) [Up 专用]
- 强制重建容器
- true: 即使配置未变也重建
- false: 仅更新变更的容器

**volumes** (bool, optional, default: false) [Down 专用]
- 删除命名卷和匿名卷
- true: 执行 docker-compose down -v
- false: 保留 volumes

**rmi** (string, optional) [Down 专用]
- 删除镜像
- "none": 不删除镜像（默认）
- "local": 删除本地构建的镜像
- "all": 删除所有使用的镜像

**remove_orphans** (bool, optional, default: true) [Down 专用]
- 删除孤儿容器（不在 Compose 文件中定义的容器）
- 建议启用

## 返回值

| 字段名 | 类型 | 描述 |
|--------|------|------|
| action | string | up/down |
| containers | array | 容器列表 |
| services | array | 服务列表 |
| exit_code | int | 退出码 |
| stdout | string | 标准输出 |
| stderr | string | 标准错误 |
| elapsed_ms | int | 执行耗时 (毫秒) |

## Docker Compose 环境要求

### 1. 安装 Compose

```bash
# 新版 (Docker CLI plugin)
docker compose version

# 旧版 (独立二进制)
docker-compose --version

# 安装新版
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

### 2. 验证 Compose 可用

```bash
docker compose version
# 或
docker-compose --version
```

## 使用示例

### 示例 1: 基本 up/down

```yaml
steps:
  - name: Deploy application
    uses: docker/compose@v1
    with:
      action: up
      file: docker-compose.yml
      project_name: myapp
  
  - name: Run tests
    uses: exec/shell@v1
    with:
      command: ./run-tests.sh
  
  - name: Cleanup
    uses: docker/compose@v1
    with:
      action: down
      file: docker-compose.yml
      project_name: myapp
```

### 示例 2: 带构建的部署

```yaml
steps:
  - name: Deploy with build
    uses: docker/compose@v1
    with:
      action: up
      file: docker-compose.yml
      build: true  # 启动前构建镜像
      force_recreate: true  # 强制重建容器
      workdir: /app
```

### 示例 3: 完整清理（删除 volumes 和镜像）

```yaml
steps:
  - name: Complete cleanup
    uses: docker/compose@v1
    with:
      action: down
      file: docker-compose.yml
      volumes: true  # 删除 volumes
      rmi: all  # 删除所有镜像
      remove_orphans: true  # 删除孤儿容器
```

### 示例 4: 使用自定义项目名称

```yaml
steps:
  - name: Deploy production
    uses: docker/compose@v1
    with:
      action: up
      file: docker-compose.prod.yml
      project_name: myapp-prod
      env:
        VERSION: "1.2.3"
        ENVIRONMENT: production
```

### 示例 5: 环境变量注入

```yaml
steps:
  - name: Deploy with env vars
    uses: docker/compose@v1
    with:
      action: up
      file: docker-compose.yml
      env:
        DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
        API_KEY: ${{ secrets.API_KEY }}
        APP_VERSION: "2.0.0"
```

### 示例 6: 完整部署流程

```yaml
steps:
  - name: Stop old version
    uses: docker/compose@v1
    with:
      action: down
      file: docker-compose.yml
      project_name: myapp
    continue-on-error: true
  
  - name: Deploy new version
    uses: docker/compose@v1
    with:
      action: up
      file: docker-compose.yml
      project_name: myapp
      build: true
      force_recreate: true
      env:
        VERSION: "2.0.0"
  
  - name: Wait for services
    uses: flow/sleep@v1
    with:
      duration: 30s
  
  - name: Health check
    uses: http/request@v1
    with:
      url: http://localhost:8080/health
```

## Compose 文件示例

### 基本 Compose 文件

```yaml
# docker-compose.yml
version: '3.8'

services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    volumes:
      - ./html:/usr/share/nginx/html
  
  db:
    image: postgres:13
    environment:
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - db_data:/var/lib/postgresql/data

volumes:
  db_data:
```

### 使用环境变量

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    image: myapp:${VERSION}
    environment:
      DATABASE_URL: postgres://db:5432/mydb
      API_KEY: ${API_KEY}
```

## 常见错误

### 错误 1: Compose 文件不存在

**现象:**
```
Error: file not found: docker-compose.yml
```

**解决方法:**
1. 检查文件路径
2. 使用 workdir 参数指定目录
3. 使用绝对路径

### 错误 2: YAML 语法错误

**现象:**
```
Error: yaml: line 10: mapping values are not allowed in this context
```

**解决方法:**
验证 YAML 语法:
```bash
docker-compose config
```

### 错误 3: 端口冲突

**现象:**
```
Error: Bind for 0.0.0.0:80 failed: port is already allocated
```

**解决方法:**
1. 停止占用端口的容器
2. 修改 Compose 文件中的端口映射
3. 使用不同的项目名称

### 错误 4: 镜像拉取失败

**现象:**
```
Error: pull access denied for myimage, repository does not exist
```

**解决方法:**
1. 检查镜像名称
2. 先使用 docker login 登录私有仓库
3. 使用 build: true 本地构建

### 错误 5: 服务依赖错误

**现象:**
```
Error: service "web" depends on service "db" which is undefined
```

**解决方法:**
检查 Compose 文件中的 depends_on 配置。

## 最佳实践

### 1. 使用项目名称隔离环境

```yaml
# 生产环境
project_name: myapp-prod

# 测试环境
project_name: myapp-test
```

### 2. 设置合理超时

```yaml
# 简单栈
timeout: 10m  # 默认值

# 复杂栈/大镜像
timeout: 20m
```

### 3. 清理时删除 volumes

```yaml
# ✅ 完整清理（测试环境）
volumes: true
rmi: all

# ❌ 保留数据（生产环境）
volumes: false
```

### 4. 使用环境变量管理配置

```yaml
env:
  VERSION: "1.2.3"
  ENVIRONMENT: production
  DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
```

### 5. 构建前先清理

```yaml
- name: Cleanup old containers
  uses: docker/compose@v1
  with:
    action: down
    volumes: true
  continue-on-error: true

- name: Deploy new version
  uses: docker/compose@v1
  with:
    action: up
    build: true
    force_recreate: true
```

### 6. 验证部署成功

```yaml
- name: Deploy
  uses: docker/compose@v1
  with:
    action: up
  id: deploy

- name: Verify deployment
  if: steps.deploy.outputs.exit_code == 0
  uses: http/request@v1
  with:
    url: http://localhost:8080/health
```

## Up vs Down 操作对比

| 特性 | Up 操作 | Down 操作 |
|------|---------|-----------|
| **目的** | 启动服务栈 | 停止并清理栈 |
| **专用参数** | detach, build, force_recreate | volumes, rmi, remove_orphans |
| **典型场景** | 部署应用 | 环境清理 |
| **默认行为** | 后台运行 | 删除容器，保留 volumes |

## 安全注意事项

1. **Compose 文件验证**: 执行前验证文件格式
2. **资源限制**: 在 Compose 文件中配置资源限制
3. **网络隔离**: 使用自定义网络隔离服务
4. **环境变量**: 使用 Secrets 管理敏感配置
5. **超时控制**: 防止长时间运行阻塞工作流

## 错误分类

| 错误类型 | 示例 | 可重试 |
|----------|------|--------|
| **永久错误** | 文件不存在、YAML 错误、端口冲突 | ❌ |
| **临时错误** | 镜像拉取失败、网络错误 | ✅ |

## 相关节点

- [docker/exec](exec.md) - 执行 Docker CLI 命令
- [exec/shell](../exec/shell.md) - 执行系统命令

## 参考文档

- [Story 3.8: Docker Compose 节点](../../sprint-artifacts/3-8-docker-compose-node.md)
- [Docker Compose Documentation](https://docs.docker.com/compose/)

---

**文档版本**: v1  
**最后更新**: 2025-12-31  
**状态**: 稳定

# 服务器组命名指南

本文档提供 Waterflow 服务器组 (Server Group) 命名的最佳实践和规范。

## 概述

服务器组是 Waterflow 分布式系统的核心概念,通过 `runs-on` 字段将工作流任务路由到特定的 Agent 执行。服务器组名称直接映射为 Temporal Task Queue,无需额外配置。

## 命名规则

### 技术要求 (必须遵守)

根据 [ADR-0006: Task Queue 路由机制](../adr/0006-task-queue-routing.md),Task Queue 名称必须满足:

- ✅ **只包含字母、数字和连字符** (`a-z`, `A-Z`, `0-9`, `-`)
- ✅ **必须以字母或数字开头和结尾** (不能以 `-` 开头或结尾)
- ✅ **长度小于 256 个字符**
- ❌ **不能包含** 下划线 (`_`)、空格、斜杠 (`/`) 或其他特殊字符

### 验证示例

**✅ 有效名称:**
```yaml
runs-on: linux-amd64      # 操作系统 + 架构
runs-on: web-servers      # 功能组
runs-on: gpu-a100         # 硬件特性
runs-on: prod-us-west     # 环境 + 区域
runs-on: a                # 单字符
```

**❌ 无效名称:**
```yaml
runs-on: linux_amd64      # ❌ 包含下划线
runs-on: web servers      # ❌ 包含空格
runs-on: linux@amd64      # ❌ 包含特殊字符
runs-on: -linux           # ❌ 以连字符开头
runs-on: linux-           # ❌ 以连字符结尾
```

## 推荐命名模式

### 1. 操作系统 + 架构

适用于按操作系统和CPU架构分组的服务器。

**模式:** `{os}-{arch}`

```yaml
jobs:
  build-linux:
    runs-on: linux-amd64
    
  build-mac:
    runs-on: macos-arm64
    
  build-windows:
    runs-on: windows-x64
```

**常见组合:**
- `linux-amd64` - Linux x86_64
- `linux-arm64` - Linux ARM64 (如 Apple Silicon, AWS Graviton)
- `macos-arm64` - macOS ARM64
- `windows-x64` - Windows x86_64

### 2. 硬件特性

适用于具有特定硬件的服务器 (GPU, 高内存, NVMe 等)。

**模式:** `{feature}-{model}` 或 `{feature}`

```yaml
jobs:
  train-model:
    runs-on: gpu-a100       # NVIDIA A100 GPU
    
  large-dataset:
    runs-on: high-memory    # 高内存服务器
    
  fast-io:
    runs-on: nvme-storage   # NVMe 存储
```

**常见组合:**
- `gpu-a100`, `gpu-v100`, `gpu-t4` - 不同GPU型号
- `high-memory` - 大内存 (64GB+)
- `nvme-storage` - NVMe SSD
- `high-cpu` - 高CPU核心数

### 3. 环境/用途

适用于按部署环境或功能用途分组的服务器。

**模式:** `{env}` 或 `{purpose}`

```yaml
jobs:
  deploy-prod:
    runs-on: production     # 生产环境
    
  deploy-staging:
    runs-on: staging        # 测试环境
    
  build-app:
    runs-on: build-servers  # 构建服务器
    
  deploy-web:
    runs-on: web-servers    # Web 服务器
```

**常见组合:**
- `production`, `staging`, `development` - 环境
- `build-servers`, `test-servers` - 功能
- `web-servers`, `db-servers`, `cache-servers` - 服务类型

### 4. 地理位置

适用于按地理位置分组的服务器 (多地域部署)。

**模式:** `{region}` 或 `{region}-{zone}`

```yaml
jobs:
  deploy-us:
    runs-on: us-west-1      # 美国西部
    
  deploy-eu:
    runs-on: eu-central-1   # 欧洲中部
    
  deploy-asia:
    runs-on: asia-east-1    # 亚洲东部
```

**常见组合:**
- `us-west-1`, `us-east-1` - 美国
- `eu-central-1`, `eu-west-1` - 欧洲
- `asia-east-1`, `asia-southeast-1` - 亚洲

### 5. 组合命名

结合多个维度创建精确的服务器组。

```yaml
jobs:
  gpu-training-prod:
    runs-on: linux-amd64-gpu-a100-us-west
    # OS + Arch + Hardware + Region
    
  web-deploy-staging:
    runs-on: staging-web-servers
    # Environment + Purpose
```

**注意:** 保持名称简洁易读,避免过长的组合。

## 命名最佳实践

### ✅ 推荐做法

1. **保持简洁** - 通常 2-4 个单词最佳
   ```yaml
   runs-on: gpu-servers           # ✅ 简洁清晰
   runs-on: linux-ubuntu-22-04-amd64-with-docker  # ❌ 过长
   ```

2. **使用小写** - 虽然支持大写,但小写更规范
   ```yaml
   runs-on: linux-amd64           # ✅ 小写
   runs-on: Linux-AMD64           # ⚠️ 可用但不推荐
   ```

3. **见名知意** - 他人能理解服务器组用途
   ```yaml
   runs-on: web-servers           # ✅ 清晰
   runs-on: group-a               # ❌ 不明确
   ```

4. **保持一致** - 项目内统一命名风格
   ```yaml
   # ✅ 一致的模式
   runs-on: build-servers
   runs-on: test-servers
   runs-on: deploy-servers
   
   # ❌ 不一致
   runs-on: build-servers
   runs-on: testServers
   runs-on: deploy_servers
   ```

5. **避免歧义** - 使用明确的分隔
   ```yaml
   runs-on: web-prod-us-west      # ✅ 清晰层次
   runs-on: webproduswest         # ❌ 难以阅读
   ```

### ❌ 避免的做法

1. **不要使用下划线** - 使用连字符代替
   ```yaml
   runs-on: linux_amd64           # ❌ 无效
   runs-on: linux-amd64           # ✅ 有效
   ```

2. **不要使用特殊字符** - 只用字母数字和连字符
   ```yaml
   runs-on: linux@amd64           # ❌ 无效
   runs-on: web.servers           # ❌ 无效
   runs-on: build/test            # ❌ 无效
   ```

3. **不要过度组合** - 避免超长名称
   ```yaml
   runs-on: linux-ubuntu-22-04-amd64-docker-gpu-a100-prod-us-west-1-zone-a
   # ❌ 过长且难以维护
   ```

## 实际应用示例

### 示例 1: 多架构构建

```yaml
name: Multi-Architecture Build

jobs:
  build-linux-amd64:
    runs-on: linux-amd64
    steps:
      - name: Build
        uses: shell@v1
        with:
          command: make build
  
  build-linux-arm64:
    runs-on: linux-arm64
    steps:
      - name: Build
        uses: shell@v1
        with:
          command: make build
  
  build-macos-arm64:
    runs-on: macos-arm64
    steps:
      - name: Build
        uses: shell@v1
        with:
          command: make build
```

### 示例 2: 环境隔离部署

```yaml
name: Multi-Environment Deploy

jobs:
  deploy-staging:
    runs-on: staging-web-servers
    steps:
      - name: Deploy to Staging
        uses: deploy@v1
        with:
          environment: staging
  
  deploy-production:
    runs-on: prod-web-servers
    needs: [deploy-staging]
    steps:
      - name: Deploy to Production
        uses: deploy@v1
        with:
          environment: production
```

### 示例 3: GPU 训练任务

```yaml
name: Model Training

jobs:
  train-small-model:
    runs-on: gpu-t4
    steps:
      - name: Train Model
        uses: python@v1
        with:
          script: train.py
          args: --model small
  
  train-large-model:
    runs-on: gpu-a100
    steps:
      - name: Train Large Model
        uses: python@v1
        with:
          script: train.py
          args: --model large
```

### 示例 4: 地理分布式部署

```yaml
name: Global Deploy

jobs:
  deploy-us:
    runs-on: us-west-1
    steps:
      - name: Deploy US Region
        uses: deploy@v1
  
  deploy-eu:
    runs-on: eu-central-1
    steps:
      - name: Deploy EU Region
        uses: deploy@v1
  
  deploy-asia:
    runs-on: asia-east-1
    steps:
      - name: Deploy Asia Region
        uses: deploy@v1
```

## Agent 配置

Agent 通过配置文件注册到一个或多个服务器组:

```yaml
# config.agent.yaml
agent:
  task_queues:
    - linux-amd64          # 主要队列
    - linux-common         # 通用队列
    - build-servers        # 构建组
```

**多队列策略:**
- **特定 + 通用** - Agent 同时注册到特定队列 (如 `linux-amd64`) 和通用队列 (如 `linux-common`)
- **专用硬件** - GPU 服务器注册到 `gpu-a100` 和 `gpu-common`
- **环境隔离** - 生产服务器只注册 `production`,测试服务器只注册 `staging`

## 错误处理

### 验证错误

如果 `runs-on` 值不符合命名规则,Server 会拒绝工作流并返回详细错误:

```json
{
  "error": {
    "code": "validation_error",
    "message": "YAML validation failed",
    "details": {
      "errors": [
        {
          "field": "jobs.build.runs-on",
          "line": 8,
          "error": "invalid task queue name: must contain only alphanumeric characters and hyphens",
          "current_value": "linux_amd64",
          "suggestion": "Use only alphanumeric characters and hyphens (e.g., 'linux-amd64', 'web-servers')"
        }
      ]
    }
  }
}
```

### Queue 不存在

如果工作流指定的 Queue 没有 Agent 注册:

- Job 进入等待状态
- 等待直到 Agent 上线或 Job 超时
- 建议提前启动 Agent 并通过 Temporal UI 验证

```yaml
jobs:
  special-task:
    runs-on: special-hardware
    timeout-minutes: 60        # 设置合理的超时时间
```

## 查询可用服务器组

### 方法 1: Temporal UI

访问 Temporal UI 查看所有活跃的 Task Queue 和 Worker:

```
http://localhost:8088 → Workers → 查看 Task Queue 列表
```

### 方法 2: API 查询 (Story 2.7)

```bash
curl http://localhost:8080/v1/task-queues
```

### 方法 3: 文档约定

在项目中维护服务器组清单:

```markdown
# 服务器组清单

## 构建服务器
- `linux-amd64` - Linux x86_64 构建
- `linux-arm64` - Linux ARM64 构建

## 部署服务器
- `staging-web-servers` - 测试环境 Web 服务器
- `prod-web-servers` - 生产环境 Web 服务器

## GPU 服务器
- `gpu-a100` - NVIDIA A100 GPU (训练)
- `gpu-t4` - NVIDIA T4 GPU (推理)
```

## 相关文档

- [ADR-0006: Task Queue 路由机制](../adr/0006-task-queue-routing.md)
- [快速开始](../quick-start.md)
- [架构设计](../architecture.md)
- [Agent 配置指南](../../config.agent.example.yaml)

## 常见问题

### Q: 服务器组需要预先定义吗?

**A:** 不需要。Waterflow 采用零配置路由,只需在 Agent 配置中指定 Task Queue 名称并启动即可。

### Q: 一个 Agent 可以属于多个服务器组吗?

**A:** 可以。Agent 可以配置多个 Task Queue,同时轮询并接收任务。

### Q: 如何动态添加新服务器组?

**A:** 只需在新服务器上部署 Agent 并配置新的 Task Queue 名称,无需修改 Server 或重启。

### Q: 多个 Agent 注册到同一 Queue 会怎样?

**A:** Temporal 自动在多个 Agent 之间负载均衡,使用轮询策略分发任务。

### Q: Queue 名称可以包含版本号吗?

**A:** 可以,如 `gpu-v100` 或 `api-v2-servers`,只要符合命名规则即可。

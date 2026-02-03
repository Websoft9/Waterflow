# Waterflow YAML DSL 语法参考

本文档是 Waterflow 工作流 YAML DSL 的完整语法参考。

## 概述

Waterflow 使用声明式 YAML DSL 定义工作流，语法设计参考 GitHub Actions，让用户可以快速上手。工作流由 Jobs 和 Steps 组成，支持变量引用、条件执行、并行策略、超时重试等高级特性。

### 设计理念

- **声明式定义** - 描述"做什么"而非"怎么做"
- **GitHub Actions 兼容** - 熟悉的语法，降低学习成本
- **表达式系统** - 使用 `${{ }}` 语法动态求值
- **Event Sourcing** - 完整执行历史追踪（基于 Temporal）
- **单节点执行** - 每个 Step 独立超时/重试配置
- **零配置路由** - `runs-on` 直接映射到 Task Queue

### 快速示例

```yaml
name: deploy-app
vars:
  environment: production
  image: nginx:latest

jobs:
  build:
    runs-on: build-servers
    steps:
      - name: Build Docker Image
        uses: shell@v1
        with:
          command: docker
          args: ["build", "-t", "${{ vars.image }}", "."]
  
  deploy:
    runs-on: web-servers
    needs: [build]
    steps:
      - name: Deploy Container
        uses: shell@v1
        with:
          command: docker
          args: ["run", "-d", "${{ vars.image }}"]
```

---

## 顶层结构

工作流文件的顶层字段定义工作流的基本信息和全局配置。

### name

**类型:** `string`  
**必需:** 是  
**说明:** 工作流名称，用于标识和显示。

**命名规范:**
- 使用小写字母、数字、连字符
- 建议使用语义化名称（如 `deploy-app`, `health-check`）
- 长度建议 1-50 字符

```yaml
name: deploy-production-app
```

### on

**类型:** `string` 或 `object`  
**必需:** 否  
**默认值:** `workflow_dispatch`（手动触发）  
**说明:** 工作流触发器配置。

**简单触发器（字符串）:**

```yaml
on: workflow_dispatch  # 手动触发
on: push              # Git Push 触发（预留）
```

**高级触发器（对象）:**

```yaml
on:
  push:
    branches: [main, develop]
  schedule:
    cron: "0 0 * * *"  # 每天午夜
  webhook:
    events: [deployment]
```

> **注意:** MVP 版本主要支持 `workflow_dispatch` 手动触发。`push`、`schedule`、`webhook` 为预留字段。

### vars

**类型:** `map[string]interface{}`  
**必需:** 否  
**说明:** 全局变量定义，可在所有 Job 和 Step 中通过 `${{ vars.name }}` 引用。

**支持的数据类型:**
- 字符串: `"value"`
- 数字: `123`, `3.14`
- 布尔: `true`, `false`
- 数组: `["item1", "item2"]`
- 对象: `{key: value}`

```yaml
vars:
  environment: production
  replicas: 3
  enabled: true
  servers:
    - web-1
    - web-2
  config:
    timeout: 30
    retry: true
```

**变量引用:**

```yaml
steps:
  - uses: shell@v1
    with:
      command: echo
      args: ["Environment: ${{ vars.environment }}"]
```

### env

**类型:** `map[string]string`  
**必需:** 否  
**说明:** 全局环境变量，继承到所有 Job 和 Step。

**特性:**
- 支持表达式求值
- 三级继承：Workflow → Job → Step
- Step 级优先级最高

```yaml
env:
  APP_ENV: production
  LOG_LEVEL: info
  DATABASE_URL: ${{ secrets.db_url }}
```

**继承和覆盖规则:**

```yaml
env:
  GLOBAL: "workflow"
  
jobs:
  test:
    env:
      GLOBAL: "job"      # 覆盖 workflow 级
      JOB_VAR: "value"   # Job 级新变量
    steps:
      - env:
          GLOBAL: "step" # 覆盖 job 级
          # 最终环境变量: GLOBAL=step, JOB_VAR=value
```

### jobs

**类型:** `map[string]Job`  
**必需:** 是  
**说明:** Job 定义映射表，key 为 Job ID。

```yaml
jobs:
  build:
    runs-on: build-servers
    steps: [...]
  
  deploy:
    runs-on: web-servers
    needs: [build]
    steps: [...]
```

---

## Job 结构

Job 是工作流的执行单元，包含一组 Steps。

### runs-on

**类型:** `string`  
**必需:** 是  
**说明:** 指定执行 Job 的 Agent Task Queue 名称。

**Task Queue 直接映射机制:**
- `runs-on` 值直接作为 Temporal Task Queue 名称
- Agent 启动时注册到指定 Task Queue
- Temporal 自动负载均衡到 Task Queue 内的多个 Agent
- 零配置路由，无需额外配置

```yaml
jobs:
  deploy:
    runs-on: web-servers  # 路由到 web-servers Task Queue
```

**支持变量引用 (ADR-0009):**

`runs-on` 支持 `${{ vars.xxx }}` 表达式，允许通过变量动态配置执行环境。变量在工作流启动前解析。

```yaml
vars:
  target_queue: web-servers

jobs:
  deploy:
    runs-on: ${{ vars.target_queue }}  # 动态路由
```

**触发时参数覆盖:**

通过 Schedule 或 Webhook 触发时，可覆盖 `vars` 实现动态路由：

```yaml
# YAML 定义默认值
vars:
  target_queue: staging-servers

# POST /v1/executions 或 Schedule/Webhook 触发时覆盖:
# { "vars": { "target_queue": "production-servers" } }
```

> **求值时机:** `runs-on` 在工作流启动时求值（定义解析阶段），而非 Step 执行时。这确保了 Job 路由在工作流开始前确定。

**常见 Task Queue 命名:**
- `default` - 默认队列
- `build-servers` - 构建服务器
- `web-servers` - Web 服务器组
- `db-servers` - 数据库服务器组
- `linux-amd64` - 按架构分组

### needs

**类型:** `[]string`  
**必需:** 否  
**说明:** Job 依赖关系，指定当前 Job 需要等待哪些 Job 完成。

**并行/串行执行:**
- 无 `needs` - 并行执行
- 有 `needs` - 串行执行（等待依赖完成）

```yaml
jobs:
  build:
    runs-on: default
    steps: [...]
  
  test:
    runs-on: default
    needs: [build]  # 等待 build 完成
    steps: [...]
  
  deploy:
    runs-on: default
    needs: [build, test]  # 等待 build 和 test 都完成
    steps: [...]
```

**依赖输出引用:**

```yaml
jobs:
  build:
    outputs:
      image_tag: ${{ steps.build.outputs.tag }}
  
  deploy:
    needs: [build]
    steps:
      - uses: shell@v1
        with:
          command: docker
          args: ["pull", "${{ needs.build.outputs.image_tag }}"]
```

### if

**类型:** `string`  
**必需:** 否  
**说明:** Job 级条件执行，表达式求值为 `true` 时执行。

```yaml
jobs:
  deploy:
    if: ${{ vars.environment == 'production' }}
    runs-on: web-servers
```

**常见条件模式:**

```yaml
# 环境判断
if: ${{ vars.env == 'production' }}

# 依赖状态判断
if: ${{ needs.build.result == 'success' }}

# 组合条件
if: ${{ success() && vars.deploy_enabled }}

# 条件函数
if: ${{ always() }}  # 总是执行
```

### timeout-minutes

**类型:** `int` 或 `string`（表达式）  
**必需:** 否  
**默认值:** `360` (6 小时)  
**说明:** Job 级超时配置，单位：分钟。

```yaml
jobs:
  deploy:
    timeout-minutes: 30  # 30 分钟超时
```

**支持变量引用 (ADR-0009):**

`timeout-minutes` 支持 `${{ vars.xxx }}` 表达式，允许通过变量动态配置超时时间。

```yaml
vars:
  job_timeout: 60

jobs:
  deploy:
    timeout-minutes: ${{ vars.job_timeout }}  # 动态超时
```

**触发时参数覆盖:**

不同环境或触发器可配置不同超时：

```yaml
# YAML 定义默认值
vars:
  job_timeout: 30

# Schedule 触发时可绑定更长超时:
# { "vars": { "job_timeout": 120 } }
```

> **求值时机:** 在 Job 启动时解析，确保超时配置在执行前确定。

**超时机制:**
- Job 超时 → Temporal 自动终止所有 Steps
- 超时后 Job 状态为 `failed`
- 支持 Step 级独立超时配置（见 Step 结构）

### strategy

**类型:** `Strategy`  
**必需:** 否  
**说明:** Matrix 并行策略配置，用于并行执行多个相似任务。

```yaml
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu, centos, debian]
        version: ['20.04', '22.04']
      max-parallel: 3
      fail-fast: true
```

详见 [Matrix 并行策略](#matrix-并行策略) 章节。

### outputs

**类型:** `map[string]string`  
**必需:** 否  
**说明:** Job 输出定义，用于 `needs` 引用。

```yaml
jobs:
  build:
    outputs:
      image_tag: ${{ steps.build.outputs.tag }}
      commit_sha: ${{ steps.info.outputs.sha }}
    steps:
      - id: build
        uses: shell@v1
        # ... outputs.tag 由 Step 生成
```

### env

**类型:** `map[string]string`  
**必需:** 否  
**说明:** Job 级环境变量，继承 Workflow 级环境变量并可覆盖。

```yaml
jobs:
  deploy:
    env:
      DEPLOY_ENV: production
      LOG_LEVEL: debug  # 覆盖 Workflow 级
```

### continue-on-error

**类型:** `bool`  
**必需:** 否  
**默认值:** `false`  
**说明:** Job 失败容忍，设为 `true` 时 Job 失败不影响工作流。

```yaml
jobs:
  optional-task:
    continue-on-error: true
    steps: [...]
```

### steps

**类型:** `[]Step`  
**必需:** 是  
**说明:** Step 列表，按顺序执行。

```yaml
jobs:
  deploy:
    steps:
      - name: Pull Image
        uses: shell@v1
        with:
          command: docker
          args: ["pull", "nginx:latest"]
      
      - name: Start Container
        uses: shell@v1
        with:
          command: docker
          args: ["run", "-d", "nginx:latest"]
```

---

## Step 结构

Step 是 Job 中的最小执行单元，每个 Step 调用一个节点（Node）。

### id

**类型:** `string`  
**必需:** 否  
**说明:** Step 标识符，用于输出引用。

```yaml
steps:
  - id: build
    name: Build Image
    uses: shell@v1
  
  - name: Use Output
    uses: shell@v1
    with:
      command: echo
      args: ["Tag: ${{ steps.build.outputs.tag }}"]
```

**ID 命名规范:**
- 字母、数字、下划线
- 必须以字母开头
- 建议语义化（如 `build`, `deploy`, `check_health`）

### name

**类型:** `string`  
**必需:** 否  
**说明:** Step 显示名称，支持表达式。

```yaml
steps:
  - name: Deploy to ${{ vars.environment }}
    uses: shell@v1
```

### uses

**类型:** `string`  
**必需:** 是  
**格式:** `<category>/<node>@<version>` 或 `<node>@<version>`  
**说明:** 节点调用语法。

**内置节点:**

| 节点 | 说明 |
|------|------|
| `exec/shell@v1` 或 `shell@v1` | Shell 命令执行 |
| `exec/script@v1` 或 `script@v1` | 脚本文件执行 |
| `http/request@v1` 或 `http@v1` | HTTP 请求 |
| `file/transfer@v1` | 文件传输（SCP） |
| `docker/exec@v1` 或 `docker@v1` | Docker 命令 |
| `docker/compose@v1` | Docker Compose |
| `flow/sleep@v1` 或 `sleep@v1` | 延迟等待 |

```yaml
steps:
  - uses: shell@v1
  - uses: exec/shell@v1  # 完整路径
  - uses: http@v1
  - uses: docker@v1
```

### with

**类型:** `map[string]interface{}`  
**必需:** 否（部分节点必需）  
**说明:** 节点参数传递，支持表达式求值。

**Shell 节点示例:**

```yaml
- uses: shell@v1
  with:
    command: docker
    args:
      - pull
      - ${{ vars.image }}
```

**HTTP 节点示例:**

```yaml
- uses: http@v1
  with:
    url: https://api.example.com/deploy
    method: POST
    body:
      environment: ${{ vars.environment }}
      version: ${{ vars.version }}
```

**文件传输节点示例:**

```yaml
- uses: file/transfer@v1
  with:
    source: /local/app.tar.gz
    destination: /remote/app.tar.gz
    host: ${{ vars.target_host }}
    user: deploy
    private_key: ${{ secrets.ssh_key }}
```

### if

**类型:** `string`  
**必需:** 否  
**说明:** Step 级条件执行。

```yaml
steps:
  - name: Deploy to Production
    if: ${{ vars.environment == 'production' }}
    uses: shell@v1
  
  - name: Rollback
    if: ${{ failure() }}  # 仅在前面 Step 失败时执行
    uses: shell@v1
```

### timeout-minutes

**类型:** `int`  
**必需:** 否  
**说明:** Step 超时配置，继承 Job 超时或独立配置。

**单节点执行模式优势:**
- 每个 Step = 1 个 Temporal Activity
- 每个 Step 独立超时控制
- HTTP 请求和 Docker 部署可用不同超时策略

```yaml
steps:
  - name: Quick HTTP Check
    uses: http@v1
    timeout-minutes: 1  # 1 分钟超时
  
  - name: Long Docker Build
    uses: docker@v1
    timeout-minutes: 30  # 30 分钟超时
```

### retry-strategy

**类型:** `RetryStrategy`  
**必需:** 否  
**说明:** Step 重试策略配置。

**字段:**

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `max-attempts` | `int` | `3` | 最大重试次数（包含首次） |
| `initial-interval` | `string` | `1s` | 首次重试间隔 |
| `backoff-coefficient` | `float64` | `2.0` | 指数退避系数 |
| `max-interval` | `string` | `60s` | 最大重试间隔 |

**指数退避算法:**
```
重试间隔 = min(initial-interval * (backoff-coefficient ^ 重试次数), max-interval)

示例:
  第 1 次重试: 1s
  第 2 次重试: 2s (1s * 2^1)
  第 3 次重试: 4s (1s * 2^2)
  第 4 次重试: 8s (1s * 2^3)
```

**HTTP 请求重试示例:**

```yaml
- uses: http@v1
  with:
    url: https://api.example.com/status
  retry-strategy:
    max-attempts: 5
    initial-interval: 2s
    backoff-coefficient: 2.0
    max-interval: 30s
```

**可重试 vs 不可重试错误:**
- **可重试**: 网络超时、5xx 服务器错误、临时连接失败
- **不可重试**: 参数错误、404 Not Found、401 认证失败、4xx 客户端错误

### continue-on-error

**类型:** `bool`  
**必需:** 否  
**默认值:** `false`  
**说明:** Step 失败容忍。

```yaml
steps:
  - name: Optional Cleanup
    continue-on-error: true
    uses: shell@v1
```

### env

**类型:** `map[string]string`  
**必需:** 否  
**说明:** Step 级环境变量，优先级最高。

```yaml
steps:
  - uses: shell@v1
    env:
      DEBUG: "true"
      STEP_VAR: "value"
```

---

## 表达式语法

表达式使用 `${{ expression }}` 语法，运行时动态求值。

### 基础语法

**格式:** `${{ expression }}`  
**引擎:** antonmedv/expr (安全沙箱)  
**可用位置:** 所有字段值（name、if、with 参数等）

```yaml
name: Deploy to ${{ vars.environment }}
if: ${{ vars.enabled && success() }}
with:
  command: ${{ vars.deploy_command }}
  args: ["--env", "${{ vars.environment }}"]
```

### 求值时机 (ADR-0009)

表达式求值分为两个阶段：

**1. 定义解析阶段（工作流启动时）:**

以下结构性字段在工作流启动时解析，确保执行计划在运行前确定：

| 字段 | 说明 |
|------|------|
| `runs-on` | Job 路由到哪个 Task Queue |
| `timeout-minutes` | Job/Step 超时配置 |
| `strategy.matrix` | Matrix 并行维度 |

```yaml
vars:
  target_queue: web-servers
  timeout: 60
  servers: [web-1, web-2]

jobs:
  deploy:
    runs-on: ${{ vars.target_queue }}      # 启动时解析
    timeout-minutes: ${{ vars.timeout }}   # 启动时解析
    strategy:
      matrix:
        server: ${{ vars.servers }}        # 启动时解析
```

**2. Step 执行阶段（每个 Step 执行前）:**

其他字段在 Step 执行时解析，可使用完整上下文（包括前序 Step 输出）：

| 字段 | 说明 |
|------|------|
| `name` | Step 显示名称 |
| `if` | 条件执行 |
| `with` | 节点参数 |
| `env` | 环境变量 |

```yaml
steps:
  - id: config
    uses: shell@v1
    with:
      command: echo
      args: ["Getting config..."]
    # outputs.version 由节点生成

  - name: Deploy version ${{ steps.config.outputs.version }}  # 执行时解析
    if: ${{ success() }}                                       # 执行时解析
    with:
      version: ${{ steps.config.outputs.version }}            # 执行时解析
```

> **三层参数覆盖:** vars 支持三层覆盖机制：YAML 默认值 → 触发器绑定（Schedule/Webhook）→ 执行时参数。详见 [ADR-0009](adr/0009-workflow-definition-execution-separation.md)。

### 上下文变量

所有可用的上下文变量：

#### vars.*

全局变量引用，支持嵌套访问和数组索引。

```yaml
vars:
  environment: production
  servers: [web-1, web-2]
  config:
    timeout: 30

# 引用
${{ vars.environment }}        # "production"
${{ vars.servers[0] }}         # "web-1"
${{ vars.config.timeout }}     # 30
```

#### env.*

环境变量引用。

```yaml
env:
  LOG_LEVEL: info

# 引用
${{ env.LOG_LEVEL }}  # "info"
```

#### matrix.*

Matrix 变量引用（仅在 Matrix Job 中可用）。

```yaml
strategy:
  matrix:
    os: [ubuntu, centos]
    version: ['20.04', '22.04']

# 引用
${{ matrix.os }}       # "ubuntu" 或 "centos"
${{ matrix.version }}  # "20.04" 或 "22.04"
```

#### steps.<id>.outputs.*

Step 输出引用。

```yaml
- id: build
  # ... outputs.tag 由节点生成

- name: Use Output
  with:
    image: ${{ steps.build.outputs.tag }}
```

#### needs.<job>.outputs.*

Job 依赖输出引用。

```yaml
jobs:
  build:
    outputs:
      version: ${{ steps.info.outputs.ver }}
  
  deploy:
    needs: [build]
    steps:
      - with:
          version: ${{ needs.build.outputs.version }}
```

#### workflow.*

工作流元数据。

```yaml
${{ workflow.name }}  # 工作流名称
${{ workflow.id }}    # 工作流 ID (运行时生成)
```

#### job.*

Job 元数据。

```yaml
${{ job.name }}    # Job 名称
${{ job.status }}  # Job 状态 (pending/running/completed/failed)
```

#### runner.*

Agent 信息。

```yaml
${{ runner.os }}    # 操作系统 (linux/darwin/windows)
${{ runner.arch }}  # 架构 (amd64/arm64)
```

#### secrets.*

密钥引用（通过 SecretProvider 接口获取）。

```yaml
with:
  password: ${{ secrets.db_password }}
  api_key: ${{ secrets.api_key }}
```

> **安全注意:** 密钥在日志中自动脱敏，显示为 `***`。

### 运算符

#### 算术运算符

```yaml
${{ 1 + 2 }}        # 3
${{ 10 - 3 }}       # 7
${{ 4 * 5 }}        # 20
${{ 20 / 4 }}       # 5
${{ 10 % 3 }}       # 1
${{ 2 ** 3 }}       # 8 (幂运算)
```

#### 比较运算符

```yaml
${{ 5 == 5 }}       # true
${{ 5 != 3 }}       # true
${{ 5 > 3 }}        # true
${{ 3 < 5 }}        # true
${{ 5 >= 5 }}       # true
${{ 3 <= 5 }}       # true
```

#### 逻辑运算符

```yaml
${{ true && false }}     # false
${{ true || false }}     # true
${{ !false }}            # true
```

#### 字符串运算符

```yaml
${{ contains('hello world', 'world') }}      # true
${{ startsWith('hello', 'hel') }}            # true
${{ endsWith('hello', 'lo') }}               # true
```

### 内置函数

#### 字符串处理

```yaml
${{ len("hello") }}              # 5
${{ upper("hello") }}            # "HELLO"
${{ lower("HELLO") }}            # "hello"
${{ trim("  hello  ") }}         # "hello"
${{ split("a,b,c", ",") }}       # ["a", "b", "c"]
${{ join(["a", "b"], ",") }}     # "a,b"
${{ format("Hello %s", "World") }}  # "Hello World"
```

#### 数组操作

```yaml
${{ len([1, 2, 3]) }}            # 3
${{ vars.servers[0] }}           # 数组索引
```

#### JSON 转换

```yaml
${{ toJSON({key: "value"}) }}    # '{"key":"value"}'
${{ fromJSON('{"key":"value"}') }}  # {key: "value"}
```

#### 条件函数

```yaml
${{ success() }}      # 前面所有 Step 成功
${{ failure() }}      # 任一 Step 失败
${{ cancelled() }}    # 工作流被取消
${{ always() }}       # 总是返回 true
```

**使用场景:**

```yaml
steps:
  - name: Build
    uses: shell@v1
  
  - name: Cleanup on Failure
    if: ${{ failure() }}
    uses: shell@v1
  
  - name: Always Run
    if: ${{ always() }}
    uses: shell@v1
```

### 复杂表达式示例

**嵌套运算:**

```yaml
${{ (vars.replicas * 2) + 1 }}
```

**条件组合:**

```yaml
if: ${{ vars.environment == 'production' && success() }}
```

**字符串拼接:**

```yaml
${{ 'https://' + vars.domain + '/api' }}
```

**数组/对象访问:**

```yaml
${{ vars.config.servers[0].host }}
```

---

## Matrix 并行策略

Matrix 策略用于并行执行多个相似任务，如多服务器部署、跨环境测试。

### 基础语法

```yaml
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu, centos, debian]
        version: ['20.04', '22.04']
```

### 动态 Matrix 定义 (ADR-0009)

Matrix 值支持 `${{ vars.xxx }}` 表达式，允许通过变量动态配置并行维度。

**动态服务器列表:**

```yaml
vars:
  target_servers:
    - web-1.example.com
    - web-2.example.com

jobs:
  deploy:
    strategy:
      matrix:
        server: ${{ vars.target_servers }}  # 动态 Matrix
    steps:
      - name: Deploy to ${{ matrix.server }}
        uses: http@v1
        with:
          url: "https://${{ matrix.server }}/deploy"
```

**触发时覆盖服务器列表:**

```yaml
# YAML 定义默认（staging 环境）
vars:
  target_servers:
    - staging-1.example.com
    - staging-2.example.com

# Schedule/Webhook 触发时可覆盖为生产环境:
# POST /v1/executions
# { 
#   "workflow_name": "deploy-app",
#   "vars": { 
#     "target_servers": ["prod-1.example.com", "prod-2.example.com", "prod-3.example.com"]
#   }
# }
```

> **求值时机:** Matrix 表达式在工作流启动时解析（定义解析阶段），确保 Matrix 展开在 Job 执行前完成。

### Matrix 展开规则

Matrix 使用笛卡尔积算法展开所有组合：

```yaml
strategy:
  matrix:
    os: [ubuntu, centos]
    version: ['20.04', '22.04']

# 展开为 4 个实例:
# 1. os=ubuntu, version=20.04
# 2. os=ubuntu, version=22.04
# 3. os=centos, version=20.04
# 4. os=centos, version=22.04
```

**组合数限制:** 最大 256 个组合

### Matrix 变量引用

在 Matrix Job 的 Steps 中使用 `${{ matrix.* }}` 引用：

```yaml
strategy:
  matrix:
    server: [web-1, web-2, web-3]

steps:
  - name: Deploy to ${{ matrix.server }}
    uses: shell@v1
    with:
      command: ssh
      args: ["${{ matrix.server }}", "deploy.sh"]
```

### max-parallel

控制并发执行数量：

```yaml
strategy:
  matrix:
    server: [web-1, web-2, web-3, web-4, web-5]
  max-parallel: 2  # 每次最多并行 2 个
```

- **默认值:** 无限制（全并行）
- **推荐值:** 根据资源限制设置（如 Agent 数量）

### fail-fast

失败策略配置：

```yaml
strategy:
  matrix:
    server: [web-1, web-2, web-3]
  fail-fast: true  # 任一失败立即取消其他实例
```

- **默认值:** `true`
- **true:** 任一实例失败立即取消其他实例
- **false:** 所有实例独立执行，不受其他实例影响

**对比示例:**

```yaml
# fail-fast: true (默认)
# web-1 失败 → web-2, web-3 立即取消

# fail-fast: false
# web-1 失败 → web-2, web-3 继续执行
```

### Matrix 实例追踪

每个 Matrix 实例：
- 独立的 Temporal Activity
- 独立的状态追踪
- 独立的日志流
- 独立的超时/重试配置

### 完整 Matrix 示例

**多服务器并行健康检查:**

```yaml
name: multi-server-health-check
vars:
  timeout: 30

jobs:
  health-check:
    runs-on: default
    strategy:
      matrix:
        server:
          - web-1.example.com
          - web-2.example.com
          - db-1.example.com
      max-parallel: 3
      fail-fast: false  # 所有服务器独立检查
    
    steps:
      - name: Check ${{ matrix.server }}
        uses: http@v1
        with:
          url: "https://${{ matrix.server }}/health"
          method: GET
        timeout-minutes: 1
        retry-strategy:
          max-attempts: 3
          initial-interval: 2s
```

**多维 Matrix 示例:**

```yaml
jobs:
  cross-platform-test:
    strategy:
      matrix:
        os: [ubuntu, centos]
        version: ['20.04', '22.04']
        arch: [amd64, arm64]
    
    steps:
      - name: Test on ${{ matrix.os }}-${{ matrix.version }}-${{ matrix.arch }}
        uses: shell@v1
        with:
          command: ./test.sh
          args:
            - --os=${{ matrix.os }}
            - --version=${{ matrix.version }}
            - --arch=${{ matrix.arch }}
```

### MVP 限制

**暂不支持 include/exclude:**

```yaml
# ❌ MVP 不支持
strategy:
  matrix:
    os: [ubuntu, centos]
  include:
    - os: ubuntu
      extra: value
  exclude:
    - os: centos
```

**替代方案:** 使用多个 Job 或条件执行

```yaml
jobs:
  ubuntu-job:
    if: ${{ vars.os == 'ubuntu' }}
  
  centos-job:
    if: ${{ vars.os == 'centos' }}
```

---

## 超时和重试配置

### Job 超时

**字段:** `timeout-minutes`  
**默认值:** `360` (6 小时)  
**说明:** Job 级超时，所有 Steps 累计执行时间不超过此值。

```yaml
jobs:
  long-running-job:
    timeout-minutes: 120  # 2 小时
```

### Step 超时

**字段:** `timeout-minutes`  
**默认值:** 继承 Job 超时  
**说明:** Step 独立超时配置。

```yaml
steps:
  - name: Quick Check
    timeout-minutes: 1
  
  - name: Long Build
    timeout-minutes: 30
```

**单节点执行模式优势:**
- 每个 Step = 1 个 Activity
- 每个 Step 独立超时
- HTTP 请求 1 分钟 vs Docker 构建 30 分钟

### 重试策略

**字段:** `retry-strategy`  
**适用:** Step 级  
**默认值:**
- `max-attempts: 3`
- `initial-interval: 1s`
- `backoff-coefficient: 2.0`
- `max-interval: 60s`

**完整配置:**

```yaml
steps:
  - uses: http@v1
    retry-strategy:
      max-attempts: 5
      initial-interval: 2s
      backoff-coefficient: 2.0
      max-interval: 30s
```

**指数退避示例:**

```
重试间隔序列 (initial=2s, coefficient=2.0, max=30s):
  重试 1: 2s
  重试 2: 4s (2s * 2^1)
  重试 3: 8s (2s * 2^2)
  重试 4: 16s (2s * 2^3)
  重试 5: 30s (min(32s, 30s))
```

### 可重试错误分类

**Temporal 自动重试:**
- 网络超时
- 5xx 服务器错误
- 临时连接失败
- DNS 解析失败

**不可重试错误（NonRetryable）:**
- 参数验证错误
- 404 Not Found
- 401/403 认证/授权错误
- 4xx 客户端错误
- YAML 语法错误

### 超时和重试最佳实践

**HTTP 请求:**

```yaml
- uses: http@v1
  timeout-minutes: 1
  retry-strategy:
    max-attempts: 3
    initial-interval: 1s
```

**Docker 部署:**

```yaml
- uses: docker@v1
  timeout-minutes: 30
  retry-strategy:
    max-attempts: 2
    initial-interval: 5s
```

**分布式任务:**

```yaml
- uses: shell@v1
  timeout-minutes: 60
  retry-strategy:
    max-attempts: 5
    initial-interval: 10s
    backoff-coefficient: 1.5
    max-interval: 60s
```

---

## 完整工作流示例

### 示例 1: 简单工作流

单 Job，多 Step，变量引用和环境变量。

```yaml
name: simple-deployment
vars:
  app_name: myapp
  version: v1.0.0

env:
  LOG_LEVEL: info

jobs:
  deploy:
    runs-on: web-servers
    env:
      DEPLOY_ENV: production
    
    steps:
      - name: Pull Docker Image
        uses: shell@v1
        with:
          command: docker
          args:
            - pull
            - "${{ vars.app_name }}:${{ vars.version }}"
      
      - name: Stop Old Container
        uses: shell@v1
        with:
          command: docker
          args: ["stop", "${{ vars.app_name }}"]
        continue-on-error: true
      
      - name: Start New Container
        uses: shell@v1
        with:
          command: docker
          args:
            - run
            - -d
            - --name=${{ vars.app_name }}
            - -e
            - LOG_LEVEL=${{ env.LOG_LEVEL }}
            - "${{ vars.app_name }}:${{ vars.version }}"
        timeout-minutes: 5
```

**执行流程:**
1. 拉取 Docker 镜像
2. 停止旧容器（失败容忍）
3. 启动新容器（5 分钟超时）

---

### 示例 2: 复杂工作流

多 Job 依赖，条件执行，输出引用。

```yaml
name: build-test-deploy
vars:
  environment: production
  notify_slack: true

jobs:
  build:
    runs-on: build-servers
    outputs:
      image_tag: ${{ steps.build.outputs.tag }}
    
    steps:
      - id: build
        name: Build Docker Image
        uses: shell@v1
        with:
          command: docker
          args: ["build", "-t", "myapp:latest", "."]
      
      - name: Push to Registry
        uses: shell@v1
        with:
          command: docker
          args: ["push", "myapp:latest"]
  
  test:
    runs-on: test-servers
    needs: [build]
    if: ${{ vars.environment == 'production' }}
    
    steps:
      - name: Run Integration Tests
        uses: shell@v1
        with:
          command: ./run-tests.sh
        timeout-minutes: 10
  
  deploy:
    runs-on: web-servers
    needs: [build, test]
    if: ${{ success() }}
    
    steps:
      - name: Deploy Container
        uses: shell@v1
        with:
          command: docker
          args:
            - run
            - -d
            - "${{ needs.build.outputs.image_tag }}"
      
      - name: Health Check
        uses: http@v1
        with:
          url: http://localhost:8080/health
        retry-strategy:
          max-attempts: 5
          initial-interval: 2s
  
  notify:
    runs-on: default
    needs: [deploy]
    if: ${{ vars.notify_slack && always() }}
    
    steps:
      - name: Send Slack Notification
        uses: http@v1
        with:
          url: ${{ secrets.slack_webhook }}
          method: POST
          body:
            status: ${{ job.status }}
            message: "Deployment ${{ job.status }}"
```

**执行流程:**
1. **build**: 构建镜像 → 推送到仓库
2. **test**: 等待 build → 运行集成测试（仅 production）
3. **deploy**: 等待 build + test → 部署容器 → 健康检查
4. **notify**: 等待 deploy → 发送 Slack 通知（总是执行）

**DAG 执行图:**
```
build → test → deploy → notify
```

---

### 示例 3: Matrix 并行工作流

多服务器并行部署，fail-fast 策略。

```yaml
name: parallel-server-deployment
vars:
  app_version: v2.0.0

jobs:
  deploy:
    runs-on: default
    strategy:
      matrix:
        server:
          - web-1.example.com
          - web-2.example.com
          - web-3.example.com
        environment: [staging, production]
      max-parallel: 3
      fail-fast: true  # 任一失败立即停止
    
    steps:
      - name: Deploy to ${{ matrix.server }} (${{ matrix.environment }})
        uses: shell@v1
        with:
          command: ssh
          args:
            - "${{ matrix.server }}"
            - "deploy.sh --version=${{ vars.app_version }} --env=${{ matrix.environment }}"
        timeout-minutes: 10
        retry-strategy:
          max-attempts: 3
          initial-interval: 5s
      
      - name: Verify Deployment
        uses: http@v1
        with:
          url: "https://${{ matrix.server }}/health"
        retry-strategy:
          max-attempts: 5
          initial-interval: 2s
      
      - name: Update Load Balancer
        if: ${{ matrix.environment == 'production' }}
        uses: http@v1
        with:
          url: ${{ secrets.lb_api_url }}
          method: POST
          body:
            action: enable
            server: ${{ matrix.server }}
```

**执行流程:**
- 6 个并行实例（3 servers × 2 environments）
- 最多 3 个并发执行
- 任一失败立即取消其他
- 每个实例：部署 → 验证 → 更新负载均衡器（仅 production）

---

## 最佳实践

### 命名规范

**Workflow 名称:**
- 小写字母、数字、连字符
- 语义化：`deploy-app`, `health-check`, `backup-database`
- 长度：1-50 字符

**Job ID:**
- 字母、数字、连字符、下划线
- 语义化：`build`, `test`, `deploy_production`

**Step ID:**
- 字母、数字、下划线
- 必须以字母开头
- 语义化：`build_image`, `run_tests`

**变量名称:**
- 语义化：`environment`, `app_version`
- 避免保留字：`name`, `on`, `jobs`, `steps`

### 变量和环境变量

**变量作用域:**
- `vars.*` - 全局变量，所有 Job/Step 可见
- `env.*` - 环境变量，三级继承
- `matrix.*` - Matrix 变量，仅 Matrix Job 可用

**环境变量优先级:**
```
Step env > Job env > Workflow env
```

**密钥引用:**
- 使用 `secrets.*` 而非硬编码
- 密钥自动脱敏（日志中显示 `***`）
- 配置 SecretProvider 接口

```yaml
# ✅ 推荐
password: ${{ secrets.db_password }}

# ❌ 不推荐
password: "hardcoded_password"
```

### Matrix 策略

**使用场景:**
- 多服务器并行部署
- 跨环境测试（dev/staging/production）
- 跨平台构建（Linux/macOS/Windows）

**组合数限制:**
- 最大 256 个组合
- 建议 < 50 个（避免资源耗尽）

**max-parallel 设置:**
- 根据 Agent 数量设置
- 避免过高（资源竞争）
- 示例：10 个 Agent → max-parallel: 8

**fail-fast 策略:**
- 部署场景：`fail-fast: true`（任一失败立即停止）
- 测试场景：`fail-fast: false`（收集所有结果）

### 超时和重试

**超时配置建议:**
- HTTP 请求：1-5 分钟
- Shell 命令：5-30 分钟
- Docker 构建：10-60 分钟
- 数据库备份：30-120 分钟

**重试策略建议:**
- HTTP 请求：max-attempts: 3-5
- 网络操作：max-attempts: 5-10
- 幂等操作：可多次重试
- 非幂等操作：max-attempts: 1-2

**避免过度重试:**
- max-attempts 建议 ≤ 10
- 总重试时间建议 < 10 分钟
- 配置 max-interval 避免长时间等待

### 条件执行

**常见 if 模式:**

```yaml
# 环境判断
if: ${{ vars.environment == 'production' }}

# 状态判断
if: ${{ success() }}        # 前面成功
if: ${{ failure() }}        # 前面失败
if: ${{ always() }}         # 总是执行

# 组合条件
if: ${{ success() && vars.deploy_enabled }}

# 依赖状态
if: ${{ needs.build.result == 'success' }}
```

**避免复杂条件:**

```yaml
# ❌ 复杂难读
if: ${{ (vars.env == 'prod' && vars.region == 'us') || (vars.env == 'staging' && success()) }}

# ✅ 拆分为多个 Step
- name: Deploy to US Production
  if: ${{ vars.env == 'prod' && vars.region == 'us' }}

- name: Deploy to Staging
  if: ${{ vars.env == 'staging' && success() }}
```

---

## 常见错误

### 循环依赖

**错误示例:**

```yaml
jobs:
  job-a:
    needs: [job-b]
  
  job-b:
    needs: [job-a]  # ❌ 循环依赖
```

**错误信息:**
```
validation_failed: Circular dependency detected: job-a → job-b → job-a
```

**解决方案:** 检查 `needs` 字段，消除循环。

### 未定义变量引用

**错误示例:**

```yaml
steps:
  - uses: shell@v1
    with:
      command: echo
      args: ["${{ vars.undefined_var }}"]  # ❌ 未定义
```

**错误信息:**
```
validation_failed: Undefined variable: vars.undefined_var
```

**解决方案:** 在 `vars` 中定义变量或检查拼写。

### Matrix 组合数超限

**错误示例:**

```yaml
strategy:
  matrix:
    server: [1, 2, ..., 100]  # 100 个
    environment: [dev, staging, prod]  # 3 个
    # 100 × 3 = 300 > 256 ❌
```

**错误信息:**
```
validation_failed: Matrix combinations 300 exceed limit 256
```

**解决方案:**
- 减少 Matrix 维度数量
- 拆分为多个 Job
- 使用条件执行过滤

### 超时配置不合理

**错误示例:**

```yaml
# ❌ Job 超时小于 Step 超时
jobs:
  deploy:
    timeout-minutes: 5
    steps:
      - timeout-minutes: 10  # Step 超时 > Job 超时
```

**解决方案:** 确保 Job 超时 ≥ 所有 Steps 超时之和。

### 表达式语法错误

**错误示例:**

```yaml
# ❌ 引号不匹配
name: Deploy to ${{ vars.env }

# ❌ 括号不匹配
if: ${{ vars.enabled && success(}

# ❌ 字段不存在
if: ${{ vars.config.nonexistent.field }}
```

**错误信息:**
```
validation_failed: Expression syntax error: unexpected token
```

**解决方案:** 检查表达式语法，使用 `waterflow validate` 验证。

---

## 语法速查表

### 顶层字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `name` | `string` | 是 | 工作流名称 |
| `on` | `string\|object` | 否 | 触发器 (默认 workflow_dispatch) |
| `vars` | `map` | 否 | 全局变量 |
| `env` | `map[string]string` | 否 | 全局环境变量 |
| `jobs` | `map[string]Job` | 是 | Job 映射表 |

### Job 字段

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `runs-on` | `string` | 是 | - | Task Queue 名称 |
| `needs` | `[]string` | 否 | `[]` | Job 依赖 |
| `if` | `string` | 否 | - | 条件执行 |
| `timeout-minutes` | `int` | 否 | `360` | Job 超时 |
| `strategy` | `Strategy` | 否 | - | Matrix 策略 |
| `env` | `map[string]string` | 否 | `{}` | Job 环境变量 |
| `outputs` | `map[string]string` | 否 | `{}` | Job 输出 |
| `continue-on-error` | `bool` | 否 | `false` | 失败容忍 |
| `steps` | `[]Step` | 是 | - | Step 列表 |

### Step 字段

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `id` | `string` | 否 | - | Step 标识符 |
| `name` | `string` | 否 | - | Step 名称 |
| `uses` | `string` | 是 | - | 节点调用 |
| `with` | `map` | 否 | `{}` | 节点参数 |
| `if` | `string` | 否 | - | 条件执行 |
| `timeout-minutes` | `int` | 否 | - | Step 超时 |
| `retry-strategy` | `RetryStrategy` | 否 | - | 重试策略 |
| `continue-on-error` | `bool` | 否 | `false` | 失败容忍 |
| `env` | `map[string]string` | 否 | `{}` | Step 环境变量 |

### 表达式函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `len(x)` | 长度 | `len([1,2,3])` → 3 |
| `upper(s)` | 大写 | `upper("hello")` → "HELLO" |
| `lower(s)` | 小写 | `lower("HELLO")` → "hello" |
| `trim(s)` | 去空格 | `trim("  hi  ")` → "hi" |
| `split(s, sep)` | 分割 | `split("a,b", ",")` → ["a","b"] |
| `join(arr, sep)` | 连接 | `join(["a","b"], ",")` → "a,b" |
| `format(fmt, ...)` | 格式化 | `format("Hello %s", "World")` |
| `contains(s, sub)` | 包含 | `contains("hello", "ll")` → true |
| `startsWith(s, prefix)` | 起始 | `startsWith("hello", "he")` → true |
| `endsWith(s, suffix)` | 结束 | `endsWith("hello", "lo")` → true |
| `toJSON(obj)` | 转 JSON | `toJSON({k:"v"})` → '{"k":"v"}' |
| `fromJSON(str)` | 解析 JSON | `fromJSON('{"k":"v"}')` → {k:"v"} |
| `success()` | 成功判断 | 前面 Steps 都成功 |
| `failure()` | 失败判断 | 任一 Step 失败 |
| `cancelled()` | 取消判断 | 工作流被取消 |
| `always()` | 总是 true | 总是执行 |

### 常用代码片段

**变量引用:**
```yaml
${{ vars.name }}
${{ vars.config.timeout }}
${{ vars.servers[0] }}
```

**环境变量:**
```yaml
${{ env.LOG_LEVEL }}
```

**Matrix 变量:**
```yaml
${{ matrix.os }}
${{ matrix.version }}
```

**Step 输出:**
```yaml
${{ steps.build.outputs.tag }}
```

**Job 输出:**
```yaml
${{ needs.build.outputs.version }}
```

**条件执行:**
```yaml
if: ${{ success() }}
if: ${{ failure() }}
if: ${{ vars.env == 'production' }}
```

---

## 参考资源

### 架构文档

- [ADR-0002: 单节点执行模式](adr/0002-single-node-execution-pattern.md) - 每个 Step 独立超时/重试
- [ADR-0006: Task Queue 直接映射](adr/0006-task-queue-routing.md) - runs-on 路由机制

### 实现文档

- [Story 1.3: YAML DSL 解析和验证](sprint-artifacts/1-3-yaml-dsl-parsing-and-validation.md)
- [Story 1.4: 表达式引擎和变量系统](sprint-artifacts/1-4-expression-engine-and-variables.md)
- [Story 1.5: 条件执行和控制流](sprint-artifacts/1-5-conditional-execution-and-control-flow.md)
- [Story 1.6: Matrix 并行执行策略](sprint-artifacts/1-6-matrix-parallel-execution.md)
- [Story 1.7: 超时和重试策略](sprint-artifacts/1-7-timeout-and-retry-strategies.md)

### 代码参考

- [pkg/dsl/types.go](../pkg/dsl/types.go) - 数据结构定义
- [pkg/dsl/expr_context.go](../pkg/dsl/expr_context.go) - 表达式上下文
- [pkg/dsl/expander.go](../pkg/dsl/expander.go) - Matrix 展开器

### 示例文件

- [examples/hello-world.yaml](../examples/hello-world.yaml) - 简单工作流
- [examples/matrix.yaml](../examples/matrix.yaml) - Matrix 示例
- [examples/workflows/multi-server-health-check.yaml](../examples/workflows/multi-server-health-check.yaml) - 复杂工作流

### 相关文档

- [快速开始指南](quick-start.md)
- [API 文档](http://localhost:8080/docs)
- [节点参考文档](nodes/README.md)
- [工作流模板库](templates/README.md)

---

**文档版本:** 1.0.0  
**更新日期:** 2026-01-15  
**维护:** Websoft9 Team

# ADR-0004: YAML DSL 语法设计

**状态:** ✅ 已采纳  
**日期:** 2025-12-13  
**决策者:** 架构团队  

## 背景

Waterflow 需要一个用户友好的 DSL 来定义工作流。目标用户群体:
- 开发者(熟悉 CI/CD 工具)
- DevOps 工程师
- 自动化测试人员

需要决定 DSL 的格式和语法风格:
1. **YAML** - 类似 GitHub Actions / GitLab CI
2. **JSON** - 机器友好,易于生成
3. **HCL** - Terraform 风格
4. **自定义语法** - 完全控制

## 决策

采用 **YAML 格式** + **GitHub Actions 风格语法**。

## 理由

### 核心优势:

1. **用户熟悉度**
   - GitHub Actions 已被广泛使用
   - 降低学习成本,减少文档需求
   - 用户可以复用 GitHub Actions 经验

2. **人类可读性**
   - YAML 简洁清晰
   - 支持注释
   - 层次结构直观

3. **生态兼容性**
   - 可以参考 GitHub Actions 的节点命名
   - 用户习惯迁移更平滑
   - 第三方工具(编辑器插件)可复用

4. **社区实践**
   - GitHub Actions, GitLab CI, Azure Pipelines 都使用 YAML
   - 大量最佳实践和模式可借鉴

### 与其他方案对比:

| 方案 | 优点 | 缺点 | 决策 |
|------|------|------|------|
| **YAML + GHA 风格** | 熟悉,可读,生态好 | YAML 陷阱(缩进,类型) | ✅ 选择 |
| JSON | 机器友好,严格 | 人类不友好,无注释 | ❌ |
| HCL | 表达力强,类型安全 | 学习成本高,生态小 | ❌ |
| 自定义语法 | 完全控制 | 开发成本高,生态为零 | ❌ |

## 后果

### 正面影响:

✅ **低学习成本** - 用户无需学习新语法  
✅ **快速上手** - 复制 GitHub Actions 配置稍作修改即可  
✅ **工具支持** - IDE 插件,Linter 可复用  
✅ **社区资源** - 大量示例和最佳实践  

### 负面影响:

⚠️ **YAML 陷阱**
   - 缩进敏感,容易出错
   - 类型推断问题(`on`/`yes`/`no` → bool, `1.0` → string)
   - 锚点和合并键复杂

⚠️ **表达能力限制**
   - 不如编程语言灵活
   - 复杂逻辑需要通过表达式或脚本

### 风险缓解:

- 提供详细的语法检查和错误提示
- Schema 验证(IDE 自动补全)
- 常见错误的友好提示

## 语法规范

### 基本结构:

```yaml
name: Build and Test

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: linux-amd64
    timeout-minutes: 30
    
    steps:
      - name: Checkout Code
        uses: checkout@v1
        with:
          repository: ${{ workflow.repository }}
      
      - name: Run Tests
        uses: run@v1
        with:
          command: go test ./...
        timeout-minutes: 10
        retry:
          attempts: 3
          backoff: 2s
```

### 核心概念:

1. **Workflow** - 顶层对象
   - `name`: 工作流名称
   - `on`: 触发条件
   - `jobs`: 任务列表

2. **Job** - 一组 Steps
   - `runs-on`: 服务器组(映射到 Task Queue)
   - `timeout-minutes`: Job 级别超时
   - `needs`: 依赖的其他 Jobs
   - `steps`: Step 列表

3. **Step** - 单个执行单元
   - `name`: 步骤名称
   - `uses`: 节点类型(如 `checkout@v1`)
   - `with`: 节点参数
   - `timeout-minutes`: Step 级别超时
   - `retry`: 重试配置
   - `if`: 条件执行
   - `continue-on-error`: 失败继续

### 表达式语法:

```yaml
steps:
  - name: Use Variable
    uses: run@v1
    with:
      command: echo ${{ steps.checkout.outputs.commit }}
  
  - name: Conditional Step
    if: ${{ job.status == 'success' }}
    uses: notify@v1
```

详见 [ADR-0005: 表达式系统语法](0005-expression-system-syntax.md)

## 与 GitHub Actions 的差异

### 相似之处:

✅ 基本结构完全一致(`jobs`/`steps`/`uses`)  
✅ 表达式语法相同(`${{ }}`)  
✅ 节点命名风格一致(`action-name@version`)  

### 差异之处:

| 特性 | GitHub Actions | Waterflow | 原因 |
|------|----------------|-----------|------|
| **触发器** | 20+ 种事件 | 简化版(push/schedule/webhook) | MVP 范围 |
| **Matrix** | 支持 | 暂不支持 | 后续版本 |
| **Secrets** | 内置 KV | 集成外部(Vault) | 安全性 |
| **runs-on** | GitHub 托管 Runner | Task Queue 名称 | 架构差异 |
| **Container** | 支持 | 暂不支持 | 后续版本 |

## Schema 定义

提供 JSON Schema 用于 IDE 自动补全:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["name", "on", "jobs"],
  "properties": {
    "name": {
      "type": "string",
      "description": "工作流名称"
    },
    "on": {
      "description": "触发条件",
      "oneOf": [
        {"type": "string"},
        {"type": "object"}
      ]
    },
    "jobs": {
      "type": "object",
      "patternProperties": {
        "^[a-z][a-z0-9-]*$": {
          "$ref": "#/definitions/job"
        }
      }
    }
  },
  "definitions": {
    "job": {
      "type": "object",
      "required": ["runs-on", "steps"],
      "properties": {
        "runs-on": {"type": "string"},
        "timeout-minutes": {"type": "integer"},
        "steps": {
          "type": "array",
          "items": {"$ref": "#/definitions/step"}
        }
      }
    },
    "step": {
      "type": "object",
      "required": ["uses"],
      "properties": {
        "name": {"type": "string"},
        "uses": {"type": "string"},
        "with": {"type": "object"},
        "timeout-minutes": {"type": "integer"}
      }
    }
  }
}
```

## 实现示例

### YAML 解析:

```go
// pkg/dsl/parser.go
import "gopkg.in/yaml.v3"

type Workflow struct {
    Name string                 `yaml:"name"`
    On   map[string]interface{} `yaml:"on"`
    Jobs map[string]Job         `yaml:"jobs"`
}

type Job struct {
    RunsOn         string `yaml:"runs-on"`
    TimeoutMinutes int    `yaml:"timeout-minutes"`
    Steps          []Step `yaml:"steps"`
}

type Step struct {
    Name           string                 `yaml:"name"`
    Uses           string                 `yaml:"uses"`
    With           map[string]interface{} `yaml:"with"`
    TimeoutMinutes int                    `yaml:"timeout-minutes,omitempty"`
    Retry          *RetryConfig           `yaml:"retry,omitempty"`
}

func ParseWorkflow(data []byte) (*Workflow, error) {
    var wf Workflow
    if err := yaml.Unmarshal(data, &wf); err != nil {
        return nil, err
    }
    return &wf, nil
}
```

## 替代方案

### 方案 A: JSON 格式 (被拒绝)

```json
{
  "name": "Build",
  "on": {"push": {"branches": ["main"]}},
  "jobs": {
    "build": {
      "runs-on": "linux-amd64",
      "steps": [
        {
          "name": "Checkout",
          "uses": "checkout@v1"
        }
      ]
    }
  }
}
```

**被拒绝原因:**
- ❌ 冗长,可读性差
- ❌ 不支持注释
- ❌ 用户不熟悉(CI/CD 工具都用 YAML)

### 方案 B: HCL (Terraform 风格) (被拒绝)

```hcl
workflow "build" {
  on = {
    push = {
      branches = ["main"]
    }
  }
}

job "build" {
  runs_on = "linux-amd64"
  
  step "checkout" {
    uses = "checkout@v1"
  }
}
```

**被拒绝原因:**
- ❌ 学习成本高(用户不熟悉 HCL)
- ❌ 解析库成熟度不如 YAML
- ❌ 社区资源少

### 方案 C: 编程语言 DSL (被拒绝)

```go
// Go DSL
workflow.New("build").
    On(trigger.Push("main")).
    Job("build", func(j *Job) {
        j.Step("checkout").Uses("checkout@v1")
        j.Step("test").Run("go test ./...")
    })
```

**被拒绝原因:**
- ❌ 需要编译,不适合配置文件
- ❌ 学习成本高
- ❌ 版本控制困难

## 参考资料

- [GitHub Actions 语法规范](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions)
- [GitLab CI YAML 规范](https://docs.gitlab.com/ee/ci/yaml/)
- [PRD: DSL 设计](../prd.md#dsl-设计)

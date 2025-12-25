# Waterflow 示例工作流

本目录包含 Waterflow YAML DSL 示例工作流，帮助您快速了解和验证 Waterflow 功能。

## 📁 示例列表

### 1. hello-world.yaml - 基础示例
最简单的 Waterflow 工作流，演示：
- 基本 YAML 语法
- 变量使用 (`vars`)
- 表达式插值 (`${{ }}`)
- 工作流手动触发 (`workflow_dispatch`)

**运行示例：**
```bash
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d @- <<EOF
{
  "yaml": "$(cat examples/hello-world.yaml | sed 's/"/\\"/g' | tr '\n' ' ')"
}
EOF
```

### 2. multi-step.yaml - 多步骤工作流
演示多个步骤和任务依赖：
- 多个 jobs
- 任务依赖 (`needs`)
- 多步骤执行
- 顺序编排

### 3. matrix.yaml - 矩阵并行执行
演示矩阵策略并行执行：
- Matrix 策略 (`strategy.matrix`)
- 并行任务执行
- 矩阵变量引用

## 🚀 快速测试

### 方法 1: 使用 curl 提交工作流

```bash
# 提交 hello-world 示例
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d "{\"yaml\": \"$(cat examples/hello-world.yaml | sed 's/"/\\"/g' | tr '\n' ' ')\"}"
```

### 方法 2: 使用测试脚本

```bash
# 运行完整部署测试（包含工作流提交）
./scripts/test-deployment.sh
```

## 📊 查看执行结果

### 1. 通过 API 查询状态

```bash
# 列出所有工作流
curl http://localhost:8080/v1/workflows

# 查询特定工作流
curl http://localhost:8080/v1/workflows/{workflow-id}
```

### 2. 通过 Temporal UI

访问 http://localhost:8088 查看工作流执行详情、历史记录和可视化流程。

## 📖 扩展阅读

- [Waterflow YAML DSL 语法](../docs/adr/0004-yaml-dsl-syntax.md)
- [表达式系统](../docs/adr/0005-expression-system-syntax.md)
- [部署指南](../docs/deployment.md)
- [开发文档](../docs/development.md)

## 💡 自定义工作流

您可以基于这些示例创建自己的工作流。基本结构：

```yaml
name: My Workflow
on:
  workflow_dispatch:  # 手动触发

vars:
  my_var: "value"

jobs:
  my_job:
    runs-on: waterflow-server
    steps:
      - name: My Step
        run: echo "Hello ${{ vars.my_var }}"
```

更多语法和功能请参考文档。

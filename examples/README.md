# Waterflow 示例

本目录包含 Waterflow 的示例工作流和配置文件。

## 📁 目录结构

```
examples/
├── configs/                    # 配置文件模板
│   ├── config.example.yaml          # Server 配置模板
│   ├── config.agent.example.yaml    # Agent 配置模板
│   └── server-groups.example.yaml   # Server Groups 配置模板
├── workflows/                  # 工作流模板
│   ├── single-server-deployment.yaml  # 单服务器部署模板 (生产级)
│   ├── shell-examples.yaml            # Shell 节点示例
│   ├── script-examples.yaml           # Script 节点示例
│   ├── docker-exec-examples.yaml      # Docker 节点示例
│   └── ...                            # 其他节点示例
├── hello-world.yaml            # 基础工作流示例
├── multi-step.yaml
├── matrix.yaml
├── multi-server.yaml
├── providers/                  # Provider 插件示例
└── README.md
```

## 🔧 配置文件模板

### Server 配置

```bash
# 复制模板创建配置文件
cp examples/configs/config.example.yaml config.yaml

# 编辑配置
vim config.yaml
```

详细配置说明参见：[配置指南](../docs/configuration.md)

### Agent 配置

```bash
# 复制模板
cp examples/configs/config.agent.example.yaml config.agent.yaml

# 编辑配置
vim config.agent.yaml
```

详细说明参见：[docs/guides/agent-README.md](../docs/guides/agent-README.md)

## 📋 工作流示例

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

## 🎯 生产级模板

### workflows/ 目录 - 生产就绪的工作流模板

`workflows/` 文件夹包含可直接用于生产的、参数化的工作流模板。这些模板展示了 Waterflow 的最佳实践，并包含完整的错误处理和回滚逻辑。

#### 1. single-server-deployment.yaml - 单服务器应用部署

**适用场景：**
- 小型 Web 应用
- 内部工具
- MVP 产品
- 开发/测试环境

**功能特性：**
- ✅ 基于 Git 的代码拉取
- ✅ Docker 镜像构建
- ✅ 优雅停止旧版本
- ✅ 启动新版本
- ✅ HTTP 健康检查（带重试）
- ✅ 失败自动回滚

**快速开始：**
```bash
# 1. 复制模板
cp examples/workflows/single-server-deployment.yaml my-app-deployment.yaml

# 2. 编辑配置（修改 vars 部分）
vim my-app-deployment.yaml
# 必需修改:
#   repo_url: "https://github.com/your-org/your-app.git"
#   app_name: "your-app"

# 3. 提交工作流
waterflow submit my-app-deployment.yaml

# 或使用 API:
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d @my-app-deployment.yaml

# 4. 查看部署状态
waterflow status <workflow-id>
```

**参数说明：**

| 参数 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `repo_url` | ✅ | - | Git 仓库地址 |
| `app_name` | ✅ | - | 应用名称（用于容器命名） |
| `app_port` | ⚙️ | 3000 | 应用监听端口 |
| `branch` | ⚙️ | main | Git 分支 |
| `deploy_path` | ⚙️ | /opt/{app_name} | 部署目录 |
| `health_check_url` | ⚙️ | http://localhost:{app_port}/health | 健康检查 URL |
| `health_check_retries` | ⚙️ | 5 | 健康检查重试次数 |
| `health_check_delay` | ⚙️ | 10 | 健康检查延迟（秒） |

**使用示例：**

```yaml
# 示例 1: 部署 Node.js Express 应用
vars:
  repo_url: "https://github.com/expressjs/express.git"
  app_name: "express-app"
  app_port: "3000"
  health_check_url: "http://localhost:3000/health"

# 示例 2: 部署 Python Flask 应用
vars:
  repo_url: "https://github.com/pallets/flask.git"
  app_name: "flask-app"
  app_port: "5000"
  health_check_url: "http://localhost:5000/health"

# 示例 3: 部署到特定分支
vars:
  repo_url: "https://github.com/user/app.git"
  app_name: "my-app"
  branch: "develop"
  deploy_path: "/opt/my-app-dev"
```

**与 CI/CD 集成（GitHub Actions）：**

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Submit deployment to Waterflow
        run: |
          WORKFLOW_ID=$(curl -X POST http://waterflow-server:8080/v1/workflows \
            -H "Content-Type: application/json" \
            -d '{
              "workflow_file": "single-server-deployment",
              "vars": {
                "repo_url": "${{ github.repositoryUrl }}",
                "app_name": "my-app",
                "branch": "${{ github.ref_name }}"
              }
            }' | jq -r '.id')
          echo "Deployment workflow ID: $WORKFLOW_ID"
      
      - name: Wait for deployment
        run: waterflow status $WORKFLOW_ID --wait
```

**前置条件：**
- Waterflow Agent 已安装并运行
- Agent 服务器已安装 Git 和 Docker
- 应用仓库可访问（公开仓库或已配置凭证）
- 目标端口未被占用

**详细文档：** 查看模板文件顶部注释获取完整参数说明和故障排查指南。

#### 其他模板（即将推出）

- **multi-server-health-check.yaml** - 多服务器健康检查模板
- **distributed-stack-deployment.yaml** - 分布式应用栈部署模板



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

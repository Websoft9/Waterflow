# Waterflow 工作流模板库

## 概述

Waterflow 提供生产就绪的工作流模板,帮助您快速实现常见的部署和运维场景。这些模板经过精心设计和测试,可以直接使用或作为自定义工作流的起点。

**模板优势:**
- ✅ **开箱即用** - 无需从零编写,复制即可使用
- ✅ **生产级质量** - 包含错误处理、重试、回滚等最佳实践
- ✅ **参数化配置** - 通过变量轻松适配不同环境
- ✅ **完整文档** - 每个模板都有详细的使用说明和示例

## 可用模板

### 1. 单服务器部署 (Single Server Deployment)

**用途:** 将应用部署到单台服务器,包含构建、健康检查和回滚功能

**适用场景:**
- 简单的 Web 应用
- MVP 产品快速部署
- 开发/测试环境
- 单体架构应用

**关键特性:**
- Git 仓库克隆/更新
- Docker 镜像构建
- 容器启动和健康检查
- 失败自动回滚

**📖 完整文档:** [single-server-deployment.md](./single-server-deployment.md)

**快速开始:**
```bash
# 1. 修改变量
cp examples/workflows/single-server-deployment.yaml my-deployment.yaml
# 编辑 vars.repo_url 和 vars.app_name

# 2. 提交工作流
waterflow submit my-deployment.yaml
```

---

### 2. 多服务器健康检查 (Multi-Server Health Check)

**用途:** 并行检查多台服务器的健康状态,收集 CPU、内存、磁盘指标

**适用场景:**
- 定期服务器巡检
- 部署前环境验证
- 问题服务器快速定位
- 容量规划和监控

**关键特性:**
- SSH 远程执行,无需 Agent
- Matrix 策略并行检查
- 跨平台兼容 (支持多种 Linux 发行版)
- Python 生成 Markdown 报告

**📖 完整文档:** [multi-server-health-check.md](./multi-server-health-check.md)

**快速开始:**
```bash
# 1. 配置 SSH 免密登录
ssh-copy-id user@server1
ssh-copy-id user@server2

# 2. 修改服务器列表
cp examples/workflows/multi-server-health-check.yaml my-health-check.yaml
# 编辑 vars.servers

# 3. 提交工作流
waterflow submit my-health-check.yaml
```

---

### 3. 分布式栈部署 (Distributed Stack Deployment)

**用途:** 部署多层应用栈,支持数据库和应用的依赖管理

**适用场景:**
- Web 应用 + 数据库
- 微服务架构
- 生产环境部署
- 多服务器编排

**关键特性:**
- Job 依赖管理 (数据库先启动)
- 跨服务器部署编排
- 数据库初始化和健康检查
- 应用与数据库连接验证
- 失败回滚选项

**📖 完整文档:** [distributed-stack-deployment.md](./distributed-stack-deployment.md)

**快速开始:**
```bash
# 1. 修改配置
cp examples/workflows/distributed-stack-deployment.yaml my-stack.yaml
# 编辑数据库和应用配置

# 2. 提交工作流
waterflow submit my-stack.yaml
```

---

## 模板选择指南

根据您的部署场景选择合适的模板:

| 场景 | 推荐模板 | 原因 |
|------|----------|------|
| 部署单个应用到一台服务器 | 单服务器部署 | 简单直接,包含完整部署流程 |
| 检查多台服务器状态 | 多服务器健康检查 | 并行执行,快速获取所有服务器指标 |
| 部署应用和数据库 | 分布式栈部署 | 管理服务依赖,确保启动顺序 |
| 部署微服务到多台服务器 | 分布式栈部署 + 定制 | 基于模板扩展,添加更多服务 |
| 定期监控服务器 | 多服务器健康检查 + Cron | 设置定时任务定期执行 |

## 通用使用流程

所有模板遵循相同的使用流程:

### 步骤 1: 选择模板

从 `examples/workflows/` 目录选择合适的模板:
```bash
ls examples/workflows/*.yaml
```

### 步骤 2: 复制和定制

复制模板并修改变量:
```bash
cp examples/workflows/<template-name>.yaml my-workflow.yaml
```

编辑 `vars` 部分,配置您的参数:
```yaml
vars:
  repo_url: "https://github.com/your-org/your-app.git"
  app_name: "your-app"
  # ... 其他参数
```

### 步骤 3: 验证配置

使用 CLI 验证工作流语法:
```bash
waterflow validate my-workflow.yaml
```

### 步骤 4: 提交工作流

提交到 Waterflow Server:
```bash
# 使用 CLI
waterflow submit my-workflow.yaml

# 或使用 API
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d @my-workflow.yaml
```

### 步骤 5: 监控执行

查看工作流状态:
```bash
# 查看状态
waterflow status <workflow-id>

# 查看日志
waterflow logs <workflow-id>

# 实时跟踪 (follow)
waterflow logs -f <workflow-id>
```

### 步骤 6: 验证结果

根据工作流类型验证结果:
```bash
# 对于部署类模板,验证应用运行
curl http://your-server:port/health

# 对于健康检查模板,查看报告
cat /tmp/health_report_*.md
```

## 获取模板列表 (API)

通过 API 获取所有可用模板:

```bash
# 列出所有模板
curl http://localhost:8080/v1/templates

# 获取特定模板详情
curl http://localhost:8080/v1/templates/single-server-deployment

# 仅获取元数据 (不含 YAML 内容)
curl http://localhost:8080/v1/templates/single-server-deployment?content=false

# 按类别过滤
curl http://localhost:8080/v1/templates?category=deployment
```

## 定制模板

所有模板都可以定制以适配您的具体需求:

### 基本定制: 修改变量

最简单的方式是修改 `vars` 部分的默认值:

```yaml
vars:
  app_port: 8080        # 修改默认端口
  branch: "develop"     # 使用不同分支
```

### 高级定制: 添加步骤

在模板中添加自定义步骤:

```yaml
jobs:
  deploy:
    steps:
      # ... 现有步骤 ...
      
      # 添加新步骤
      - name: Run database migrations
        uses: exec/shell@v1
        with:
          command: |
            cd ${{ vars.deploy_path }}
            npm run migrate
```

### 深度定制: 创建新模板

1. 复制现有模板作为起点
2. 修改工作流逻辑
3. 添加到 `examples/workflows/`
4. 更新 `templates-metadata.json`
5. 编写文档

**参考:** [自定义节点开发指南](../../docs/guides/custom-node-development.md)

## 故障排查

### 常见问题

**1. 工作流验证失败**
```bash
# 问题: YAML 语法错误
# 解决: 检查缩进,确保使用空格而非 Tab
waterflow validate my-workflow.yaml
```

**2. Agent 连接失败**
```bash
# 问题: 找不到指定的 Task Queue
# 解决: 确认 Agent 已启动且 Task Queue 名称正确
docker ps | grep waterflow-agent
```

**3. SSH 连接超时 (多服务器健康检查)**
```bash
# 问题: 无法 SSH 到目标服务器
# 解决: 配置 SSH 免密登录
ssh-copy-id user@server
ssh user@server  # 测试连接
```

**4. Docker 构建失败 (单服务器部署)**
```bash
# 问题: Dockerfile 不存在或构建错误
# 解决: 确保仓库根目录有 Dockerfile
cd /path/to/repo && docker build -t test .
```

**5. 数据库连接失败 (分布式栈部署)**
```bash
# 问题: 应用无法连接数据库
# 解决: 检查网络配置和数据库凭证
docker exec -it postgres pg_isready -U appuser
```

### 获取帮助

- 📖 **完整文档:** [Waterflow Documentation](../../README.md)
- 🐛 **问题反馈:** [GitHub Issues](https://github.com/Websoft9/waterflow/issues)
- 💬 **社区讨论:** [GitHub Discussions](https://github.com/Websoft9/waterflow/discussions)
- 📘 **API 文档:** [API Reference](../../api/README.md)

## 贡献模板

欢迎贡献新的工作流模板!

**贡献步骤:**
1. Fork 仓库
2. 在 `examples/workflows/` 创建新模板 YAML
3. 更新 `examples/workflows/templates-metadata.json`
4. 在 `docs/templates/` 创建模板文档
5. 添加测试用例
6. 提交 Pull Request

**模板质量要求:**
- ✅ 包含完整的参数说明
- ✅ 提供至少 2 个使用示例
- ✅ 包含错误处理和重试逻辑
- ✅ 通过测试验证
- ✅ 编写详细的文档

参考现有模板的结构和风格。

## 相关资源

- [快速开始指南](../../docs/quick-start.md)
- [YAML DSL 语法参考](../../docs/yaml-dsl-syntax-reference.md)
- [节点参考文档](../../docs/nodes/README.md)
- [表达式系统文档](../../docs/expression-system.md)
- [API 参考](../../api/README.md)
- [示例工作流](../../examples/README.md)

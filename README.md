# Waterflow

基于 Temporal 的声明式工作流编排引擎。

## 项目简介

Waterflow 是一个现代化的工作流编排系统，通过声明式 YAML DSL 定义工作流，利用 Temporal 实现 100% 状态持久化和分布式执行。

### 核心特性

- ✅ **Event Sourcing 状态管理** - 基于 Temporal Event History 实现工作流状态 100% 持久化
- ✅ **声明式 YAML DSL** - 简洁直观的工作流定义语法
- 🚧 **分布式 Agent 执行** - 跨多台服务器并行执行任务（开发中）
- 🚧 **插件化节点系统** - 丰富的内置节点和自定义扩展能力（开发中）

## 快速开始

### 前置要求

- Go 1.24+
- Docker（可选，用于容器化部署）
- Temporal Server（用于生产环境）

### 安装

```bash
# 克隆仓库
git clone https://github.com/Websoft9/waterflow.git
cd waterflow

# 安装依赖
go mod download

# 构建
make build

# 运行
./bin/server
```

### 使用 Docker

```bash
# 构建镜像
make docker-build

# 运行容器
docker run -p 8080:8080 waterflow:latest
```

### 配置

复制配置示例并根据需要修改：

```bash
cp config.example.yaml config.yaml
```

支持通过环境变量覆盖配置：

```bash
export WATERFLOW_SERVER_PORT=9090
export WATERFLOW_LOG_LEVEL=debug
./bin/server
```

查看完整配置说明：[docs/configuration.md](docs/configuration.md)

## 开发指南

### 克隆和构建

```bash
git clone https://github.com/Websoft9/waterflow.git
cd waterflow
make build
```

### 运行测试

```bash
# 运行所有测试
make test

# 生成覆盖率报告
make coverage
```

### 代码检查

```bash
# 运行 linter
make lint

# 格式化代码
make fmt
```

详细开发指南：[docs/development.md](docs/development.md)

## 架构

Waterflow 采用 Event Sourcing 架构，Server 完全无状态，所有工作流状态存储在 Temporal Event History 中。

```
┌─────────────────┐
│   Waterflow     │
│     Server      │ ← REST API
└────────┬────────┘
         │
         ↓
┌─────────────────┐
│    Temporal     │
│     Server      │ ← Event Sourcing
└─────────────────┘
```

详细架构文档：[docs/architecture.md](docs/architecture.md)

## 贡献

欢迎贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解如何参与项目。

## License

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 相关链接

- [项目文档](docs/)
- [架构设计决策 (ADR)](docs/adr/)
- [产品需求文档 (PRD)](docs/prd.md)

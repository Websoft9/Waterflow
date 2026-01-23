# CI/CD 工作流文档

本目录包含 Waterflow 项目的 GitHub Actions CI/CD 配置。

## 工作流概览

| 工作流 | 文件 | 触发条件 | 用途 |
|--------|------|----------|------|
| CI | `ci.yml` | Push/PR to main/develop | 代码质量检查、测试、构建 |
| Docker Build | `docker.yml` | Push to main/develop, Tags | 构建和推送 Docker 镜像 |
| Release | `release.yml` | Tag v*.*.* | 创建 GitHub Release |
| Stress Test | `stress-test.yml` | Schedule/Manual | 压力测试 |
| Semantic PR | `semantic-pr.yml` | PR | PR 标题规范检查 |
| Labels | `label.yml` | PR | 自动添加标签 |

## CI 工作流 (ci.yml)

### 阶段流程

```
Stage 1 (并行):
├── lint          - golangci-lint 代码检查
└── security      - govulncheck 漏洞扫描

Stage 2 (需要 lint 通过):
├── unit-test     - 单元测试 + 覆盖率
└── build         - 构建 server/agent/cli

Stage 3 (需要 Stage 2 通过):
└── integration-tests  - 集成测试 (Temporal + PostgreSQL)

Stage 4 (需要集成测试通过):
└── acceptance-tests   - 验收测试 (多 Agent 环境)
```

### Jobs 详情

#### lint
- 运行 golangci-lint
- 超时: 5 分钟
- 配置: `.golangci.yml`

#### security
- 运行 govulncheck 检查依赖漏洞
- 与 lint 并行执行

#### unit-test
- 运行 `make test`
- 生成覆盖率报告
- 上传到 Codecov
- 依赖: lint 通过

#### build
- 构建 server、agent、CLI 二进制
- 验证二进制版本信息
- 上传 artifacts
- 依赖: lint 通过

#### integration-tests
- 使用 `docker-compose.test.yaml` 启动环境
- 需要: Temporal, PostgreSQL, Server, Agent
- 超时: 15 分钟
- 依赖: unit-test, build

#### acceptance-tests
- 使用 `docker-compose.acceptance.yaml`
- 多 Agent 测试环境 (web-1, web-2, db-1)
- 运行 PRD 验收场景
- 超时: 25 分钟
- 依赖: integration-tests

## Docker 工作流 (docker.yml)

### 构建矩阵

- **平台**: linux/amd64, linux/arm64
- **镜像**:
  - Server: `waterflow/server`
  - Agent: `waterflow/agent`

### 镜像标签策略

| 触发条件 | 标签格式 |
|----------|----------|
| Branch push | `{branch}`, `{branch}-{sha}` |
| Tag push | `{version}`, `{major}.{minor}`, `latest` |
| PR | `pr-{number}` |

### 安全扫描

- 使用 Trivy 扫描镜像漏洞
- 结果上传到 GitHub Security

### Registry

- Docker Hub: `docker.io/waterflow/*`
- GHCR: `ghcr.io/{owner}/waterflow-*`

## Release 工作流 (release.yml)

### 触发条件

仅在推送 `v*.*.*` 格式的 tag 时触发。

### 构建矩阵

| 平台 | Server | Agent | CLI |
|------|--------|-------|-----|
| Linux amd64 | ✅ | ✅ | ✅ |
| Linux arm64 | ✅ | ✅ | ✅ |
| macOS amd64 | ✅ | ✅ | ✅ |
| macOS arm64 | ✅ | ✅ | ✅ |
| Windows amd64 | ✅ | ✅ | ✅ |

### Release 资产

- 15 个二进制文件 (5 平台 × 3 组件)
- `checksums.txt` (SHA256)
- 自动生成的 Release Notes

## Secrets 配置

### 必需 Secrets

| Secret | 用途 | 设置位置 |
|--------|------|----------|
| `DOCKER_USERNAME` | Docker Hub 登录 | Repository Settings |
| `DOCKER_PASSWORD` | Docker Hub 密码/Token | Repository Settings |

### 可选 Secrets

| Secret | 用途 | 默认行为 |
|--------|------|----------|
| `CODECOV_TOKEN` | Codecov 上传 | 上传失败但不阻塞 CI |
| `SLACK_WEBHOOK` | Slack 通知 | 跳过通知 |

### 自动提供的 Secrets

- `GITHUB_TOKEN`: 自动提供，用于 GHCR 和 Release

## 分支保护规则

### main 分支 (推荐配置)

在 GitHub Repository Settings → Branches → Add branch protection rule:

```
Branch name pattern: main

☑ Require a pull request before merging
  ☑ Require approvals: 1
  ☑ Dismiss stale pull request approvals when new commits are pushed

☑ Require status checks to pass before merging
  ☑ Require branches to be up to date before merging
  Required status checks:
  - lint
  - unit-test
  - build

☑ Require conversation resolution before merging

☐ Require signed commits (可选)

☑ Do not allow bypassing the above settings
```

### develop 分支 (推荐配置)

```
Branch name pattern: develop

☑ Require a pull request before merging

☑ Require status checks to pass before merging
  Required status checks:
  - lint
  - unit-test
```

## 本地运行

### 运行完整 CI 检查

```bash
# 代码检查
make lint

# 单元测试
make test

# 覆盖率
make coverage

# 构建
make build build-agent cli

# 集成测试
make integration-test

# 验收测试
make acceptance-test
```

### 模拟 CI 环境

```bash
# 使用 act 本地运行 GitHub Actions (需要 Docker)
# https://github.com/nektos/act
act -j lint
act -j unit-test
```

## 手动触发

### 触发 Release

```bash
# 创建并推送 tag
git tag v1.2.3
git push origin v1.2.3
```

### 手动运行压力测试

1. 进入 Actions 页面
2. 选择 "Stress Test" 工作流
3. 点击 "Run workflow"

## 故障排查

### CI 失败

1. 检查失败的 job logs
2. 本地运行相同命令复现
3. 检查最近的代码变更

### Docker 构建失败

1. 检查 Dockerfile 语法
2. 验证基础镜像可用性
3. 检查构建参数

### Release 失败

1. 确认 tag 格式正确 (`v*.*.*`)
2. 检查 secrets 配置
3. 验证构建步骤

## 相关文档

- [开发指南](../../docs/development.md)
- [部署文档](../../docs/deployment.md)
- [故障排查](../../docs/troubleshooting.md)

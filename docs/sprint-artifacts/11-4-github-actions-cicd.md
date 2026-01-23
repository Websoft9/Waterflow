# Story 11.4: GitHub Actions CI/CD

**Status:** ready-for-dev

## Story

As a **开发者**,
I want **自动化 CI/CD 流程**,
So that **保证代码质量和自动发布**。

## Acceptance Criteria

### AC1: 自动运行 lint、单元测试、集成测试

**Given** GitHub Actions 配置  
**When** 提交代码或创建 PR  
**Then** 自动运行 golangci-lint 代码检查  
**And** 自动运行单元测试 (make test)  
**And** 自动运行集成测试 (make test-integration)  
**And** 生成测试覆盖率报告  
**And** 测试结果显示在 PR 检查中

### AC2: 构建 Docker 镜像并推送到 Registry

**Given** 代码合并到 main/develop 分支或创建 Tag  
**When** CI 流程执行  
**Then** 构建 Server Docker 镜像 (linux/amd64, linux/arm64)  
**And** 构建 Agent Docker 镜像 (linux/amd64, linux/arm64)  
**And** 推送镜像到 Docker Hub  
**And** 推送镜像到 GitHub Container Registry (GHCR)  
**And** 镜像标签包含版本号和 commit SHA

### AC3: 编译多平台二进制

**Given** 创建 Release Tag (v*.*.*)  
**When** Release 流程执行  
**Then** 编译 Linux/amd64 二进制  
**And** 编译 macOS/amd64 二进制  
**And** 编译 macOS/arm64 二进制 (Apple Silicon)  
**And** 编译 Windows/amd64 二进制  
**And** 生成 checksum 文件

### AC4: 创建 GitHub Release 附带二进制

**Given** Tag 推送 (v*.*.*)  
**When** Release 工作流触发  
**Then** 自动创建 GitHub Release  
**And** 附带所有平台二进制文件  
**And** 附带 checksum.txt 文件  
**And** 自动生成 Release Notes  
**And** Release 状态为 published (非 draft)

### AC5: 代码质量检查失败时阻止合并

**Given** PR 提交到 main/develop  
**When** CI 检查失败  
**Then** PR 显示红色失败状态  
**And** GitHub 阻止 PR 合并  
**And** 失败详情可在 Actions 查看  
**And** 必须修复后才能合并

### AC6: Tag 推送时自动发布版本

**Given** 推送 Tag (v*.*.*)  
**When** Tag 创建  
**Then** 自动触发 Release 工作流  
**And** 自动构建并发布所有资产  
**And** 自动推送 Docker 镜像 (带版本标签)  
**And** 发布完成后通知 (可选)

## Tasks / Subtasks

### Task 1: 审计和增强现有 CI 配置 (AC: #1, #5)

当前: .github/workflows/ci.yml 已存在

- [ ] 1.1 审计现有 ci.yml 配置
- [ ] 1.2 添加集成测试 job (需要 Temporal 服务)
- [ ] 1.3 添加验收测试 job (可选，长时运行)
- [ ] 1.4 配置覆盖率报告上传到 Codecov
- [ ] 1.5 配置分支保护规则要求 CI 通过
- [ ] 1.6 添加并行测试优化

### Task 2: 增强 Docker 构建流程 (AC: #2)

当前: .github/workflows/docker.yml 已存在

- [ ] 2.1 审计现有 docker.yml 配置
- [ ] 2.2 确认多平台构建 (linux/amd64, linux/arm64)
- [ ] 2.3 确认 Docker Hub 和 GHCR 推送
- [ ] 2.4 添加镜像安全扫描 (Trivy)
- [ ] 2.5 添加镜像签名 (cosign, 可选)
- [ ] 2.6 优化构建缓存

### Task 3: 增强 Release 流程 (AC: #3, #4, #6)

当前: .github/workflows/release.yml 已存在

- [ ] 3.1 审计现有 release.yml 配置
- [ ] 3.2 添加 macOS/arm64 (Apple Silicon) 构建
- [ ] 3.3 添加 Linux/arm64 构建
- [ ] 3.4 生成 checksum.txt (SHA256)
- [ ] 3.5 添加 CLI 二进制构建
- [ ] 3.6 改进 Release Notes 生成
- [ ] 3.7 添加发布前测试验证

### Task 4: 添加集成测试 CI Job (AC: #1)

- [ ] 4.1 创建 .github/workflows/integration.yml
- [ ] 4.2 配置 Temporal 服务容器
- [ ] 4.3 配置 PostgreSQL 服务容器
- [ ] 4.4 运行集成测试 (make test-integration)
- [ ] 4.5 上传测试结果 artifact
- [ ] 4.6 配置测试超时 (15 分钟)

### Task 5: 添加验收测试 CI Job (AC: #1)

- [ ] 5.1 创建验收测试 workflow (独立或合并到 CI)
- [ ] 5.2 配置多 Agent 测试环境
- [ ] 5.3 运行验收测试场景
- [ ] 5.4 生成验收测试报告
- [ ] 5.5 配置为发布门禁

### Task 6: 配置分支保护 (AC: #5)

- [ ] 6.1 文档化分支保护规则配置
- [ ] 6.2 要求 CI 检查通过
- [ ] 6.3 要求 PR 审核
- [ ] 6.4 禁止直接推送到 main
- [ ] 6.5 要求签名提交 (可选)

### Task 7: 添加代码质量工具 (AC: #1)

- [ ] 7.1 配置 golangci-lint (已有)
- [ ] 7.2 添加 gosec 安全扫描
- [ ] 7.3 添加 staticcheck
- [ ] 7.4 配置 .golangci.yml 规则
- [ ] 7.5 添加依赖漏洞扫描 (govulncheck)

### Task 8: CI 性能优化 (AC: #1)

- [ ] 8.1 优化 Go 模块缓存
- [ ] 8.2 并行运行独立 jobs
- [ ] 8.3 使用 matrix 策略测试多 Go 版本
- [ ] 8.4 配置构建缓存
- [ ] 8.5 分析 CI 耗时瓶颈

### Task 9: 通知和监控 (可选)

- [ ] 9.1 配置 Slack 通知 (构建失败)
- [ ] 9.2 配置邮件通知
- [ ] 9.3 添加 CI 状态徽章到 README
- [ ] 9.4 配置 Dependabot 依赖更新

### Task 10: 文档和说明

- [ ] 10.1 创建 .github/README.md (CI/CD 说明)
- [ ] 10.2 文档化 secrets 配置要求
- [ ] 10.3 文档化手动触发流程
- [ ] 10.4 文档化版本发布流程

## Dev Notes

### 现有 GitHub Actions 工作流

| 文件 | 用途 | 状态 |
|------|------|------|
| ci.yml | 基础 CI (lint, test, build) | ✅ 已有 |
| docker.yml | Docker 镜像构建 | ✅ 已有 |
| docker-build.yml | Docker 构建 (备用) | ✅ 已有 |
| release.yml | GitHub Release 发布 | ✅ 已有 |
| stress-test.yml | 压力测试 (定时) | ✅ 已有 |
| semantic-pr.yml | PR 标题规范检查 | ✅ 已有 |
| label.yml | 自动标签 | ✅ 已有 |

### 现有 CI 配置分析

**ci.yml 特性:**
- Go 1.24 设置
- Go 模块缓存
- golangci-lint 代码检查
- 单元测试 (make test)
- 覆盖率报告生成
- Server 和 Agent 二进制构建
- Artifact 上传

**docker.yml 特性:**
- 多平台构建 (linux/amd64, linux/arm64)
- Docker Hub 推送
- GHCR 推送
- 语义化标签 (version, major.minor, sha, latest)
- BuildKit 缓存

**release.yml 特性:**
- Tag 触发 (v*.*.*)
- 多平台二进制构建 (Linux, macOS, Windows)
- 自动 Release 创建
- Release Notes 自动生成

### 缺失功能分析

| 功能 | 状态 | 优先级 |
|------|------|--------|
| 集成测试 CI | ❌ 缺失 | 高 |
| 验收测试 CI | ❌ 缺失 | 高 |
| Codecov 集成 | ❌ 缺失 | 中 |
| macOS/arm64 构建 | ❌ 缺失 | 中 |
| Linux/arm64 二进制 | ❌ 缺失 | 中 |
| Checksum 文件 | ❌ 缺失 | 中 |
| 镜像安全扫描 | ❌ 缺失 | 中 |
| gosec 安全扫描 | ❌ 缺失 | 低 |
| CLI 二进制发布 | ❌ 缺失 | 低 |

### Makefile 目标参考

```makefile
# 测试相关
make test           # 单元测试
make test-quick     # 快速测试
make test-integration # 集成测试
make coverage       # 覆盖率报告

# 构建相关
make build          # 构建 server
make build-agent    # 构建 agent
make cli            # 构建 CLI

# 检查相关
make lint           # 代码检查
make fmt            # 格式化
make check          # fmt + lint + test-quick
make verify         # fmt + lint + build
```

### CI 工作流建议结构

```yaml
# .github/workflows/ci.yml (增强版)
jobs:
  lint:
    runs-on: ubuntu-latest
    steps: [checkout, go, golangci-lint]
  
  unit-test:
    runs-on: ubuntu-latest
    steps: [checkout, go, make test, coverage]
  
  integration-test:
    runs-on: ubuntu-latest
    needs: [lint, unit-test]
    services:
      temporal: ...
      postgresql: ...
    steps: [checkout, go, make test-integration]
  
  build:
    runs-on: ubuntu-latest
    needs: [lint]
    steps: [checkout, go, make build build-agent cli]
```

### Secrets 配置要求

| Secret | 用途 | 必需 |
|--------|------|------|
| DOCKER_USERNAME | Docker Hub 登录 | 是 |
| DOCKER_PASSWORD | Docker Hub 密码 | 是 |
| GITHUB_TOKEN | GHCR/Release (自动) | 自动 |
| CODECOV_TOKEN | Codecov 上传 | 否 |
| SLACK_WEBHOOK | Slack 通知 | 否 |

### 分支保护规则建议

```
main 分支:
- Require pull request before merging
- Require status checks to pass: [lint, unit-test, build]
- Require branches to be up to date
- Require conversation resolution
- Require signed commits (optional)

develop 分支:
- Require pull request before merging
- Require status checks to pass: [lint, unit-test]
```

### Project Structure Notes

- 现有 CI 配置基础良好，需要增强集成测试
- Docker 构建已支持多平台和多 registry
- Release 流程需要添加更多平台和 checksum
- 需要配置分支保护规则

### References

- [Source: docs/architecture.md#AR9] - CI/CD: GitHub Actions 自动化构建
- [Source: docs/epics.md#story-11.4] - CI/CD 验收标准
- [Source: .github/workflows/ci.yml] - 现有 CI 配置
- [Source: .github/workflows/release.yml] - 现有 Release 配置
- [Source: .github/workflows/docker.yml] - 现有 Docker 构建配置

---

## Dev Agent Record

### Context Reference

<!-- Path(s) to story context XML will be added here by context workflow -->

### Agent Model Used

Claude Opus 4.5

### Debug Log References

无

### Completion Notes List

1. **CI 工作流增强** (ci.yml)
   - 添加并行化: lint 和 security 并行执行
   - 新增 security job (govulncheck)
   - 分离 unit-test 和 build jobs
   - 集成测试和验收测试已包含

2. **Docker 构建增强** (docker.yml)
   - 添加 QEMU 支持多平台构建
   - 添加 Trivy 安全扫描
   - Agent 镜像也推送到 GHCR
   - 优化标签策略

3. **Release 流程增强** (release.yml)
   - 使用 matrix 策略构建 15 个二进制
   - 添加 Linux/arm64, macOS/arm64 支持
   - 添加 CLI 二进制构建
   - 生成 SHA256 checksums
   - 改进 Release Notes

4. **文档完善**
   - 创建 .github/README.md CI/CD 文档
   - 文档化分支保护规则配置
   - 更新 README 徽章

### File List

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `.github/workflows/ci.yml` | 重写 | 增强版 CI 工作流 (lint/security/test/build/integration/acceptance) |
| `.github/workflows/docker.yml` | 修改 | 添加 QEMU, Trivy 扫描, GHCR 推送 |
| `.github/workflows/release.yml` | 重写 | 多平台构建 (15 二进制), checksum, 改进 Release Notes |
| `.github/README.md` | 新建 | CI/CD 工作流文档 (分支保护/Secrets/故障排查) |
| `README.md` | 修改 | 添加 CI/Docker/Release/Codecov/Go Report 徽章 |
| `README_zh.md` | 修改 | 添加 CI/Docker/Release/Codecov/Go Report 徽章 |

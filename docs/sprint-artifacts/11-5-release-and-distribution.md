# Story 11.5: 发布和分发

**Status:** ready-for-dev

## Story

As a **用户**,
I want **多种方式获取 Waterflow**,
So that **选择最适合的安装方式**。

## Acceptance Criteria

### AC1: Docker Hub 提供最新镜像

**Given** 新版本发布  
**When** 用户从 Docker Hub 拉取镜像  
**Then** docker pull websoft9/waterflow-server:latest 成功  
**And** docker pull websoft9/waterflow-agent:latest 成功  
**And** 镜像带有版本标签 (v1.0.0, v1.0, v1)  
**And** 镜像支持 linux/amd64 和 linux/arm64 平台  
**And** 镜像小于 100MB (优化后)

### AC2: GitHub Releases 提供二进制下载

**Given** 新版本发布  
**When** 用户访问 GitHub Releases 页面  
**Then** 提供 Server 二进制 (Linux/macOS/Windows)  
**And** 提供 Agent 二进制 (Linux/macOS/Windows)  
**And** 提供 CLI 二进制 (Linux/macOS/Windows)  
**And** 每个平台提供 amd64 和 arm64 架构  
**And** 文件命名规范: waterflow-{component}-{os}-{arch}[.exe]

### AC3: Go modules 可通过 go get 安装

**Given** Go 开发环境  
**When** 执行 go get github.com/Websoft9/waterflow  
**Then** 成功下载和安装模块  
**And** SDK 包可正常导入使用  
**And** 版本号与 Release 一致  
**And** go.mod 配置正确

### AC4: 提供版本号和 Changelog

**Given** 新版本发布  
**When** 查看发布信息  
**Then** Release 标题包含版本号 (v1.0.0)  
**And** Release Notes 包含变更内容  
**And** CHANGELOG.md 文件同步更新  
**And** 二进制 --version 输出正确版本号  
**And** Docker 镜像 LABEL 包含版本信息

### AC5: 提供 checksum 文件验证完整性

**Given** GitHub Releases 发布  
**When** 下载二进制文件  
**Then** 提供 checksums.txt 文件 (SHA256)  
**And** 每个二进制文件有对应的 checksum  
**And** 文档说明如何验证 checksum  
**And** 可选: 提供 GPG 签名

### AC6: 所有分发渠道版本同步

**Given** 版本 v1.0.0 发布  
**When** 检查所有分发渠道  
**Then** Docker Hub 镜像版本为 v1.0.0  
**And** GHCR 镜像版本为 v1.0.0  
**And** GitHub Release 版本为 v1.0.0  
**And** Go module 版本为 v1.0.0  
**And** 所有渠道发布时间相近 (<1 小时)

## Tasks / Subtasks

### Task 1: Docker 镜像分发 (AC: #1)

- [ ] 1.1 确认 Docker Hub 仓库配置 (websoft9/waterflow-server, websoft9/waterflow-agent)
- [ ] 1.2 确认 GHCR 仓库配置 (ghcr.io/websoft9/waterflow-server)
- [ ] 1.3 配置多平台构建 (linux/amd64, linux/arm64)
- [ ] 1.4 配置版本标签策略 (latest, v1.0.0, v1.0, v1)
- [ ] 1.5 优化镜像大小 (<100MB)
- [ ] 1.6 添加镜像元数据 LABEL

### Task 2: GitHub Releases 二进制分发 (AC: #2)

- [ ] 2.1 配置多平台二进制构建矩阵
- [ ] 2.2 添加 Server 二进制 (6 个: linux/darwin/windows × amd64/arm64)
- [ ] 2.3 添加 Agent 二进制 (6 个: linux/darwin/windows × amd64/arm64)
- [ ] 2.4 添加 CLI 二进制 (6 个: linux/darwin/windows × amd64/arm64)
- [ ] 2.5 统一文件命名规范
- [ ] 2.6 压缩二进制 (tar.gz for Linux/macOS, zip for Windows)

### Task 3: Go Modules 分发 (AC: #3)

- [ ] 3.1 验证 go.mod 配置正确
- [ ] 3.2 确保 module 路径与仓库 URL 一致
- [ ] 3.3 测试 go get 安装流程
- [ ] 3.4 测试 SDK 包导入和使用
- [ ] 3.5 添加 pkg.go.dev 文档徽章

### Task 4: 版本管理和 Changelog (AC: #4)

- [ ] 4.1 创建 CHANGELOG.md (如不存在)
- [ ] 4.2 配置 Release Notes 自动生成
- [ ] 4.3 添加版本号注入到二进制 (-ldflags)
- [ ] 4.4 添加版本号到 Docker 镜像 LABEL
- [ ] 4.5 配置 --version 命令输出格式
- [ ] 4.6 建立版本发布规范文档

### Task 5: Checksum 和完整性验证 (AC: #5)

- [ ] 5.1 在 Release 工作流生成 checksums.txt
- [ ] 5.2 使用 SHA256 算法
- [ ] 5.3 格式: `<sha256sum>  <filename>`
- [ ] 5.4 文档化验证方法 (sha256sum -c)
- [ ] 5.5 可选: 添加 GPG 签名
- [ ] 5.6 可选: 添加 cosign 签名 (容器镜像)

### Task 6: 分发渠道同步 (AC: #6)

- [ ] 6.1 确保所有工作流由同一 Tag 触发
- [ ] 6.2 配置工作流依赖顺序 (测试 → 构建 → 发布)
- [ ] 6.3 添加发布验证脚本
- [ ] 6.4 创建发布检查清单

### Task 7: 安装脚本和便捷工具

- [ ] 7.1 创建一键安装脚本 (scripts/install.sh)
- [ ] 7.2 支持自动检测平台和架构
- [ ] 7.3 支持指定版本安装
- [ ] 7.4 支持安装到自定义路径
- [ ] 7.5 添加卸载脚本

### Task 8: 文档和用户指南

- [ ] 8.1 更新 README 安装部分
- [ ] 8.2 创建详细安装指南 (docs/installation.md)
- [ ] 8.3 添加各平台安装示例
- [ ] 8.4 添加 Docker Compose 快速开始
- [ ] 8.5 添加从源码构建说明

### Task 9: 发布自动化增强

- [ ] 9.1 创建 release 检查脚本
- [ ] 9.2 自动更新 CHANGELOG
- [ ] 9.3 自动创建 Release PR (可选)
- [ ] 9.4 配置发布通知 (Slack/Discord)
- [ ] 9.5 添加发布后验证步骤

### Task 10: 版本发布流程文档

- [ ] 10.1 创建 RELEASING.md 发布指南
- [ ] 10.2 文档化版本号规范 (SemVer)
- [ ] 10.3 文档化发布步骤
- [ ] 10.4 文档化回滚流程
- [ ] 10.5 创建发布检查清单模板

## Dev Notes

### 当前分发渠道状态

| 渠道 | 状态 | 配置文件 |
|------|------|----------|
| Docker Hub | ✅ 已配置 | .github/workflows/docker.yml |
| GHCR | ✅ 已配置 | .github/workflows/docker.yml |
| GitHub Releases | ✅ 已配置 | .github/workflows/release.yml |
| Go Modules | ✅ 可用 | go.mod (github.com/Websoft9/waterflow) |

### 二进制构建矩阵

```
组件: server, agent, cli
平台: linux, darwin, windows
架构: amd64, arm64

总计: 3 × 3 × 2 = 18 个二进制文件

命名规范:
- waterflow-server-linux-amd64
- waterflow-server-linux-arm64
- waterflow-server-darwin-amd64
- waterflow-server-darwin-arm64
- waterflow-server-windows-amd64.exe
- waterflow-server-windows-arm64.exe
- waterflow-agent-linux-amd64
- waterflow-agent-linux-arm64
- ...
- waterflow-cli-linux-amd64
- waterflow-cli-linux-arm64
- ...
```

### Docker 镜像标签策略

```
发布 v1.2.3 时:
- websoft9/waterflow-server:v1.2.3
- websoft9/waterflow-server:v1.2
- websoft9/waterflow-server:v1
- websoft9/waterflow-server:latest (仅主分支)
- websoft9/waterflow-server:develop (开发分支)
- websoft9/waterflow-server:sha-abc1234 (commit SHA)
```

### 版本号注入

```go
// cmd/server/main.go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildTime = "unknown"
)

// 构建时注入
go build -ldflags="-X main.Version=v1.0.0 -X main.Commit=abc1234 -X main.BuildTime=2026-01-23"
```

### Checksum 文件格式

```
# checksums.txt (SHA256)
a1b2c3d4e5f6...  waterflow-server-linux-amd64
b2c3d4e5f6a1...  waterflow-server-linux-arm64
c3d4e5f6a1b2...  waterflow-server-darwin-amd64
d4e5f6a1b2c3...  waterflow-server-darwin-arm64
...

# 验证方法
sha256sum -c checksums.txt
```

### 安装脚本示例

```bash
#!/bin/bash
# scripts/install.sh

VERSION="${1:-latest}"
COMPONENT="${2:-server}"
INSTALL_DIR="${3:-/usr/local/bin}"

# 检测平台
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac

# 下载
DOWNLOAD_URL="https://github.com/Websoft9/waterflow/releases/download/${VERSION}/waterflow-${COMPONENT}-${OS}-${ARCH}"
curl -L -o waterflow-${COMPONENT} "$DOWNLOAD_URL"
chmod +x waterflow-${COMPONENT}
sudo mv waterflow-${COMPONENT} ${INSTALL_DIR}/

echo "Waterflow ${COMPONENT} ${VERSION} installed to ${INSTALL_DIR}"
```

### 发布检查清单

```markdown
## Release Checklist v1.0.0

### 发布前
- [ ] 所有测试通过 (unit, integration, acceptance)
- [ ] CHANGELOG.md 已更新
- [ ] 版本号已确认
- [ ] 依赖已更新 (go mod tidy)

### 发布中
- [ ] 创建 Tag: git tag v1.0.0
- [ ] 推送 Tag: git push origin v1.0.0
- [ ] 等待 CI 完成

### 发布后验证
- [ ] GitHub Release 已创建
- [ ] 所有二进制已上传
- [ ] checksums.txt 已生成
- [ ] Docker Hub 镜像可拉取
- [ ] GHCR 镜像可拉取
- [ ] go get 可安装
- [ ] 版本号正确
```

### Project Structure Notes

- Go module 路径: github.com/Websoft9/waterflow
- Docker 镜像: websoft9/waterflow-server, websoft9/waterflow-agent
- 发布触发: Tag v*.*.* 推送
- 构建目录: build/ (Dockerfile.server, Dockerfile.agent)

### References

- [Source: docs/architecture.md#AR10] - 分发方式: Docker Hub, GitHub Releases, Go modules
- [Source: docs/epics.md#story-11.5] - 发布和分发验收标准
- [Source: .github/workflows/release.yml] - Release 工作流
- [Source: .github/workflows/docker.yml] - Docker 构建工作流
- [Source: go.mod] - Go module 配置

---

## Dev Agent Record

### Context Reference

<!-- Path(s) to story context XML will be added here by context workflow -->

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List


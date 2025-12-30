# File/Transfer Node

文件传输节点 - 通过 SFTP/SCP 协议在本地与远程服务器之间传输文件。

## 功能描述

`file/transfer` 节点提供安全的文件上传/下载功能，支持 SFTP 和 SCP 协议，适用于跨服务器文件分发、备份、配置同步等场景。

## 使用场景

1. **配置文件分发** - 上传配置到多个服务器
2. **日志收集** - 从远程服务器下载日志文件  
3. **备份传输** - 传输备份文件到备份服务器
4. **代码部署** - 上传应用包到目标服务器
5. **数据同步** - 在服务器之间同步数据文件

## 参数

### 输入参数

| 参数名 | 类型 | 必需 | 默认值 | 描述 |
|--------|------|------|--------|------|
| `mode` | string | 是 | - | 传输模式: 'upload' (本地→远程) 或 'download' (远程→本地) |
| `protocol` | string | 否 | sftp | 传输协议: 'sftp' (推荐) 或 'scp' |
| `host` | string | 是 | - | 远程主机 (IP 或域名) |
| `port` | int | 否 | 22 | SSH 端口 |
| `user` | string | 是 | - | SSH 用户名 |
| `password` | string | 否 | - | SSH 密码 (与 private_key 二选一) |
| `private_key` | string | 否 | - | SSH 私钥 (文件路径或 PEM 内容) |
| `source_path` | string | 是 | - | 源文件路径 (upload: 本地, download: 远程) |
| `target_path` | string | 是 | - | 目标文件路径 (upload: 远程, download: 本地) |
| `permissions` | string | 否 | - | 上传文件权限 (如 "0644", "0755") |
| `timeout` | string | 否 | 30s | 连接超时 (如 "30s", "1m") |

### 输出参数

| 参数名 | 类型 | 描述 |
|--------|------|------|
| `bytes_transferred` | int | 传输字节数 |
| `protocol` | string | 使用的协议 (sftp/scp) |
| `mode` | string | 传输模式 (upload/download) |
| `duration_ms` | int | 传输耗时 (毫秒) |

## 使用示例

### 示例 1: 上传配置文件

```yaml
name: Upload Config
jobs:
  deploy-config:
    runs-on: linux-amd64
    steps:
      - name: Upload nginx config
        uses: file/transfer@v1
        with:
          mode: upload
          host: 192.168.1.100
          user: admin
          password: ${{ secrets.SSH_PASSWORD }}
          source_path: /local/nginx.conf
          target_path: /etc/nginx/nginx.conf
          permissions: "0644"
```

### 示例 2: 下载日志文件

```yaml
- name: Download application logs
  uses: file/transfer@v1
  with:
    mode: download
    host: app-server.example.com
    user: appuser
    private_key: ${{ secrets.SSH_PRIVATE_KEY }}
    source_path: /var/log/app/application.log
    target_path: ./logs/app-{{ steps.context.outputs.date }}.log
```

### 示例 3: 使用私钥认证

```yaml
- name: Deploy application package
  uses: file/transfer@v1
  with:
    mode: upload
    protocol: sftp
    host: ${{ matrix.server }}
    port: 22
    user: deploy
    private_key: /home/deploy/.ssh/id_rsa
    source_path: ./dist/app-v1.2.3.tar.gz
    target_path: /opt/app/releases/app-v1.2.3.tar.gz
```

### 示例 4: 批量文件传输

```yaml
- name: Upload multiple configs
  uses: file/transfer@v1
  with:
    mode: upload
    host: ${{ matrix.server }}
    user: admin
    password: ${{ secrets.SSH_PASSWORD }}
    source_path: ${{ matrix.config_file }}
    target_path: /etc/app/${{ matrix.config_file }}
```

### 示例 5: 备份下载

```yaml
- name: Download database backup
  uses: file/transfer@v1
  with:
    mode: download
    host: db-server.local
    user: backup
    private_key: ${{ secrets.BACKUP_KEY }}
    source_path: /backup/db-{{ steps.date.outputs.today }}.sql.gz
    target_path: ./backups/db-latest.sql.gz
    timeout: 5m
```

### 示例 6: 跨服务器文件同步

```yaml
name: Sync Files
jobs:
  sync:
    runs-on: linux-amd64
    steps:
      - name: Download from source
        uses: file/transfer@v1
        with:
          mode: download
          host: source-server.com
          user: sync
          password: ${{ secrets.SOURCE_PASSWORD }}
          source_path: /data/export.tar
          target_path: /tmp/export.tar
        id: download

      - name: Upload to target
        uses: file/transfer@v1
        with:
          mode: upload
          host: target-server.com
          user: sync
          password: ${{ secrets.TARGET_PASSWORD }}
          source_path: /tmp/export.tar
          target_path: /data/import.tar
```

## 错误处理和重试

节点自动分类传输错误：

| 错误类型 | 条件 | Temporal 行为 | 说明 |
|----------|------|---------------|------|
| **永久错误** | 认证失败、权限拒绝、文件不存在 | 不重试 | 需要修复配置 |
| **临时错误** | 网络超时、连接被拒绝、服务器错误 | 可重试 | 可能是临时故障 |

## 限制和注意事项

### 协议选择

- **SFTP (推荐)**: 更稳定，支持更多文件操作，推荐用于生产环境
- **SCP**: 更快但功能较少，适合简单文件传输

### 认证方式

- **密码认证**: 简单但安全性较低，建议仅用于测试环境
- **私钥认证**: 推荐用于生产环境，更安全

### 文件路径

- **上传**: source_path 为本地路径，target_path 为远程路径
- **下载**: source_path 为远程路径，target_path 为本地路径
- 自动创建目标目录（如果不存在）

### 文件权限

- `permissions` 参数仅在上传时生效
- 使用八进制格式 (如 "0644", "0755")
- 下载时保留原始权限

### 超时设置

- 默认超时：30秒
- 建议根据文件大小调整：
  - 小文件 (<10MB): 30s
  - 中等文件 (10MB-100MB): 1-5m
  - 大文件 (>100MB): 5-10m

### 性能考虑

- 大文件传输可能耗时较长，建议：
  - 压缩文件后传输
  - 使用合适的超时时间
  - 在 retry 策略中配置适当的重试次数

## 编译和测试

### 快速开始

```bash
# 编译插件
make build

# 运行单元测试
make test

# 清理
make clean

# 安装
make install
```

### 测试说明

**单元测试** (不需要 SSH 服务器):
```bash
make test
```
- 测试参数验证、元数据、错误分类等
- 11 个测试用例
- 覆盖率约 27% (核心传输逻辑需要 SSH 环境)

**集成测试** (需要 Docker):
```bash
make integration-test
```
- 自动启动 SFTP 测试服务器 (Docker 容器)
- 测试真实文件上传/下载、权限设置、大文件传输等
- 8 个集成测试场景
- 测试完成后自动清理容器

**手动运行集成测试**:
```bash
# 1. 启动 SFTP 测试服务器
docker run -d --name sftp-test -p 2222:22 \
  -v /tmp/sftp-upload:/home/testuser/upload \
  atmoz/sftp testuser:testpass:::upload

# 2. 运行集成测试
go test -v -tags=integration -timeout 5m

# 3. 清理
docker rm -f sftp-test
```

### 覆盖率报告

单元测试覆盖了参数解析、验证、错误分类等逻辑。集成测试覆盖核心传输功能。

```bash
# 查看详细覆盖率
go tool cover -html=coverage.out
```

## 技术实现

- **SSH 客户端**: golang.org/x/crypto/ssh
- **SFTP**: github.com/pkg/sftp
- **错误分类**: Temporal SDK `temporal.NewApplicationError` / `temporal.NewNonRetryableApplicationError`

## 安全建议

1. **使用密钥认证**: 避免在 YAML 中硬编码密码
2. **使用 Secrets**: 敏感信息存储在 Waterflow secrets
3. **最小权限原则**: SSH 用户仅授予必要的权限
4. **审计日志**: 记录所有文件传输操作
5. **网络隔离**: 限制 SSH 访问的源 IP

## 许可证

与 Waterflow 项目保持一致

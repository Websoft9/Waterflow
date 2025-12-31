# file/transfer@v1

**分类**: file (文件操作)  
**版本**: v1  
**状态**: stable

## 概述

file/transfer 节点用于在服务器间传输文件，支持 SFTP/SCP 协议的双向传输（上传/下载）。支持密码和私钥认证，提供安全的文件传输能力。

## 使用场景

- **配置分发**: 上传配置文件到多台应用服务器
- **日志收集**: 从远程服务器下载日志进行分析
- **部署资产**: 上传构建产物、脚本到目标服务器
- **备份传输**: 下载数据库备份到中心存储
- **证书分发**: 安全分发 SSL 证书和密钥

## 参数

| 参数名 | 类型 | 必需 | 默认值 | 描述 |
|--------|------|------|--------|------|
| mode | string | ✅ | - | 传输模式: upload/download |
| protocol | string | ❌ | sftp | 协议: sftp/scp |
| host | string | ✅ | - | 远程主机 |
| port | int | ❌ | 22 | SSH 端口 |
| user | string | ✅ | - | SSH 用户名 |
| password | string | ❌* | - | SSH 密码 |
| private_key | string | ❌* | - | 私钥路径或内容 |
| source_path | string | ✅ | - | 源文件路径 |
| target_path | string | ✅ | - | 目标文件路径 |
| permissions | string | ❌ | - | 文件权限 (如 "0644") |
| timeout | string | ❌ | 30s | 连接超时 |
| host_key_check | bool | ❌ | true | 验证主机密钥 |

*注: password 和 private_key 必须提供其中一个。

### 参数详细说明

**mode** (string, required)
- 传输模式
- upload: 本地→远程
- download: 远程→本地
- 示例: `"upload"`

**protocol** (string, optional, default: sftp)
- 传输协议
- sftp: 推荐，功能完整
- scp: 较快但功能有限（当前使用 SFTP 实现）
- 示例: `"sftp"`

**host** (string, required)
- 远程服务器主机名或 IP
- 示例: `"192.168.1.100"`, `"app.example.com"`

**port** (int, optional, default: 22)
- SSH 端口号
- 示例: `2222`

**user** (string, required)
- SSH 用户名
- 示例: `"deploy"`, `"ubuntu"`

**password** (string, optional)
- SSH 密码（与 private_key 二选一）
- 建议使用 Secrets
- 示例: `"${{ secrets.SSH_PASSWORD }}"`

**private_key** (string, optional)
- SSH 私钥
- 可以是文件路径或内联私钥内容
- 文件路径示例: `"/home/user/.ssh/id_rsa"`
- 内联内容示例: `"-----BEGIN RSA PRIVATE KEY-----\n..."`

**source_path** (string, required)
- 源文件路径
- upload: 本地文件路径
- download: 远程文件路径
- 示例: `"/app/config.yml"`

**target_path** (string, required)
- 目标文件路径
- upload: 远程文件路径
- download: 本地文件路径
- 自动创建目标目录（download 模式）
- 示例: `"/etc/app/config.yml"`

**permissions** (string, optional)
- 上传文件的权限（upload 模式）
- 八进制格式字符串
- 示例: `"0644"` (rw-r--r--), `"0755"` (rwxr-xr-x), `"0600"` (rw-------)

**timeout** (string, optional, default: 30s)
- 连接超时时间
- 大文件传输建议增加
- 示例: `"5m"`

**host_key_check** (bool, optional, default: true)
- 是否验证主机密钥
- 生产环境建议 true（防止中间人攻击）
- 开发环境可设为 false
- 示例: `false`

## 返回值

| 字段名 | 类型 | 描述 |
|--------|------|------|
| transferred_bytes | int | 传输字节数 |
| file_size | int | 文件大小 |
| elapsed_ms | int | 传输耗时 (毫秒) |
| mode | string | upload/download |
| remote_path | string | 远程文件路径 |

## 使用示例

### 示例 1: 上传配置文件（密码认证）

```yaml
steps:
  - name: Upload nginx config
    uses: file/transfer@v1
    with:
      mode: upload
      host: 192.168.1.100
      user: deploy
      password: ${{ secrets.SSH_PASSWORD }}
      source_path: /local/nginx.conf
      target_path: /etc/nginx/nginx.conf
      permissions: "0644"
```

### 示例 2: 上传文件（私钥认证）

```yaml
steps:
  - name: Upload application binary
    uses: file/transfer@v1
    with:
      mode: upload
      host: app.example.com
      user: deploy
      private_key: /home/runner/.ssh/id_rsa
      source_path: /build/app
      target_path: /opt/app/bin/app
      permissions: "0755"
      timeout: 5m
```

### 示例 3: 下载日志文件

```yaml
steps:
  - name: Download application logs
    uses: file/transfer@v1
    with:
      mode: download
      host: app.example.com
      user: admin
      private_key: ${{ secrets.SSH_PRIVATE_KEY }}
      source_path: /var/log/app/app.log
      target_path: /local/logs/app.log
```

### 示例 4: 内联私钥

```yaml
steps:
  - name: Upload with inline key
    uses: file/transfer@v1
    with:
      mode: upload
      host: server.com
      user: deploy
      private_key: ${{ secrets.DEPLOY_SSH_KEY }}
      source_path: /app/release.tar.gz
      target_path: /tmp/release.tar.gz
```

### 示例 5: 跳过主机密钥验证（开发环境）

```yaml
steps:
  - name: Upload to dev server
    uses: file/transfer@v1
    with:
      mode: upload
      host: 192.168.1.50
      user: dev
      password: ${{ secrets.DEV_PASSWORD }}
      source_path: /build/app.jar
      target_path: /app/app.jar
      host_key_check: false  # 仅开发环境
```

### 示例 6: 批量传输

```yaml
steps:
  - name: Upload config files
    uses: file/transfer@v1
    with:
      mode: upload
      host: server.com
      user: deploy
      private_key: ${{ secrets.SSH_KEY }}
      source_path: /config/app.yml
      target_path: /etc/app/app.yml
      permissions: "0644"
  
  - name: Upload service file
    uses: file/transfer@v1
    with:
      mode: upload
      host: server.com
      user: deploy
      private_key: ${{ secrets.SSH_KEY }}
      source_path: /config/app.service
      target_path: /etc/systemd/system/app.service
      permissions: "0644"
```

## 常见错误

### 错误 1: SSH 认证失败

**现象:**
```
Error: ssh: unable to authenticate, attempted methods [none password]
```

**解决方法:**
1. 检查用户名和密码正确性
2. 确认私钥格式正确
3. 验证私钥权限: `chmod 600 ~/.ssh/id_rsa`

### 错误 2: 文件不存在

**现象:**
```
Error: file not found: /path/to/file
```

**解决方法:**
1. 检查文件路径是否正确
2. 确认文件在源服务器上存在
3. 使用绝对路径

### 错误 3: 权限拒绝

**现象:**
```
Error: permission denied
```

**解决方法:**
1. 检查目标目录权限
2. 确认用户有写入权限
3. 检查 SELinux/AppArmor 策略

### 错误 4: 主机密钥验证失败

**现象:**
```
Error: ssh: handshake failed: knownhosts: key mismatch
```

**解决方法:**
1. 更新 known_hosts 文件
2. 开发环境可设置 `host_key_check: false`
3. 生产环境需验证主机密钥变更原因

### 错误 5: 连接超时

**现象:**
```
Error: dial tcp: i/o timeout
```

**解决方法:**
1. 检查网络连接
2. 确认防火墙规则
3. 验证 SSH 服务运行状态
4. 增加超时时间

## 最佳实践

### 1. 使用私钥认证（生产环境）

```yaml
# ✅ 推荐: 私钥认证更安全
private_key: ${{ secrets.SSH_PRIVATE_KEY }}

# ❌ 密码认证（仅开发环境）
password: ${{ secrets.SSH_PASSWORD }}
```

### 2. 启用主机密钥验证（生产环境）

```yaml
# ✅ 生产环境
host_key_check: true

# ❌ 仅开发环境
host_key_check: false
```

### 3. 设置正确的文件权限

```yaml
# 配置文件
permissions: "0644"  # rw-r--r--

# 可执行文件
permissions: "0755"  # rwxr-xr-x

# 密钥文件
permissions: "0600"  # rw-------
```

### 4. 使用 Secrets 管理凭据

```yaml
# ✅ 安全
password: ${{ secrets.SSH_PASSWORD }}
private_key: ${{ secrets.SSH_PRIVATE_KEY }}

# ❌ 不安全
password: "hardcoded_password"
```

### 5. 大文件传输调整超时

```yaml
# 小文件 (<10MB)
timeout: 30s

# 中等文件 (10-100MB)
timeout: 5m

# 大文件 (>100MB)
timeout: 15m
```

### 6. 使用绝对路径

```yaml
# ✅ 推荐
source_path: /app/config.yml
target_path: /etc/app/config.yml

# ❌ 相对路径可能有歧义
source_path: config.yml
```

## 安全最佳实践

1. **认证方式**: 生产环境优先使用私钥认证
2. **主机密钥验证**: 生产环境必须启用 host_key_check
3. **私钥权限**: 私钥文件权限设置为 0600
4. **凭据管理**: 使用 Secrets 管理密码和私钥
5. **文件权限**: 根据文件类型设置合适权限
6. **定期轮换**: 定期轮换 SSH 密钥和密码

## 协议选择

| 协议 | 特点 | 适用场景 |
|------|------|----------|
| **SFTP** | 功能完整、稳定可靠 | 推荐所有场景 |
| **SCP** | 理论更快（当前使用SFTP实现） | 兼容性场景 |

## 相关节点

- [exec/shell](../exec/shell.md) - 使用 scp/rsync 命令传输
- [http/request](../http/request.md) - HTTP 文件下载

## 参考文档

- [Story 3.6: 文件传输节点](../../sprint-artifacts/3-6-file-transfer-node.md)

---

**文档版本**: v1  
**最后更新**: 2025-12-31  
**状态**: 稳定

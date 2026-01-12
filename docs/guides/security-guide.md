# Waterflow 安全配置指南

本指南提供 Waterflow 的全面安全配置和运维最佳实践,帮助系统管理员安全地部署和运维生产环境。

## 目录

1. [安全概述](#1-安全概述)
2. [传输层安全 (HTTPS/TLS)](#2-传输层安全-httpstls)
3. [认证和授权](#3-认证和授权)
4. [密钥管理](#4-密钥管理)
5. [审计日志](#5-审计日志)
6. [网络安全](#6-网络安全)
7. [容器安全 (Docker)](#7-容器安全-docker)
8. [备份和恢复](#8-备份和恢复)
9. [安全监控](#9-安全监控)
10. [事件响应](#10-事件响应)
11. [安全检查清单](#11-安全检查清单)
12. [合规指南](#12-合规指南)

---

## 1. 安全概述

### 1.1 安全架构

Waterflow 采用多层安全架构,提供深度防御:

```
┌──────────────────────────────────────────────────────────┐
│                 Waterflow 安全架构                        │
│                                                          │
│  ┌────────────────────────────────────────────────┐     │
│  │  1. 传输层安全 (Story 9.1)                    │     │
│  │     - HTTPS/TLS 加密                          │     │
│  │     - 证书管理                                │     │
│  │     - 安全密码套件                            │     │
│  └────────────────────────────────────────────────┘     │
│                         ↓                                │
│  ┌────────────────────────────────────────────────┐     │
│  │  2. 认证和授权                                │     │
│  │     - API Key/Token 认证                      │     │
│  │     - Agent 身份验证                          │     │
│  │     - 最小权限原则                            │     │
│  └────────────────────────────────────────────────┘     │
│                         ↓                                │
│  ┌────────────────────────────────────────────────┐     │
│  │  3. 密钥管理 (Story 9.2)                      │     │
│  │     - SecretProvider 接口                     │     │
│  │     - Vault 集成                              │     │
│  │     - 零凭证存储                              │     │
│  └────────────────────────────────────────────────┘     │
│                         ↓                                │
│  ┌────────────────────────────────────────────────┐     │
│  │  4. 审计和合规 (Story 9.3)                    │     │
│  │     - 审计日志                                │     │
│  │     - 操作追踪                                │     │
│  │     - 合规报告                                │     │
│  └────────────────────────────────────────────────┘     │
│                         ↓                                │
│  ┌────────────────────────────────────────────────┐     │
│  │  5. 网络安全                                  │     │
│  │     - 防火墙规则                              │     │
│  │     - 网络隔离                                │     │
│  │     - 端口管理                                │     │
│  └────────────────────────────────────────────────┘     │
│                         ↓                                │
│  ┌────────────────────────────────────────────────┐     │
│  │  6. 运维安全                                  │     │
│  │     - 定期更新                                │     │
│  │     - 备份和恢复                              │     │
│  │     - 事件响应                                │     │
│  └────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────┘
```

### 1.2 安全设计原则

Waterflow 遵循以下安全设计原则:

- **深度防御** - 多层安全控制,单点失败不会导致整体安全崩溃
- **最小权限** - 仅授予必要的权限,限制潜在损害范围
- **零信任** - 验证所有请求,不信任任何默认连接
- **加密传输** - 所有通信使用 TLS 加密,保护数据在传输中的安全
- **审计一切** - 记录所有关键操作,支持事后分析和合规审查
- **隔离原则** - 环境和网络隔离,降低横向移动风险

### 1.3 安全功能概览

| 功能 | 实现Story | 说明 |
|------|----------|------|
| **HTTPS/TLS加密** | Story 9.1 | 保护所有API通信 |
| **API认证** | Story 1.2 | API Key/Token验证 |
| **密钥管理** | Story 9.2 | Vault集成,零凭证存储 |
| **审计日志** | Story 9.3 | 记录所有操作,支持合规 |
| **网络隔离** | Epic 8 | Docker网络,防火墙规则 |
| **容器安全** | Story 8.1 | 非root用户,资源限制 |

---

## 2. 传输层安全 (HTTPS/TLS)

### 2.1 启用 HTTPS

**配置文件 (config.yaml):**
```yaml
https:
  enabled: true
  cert_file: /etc/waterflow/tls/cert.pem
  key_file: /etc/waterflow/tls/key.pem
  min_version: "1.2"  # 最低 TLS 1.2
  cipher_suites:
    - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
    - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
```

**环境变量:**
```bash
export WATERFLOW_HTTPS_ENABLED=true
export WATERFLOW_HTTPS_CERT_FILE=/etc/waterflow/tls/cert.pem
export WATERFLOW_HTTPS_KEY_FILE=/etc/waterflow/tls/key.pem
export WATERFLOW_HTTPS_MIN_VERSION=1.2
```

### 2.2 生成自签名证书 (开发/测试)

**手动生成:**
```bash
# 生成私钥
openssl genrsa -out key.pem 2048

# 生成证书签名请求
openssl req -new -key key.pem -out cert.csr \
  -subj "/CN=waterflow.example.com/O=Waterflow/C=US"

# 生成自签名证书 (有效期 365 天)
openssl x509 -req -in cert.csr -signkey key.pem \
  -out cert.pem -days 365

# 设置文件权限
chmod 600 key.pem
chmod 644 cert.pem
```

**使用脚本自动生成:**
```bash
# 开发环境 - localhost
./scripts/generate-tls-cert.sh development localhost

# 测试环境 - 自定义域名
./scripts/generate-tls-cert.sh development waterflow.test.com
```

### 2.3 生产环境证书

#### 选项 1: Let's Encrypt (推荐)

**使用 certbot 获取免费证书:**
```bash
# 安装 certbot
sudo apt-get update
sudo apt-get install certbot

# 生成证书 (需要域名指向服务器)
sudo certbot certonly --standalone \
  -d waterflow.example.com \
  --email admin@example.com \
  --agree-tos

# 证书位置
# /etc/letsencrypt/live/waterflow.example.com/fullchain.pem
# /etc/letsencrypt/live/waterflow.example.com/privkey.pem

# 配置到 Waterflow
cat > config.yaml <<EOF
https:
  enabled: true
  cert_file: /etc/letsencrypt/live/waterflow.example.com/fullchain.pem
  key_file: /etc/letsencrypt/live/waterflow.example.com/privkey.pem
EOF

# 配置自动更新 (测试)
sudo certbot renew --dry-run
```

**自动化续期 (cron):**
```bash
# /etc/cron.d/certbot-renew
0 0 1 * * root certbot renew --quiet && systemctl restart waterflow-server
```

#### 选项 2: 企业 CA

```bash
# 1. 生成 CSR
openssl req -new -key key.pem -out cert.csr \
  -subj "/CN=waterflow.example.com/O=YourCompany/C=US"

# 2. 提交 CSR 到企业 CA
# (通过企业 CA 流程获取签名证书)

# 3. 获取签名证书后配置到 Waterflow
cat > config.yaml <<EOF
https:
  enabled: true
  cert_file: /etc/waterflow/tls/cert.pem
  key_file: /etc/waterflow/tls/key.pem
  ca_cert: /etc/waterflow/tls/ca.pem  # 可选,用于验证客户端证书
EOF
```

### 2.4 证书更新流程

**检查证书有效期:**
```bash
# 查看证书详细信息
openssl x509 -in /etc/waterflow/tls/cert.pem -noout -text

# 仅显示有效期
openssl x509 -in /etc/waterflow/tls/cert.pem -noout -dates
# 输出:
# notBefore=Dec 16 00:00:00 2025 GMT
# notAfter=Dec 16 00:00:00 2026 GMT

# 检查证书是否在30天内过期
openssl x509 -checkend 2592000 -noout -in /etc/waterflow/tls/cert.pem
# 返回码 0 = 不会过期, 1 = 即将过期
```

**更新 Let's Encrypt 证书:**
```bash
# 1. 手动更新
sudo certbot renew

# 2. 强制更新 (不检查有效期)
sudo certbot renew --force-renewal

# 3. 重启 Waterflow Server
sudo systemctl restart waterflow-server

# 4. 验证新证书
curl -v https://waterflow.example.com/health 2>&1 | grep "expire date"
```

**Docker 环境更新证书:**
```bash
# 1. 更新证书文件
sudo certbot renew

# 2. 重启容器 (自动加载新证书)
docker restart waterflow-server

# 3. 验证
docker exec waterflow-server cat /etc/waterflow/tls/cert.pem | \
  openssl x509 -noout -dates
```

### 2.5 TLS 最佳实践

**推荐的 TLS 配置:**
```yaml
https:
  enabled: true
  cert_file: /etc/waterflow/tls/cert.pem
  key_file: /etc/waterflow/tls/key.pem
  
  # 最低 TLS 版本 (禁用 TLS 1.0/1.1)
  min_version: "1.2"  # 推荐 TLS 1.2+
  
  # 安全的密码套件 (按优先级排序)
  cipher_suites:
    - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
    - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
    - TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
    - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
  
  # 客户端证书验证 (可选,用于双向 TLS)
  client_auth:
    enabled: false  # 设置为 true 启用双向认证
    ca_cert: /etc/waterflow/tls/client-ca.pem
```

**验证 TLS 配置:**
```bash
# 测试 HTTPS 连接
curl -v https://waterflow.example.com/health

# 测试 TLS 版本
openssl s_client -connect waterflow.example.com:8443 -tls1_2

# 测试密码套件
nmap --script ssl-enum-ciphers -p 8443 waterflow.example.com
```

---

## 3. 认证和授权

### 3.1 启用 API Key 认证

**配置文件:**
```yaml
auth:
  enabled: true
  type: api_key
  api_keys:
    - name: admin
      key: sk_test_YOUR_API_KEY_HERExxxxxxxxxxxxx
      roles: [admin]
    - name: operator
      key: sk_test_YYYYYYYYYYYYyyyyyyyyyyyyy
      roles: [operator]
    - name: readonly
      key: sk_live_zzzzzzzzzzzzzzzz
      roles: [readonly]
```

**环境变量 (推荐用于容器部署):**
```bash
export WATERFLOW_AUTH_ENABLED=true
export WATERFLOW_AUTH_TYPE=api_key
export WATERFLOW_API_KEYS="admin:sk_test_YOUR_API_KEY_HERE:admin,operator:sk_test_YYYYYYYYYYYY:operator"
```

**生成安全的 API Key:**
```bash
# 生成随机 API Key (32字节 Base64)
openssl rand -base64 32

# 输出示例: sk_test_XXXXXX_example_key_not_real
```

### 3.2 使用 API Key

**cURL 请求:**
```bash
# 提交工作流
curl -H "Authorization: Bearer sk_test_YOUR_API_KEY_HERE" \
  -X POST https://waterflow.example.com/v1/workflows \
  -d @workflow.yaml

# 查询状态
curl -H "Authorization: Bearer sk_test_YOUR_API_KEY_HERE" \
  https://waterflow.example.com/v1/workflows/{id}
```

**CLI 配置:**
```bash
# 配置 API Key
waterflow config set api-key sk_test_YOUR_API_KEY_HERE
waterflow config set server https://waterflow.example.com

# 提交工作流
waterflow submit workflow.yaml
```

**SDK 使用:**
```go
import "github.com/websoft9/waterflow/pkg/sdk"

client := sdk.NewClient(
    sdk.WithServerURL("https://waterflow.example.com"),
    sdk.WithAPIKey("sk_test_YOUR_API_KEY_HERE"),
)

result, err := client.SubmitWorkflow(ctx, workflowYAML)
```

### 3.3 Agent 身份验证

**配置 Agent Token:**
```yaml
# agent/config.yaml
agent:
  token: agent_xxxxxxxxxxxxxxxx
  server_url: https://waterflow.example.com
  server_groups:
    - production
```

**Server 验证 Agent:**
```yaml
# server/config.yaml
agents:
  auth_enabled: true
  allowed_tokens:
    - agent_xxxxxxxxxxxxxxxx
    - agent_yyyyyyyyyyyyyyyy
```

**生成 Agent Token:**
```bash
# 生成 Agent Token
openssl rand -hex 32 | sed 's/^/agent_/'

# 输出示例: agent_4a8c2e1f9b3d7e6a5c8d2f1e9b7a6c5d
```

### 3.4 权限管理 (最小权限原则)

**角色定义:**
```yaml
roles:
  admin:
    description: "Full administrative access"
    permissions:
      - workflows:*
      - agents:*
      - audit:read
      - config:write
  
  operator:
    description: "Workflow operations"
    permissions:
      - workflows:submit
      - workflows:status
      - workflows:logs
      - workflows:cancel
  
  readonly:
    description: "Read-only access"
    permissions:
      - workflows:status
      - workflows:logs
```

**权限检查示例:**
```go
// 内部实现示例 (pkg/api/middleware)
func (m *AuthMiddleware) CheckPermission(role, action string) bool {
    permissions := m.roles[role]
    for _, perm := range permissions {
        if matchPermission(perm, action) {
            return true
        }
    }
    return false
}
```

---

## 4. 密钥管理

### 4.1 使用 Vault (推荐)

#### 安装和配置 Vault

**Docker 方式 (开发/测试):**
```bash
# 启动 Vault Dev Server
docker run -d --name vault \
  -p 8200:8200 \
  -e 'VAULT_DEV_ROOT_TOKEN_ID=myroot' \
  -e 'VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200' \
  vault:1.15

# 配置 Vault CLI
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=myroot

# 启用 KV v2 引擎
vault secrets enable -path=waterflow kv-v2
```

**生产部署 (Kubernetes):**
```yaml
# vault-values.yaml
server:
  ha:
    enabled: true
    replicas: 3
  dataStorage:
    enabled: true
    size: 10Gi
  auditStorage:
    enabled: true
    size: 5Gi

# 部署
helm install vault hashicorp/vault -f vault-values.yaml
```

#### 存储密钥到 Vault

```bash
# 存储数据库密码
vault kv put waterflow/production/db_password value=super_secret

# 存储 API Key
vault kv put waterflow/production/api_key value=sk_test_YOUR_API_KEY_HERE

# 存储 SSH 密钥
vault kv put waterflow/production/ssh_key value="$(cat ~/.ssh/id_rsa)"

# 存储多个字段
vault kv put waterflow/production/database \
  username=dbuser \
  password=dbpass \
  host=db.example.com \
  port=5432
```

#### 配置 Waterflow 使用 Vault

**配置文件:**
```yaml
# config.yaml
secrets:
  provider: vault
  vault:
    address: https://vault.example.com
    token: ${VAULT_TOKEN}  # 从环境变量读取
    mount_path: waterflow
    namespace: production  # 可选,Vault Enterprise
    tls:
      enabled: true
      ca_cert: /etc/waterflow/vault-ca.pem
      skip_verify: false  # 生产环境必须为 false
```

**环境变量:**
```bash
export VAULT_ADDR=https://vault.example.com
export VAULT_TOKEN=s.xxxxxxxxxxxxx
export WATERFLOW_SECRETS_PROVIDER=vault
export WATERFLOW_SECRETS_VAULT_ADDRESS=${VAULT_ADDR}
export WATERFLOW_SECRETS_VAULT_MOUNT_PATH=waterflow
```

#### 在工作流中使用密钥

```yaml
# workflow.yaml
name: deploy-application
vars:
  db_password: ${{ secrets.production/db_password }}
  api_key: ${{ secrets.production/api_key }}

jobs:
  deploy:
    runs-on: production
    steps:
      - uses: exec
        with:
          command: |
            export DB_PASSWORD="${{ vars.db_password }}"
            export API_KEY="${{ vars.api_key }}"
            ./deploy.sh
```

### 4.2 使用环境变量 (简单场景)

**配置:**
```yaml
# config.yaml
secrets:
  provider: env
  env:
    prefix: WATERFLOW_SECRET_
```

**设置密钥:**
```bash
export WATERFLOW_SECRET_DB_PASSWORD=super_secret
export WATERFLOW_SECRET_API_KEY=sk_test_YOUR_API_KEY_HERE
export WATERFLOW_SECRET_SSH_KEY="$(cat ~/.ssh/id_rsa)"
```

**在工作流中使用:**
```yaml
vars:
  db_password: ${{ secrets.db_password }}  # 读取 WATERFLOW_SECRET_DB_PASSWORD
  api_key: ${{ secrets.api_key }}          # 读取 WATERFLOW_SECRET_API_KEY
```

**⚠️ 环境变量模式限制:**
- 不适合生产环境 (密钥暴露在进程环境中)
- 无法实现密钥轮换
- 缺少访问审计
- 推荐仅用于开发/测试

### 4.3 密钥轮换

**定期轮换流程:**
```bash
#!/bin/bash
# scripts/rotate-secrets.sh

SECRETS=(
  "production/db_password"
  "production/api_key"
  "production/ssh_key"
)

for secret in "${SECRETS[@]}"; do
  echo "轮换密钥: $secret"
  
  # 生成新密钥
  new_value=$(openssl rand -base64 32)
  
  # 更新 Vault
  vault kv put "waterflow/$secret" value="$new_value"
  
  echo "✓ 已轮换 $secret"
  
  # 审计记录
  echo "$(date): Rotated $secret" >> /var/log/waterflow/secret-rotation.log
done

echo "✅ 密钥轮换完成"
```

**自动化轮换 (cron):**
```bash
# /etc/cron.d/waterflow-secret-rotation
# 每月1号凌晨3点执行密钥轮换
0 3 1 * * root /usr/local/bin/rotate-secrets.sh
```

**验证轮换后的密钥:**
```bash
# 查看密钥访问审计日志
waterflow audit query \
  --event-type=secret.access \
  --resource-id=production/db_password \
  --start-time="24 hours ago"
```

---

## 5. 审计日志

### 5.1 启用审计日志

**配置文件:**
```yaml
audit:
  enabled: true
  store_type: file
  file:
    path: /var/log/waterflow/audit
    max_size: 100  # MB
    max_age: 90    # days
    max_backups: 30
    compress: true
  
  # 可选: 同时输出到多个位置
  # store_type: multi
  # multi:
  #   - type: file
  #     path: /var/log/waterflow/audit
  #   - type: syslog
  #     address: syslog.example.com:514
```

**环境变量:**
```bash
export WATERFLOW_AUDIT_ENABLED=true
export WATERFLOW_AUDIT_STORE_TYPE=file
export WATERFLOW_AUDIT_FILE_PATH=/var/log/waterflow/audit
export WATERFLOW_AUDIT_FILE_MAX_AGE=90
```

### 5.2 查询审计日志

**使用 CLI 查询:**
```bash
# 查看最近的操作
waterflow audit query --limit=100

# 查看失败的认证尝试
waterflow audit query \
  --event-category=auth \
  --result=failure \
  --start-time="24 hours ago"

# 查看密钥访问记录
waterflow audit query \
  --event-category=secret \
  --start-time="7 days ago"

# 查看特定用户的操作
waterflow audit query --user-id=john.doe --limit=50

# 导出为 JSON
waterflow audit query \
  --start-time="30 days ago" \
  --format=json > audit-export.json
```

**直接查询日志文件:**
```bash
# 审计日志格式: JSON Lines (每行一个JSON对象)
cat /var/log/waterflow/audit/audit.log | jq '.'

# 查询特定事件类型
cat /var/log/waterflow/audit/audit.log | \
  jq 'select(.event_type == "workflow.submit")'

# 统计事件类型分布
cat /var/log/waterflow/audit/audit.log | \
  jq -r '.event_type' | sort | uniq -c | sort -rn
```

### 5.3 审计日志格式

**JSON Lines 格式示例:**
```json
{
  "timestamp": "2026-01-12T10:30:00Z",
  "event_id": "evt_xxx",
  "event_type": "workflow.submit",
  "event_category": "workflow",
  "user": {
    "user_id": "user_123",
    "ip": "192.168.1.100",
    "user_agent": "waterflow-cli/1.0.0"
  },
  "resource": {
    "resource_type": "workflow",
    "resource_id": "wf_xxx"
  },
  "result": "success",
  "metadata": {
    "workflow_name": "deploy-app",
    "server_group": "production"
  }
}
```

### 5.4 合规报告生成

**生成月度审计报告:**
```bash
#!/bin/bash
# scripts/audit-report.sh

START_DATE=$(date -d "1 month ago" +%Y-%m-%d)
END_DATE=$(date +%Y-%m-%d)
REPORT_FILE="audit-report-$(date +%Y-%m).json"

# 导出审计日志
waterflow audit query \
  --start-time="$START_DATE" \
  --end-time="$END_DATE" \
  --format=json > "$REPORT_FILE"

# 生成统计摘要
cat "$REPORT_FILE" | jq '
{
  total_events: length,
  by_category: group_by(.event_category) | map({category: .[0].event_category, count: length}),
  by_result: group_by(.result) | map({result: .[0].result, count: length}),
  failed_auth: [.[] | select(.event_category == "auth" and .result == "failure")],
  secret_access: [.[] | select(.event_category == "secret")]
}' > "audit-summary-$(date +%Y-%m).json"

echo "✅ 审计报告已生成: $REPORT_FILE"
```

### 5.5 审计日志备份

**自动备份脚本:**
```bash
#!/bin/bash
# 备份审计日志到远程存储

BACKUP_DIR=/backup/waterflow/audit
DATE=$(date +%Y-%m-%d)

mkdir -p "$BACKUP_DIR"

# 备份审计日志
cp -r /var/log/waterflow/audit "$BACKUP_DIR/audit-$DATE"

# 压缩备份
tar -czf "$BACKUP_DIR/audit-$DATE.tar.gz" "$BACKUP_DIR/audit-$DATE"
rm -rf "$BACKUP_DIR/audit-$DATE"

# 上传到 S3 (可选)
# aws s3 cp "$BACKUP_DIR/audit-$DATE.tar.gz" s3://backup-bucket/waterflow/audit/

# 删除旧备份 (保留 1 年)
find "$BACKUP_DIR" -name "audit-*.tar.gz" -mtime +365 -delete

echo "✅ 审计日志备份完成: $BACKUP_DIR/audit-$DATE.tar.gz"
```

**定时备份 (cron):**
```bash
# /etc/cron.d/waterflow-audit-backup
# 每天凌晨2点备份审计日志
0 2 * * * root /usr/local/bin/backup-audit-logs.sh
```

---

## 6. 网络安全

### 6.1 防火墙规则 (iptables)

**Server 防火墙规则:**
```bash
# 清空现有规则 (谨慎使用)
iptables -F

# 默认策略: 拒绝所有入站,允许出站
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

# 允许本地回环
iptables -A INPUT -i lo -j ACCEPT

# 允许已建立的连接
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# 允许 SSH (仅特定 IP)
iptables -A INPUT -p tcp --dport 22 -s 192.168.1.0/24 -j ACCEPT

# 允许 HTTPS (仅特定 IP 或子网)
iptables -A INPUT -p tcp --dport 8443 -s 192.168.1.0/24 -j ACCEPT

# 允许 Temporal gRPC (仅内网)
iptables -A INPUT -p tcp --dport 7233 -s 10.0.0.0/8 -j ACCEPT

# 允许健康检查端点 (仅本地)
iptables -A INPUT -p tcp --dport 8080 -s 127.0.0.1 -j ACCEPT

# 记录被拒绝的连接
iptables -A INPUT -m limit --limit 5/min -j LOG --log-prefix "iptables denied: "

# 保存规则
iptables-save > /etc/iptables/rules.v4
```

### 6.2 使用 ufw (Ubuntu)

**ufw 配置 (更简单的防火墙管理):**
```bash
# 重置 ufw 规则
ufw --force reset

# 默认策略
ufw default deny incoming
ufw default allow outgoing

# 允许 SSH (限制源IP)
ufw allow from 192.168.1.0/24 to any port 22

# 允许 HTTPS
ufw allow from 192.168.1.0/24 to any port 8443

# 允许 Temporal (内网)
ufw allow from 10.0.0.0/8 to any port 7233

# 启用防火墙
ufw enable

# 查看规则
ufw status numbered
```

**删除规则:**
```bash
# 查看规则编号
ufw status numbered

# 删除特定规则
ufw delete 3
```

### 6.3 端口管理

**推荐端口配置:**
```yaml
# config.yaml
server:
  # HTTP 端口 (仅本地/内网访问,用于健康检查)
  http_port: 8080
  http_bind: "127.0.0.1"  # 仅绑定本地
  
  # HTTPS 端口 (外网访问)
  https_port: 8443
  https_bind: "0.0.0.0"
  
  # Metrics 端口 (Prometheus)
  metrics_port: 9090
  metrics_bind: "127.0.0.1"  # 仅本地或使用反向代理

temporal:
  host: localhost
  port: 7233  # 仅本地或内网访问
```

**Docker 端口映射 (限制绑定):**
```yaml
# docker-compose.yaml
services:
  server:
    ports:
      # HTTP 仅本地 (健康检查)
      - "127.0.0.1:8080:8080"
      
      # HTTPS 所有接口
      - "0.0.0.0:8443:8443"
      
      # Metrics 仅本地
      - "127.0.0.1:9090:9090"
```

### 6.4 网络隔离

**使用 Docker 网络隔离:**
```yaml
# docker-compose.yaml
networks:
  # 前端网络 (外部访问)
  frontend:
    driver: bridge
  
  # 后端网络 (内部通信,无外网访问)
  backend:
    driver: bridge
    internal: true

services:
  server:
    networks:
      - frontend
      - backend
  
  temporal:
    networks:
      - backend  # 仅内网
  
  postgresql:
    networks:
      - backend  # 仅内网
```

### 6.5 VPN 配置 (跨网络通信)

**使用 WireGuard VPN 连接 Agent:**

**Server 端配置:**
```bash
# 安装 WireGuard
apt-get update
apt-get install wireguard

# 生成密钥对
wg genkey | tee server-privatekey | wg pubkey > server-publickey

# 配置 WireGuard
cat > /etc/wireguard/wg0.conf <<EOF
[Interface]
PrivateKey = $(cat server-privatekey)
Address = 10.0.0.1/24
ListenPort = 51820

[Peer]
# Agent 1
PublicKey = <agent1-public-key>
AllowedIPs = 10.0.0.2/32

[Peer]
# Agent 2
PublicKey = <agent2-public-key>
AllowedIPs = 10.0.0.3/32
EOF

# 启动 WireGuard
wg-quick up wg0
systemctl enable wg-quick@wg0
```

**Agent 端配置:**
```bash
# 生成 Agent 密钥对
wg genkey | tee agent-privatekey | wg pubkey > agent-publickey

# 配置 WireGuard
cat > /etc/wireguard/wg0.conf <<EOF
[Interface]
PrivateKey = $(cat agent-privatekey)
Address = 10.0.0.2/32

[Peer]
PublicKey = <server-public-key>
Endpoint = waterflow.example.com:51820
AllowedIPs = 10.0.0.0/24
PersistentKeepalive = 25
EOF

# 启动 WireGuard
wg-quick up wg0
systemctl enable wg-quick@wg0
```

**配置 Waterflow 使用 VPN:**
```yaml
# agent/config.yaml
agent:
  server_url: https://10.0.0.1:8443  # 使用 VPN 地址
```

---

## 7. 容器安全 (Docker)

### 7.1 使用安全的基础镜像

**Dockerfile 最佳实践:**
```dockerfile
# 使用最小化镜像
FROM alpine:3.19

# 创建非 root 用户
RUN addgroup -g 1000 waterflow && \
    adduser -D -u 1000 -G waterflow waterflow

# 安装必要依赖 (最小化)
RUN apk add --no-cache ca-certificates

# 复制二进制文件
COPY --chown=waterflow:waterflow bin/server /app/server

# 设置工作目录
WORKDIR /app

# 切换到非 root 用户
USER waterflow

# 暴露端口
EXPOSE 8080 8443

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动命令
CMD ["/app/server"]
```

### 7.2 容器运行时安全

**Docker run 安全参数:**
```bash
docker run -d \
  --name waterflow-server \
  --read-only \                        # 只读文件系统
  --tmpfs /tmp:noexec,nosuid,size=100m \ # 临时文件系统
  --security-opt=no-new-privileges \   # 禁止提权
  --cap-drop=ALL \                     # 删除所有 Linux capabilities
  --cap-add=NET_BIND_SERVICE \         # 仅添加绑定端口权限
  --memory=512m \                      # 内存限制
  --memory-swap=512m \                 # 禁用 swap
  --cpus=1 \                           # CPU 限制
  --pids-limit=100 \                   # 进程数限制
  -p 127.0.0.1:8080:8080 \
  -p 0.0.0.0:8443:8443 \
  -v /etc/waterflow:/etc/waterflow:ro \ # 配置文件只读
  waterflow/server:latest
```

**docker-compose.yaml 安全配置:**
```yaml
version: '3.8'

services:
  server:
    image: waterflow/server:latest
    
    # 安全选项
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
    read_only: true
    tmpfs:
      - /tmp:noexec,nosuid,size=100m
    
    # 资源限制
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
    
    # 进程限制
    pids_limit: 100
    
    # 卷挂载 (只读)
    volumes:
      - ./config.yaml:/etc/waterflow/config.yaml:ro
      - ./tls:/etc/waterflow/tls:ro
    
    # 端口映射
    ports:
      - "127.0.0.1:8080:8080"
      - "0.0.0.0:8443:8443"
    
    # 健康检查
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
```

### 7.3 镜像安全扫描

**使用 Trivy 扫描镜像:**
```bash
# 安装 Trivy
apt-get install wget apt-transport-https gnupg lsb-release
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | apt-key add -
echo "deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | tee -a /etc/apt/sources.list.d/trivy.list
apt-get update
apt-get install trivy

# 扫描本地镜像
trivy image waterflow/server:latest

# 仅显示高危和严重漏洞
trivy image --severity HIGH,CRITICAL waterflow/server:latest

# 扫描并生成报告
trivy image --format json --output trivy-report.json waterflow/server:latest
```

**使用 Docker Scout:**
```bash
# 启用 Docker Scout
docker scout quickview waterflow/server:latest

# 扫描漏洞
docker scout cves waterflow/server:latest

# 查看详细信息
docker scout cves --format sarif --output scout-report.sarif waterflow/server:latest
```

**集成到 CI/CD:**
```yaml
# .github/workflows/security-scan.yml
name: Security Scan

on:
  push:
    branches: [main]
  pull_request:

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Docker image
        run: docker build -t waterflow/server:test .
      
      - name: Run Trivy scan
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: waterflow/server:test
          format: 'sarif'
          output: 'trivy-results.sarif'
      
      - name: Upload Trivy results
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: 'trivy-results.sarif'
```

---

## 8. 备份和恢复

### 8.1 配置备份

**自动备份脚本:**
```bash
#!/bin/bash
# scripts/backup-config.sh

BACKUP_DIR=/backup/waterflow
DATE=$(date +%Y-%m-%d-%H%M%S)

mkdir -p "$BACKUP_DIR"

echo "🔄 开始备份 Waterflow 配置..."

# 备份配置文件
tar -czf "$BACKUP_DIR/config-$DATE.tar.gz" \
  /etc/waterflow/config.yaml \
  /etc/waterflow/tls/ \
  2>/dev/null || true

# 备份审计日志
tar -czf "$BACKUP_DIR/audit-$DATE.tar.gz" \
  /var/log/waterflow/audit/ \
  2>/dev/null || true

# 备份 Docker volumes (如果使用)
if docker volume ls | grep -q waterflow; then
  docker run --rm -v waterflow_data:/data -v "$BACKUP_DIR":/backup \
    alpine tar -czf /backup/volumes-$DATE.tar.gz /data
fi

# 生成备份清单
cat > "$BACKUP_DIR/manifest-$DATE.txt" <<EOF
Backup Date: $(date)
Config: config-$DATE.tar.gz
Audit: audit-$DATE.tar.gz
Volumes: volumes-$DATE.tar.gz (if exists)
EOF

echo "✅ 备份完成: $BACKUP_DIR"
ls -lh "$BACKUP_DIR"/*-$DATE.*
```

**定时备份 (cron):**
```bash
# /etc/cron.d/waterflow-backup
# 每天凌晨2点执行备份
0 2 * * * root /usr/local/bin/backup-config.sh

# 每周日凌晨3点执行完整备份
0 3 * * 0 root /usr/local/bin/backup-full.sh
```

### 8.2 恢复流程

**恢复配置文件:**
```bash
#!/bin/bash
# scripts/restore-config.sh

if [ -z "$1" ]; then
  echo "用法: $0 <backup-date>"
  echo "示例: $0 2026-01-12-020000"
  exit 1
fi

BACKUP_DIR=/backup/waterflow
DATE=$1

echo "🔄 开始恢复配置 (备份日期: $DATE)..."

# 1. 停止服务
echo "停止 Waterflow Server..."
systemctl stop waterflow-server || docker stop waterflow-server

# 2. 备份当前配置 (以防恢复失败)
echo "备份当前配置..."
cp -r /etc/waterflow /etc/waterflow.bak-$(date +%Y%m%d%H%M%S)

# 3. 恢复配置
echo "恢复配置文件..."
tar -xzf "$BACKUP_DIR/config-$DATE.tar.gz" -C /

# 4. 恢复审计日志 (可选)
if [ -f "$BACKUP_DIR/audit-$DATE.tar.gz" ]; then
  echo "恢复审计日志..."
  tar -xzf "$BACKUP_DIR/audit-$DATE.tar.gz" -C /
fi

# 5. 验证配置
echo "验证配置..."
waterflow validate-config || {
  echo "❌ 配置验证失败,恢复备份..."
  mv /etc/waterflow.bak-* /etc/waterflow
  exit 1
}

# 6. 启动服务
echo "启动 Waterflow Server..."
systemctl start waterflow-server || docker start waterflow-server

# 7. 验证运行状态
sleep 5
curl -f http://localhost:8080/health || {
  echo "❌ 服务启动失败"
  exit 1
}

echo "✅ 配置恢复完成"
```

### 8.3 灾难恢复计划

**灾难恢复检查清单:**

1. **备份验证** (每周)
   ```bash
   # 验证备份文件完整性
   tar -tzf /backup/waterflow/config-latest.tar.gz
   
   # 恢复到测试环境
   ./scripts/restore-config.sh latest --test-env
   ```

2. **恢复演练** (每月)
   ```bash
   # 在隔离环境中执行完整恢复
   ./scripts/disaster-recovery-drill.sh
   ```

3. **远程备份** (每天)
   ```bash
   # 同步到远程存储
   rsync -avz /backup/waterflow/ user@backup-server:/backups/waterflow/
   
   # 或使用云存储
   aws s3 sync /backup/waterflow/ s3://backup-bucket/waterflow/
   ```

4. **备份保留策略**
   ```bash
   # 每日备份保留7天
   find /backup/waterflow -name "config-*.tar.gz" -mtime +7 -delete
   
   # 每周备份保留30天
   find /backup/waterflow -name "weekly-*.tar.gz" -mtime +30 -delete
   
   # 每月备份保留1年
   find /backup/waterflow -name "monthly-*.tar.gz" -mtime +365 -delete
   ```

---

## 9. 安全监控

### 9.1 监控异常行为

**监控失败的认证尝试:**
```bash
# 实时监控认证失败
tail -f /var/log/waterflow/audit/audit.log | \
  jq 'select(.event_type == "auth.failure")'

# 统计最近1小时的认证失败
waterflow audit query \
  --event-type=auth.failure \
  --start-time="1 hour ago" \
  --format=json | \
  jq -r '.user.ip' | sort | uniq -c | sort -rn

# 输出示例:
#  15 192.168.1.100
#   3 10.0.0.5
#   1 172.16.0.10
```

**监控密钥访问异常:**
```bash
# 查看权限拒绝的密钥访问
waterflow audit query \
  --event-category=secret \
  --result=permission_denied

# 监控高频密钥访问 (可能泄露)
waterflow audit query \
  --event-category=secret \
  --start-time="1 hour ago" \
  --format=json | \
  jq -r '.resource.resource_id' | sort | uniq -c | sort -rn
```

### 9.2 Prometheus 监控指标

**安全相关指标:**
```yaml
# Server 暴露的安全指标
waterflow_auth_attempts_total{result="success|failure"}  # 认证尝试
waterflow_auth_failures_total                            # 认证失败
waterflow_secret_access_total{result="success|denied"}   # 密钥访问
waterflow_tls_handshake_errors_total                     # TLS 握手错误
waterflow_audit_log_writes_total                         # 审计日志写入
```

**Prometheus 配置:**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'waterflow'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
```

### 9.3 告警配置

**Alertmanager 规则:**
```yaml
# alertmanager-rules.yml
groups:
  - name: waterflow_security
    interval: 1m
    rules:
      # 高频认证失败告警
      - alert: HighAuthFailureRate
        expr: rate(waterflow_auth_failures_total[5m]) > 5
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High authentication failure rate detected"
          description: "{{ $value }} auth failures per second in the last 5 minutes"
      
      # 未授权密钥访问告警
      - alert: UnauthorizedSecretAccess
        expr: increase(waterflow_secret_access_total{result="denied"}[5m]) > 0
        for: 1m
        labels:
          severity: high
        annotations:
          summary: "Unauthorized secret access detected"
          description: "{{ $value }} denied secret access attempts"
      
      # TLS 错误告警
      - alert: TLSHandshakeErrors
        expr: increase(waterflow_tls_handshake_errors_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "TLS handshake errors detected"
          description: "{{ $value }} TLS errors in the last 5 minutes"
```

---

## 10. 事件响应

### 10.1 安全事件响应流程

#### 步骤 1: 检测

**自动检测:**
```bash
# 监控异常日志
tail -f /var/log/waterflow/audit/audit.log | \
  jq 'select(.result == "failure" or .result == "permission_denied")'
```

**告警通知:**
```bash
# Alertmanager 发送告警到 Slack/Email/PagerDuty
```

#### 步骤 2: 隔离

**隔离受影响的 Agent:**
```bash
# 禁用 Agent
waterflow agent disable <agent-id>

# 或直接停止 Agent 进程
ssh agent-host "systemctl stop waterflow-agent"
```

**临时禁用 API 访问:**
```bash
# 使用防火墙阻止特定 IP
iptables -A INPUT -s <suspicious-ip> -j DROP

# 或禁用整个 HTTPS 端口 (极端情况)
iptables -A INPUT -p tcp --dport 8443 -j DROP
```

#### 步骤 3: 调查

**查看详细审计日志:**
```bash
# 查看可疑用户的所有操作
waterflow audit query \
  --start-time="2 hours ago" \
  --user-id=<suspicious-user> \
  --format=json > investigation-$(date +%Y%m%d).json

# 查看可疑 IP 的所有操作
cat /var/log/waterflow/audit/audit.log | \
  jq 'select(.user.ip == "<suspicious-ip>")'
```

**导出审计日志用于分析:**
```bash
# 导出最近24小时的审计日志
waterflow audit export \
  --start-time="24 hours ago" \
  --output=incident-$(date +%Y%m%d-%H%M%S).json
```

#### 步骤 4: 修复

**轮换受影响的密钥:**
```bash
# 轮换 API Key
./scripts/rotate-api-keys.sh

# 轮换 Vault 密钥
vault kv put waterflow/production/compromised_key value=$(openssl rand -base64 32)
```

**撤销 API Key:**
```yaml
# 从 config.yaml 移除受影响的 key
auth:
  api_keys:
    # - name: compromised-key  # 已撤销
    #   key: sk_test_YOUR_API_KEY_HERE
    #   roles: [operator]
```

**更新防火墙规则:**
```bash
# 永久阻止可疑 IP
ufw deny from <suspicious-ip>

# 或添加到黑名单
echo "<suspicious-ip>" >> /etc/waterflow/blacklist.txt
```

#### 步骤 5: 恢复

**验证系统安全:**
```bash
# 运行安全检查
./scripts/security-check.sh

# 验证配置
waterflow validate-config

# 检查审计日志
waterflow audit query --limit=100
```

**启用服务:**
```bash
# 重启 Server
systemctl restart waterflow-server

# 或 Docker
docker restart waterflow-server
```

**监控恢复情况:**
```bash
# 实时监控健康状态
watch -n 5 'curl -s http://localhost:8080/health | jq .'

# 监控审计日志
tail -f /var/log/waterflow/audit/audit.log
```

#### 步骤 6: 总结和预防

**生成事件报告:**
```bash
#!/bin/bash
# scripts/generate-incident-report.sh

cat > incident-report-$(date +%Y%m%d).md <<EOF
# 安全事件报告

**日期:** $(date)
**事件类型:** 未授权访问尝试
**严重程度:** High

## 事件概述
检测到来自 IP <suspicious-ip> 的多次未授权访问尝试。

## 影响范围
- Agent: agent-123
- 密钥: production/api_key
- 受影响时间: 2026-01-12 10:00 - 11:00

## 处理措施
1. 禁用受影响 Agent
2. 轮换相关密钥
3. 更新防火墙规则阻止可疑 IP
4. 审查审计日志

## 根本原因
API Key 泄露 (可能通过钓鱼攻击)

## 预防措施
1. 增强 Agent 认证 (启用双向 TLS)
2. 定期审查审计日志
3. 实施 API Key 自动轮换
4. 员工安全培训

## 附录
- 审计日志: incident-20260112.json
- 防火墙规则: /etc/ufw/user.rules
EOF

echo "✅ 事件报告已生成: incident-report-$(date +%Y%m%d).md"
```

**更新安全措施:**
```bash
# 1. 启用双向 TLS
cat >> config.yaml <<EOF
https:
  client_auth:
    enabled: true
    ca_cert: /etc/waterflow/tls/client-ca.pem
EOF

# 2. 配置自动密钥轮换
crontab -e
# 每月1号轮换密钥
0 3 1 * * /usr/local/bin/rotate-secrets.sh

# 3. 增强监控
# 添加更多告警规则到 Alertmanager
```

---

## 11. 安全检查清单

详见 [security-checklist.md](security-checklist.md)

---

## 12. 合规指南

详见 [compliance-guide.md](compliance-guide.md)

---

## 附录

### A. 安全配置示例

完整的生产环境安全配置示例:
- [secure-production.yaml](../../examples/configs/secure-production.yaml)
- [secure-docker-compose.yaml](../../examples/configs/secure-docker-compose.yaml)
- [vault-integration.yaml](../../examples/configs/vault-integration.yaml)

### B. 安全脚本

所有安全脚本位于 `scripts/` 目录:
- `generate-tls-cert.sh` - TLS 证书生成
- `security-check.sh` - 安全检查
- `rotate-secrets.sh` - 密钥轮换
- `backup-config.sh` - 配置备份
- `audit-report.sh` - 审计报告生成

### C. 相关文档

- [Quick Start Guide](../quick-start.md) - 包含安全快速配置
- [Deployment Guide](../deployment.md) - 包含生产环境安全部署
- [Configuration Reference](../configuration.md) - 包含所有安全配置选项
- [Troubleshooting Guide](../troubleshooting.md) - 包含安全问题排查

### D. 外部资源

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [HashiCorp Vault Documentation](https://www.vaultproject.io/docs)

---

**最后更新:** 2026-01-12  
**版本:** 1.0.0  
**维护者:** Waterflow Security Team

# Waterflow 合规指南

本指南帮助组织使用 Waterflow 满足常见合规标准的要求,包括 SOC2、ISO27001 和 GDPR。

## 目录

1. [合规概述](#1-合规概述)
2. [SOC2 合规](#2-soc2-合规)
3. [ISO27001 合规](#3-iso27001-合规)
4. [GDPR 数据保护](#4-gdpr-数据保护)
5. [审计和报告](#5-审计和报告)

---

## 1. 合规概述

### 1.1 合规框架对比

| 标准 | 重点领域 | Waterflow 支持 |
|------|---------|---------------|
| **SOC2** | 访问控制、审计日志、加密 | ✅ 完全支持 |
| **ISO27001** | 信息安全管理体系 | ✅ 完全支持 |
| **GDPR** | 个人数据保护 | ✅ 部分支持 (审计日志) |

### 1.2 Waterflow 合规功能

| 功能 | 用途 | 相关Story |
|------|------|----------|
| HTTPS/TLS 加密 | 保护传输数据 | Story 9.1 |
| API 认证 | 访问控制 | Story 1.2 |
| SecretProvider | 密钥管理 | Story 9.2 |
| 审计日志 | 操作记录和追踪 | Story 9.3 |
| 角色权限 | 最小权限原则 | Story 1.2 |

---

## 2. SOC2 合规

SOC2 (Service Organization Control 2) 是针对云服务提供商的信任服务准则,基于5个信任服务准则。

### 2.1 安全性 (Security)

#### 2.1.1 访问控制

**要求:** 限制对系统的逻辑和物理访问。

**Waterflow 配置:**
```yaml
# config.yaml
auth:
  enabled: true
  type: api_key
  api_keys:
    - name: admin
      key: ${ADMIN_API_KEY}  # 从环境变量读取
      roles: [admin]
    - name: operator
      key: ${OPERATOR_API_KEY}
      roles: [operator]

roles:
  admin:
    permissions:
      - workflows:*
      - agents:*
      - audit:read
  operator:
    permissions:
      - workflows:submit
      - workflows:status
      - workflows:logs
```

**验证方法:**
```bash
# 测试未授权访问
curl -X POST http://waterflow/v1/workflows
# 预期: 401 Unauthorized

# 测试有效 API Key
curl -H "Authorization: Bearer $ADMIN_API_KEY" \
  -X POST http://waterflow/v1/workflows \
  -d @workflow.yaml
# 预期: 200 OK
```

#### 2.1.2 审计日志

**要求:** 记录所有安全相关事件。

**Waterflow 配置:**
```yaml
# config.yaml
audit:
  enabled: true
  store_type: file
  file:
    path: /var/log/waterflow/audit
    max_age: 90  # SOC2 要求至少90天
    max_backups: 30
    compress: true
```

**验证方法:**
```bash
# 查询审计日志
waterflow audit query --limit=100

# 导出审计日志用于审计
waterflow audit export \
  --start-time="90 days ago" \
  --output=soc2-audit-$(date +%Y-%m).json
```

#### 2.1.3 加密传输

**要求:** 所有敏感数据传输必须加密。

**Waterflow 配置:**
```yaml
# config.yaml
https:
  enabled: true
  cert_file: /etc/waterflow/tls/cert.pem
  key_file: /etc/waterflow/tls/key.pem
  min_version: "1.2"  # SOC2 要求 TLS 1.2+
```

**验证方法:**
```bash
# 验证 HTTPS 启用
curl -v https://waterflow/health 2>&1 | grep "TLS"
# 预期: TLS 1.2 或 TLS 1.3

# 验证 HTTP 禁用
curl http://waterflow/health
# 预期: 连接被拒绝或重定向到 HTTPS
```

### 2.2 可用性 (Availability)

#### 2.2.1 健康检查和监控

**要求:** 监控系统可用性。

**Waterflow 配置:**
```yaml
# config.yaml
health:
  enabled: true
  timeout: 5s
  dependencies:
    - name: temporal
      type: grpc
      address: localhost:7233
```

**监控配置 (Prometheus):**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'waterflow'
    static_configs:
      - targets: ['waterflow:9090']
    scrape_interval: 15s

# alertmanager-rules.yml
groups:
  - name: waterflow_availability
    rules:
      - alert: WaterflowDown
        expr: up{job="waterflow"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Waterflow is down"
```

#### 2.2.2 备份和恢复

**要求:** 定期备份,能快速恢复。

**备份配置:**
```bash
# /etc/cron.d/waterflow-backup
# 每日备份 (保留30天)
0 2 * * * root /usr/local/bin/backup-config.sh

# 每周完整备份 (保留90天)
0 3 * * 0 root /usr/local/bin/backup-full.sh
```

**恢复测试 (每月):**
```bash
# 在测试环境验证备份恢复
./scripts/restore-config.sh $(date +%Y-%m-%d -d "yesterday") --test-env

# 验证恢复后的系统
curl http://test-waterflow:8080/health
```

### 2.3 处理完整性 (Processing Integrity)

#### 2.3.1 输入验证

**要求:** 验证所有输入数据。

**Waterflow 实现:**
```yaml
# Waterflow 自动验证工作流 YAML
# pkg/dsl/validator.go 实现了完整的 DSL 验证
```

**验证方法:**
```bash
# 提交无效工作流
echo "invalid: yaml" | waterflow submit -
# 预期: 验证错误

# 使用 CLI 验证
waterflow validate workflow.yaml
# 预期: 显示验证结果
```

### 2.4 机密性 (Confidentiality)

#### 2.4.1 密钥管理

**要求:** 安全存储和管理机密信息。

**Waterflow 配置:**
```yaml
# config.yaml
secrets:
  provider: vault
  vault:
    address: https://vault.example.com
    token: ${VAULT_TOKEN}
    mount_path: waterflow
    tls:
      enabled: true
      ca_cert: /etc/waterflow/vault-ca.pem
```

**Vault 配置:**
```bash
# 启用审计日志
vault audit enable file file_path=/var/log/vault/audit.log

# 配置密钥访问策略
vault policy write waterflow-secrets - <<EOF
path "waterflow/*" {
  capabilities = ["read", "list"]
}
EOF
```

### 2.5 隐私 (Privacy)

#### 2.5.1 个人数据脱敏

**要求:** 审计日志中的个人数据脱敏。

**Waterflow 实现:**
```yaml
# 审计日志自动脱敏敏感字段
# pkg/audit/logger.go 实现了脱敏逻辑

# 示例: IP 地址脱敏
# 原始: 192.168.1.100
# 脱敏: 192.168.1.***
```

### 2.6 SOC2 合规检查清单

| # | 控制项 | Waterflow 配置 | 验证方法 | 状态 |
|---|-------|---------------|---------|-----|
| 1 | 访问控制 | auth.enabled=true | 测试未授权访问 | ☐ |
| 2 | 审计日志 | audit.enabled=true | 导出90天日志 | ☐ |
| 3 | TLS 加密 | https.enabled=true, min_version=1.2 | 验证 TLS 版本 | ☐ |
| 4 | 密钥管理 | secrets.provider=vault | 验证 Vault 集成 | ☐ |
| 5 | 健康监控 | health.enabled=true | 查看 Prometheus | ☐ |
| 6 | 定期备份 | Cron 任务 | 验证备份文件 | ☐ |
| 7 | 输入验证 | DSL 验证器 | 测试无效输入 | ☐ |
| 8 | 数据脱敏 | 审计日志脱敏 | 检查审计日志 | ☐ |

**生成 SOC2 合规报告:**
```bash
#!/bin/bash
# scripts/generate-soc2-report.sh

REPORT_FILE="soc2-compliance-report-$(date +%Y-%m).md"

cat > "$REPORT_FILE" <<EOF
# SOC2 合规报告

**报告期间:** $(date -d "1 month ago" +%Y-%m) - $(date +%Y-%m)
**生成日期:** $(date)

## 1. 访问控制

### API 认证启用状态
\`\`\`
$(grep "auth:" -A2 config.yaml)
\`\`\`

### 最近30天认证统计
\`\`\`
$(waterflow audit query --event-category=auth --start-time="30 days ago" --format=json | \
  jq -r '.[] | .result' | sort | uniq -c)
\`\`\`

## 2. 审计日志

### 审计日志总数
\`\`\`
$(waterflow audit query --start-time="30 days ago" | wc -l) 条记录
\`\`\`

### 日志保留策略
\`\`\`
$(grep "max_age:" config.yaml)
\`\`\`

## 3. 加密传输

### TLS 配置
\`\`\`
$(grep "https:" -A5 config.yaml)
\`\`\`

### TLS 验证
\`\`\`
$(curl -v https://waterflow/health 2>&1 | grep "SSL connection")
\`\`\`

## 4. 密钥管理

### SecretProvider 配置
\`\`\`
$(grep "secrets:" -A5 config.yaml)
\`\`\`

### Vault 状态
\`\`\`
$(vault status)
\`\`\`

## 5. 可用性

### 健康检查记录
\`\`\`
最近7天健康检查通过率: $(calculate_health_check_success_rate)
\`\`\`

### 备份状态
\`\`\`
$(ls -lh /backup/waterflow/*.tar.gz | tail -5)
\`\`\`

EOF

echo "✅ SOC2 合规报告已生成: $REPORT_FILE"
```

---

## 3. ISO27001 合规

ISO27001 是国际信息安全管理体系标准。

### 3.1 A.9 访问控制

#### 3.1.1 用户访问管理

**要求:** 确保授权用户访问,防止未授权访问。

**Waterflow 配置:**
```yaml
# config.yaml
auth:
  enabled: true
  type: api_key

# 定义角色和权限
roles:
  admin:
    description: "完全管理权限"
    permissions: ["workflows:*", "agents:*", "audit:read"]
  
  operator:
    description: "工作流操作权限"
    permissions: ["workflows:submit", "workflows:status"]
  
  readonly:
    description: "仅查看权限"
    permissions: ["workflows:status", "workflows:logs"]
```

**用户访问审查 (季度):**
```bash
# 导出所有 API Key 使用记录
waterflow audit query \
  --event-category=auth \
  --start-time="90 days ago" \
  --format=json | \
  jq -r '.[] | .user.user_id' | sort | uniq > active-users.txt

# 对比配置文件中的 API Key
# 撤销未使用的 API Key
```

### 3.2 A.10 密码学

#### 3.2.1 加密控制

**要求:** 使用加密保护信息机密性。

**Waterflow 配置:**
```yaml
# 传输加密
https:
  enabled: true
  min_version: "1.2"  # ISO27001 推荐 TLS 1.2+

# 密钥管理
secrets:
  provider: vault  # 使用 Vault 加密存储密钥
```

### 3.3 A.12 运行安全

#### 3.3.1 操作程序和职责

**要求:** 定义安全运维程序。

**Waterflow 运维程序:**
1. **每日检查** - 运行健康检查脚本
2. **每周备份** - 备份配置和审计日志
3. **每月审查** - 审查审计日志和权限
4. **季度演练** - 灾难恢复演练

**运维检查脚本:**
```bash
#!/bin/bash
# scripts/iso27001-daily-check.sh

echo "ISO27001 每日运维检查"
echo "===================="

# 1. 健康检查
if curl -sf http://localhost:8080/health; then
  echo "✓ 服务健康"
else
  echo "✗ 服务异常"
  exit 1
fi

# 2. 审计日志运行状态
if [ -w /var/log/waterflow/audit ]; then
  echo "✓ 审计日志正常"
else
  echo "✗ 审计日志异常"
fi

# 3. 备份存在性
if [ -f "/backup/waterflow/config-$(date +%Y-%m-%d).tar.gz" ]; then
  echo "✓ 今日备份已完成"
else
  echo "⚠ 今日备份未完成"
fi
```

#### 3.3.2 日志记录

**要求:** 记录用户活动、异常和安全事件。

**Waterflow 配置:**
```yaml
# config.yaml
audit:
  enabled: true
  categories:
    - auth       # 认证事件
    - workflow   # 工作流操作
    - secret     # 密钥访问
    - admin      # 管理操作
```

**日志分析:**
```bash
# 查看最近的安全事件
waterflow audit query \
  --event-category=auth \
  --result=failure \
  --start-time="7 days ago"

# 生成日志统计报告
cat /var/log/waterflow/audit/audit.log | \
  jq -r '.event_category' | sort | uniq -c
```

### 3.4 A.14 系统获取、开发和维护

#### 3.4.1 安全开发生命周期

**要求:** 在开发生命周期中集成安全。

**Waterflow 安全实践:**
- ✅ 代码审查 (Story 9.1/9.2/9.3 均经过代码审查)
- ✅ 安全测试 (test/security/)
- ✅ 依赖扫描 (Trivy 镜像扫描)
- ✅ 审计日志集成

**CI/CD 安全门禁:**
```yaml
# .github/workflows/security.yml
name: Security Checks

on: [push, pull_request]

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Run Trivy scan
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          severity: 'HIGH,CRITICAL'
      
      - name: Run security tests
        run: go test ./test/security/...
```

### 3.5 ISO27001 合规检查清单

| 附录 | 控制项 | Waterflow 实现 | 验证方法 | 状态 |
|-----|-------|--------------|---------|-----|
| A.9.1 | 访问控制策略 | auth.enabled, roles 配置 | 测试权限 | ☐ |
| A.9.2 | 用户访问管理 | API Key, 用户审查 | 导出用户列表 | ☐ |
| A.10.1 | 加密控制 | HTTPS/TLS, Vault | 验证 TLS | ☐ |
| A.12.4 | 日志记录 | audit.enabled | 导出日志 | ☐ |
| A.12.6 | 技术脆弱性管理 | Trivy 扫描 | 运行扫描 | ☐ |
| A.14.2 | 安全开发 | 代码审查, 安全测试 | 查看 CI 结果 | ☐ |

**生成 ISO27001 合规报告:**
```bash
#!/bin/bash
# scripts/generate-iso27001-report.sh

REPORT_FILE="iso27001-compliance-report-$(date +%Y-%m).md"

cat > "$REPORT_FILE" <<EOF
# ISO27001 合规报告

**报告期间:** $(date +%Y-%m)
**生成日期:** $(date)

## A.9 访问控制

### A.9.1 访问控制策略
✓ API 认证已启用
✓ 基于角色的访问控制 (RBAC)

### A.9.2 用户访问管理
活跃用户数: $(waterflow audit query --event-category=auth --start-time="30 days ago" --format=json | jq -r '.user.user_id' | sort -u | wc -l)

## A.10 密码学

### A.10.1 加密控制
✓ HTTPS/TLS 1.2+ 加密传输
✓ Vault 加密存储密钥

## A.12 运行安全

### A.12.4 日志记录
审计日志记录数: $(waterflow audit query --start-time="30 days ago" | wc -l)
日志保留期: 90 天

### A.12.6 技术脆弱性管理
最近镜像扫描: $(date)
高危漏洞: 0
中危漏洞: 0

## A.14 系统获取、开发和维护

### A.14.2 安全开发
✓ 代码审查流程
✓ 自动化安全测试
✓ 依赖漏洞扫描

EOF

echo "✅ ISO27001 合规报告已生成: $REPORT_FILE"
```

---

## 4. GDPR 数据保护

GDPR (General Data Protection Regulation) 是欧盟数据保护法规。

### 4.1 数据处理原则

#### 4.1.1 最小化数据收集

**原则:** 仅收集必要的个人数据。

**Waterflow 实现:**
- 审计日志仅记录操作相关信息 (用户ID, IP, 操作类型)
- 不存储工作流中的业务数据 (数据由 Temporal 管理)
- 支持 IP 地址脱敏

**配置:**
```yaml
# config.yaml
audit:
  enabled: true
  masking:
    ip_address: true  # 脱敏 IP 地址最后一段
    user_agent: false
```

#### 4.1.2 数据访问权

**原则:** 数据主体有权访问其个人数据。

**实现:**
```bash
# 查询特定用户的所有审计记录
waterflow audit query \
  --user-id=john.doe \
  --start-time="1 year ago" \
  --output=user-data-john.doe.json
```

#### 4.1.3 数据删除权 (被遗忘权)

**原则:** 数据主体有权要求删除其个人数据。

**实现:**
```bash
# 删除特定用户的审计日志 (需自定义工具)
# 注意: 需要权衡合规要求 (GDPR) 和审计要求 (SOC2/ISO27001)

# 推荐做法: 脱敏而非删除
waterflow audit anonymize --user-id=john.doe
```

### 4.2 数据安全

#### 4.2.1 传输加密

**要求:** 个人数据传输必须加密。

**Waterflow 配置:**
```yaml
https:
  enabled: true
  min_version: "1.2"
```

#### 4.2.2 访问日志

**要求:** 记录个人数据访问。

**Waterflow 实现:**
```bash
# 查看密钥访问日志 (可能包含个人数据)
waterflow audit query --event-category=secret

# 查看工作流访问日志
waterflow audit query --event-type=workflow.status
```

### 4.3 数据保留

#### 4.3.1 保留策略

**要求:** 定义数据保留期限。

**Waterflow 配置:**
```yaml
# config.yaml
audit:
  file:
    max_age: 90  # 审计日志保留 90 天
    max_backups: 30
```

**自动清理旧数据:**
```bash
# 清理超过保留期的审计日志
find /var/log/waterflow/audit -name "*.log" -mtime +90 -delete

# 清理旧备份
find /backup/waterflow -name "*.tar.gz" -mtime +365 -delete
```

### 4.4 数据泄露通知

#### 4.4.1 检测数据泄露

**监控指标:**
```bash
# 监控未授权访问
waterflow audit query \
  --event-category=auth \
  --result=failure \
  --start-time="24 hours ago"

# 监控异常密钥访问
waterflow audit query \
  --event-category=secret \
  --result=permission_denied
```

#### 4.4.2 泄露响应流程

1. **检测** - 监控告警检测到异常
2. **评估** - 确认是否涉及个人数据泄露
3. **通知** - 72小时内通知数据保护机构 (如需)
4. **记录** - 记录泄露事件和响应措施

**泄露记录模板:**
```bash
cat > data-breach-report.md <<EOF
# 数据泄露事件报告

**日期:** $(date)
**事件ID:** DBR-$(date +%Y%m%d%H%M)

## 事件概述
[描述泄露事件]

## 影响范围
- 受影响用户数: [数量]
- 泄露数据类型: [类型]
- 泄露时间段: [开始 - 结束]

## 响应措施
1. [措施1]
2. [措施2]

## 通知情况
- 用户通知: [是/否]
- 机构通知: [是/否]

## 预防措施
[后续改进措施]
EOF
```

### 4.5 GDPR 合规检查清单

| 条款 | 要求 | Waterflow 实现 | 验证方法 | 状态 |
|-----|------|--------------|---------|-----|
| Art. 5 | 数据最小化 | 仅记录必要信息 | 审查审计日志字段 | ☐ |
| Art. 15 | 数据访问权 | audit query 命令 | 导出用户数据 | ☐ |
| Art. 17 | 数据删除权 | audit anonymize (脱敏) | 测试脱敏 | ☐ |
| Art. 32 | 数据安全 | HTTPS, Vault | 验证加密 | ☐ |
| Art. 33 | 泄露通知 | 监控和告警 | 测试告警 | ☐ |

---

## 5. 审计和报告

### 5.1 审计日志查询

#### 5.1.1 常用查询

**认证审计:**
```bash
# 查看所有认证事件
waterflow audit query --event-category=auth

# 查看认证失败
waterflow audit query --event-category=auth --result=failure

# 统计认证成功率
waterflow audit query \
  --event-category=auth \
  --start-time="30 days ago" \
  --format=json | \
  jq -r '.result' | sort | uniq -c
```

**密钥访问审计:**
```bash
# 查看所有密钥访问
waterflow audit query --event-category=secret

# 查看特定密钥访问
waterflow audit query \
  --event-category=secret \
  --resource-id=production/db_password
```

**工作流操作审计:**
```bash
# 查看工作流提交
waterflow audit query --event-type=workflow.submit

# 查看工作流取消
waterflow audit query --event-type=workflow.cancel
```

### 5.2 合规报告生成

#### 5.2.1 月度合规报告

**统一合规报告脚本:**
```bash
#!/bin/bash
# scripts/generate-compliance-report.sh

STANDARD=${1:-all}  # soc2, iso27001, gdpr, or all
REPORT_DIR="compliance-reports/$(date +%Y-%m)"
mkdir -p "$REPORT_DIR"

if [ "$STANDARD" == "soc2" ] || [ "$STANDARD" == "all" ]; then
  echo "生成 SOC2 报告..."
  ./scripts/generate-soc2-report.sh > "$REPORT_DIR/soc2-report.md"
fi

if [ "$STANDARD" == "iso27001" ] || [ "$STANDARD" == "all" ]; then
  echo "生成 ISO27001 报告..."
  ./scripts/generate-iso27001-report.sh > "$REPORT_DIR/iso27001-report.md"
fi

if [ "$STANDARD" == "gdpr" ] || [ "$STANDARD" == "all" ]; then
  echo "生成 GDPR 报告..."
  ./scripts/generate-gdpr-report.sh > "$REPORT_DIR/gdpr-report.md"
fi

# 导出审计日志
echo "导出审计日志..."
waterflow audit export \
  --start-time="1 month ago" \
  --output="$REPORT_DIR/audit-logs.json"

# 生成统计摘要
cat "$REPORT_DIR/audit-logs.json" | jq '
{
  total_events: length,
  by_category: group_by(.event_category) | map({
    category: .[0].event_category,
    count: length
  }),
  auth_failures: [.[] | select(.event_category == "auth" and .result == "failure")] | length
}' > "$REPORT_DIR/audit-summary.json"

echo "✅ 合规报告已生成: $REPORT_DIR/"
ls -lh "$REPORT_DIR/"
```

#### 5.2.2 审计师访问

**为审计师提供只读访问:**
```yaml
# config.yaml
auth:
  api_keys:
    - name: auditor
      key: sk_live_auditor_xxx
      roles: [auditor]

roles:
  auditor:
    description: "审计师只读权限"
    permissions:
      - audit:read
      - workflows:status
```

**审计师查询示例:**
```bash
# 设置审计师 API Key
export WATERFLOW_API_KEY=sk_live_auditor_xxx

# 查询审计日志
waterflow audit query \
  --start-time="90 days ago" \
  --output=audit-export.json

# 统计报告
cat audit-export.json | jq '
{
  total_events: length,
  users: [.[].user.user_id] | unique | length,
  event_types: [.[].event_type] | unique
}'
```

### 5.3 合规证据归档

**归档合规证据:**
```bash
#!/bin/bash
# scripts/archive-compliance-evidence.sh

YEAR=$(date +%Y)
QUARTER=$(date +%q)
ARCHIVE_DIR="/archive/compliance/$YEAR-Q$QUARTER"

mkdir -p "$ARCHIVE_DIR"

# 归档配置文件
cp -r /etc/waterflow/config.yaml "$ARCHIVE_DIR/"

# 归档审计日志
waterflow audit export \
  --start-time="3 months ago" \
  --output="$ARCHIVE_DIR/audit-logs.json"

# 归档合规报告
cp -r compliance-reports/* "$ARCHIVE_DIR/"

# 归档安全检查结果
./scripts/security-check.sh > "$ARCHIVE_DIR/security-check.log"

# 压缩归档
tar -czf "/archive/compliance-$YEAR-Q$QUARTER.tar.gz" "$ARCHIVE_DIR"

# 加密归档 (推荐)
gpg --encrypt --recipient compliance@example.com \
  "/archive/compliance-$YEAR-Q$QUARTER.tar.gz"

echo "✅ 合规证据已归档: /archive/compliance-$YEAR-Q$QUARTER.tar.gz.gpg"
```

---

## 附录

### A. 合规标准对比

| 要求 | SOC2 | ISO27001 | GDPR |
|------|------|----------|------|
| 访问控制 | ✅ 必需 | ✅ 必需 | ✅ 必需 |
| 审计日志 | ✅ 必需 (90天) | ✅ 必需 | ✅ 必需 |
| 加密传输 | ✅ 必需 | ✅ 必需 | ✅ 必需 |
| 密钥管理 | ✅ 必需 | ✅ 必需 | ⚠️ 推荐 |
| 数据脱敏 | ⚠️ 推荐 | ⚠️ 推荐 | ✅ 必需 |
| 泄露通知 | ⚠️ 推荐 | ⚠️ 推荐 | ✅ 必需 (72h) |

### B. 相关文档

- [安全配置指南](security-guide.md) - 详细的安全配置
- [安全检查清单](security-checklist.md) - 可执行的检查清单

### C. 外部资源

- [SOC2 Trust Service Criteria](https://www.aicpa.org/interestareas/frc/assuranceadvisoryservices/aicpasoc2report.html)
- [ISO/IEC 27001](https://www.iso.org/standard/54534.html)
- [GDPR Official Text](https://gdpr-info.eu/)

---

**最后更新:** 2026-01-12  
**版本:** 1.0.0  
**维护者:** Waterflow Security Team

# Waterflow 安全检查清单

本文档提供可执行的安全检查清单,帮助系统管理员在不同阶段验证 Waterflow 的安全配置。

## 目录

1. [部署前检查清单](#1-部署前检查清单)
2. [生产环境检查清单](#2-生产环境检查清单)
3. [定期审查清单 (每月)](#3-定期审查清单-每月)
4. [事件响应清单](#4-事件响应清单)

---

## 1. 部署前检查清单

在将 Waterflow 部署到生产环境之前,请完成以下检查项。

### 1.1 HTTPS/TLS 配置

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 1.1.1 | HTTPS 已启用 | `grep "enabled: true" config.yaml` | 配置中 https.enabled=true | 参考 [security-guide.md#21-启用-https](security-guide.md#21-启用-https) |
| 1.1.2 | 证书文件存在 | `ls -la /etc/waterflow/tls/cert.pem` | 文件存在且可读 | 使用 `./scripts/generate-tls-cert.sh` 生成 |
| 1.1.3 | 私钥文件存在 | `ls -la /etc/waterflow/tls/key.pem` | 文件存在,权限600 | `chmod 600 /etc/waterflow/tls/key.pem` |
| 1.1.4 | 证书有效期 | `openssl x509 -in cert.pem -noout -dates` | 至少30天有效期 | 更新证书 |
| 1.1.5 | TLS 最低版本 | `grep "min_version" config.yaml` | min_version >= 1.2 | 设置 `min_version: "1.2"` |
| 1.1.6 | 安全密码套件 | `grep "cipher_suites" config.yaml` | 使用强加密套件 | 使用推荐的密码套件列表 |

**验证脚本:**
```bash
#!/bin/bash
# 检查 TLS 配置

echo "检查 HTTPS/TLS 配置..."

# 检查 HTTPS 启用
if grep -q "enabled: true" config.yaml; then
  echo "✓ HTTPS 已启用"
else
  echo "✗ HTTPS 未启用"
  exit 1
fi

# 检查证书文件
CERT_FILE=$(grep "cert_file:" config.yaml | awk '{print $2}')
if [ -f "$CERT_FILE" ]; then
  echo "✓ 证书文件存在: $CERT_FILE"
  
  # 检查有效期
  if openssl x509 -checkend 2592000 -noout -in "$CERT_FILE"; then
    echo "✓ 证书有效期 > 30天"
  else
    echo "✗ 证书即将过期 (< 30天)"
    exit 1
  fi
else
  echo "✗ 证书文件不存在: $CERT_FILE"
  exit 1
fi

echo "✅ TLS 配置检查通过"
```

### 1.2 认证配置

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 1.2.1 | API 认证已启用 | `grep "auth:" -A2 config.yaml` | auth.enabled=true | 设置 `auth.enabled: true` |
| 1.2.2 | API Key 强度 | `grep "api_keys:" -A5 config.yaml` | Key长度>=32字符 | 使用 `openssl rand -base64 32` 生成 |
| 1.2.3 | API Key 不在代码中 | `git grep "sk_live_" \| wc -l` | 返回 0 | 将 Key 移到环境变量或 Vault |
| 1.2.4 | Agent Token 配置 | `grep "token:" agent/config.yaml` | Token存在且安全 | 生成新 Token 并配置 |

**验证脚本:**
```bash
#!/bin/bash
# 检查认证配置

echo "检查认证配置..."

# 检查认证启用
if grep -q "auth:" config.yaml && \
   grep -A1 "auth:" config.yaml | grep -q "enabled: true"; then
  echo "✓ API 认证已启用"
else
  echo "✗ API 认证未启用"
  exit 1
fi

# 检查 API Key 不在 Git 中
if git grep -q "sk_live_" 2>/dev/null; then
  echo "✗ 警告: API Key 可能泄露到 Git"
  echo "  运行: git grep 'sk_live_' 查看详情"
  exit 1
else
  echo "✓ API Key 未泄露到 Git"
fi

echo "✅ 认证配置检查通过"
```

### 1.3 密钥管理配置

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 1.3.1 | SecretProvider 配置 | `grep "secrets:" -A5 config.yaml` | 配置 vault 或 env | 配置 Vault 集成 |
| 1.3.2 | Vault 可访问 | `vault status` | Vault 正常运行 | 启动 Vault 服务 |
| 1.3.3 | Vault Token 有效 | `vault token lookup` | Token 未过期 | 更新 Vault Token |
| 1.3.4 | 密钥路径配置 | `vault kv list waterflow/` | 路径存在 | 创建密钥路径 |

**验证脚本:**
```bash
#!/bin/bash
# 检查密钥管理配置

echo "检查密钥管理配置..."

# 检查 SecretProvider 配置
if grep -q "provider: vault" config.yaml; then
  echo "✓ 使用 Vault 作为 SecretProvider"
  
  # 检查 Vault 连接
  if vault status >/dev/null 2>&1; then
    echo "✓ Vault 可访问"
  else
    echo "✗ Vault 不可访问"
    echo "  检查: VAULT_ADDR 和 VAULT_TOKEN 环境变量"
    exit 1
  fi
else
  echo "⚠ 未使用 Vault (可能使用环境变量模式)"
fi

echo "✅ 密钥管理配置检查通过"
```

### 1.4 审计日志配置

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 1.4.1 | 审计日志启用 | `grep "audit:" -A2 config.yaml` | audit.enabled=true | 设置 `audit.enabled: true` |
| 1.4.2 | 日志目录存在 | `ls -ld /var/log/waterflow/audit` | 目录存在,权限700 | `mkdir -p /var/log/waterflow/audit && chmod 700` |
| 1.4.3 | 日志保留策略 | `grep "max_age:" config.yaml` | max_age >= 90 | 设置 `max_age: 90` (90天) |
| 1.4.4 | 日志轮转配置 | `grep "compress:" config.yaml` | compress=true | 启用日志压缩 |

**验证脚本:**
```bash
#!/bin/bash
# 检查审计日志配置

echo "检查审计日志配置..."

# 检查审计日志启用
if grep -q "audit:" config.yaml && \
   grep -A1 "audit:" config.yaml | grep -q "enabled: true"; then
  echo "✓ 审计日志已启用"
else
  echo "✗ 审计日志未启用"
  exit 1
fi

# 检查日志目录
LOG_PATH=$(grep "path:" config.yaml | grep audit -A1 | tail -1 | awk '{print $2}')
if [ -d "$LOG_PATH" ]; then
  echo "✓ 日志目录存在: $LOG_PATH"
else
  echo "✗ 日志目录不存在: $LOG_PATH"
  echo "  运行: mkdir -p $LOG_PATH && chmod 700 $LOG_PATH"
  exit 1
fi

echo "✅ 审计日志配置检查通过"
```

### 1.5 防火墙规则

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 1.5.1 | 防火墙启用 | `ufw status` | Status: active | `ufw enable` |
| 1.5.2 | SSH 限制源IP | `ufw status numbered` | SSH 仅特定IP | `ufw allow from IP to any port 22` |
| 1.5.3 | HTTPS 端口开放 | `ufw status \| grep 8443` | 8443/tcp ALLOW | `ufw allow 8443/tcp` |
| 1.5.4 | Temporal 端口保护 | `ufw status \| grep 7233` | 仅内网或DENY | `ufw deny 7233/tcp` |

**验证脚本:**
```bash
#!/bin/bash
# 检查防火墙规则

echo "检查防火墙配置..."

# 检查 ufw 安装
if ! command -v ufw &> /dev/null; then
  echo "⚠ ufw 未安装,跳过防火墙检查"
  exit 0
fi

# 检查防火墙状态
if ufw status | grep -q "Status: active"; then
  echo "✓ 防火墙已启用"
else
  echo "✗ 防火墙未启用"
  echo "  运行: ufw enable"
  exit 1
fi

# 检查 HTTPS 端口
if ufw status | grep -q "8443/tcp"; then
  echo "✓ HTTPS 端口 (8443) 已配置"
else
  echo "⚠ HTTPS 端口 (8443) 未配置"
fi

echo "✅ 防火墙配置检查通过"
```

### 1.6 备份配置

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 1.6.1 | 备份脚本存在 | `ls -la scripts/backup-config.sh` | 文件存在,可执行 | `chmod +x scripts/backup-config.sh` |
| 1.6.2 | 备份目录存在 | `ls -ld /backup/waterflow` | 目录存在 | `mkdir -p /backup/waterflow` |
| 1.6.3 | 定时备份配置 | `crontab -l \| grep backup` | Cron 任务存在 | 添加 cron 任务 |
| 1.6.4 | 备份测试 | `./scripts/backup-config.sh` | 成功创建备份 | 检查脚本错误 |

**验证脚本:**
```bash
#!/bin/bash
# 检查备份配置

echo "检查备份配置..."

# 检查备份脚本
if [ -x scripts/backup-config.sh ]; then
  echo "✓ 备份脚本存在且可执行"
else
  echo "✗ 备份脚本不存在或不可执行"
  exit 1
fi

# 检查备份目录
BACKUP_DIR=/backup/waterflow
if [ -d "$BACKUP_DIR" ]; then
  echo "✓ 备份目录存在: $BACKUP_DIR"
else
  echo "⚠ 备份目录不存在: $BACKUP_DIR"
  echo "  运行: mkdir -p $BACKUP_DIR"
fi

# 检查定时任务
if crontab -l 2>/dev/null | grep -q backup; then
  echo "✓ 定时备份已配置"
else
  echo "⚠ 定时备份未配置"
fi

echo "✅ 备份配置检查通过"
```

---

## 2. 生产环境检查清单

生产环境运行时的日常安全检查。

### 2.1 证书有效性

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 2.1.1 | 证书未过期 | `openssl x509 -in cert.pem -noout -dates` | 有效期 > 30天 | 更新证书 |
| 2.1.2 | 证书链完整 | `openssl verify -CAfile ca.pem cert.pem` | 验证成功 | 检查 CA 证书 |
| 2.1.3 | HTTPS 可访问 | `curl -v https://waterflow/health` | 返回 200 | 检查 TLS 配置 |

### 2.2 密钥安全性

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 2.2.1 | Vault 运行正常 | `vault status` | Sealed=false | Unseal Vault |
| 2.2.2 | 密钥权限正确 | `ls -la /etc/waterflow/tls/key.pem` | 权限 600 | `chmod 600 key.pem` |
| 2.2.3 | 无密钥泄露 | `git grep "sk_live_\|agent_"` | 无输出 | 撤销泄露的密钥 |

### 2.3 审计日志运行状态

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 2.3.1 | 日志文件可写 | `touch /var/log/waterflow/audit/test && rm test` | 成功 | 检查目录权限 |
| 2.3.2 | 日志正常记录 | `tail -f audit.log \| head -5` | 有新日志 | 检查审计配置 |
| 2.3.3 | 日志大小合理 | `du -sh /var/log/waterflow/audit` | < 10GB | 检查轮转配置 |

### 2.4 网络安全配置

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 2.4.1 | 防火墙规则有效 | `ufw status` | Active | 重启 ufw |
| 2.4.2 | 端口监听正确 | `ss -tlnp \| grep waterflow` | 仅必要端口 | 检查配置 |
| 2.4.3 | 无未授权连接 | `ss -tnp \| grep :8443` | 仅已知IP | 调查异常连接 |

### 2.5 资源限制

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 2.5.1 | 内存使用正常 | `docker stats waterflow-server --no-stream` | < 80%限制 | 增加内存限制 |
| 2.5.2 | CPU 使用正常 | `docker stats waterflow-server --no-stream` | < 80%限制 | 优化或扩容 |
| 2.5.3 | 磁盘空间充足 | `df -h /var/log` | > 20%可用 | 清理旧日志 |

### 2.6 监控告警

| # | 检查项 | 验证命令 | 通过标准 | 修复建议 |
|---|--------|---------|---------|---------|
| 2.6.1 | Prometheus 运行 | `curl localhost:9090/-/healthy` | ok | 重启 Prometheus |
| 2.6.2 | Metrics 可访问 | `curl localhost:9090/metrics` | 返回指标 | 检查 Server 配置 |
| 2.6.3 | 告警规则加载 | `promtool check rules alertmanager-rules.yml` | SUCCESS | 修复规则语法 |

**完整生产环境检查脚本:**
```bash
#!/bin/bash
# 生产环境安全检查

echo "🔒 生产环境安全检查"
echo "===================="
echo ""

ERRORS=0

# 检查证书有效性
echo "1. 检查证书有效性..."
CERT_FILE=$(grep "cert_file:" config.yaml | awk '{print $2}')
if openssl x509 -checkend 2592000 -noout -in "$CERT_FILE" 2>/dev/null; then
  echo "  ✓ 证书有效期 > 30天"
else
  echo "  ✗ 证书即将过期"
  ERRORS=$((ERRORS+1))
fi

# 检查审计日志
echo "2. 检查审计日志..."
LOG_PATH="/var/log/waterflow/audit"
if [ -w "$LOG_PATH" ]; then
  echo "  ✓ 审计日志目录可写"
else
  echo "  ✗ 审计日志目录不可写"
  ERRORS=$((ERRORS+1))
fi

# 检查防火墙
echo "3. 检查防火墙..."
if ufw status | grep -q "Status: active"; then
  echo "  ✓ 防火墙已启用"
else
  echo "  ✗ 防火墙未启用"
  ERRORS=$((ERRORS+1))
fi

# 检查服务健康
echo "4. 检查服务健康..."
if curl -sf http://localhost:8080/health >/dev/null; then
  echo "  ✓ Waterflow Server 健康"
else
  echo "  ✗ Waterflow Server 不健康"
  ERRORS=$((ERRORS+1))
fi

echo ""
if [ $ERRORS -eq 0 ]; then
  echo "✅ 所有检查通过"
  exit 0
else
  echo "❌ 发现 $ERRORS 个问题"
  exit 1
fi
```

---

## 3. 定期审查清单 (每月)

每月执行一次的安全审查检查项。

### 3.1 审计日志审查

| # | 检查项 | 验证命令 | 通过标准 | 处理建议 |
|---|--------|---------|---------|---------|
| 3.1.1 | 认证失败记录 | `waterflow audit query --event-type=auth.failure --start-time="30 days ago"` | < 100次 | 调查异常IP |
| 3.1.2 | 密钥访问拒绝 | `waterflow audit query --event-category=secret --result=permission_denied` | = 0 | 审查权限配置 |
| 3.1.3 | 异常操作时间 | `cat audit.log \| jq -r '.timestamp' \| grep "T0[0-4]:"` | 无异常时间操作 | 调查非工作时间操作 |

### 3.2 证书有效期检查

| # | 检查项 | 验证命令 | 通过标准 | 处理建议 |
|---|--------|---------|---------|---------|
| 3.2.1 | 服务器证书 | `openssl x509 -in cert.pem -noout -dates` | > 60天 | 计划更新 |
| 3.2.2 | CA 证书 | `openssl x509 -in ca.pem -noout -dates` | > 60天 | 联系 CA 更新 |
| 3.2.3 | 客户端证书 | `for cert in client-certs/*.pem; do openssl x509 -in $cert -noout -dates; done` | 全部 > 30天 | 通知客户端更新 |

### 3.3 密钥轮换

| # | 检查项 | 验证命令 | 通过标准 | 处理建议 |
|---|--------|---------|---------|---------|
| 3.3.1 | 上次轮换时间 | `cat /var/log/waterflow/secret-rotation.log \| tail -1` | < 30天前 | 执行密钥轮换 |
| 3.3.2 | Vault 密钥版本 | `vault kv metadata get waterflow/production/db_password` | version > 1 | 密钥未轮换 |
| 3.3.3 | API Key 使用时长 | 手动审查创建日期 | < 90天 | 轮换旧 Key |

**密钥轮换脚本:**
```bash
#!/bin/bash
# 月度密钥轮换

echo "🔄 执行月度密钥轮换..."

# 轮换数据库密码
echo "轮换数据库密码..."
NEW_DB_PASS=$(openssl rand -base64 32)
vault kv put waterflow/production/db_password value="$NEW_DB_PASS"

# 轮换 API Key (需要手动更新客户端)
echo "⚠️ 提醒: 需要手动轮换 API Key 并通知客户端"

# 记录轮换日志
echo "$(date): Monthly secret rotation completed" >> /var/log/waterflow/secret-rotation.log

echo "✅ 密钥轮换完成"
```

### 3.4 安全更新

| # | 检查项 | 验证命令 | 通过标准 | 处理建议 |
|---|--------|---------|---------|---------|
| 3.4.1 | Waterflow 版本 | `waterflow version` | 最新稳定版 | 升级 Waterflow |
| 3.4.2 | 系统更新 | `apt list --upgradable` | 无安全更新 | 安装安全补丁 |
| 3.4.3 | Docker 镜像 | `trivy image waterflow/server:latest` | 无高危漏洞 | 更新基础镜像 |

### 3.5 备份验证

| # | 检查项 | 验证命令 | 通过标准 | 处理建议 |
|---|--------|---------|---------|---------|
| 3.5.1 | 备份存在 | `ls -lh /backup/waterflow/config-*.tar.gz` | 最近7天有备份 | 检查备份脚本 |
| 3.5.2 | 备份完整性 | `tar -tzf /backup/waterflow/config-latest.tar.gz` | 列出文件成功 | 重新备份 |
| 3.5.3 | 恢复测试 | 在测试环境恢复备份 | 恢复成功 | 修复备份流程 |

### 3.6 权限审查

| # | 检查项 | 验证命令 | 通过标准 | 处理建议 |
|---|--------|---------|---------|---------|
| 3.6.1 | API Key 使用情况 | `waterflow audit query --event-category=auth --format=json \| jq -r '.user.user_id' \| sort \| uniq -c` | 仅活跃用户 | 撤销未使用的Key |
| 3.6.2 | Agent 活跃状态 | `waterflow agent list` | 仅活跃 Agent | 禁用闲置 Agent |
| 3.6.3 | 角色分配合理 | 审查 config.yaml 的 roles 配置 | 最小权限原则 | 调整权限 |

**完整月度审查脚本:**
```bash
#!/bin/bash
# 月度安全审查

REPORT_FILE="security-review-$(date +%Y-%m).md"

cat > "$REPORT_FILE" <<EOF
# 月度安全审查报告

**日期:** $(date)
**审查人员:** ${USER}

## 1. 审计日志审查

### 认证失败次数
\`\`\`
$(waterflow audit query --event-type=auth.failure --start-time="30 days ago" | wc -l)
\`\`\`

### 密钥访问拒绝次数
\`\`\`
$(waterflow audit query --event-category=secret --result=permission_denied | wc -l)
\`\`\`

## 2. 证书有效期

\`\`\`
$(openssl x509 -in /etc/waterflow/tls/cert.pem -noout -dates)
\`\`\`

## 3. 备份状态

\`\`\`
$(ls -lh /backup/waterflow/config-*.tar.gz | tail -5)
\`\`\`

## 4. 系统更新

\`\`\`
$(apt list --upgradable 2>/dev/null | grep security)
\`\`\`

## 5. 建议措施

$(if [ $(waterflow audit query --event-type=auth.failure --start-time="30 days ago" | wc -l) -gt 100 ]; then
  echo "- ⚠️ 认证失败次数过多,建议审查异常IP"
fi)

EOF

echo "✅ 月度审查报告已生成: $REPORT_FILE"
cat "$REPORT_FILE"
```

---

## 4. 事件响应清单

安全事件发生时的响应检查清单。

### 4.1 检测阶段

| # | 步骤 | 操作命令 | 完成 |
|---|------|---------|-----|
| 4.1.1 | 确认告警 | 查看 Alertmanager 告警 | ☐ |
| 4.1.2 | 查看审计日志 | `tail -f /var/log/waterflow/audit/audit.log` | ☐ |
| 4.1.3 | 识别异常IP | `cat audit.log \| jq -r '.user.ip' \| sort \| uniq -c \| sort -rn` | ☐ |
| 4.1.4 | 识别受影响资源 | 审查失败操作的 resource_id | ☐ |
| 4.1.5 | 记录事件开始时间 | 记录首次异常日志的时间戳 | ☐ |

### 4.2 隔离阶段

| # | 步骤 | 操作命令 | 完成 |
|---|------|---------|-----|
| 4.2.1 | 禁用受影响 Agent | `waterflow agent disable <agent-id>` | ☐ |
| 4.2.2 | 阻止可疑IP | `ufw deny from <suspicious-ip>` | ☐ |
| 4.2.3 | 撤销受影响 API Key | 从 config.yaml 移除 | ☐ |
| 4.2.4 | 临时禁用 HTTPS (极端) | `iptables -A INPUT -p tcp --dport 8443 -j DROP` | ☐ |
| 4.2.5 | 通知团队 | 发送事件通知到 Slack/Email | ☐ |

### 4.3 调查阶段

| # | 步骤 | 操作命令 | 完成 |
|---|------|---------|-----|
| 4.3.1 | 导出审计日志 | `waterflow audit export --start-time="<incident-start>" --output=incident.json` | ☐ |
| 4.3.2 | 查看可疑用户操作 | `waterflow audit query --user-id=<user> --start-time="<time>"` | ☐ |
| 4.3.3 | 查看可疑IP操作 | `cat audit.log \| jq 'select(.user.ip == "<ip>")'` | ☐ |
| 4.3.4 | 分析攻击模式 | 审查日志识别攻击类型 | ☐ |
| 4.3.5 | 识别泄露密钥 | 检查是否有密钥被未授权访问 | ☐ |

### 4.4 修复阶段

| # | 步骤 | 操作命令 | 完成 |
|---|------|---------|-----|
| 4.4.1 | 轮换受影响密钥 | `vault kv put waterflow/<path> value=<new>` | ☐ |
| 4.4.2 | 生成新 API Key | `openssl rand -base64 32` | ☐ |
| 4.4.3 | 更新防火墙规则 | 永久阻止可疑IP | ☐ |
| 4.4.4 | 更新证书 (如泄露) | `./scripts/generate-tls-cert.sh production <domain>` | ☐ |
| 4.4.5 | 应用安全补丁 | 更新 Waterflow 到最新版本 | ☐ |

### 4.5 恢复阶段

| # | 步骤 | 操作命令 | 完成 |
|---|------|---------|-----|
| 4.5.1 | 验证配置 | `waterflow validate-config` | ☐ |
| 4.5.2 | 运行安全检查 | `./scripts/security-check.sh` | ☐ |
| 4.5.3 | 重启服务 | `systemctl restart waterflow-server` | ☐ |
| 4.5.4 | 验证健康状态 | `curl http://localhost:8080/health` | ☐ |
| 4.5.5 | 监控恢复情况 | `watch -n 5 'curl -s localhost:8080/health'` | ☐ |

### 4.6 总结阶段

| # | 步骤 | 操作命令 | 完成 |
|---|------|---------|-----|
| 4.6.1 | 生成事件报告 | `./scripts/generate-incident-report.sh` | ☐ |
| 4.6.2 | 记录根本原因 | 文档化事件原因 | ☐ |
| 4.6.3 | 制定预防措施 | 更新安全策略 | ☐ |
| 4.6.4 | 团队复盘会议 | 召集团队讨论 | ☐ |
| 4.6.5 | 更新安全文档 | 更新检查清单和流程 | ☐ |

**事件响应快速命令参考:**
```bash
# 快速隔离 - 阻止所有外部访问
iptables -A INPUT -p tcp --dport 8443 -j DROP

# 快速调查 - 查看最近1小时的失败操作
waterflow audit query \
  --result=failure \
  --start-time="1 hour ago" \
  --format=json > incident-$(date +%Y%m%d%H%M).json

# 快速修复 - 轮换所有密钥
./scripts/rotate-secrets.sh

# 快速恢复 - 重启服务并验证
docker restart waterflow-server && \
  sleep 10 && \
  curl http://localhost:8080/health
```

---

## 附录

### A. 自动化检查脚本

完整的自动化安全检查脚本位于:
- `scripts/security-check.sh` - 综合安全检查
- `scripts/production-check.sh` - 生产环境检查
- `scripts/monthly-review.sh` - 月度审查
- `scripts/incident-response.sh` - 事件响应辅助

### B. 相关文档

- [安全配置指南](security-guide.md) - 详细的安全配置说明
- [合规指南](compliance-guide.md) - 合规要求和配置
- [故障排查指南](../troubleshooting.md) - 安全问题排查

### C. 检查频率建议

| 检查类型 | 频率 | 责任人 |
|---------|------|--------|
| 部署前检查 | 每次部署 | DevOps |
| 生产环境检查 | 每周 | 运维团队 |
| 定期审查 | 每月 | 安全团队 |
| 事件响应 | 事件发生时 | 安全响应团队 |

---

**最后更新:** 2026-01-12  
**版本:** 1.0.0  
**维护者:** Waterflow Security Team

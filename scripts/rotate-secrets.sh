#!/bin/bash
# Waterflow 密钥轮换脚本
#
# 定期轮换 Vault 中的密钥,提高安全性
#
# 使用方法:
#   ./rotate-secrets.sh [environment]
#
# 参数:
#   environment - production, staging, development (默认: production)

set -e

ENV=${1:-production}
LOG_FILE="/var/log/waterflow/secret-rotation.log"
VAULT_MOUNT="waterflow"

echo "========================================="
echo " Waterflow 密钥轮换"
echo "========================================="
echo "环境: $ENV"
echo "时间: $(date)"
echo "========================================="
echo ""

# 检查 Vault
if ! command -v vault &> /dev/null; then
    echo "❌ vault CLI 未安装"
    exit 1
fi

if ! vault status >/dev/null 2>&1; then
    echo "❌ Vault 不可访问"
    echo "   检查 VAULT_ADDR 和 VAULT_TOKEN"
    exit 1
fi

# 密钥列表
SECRETS=(
  "$ENV/db_password"
  "$ENV/api_key"
  "$ENV/ssh_key"
)

# 轮换密钥
for secret in "${SECRETS[@]}"; do
  echo "🔄 轮换密钥: $secret"
  
  # 生成新密钥
  new_value=$(openssl rand -base64 32)
  
  # 更新 Vault
  vault kv put "$VAULT_MOUNT/$secret" value="$new_value"
  
  echo "✓ 已轮换 $secret"
  
  # 记录日志
  echo "$(date): Rotated $secret" >> "$LOG_FILE"
done

echo ""
echo "✅ 密钥轮换完成!"
echo "日志: $LOG_FILE"

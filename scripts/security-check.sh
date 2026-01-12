#!/bin/bash
# Waterflow 安全检查脚本
#
# 运行全面的安全检查,验证 Waterflow 配置的安全性
# 
# 使用方法:
#   ./security-check.sh [config_file]
#
# 返回值:
#   0 - 所有检查通过
#   1 - 发现安全问题

set -e

CONFIG_FILE=${1:-config.yaml}
ERRORS=0
WARNINGS=0

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

check_pass() {
    echo -e "  ${GREEN}✓${NC} $1"
}

check_fail() {
    echo -e "  ${RED}✗${NC} $1"
    ERRORS=$((ERRORS+1))
}

check_warn() {
    echo -e "  ${YELLOW}⚠${NC} $1"
    WARNINGS=$((WARNINGS+1))
}

section() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

echo "========================================="
echo "  Waterflow 安全检查"
echo "========================================="
echo "配置文件: $CONFIG_FILE"
echo "检查时间: $(date)"
echo ""

# ========================================
# 检查 0: 配置文件存在
# ========================================
section "0. 配置文件检查"

if [ ! -f "$CONFIG_FILE" ]; then
    check_fail "配置文件不存在: $CONFIG_FILE"
    echo ""
    echo "请设置 WATERFLOW_CONFIG 环境变量或确保文件存在"
    exit 1
fi
check_pass "配置文件存在"

# ========================================
# 检查 1: HTTPS/TLS 配置
# ========================================
section "1. HTTPS/TLS 配置"

if grep -q "https:" "$CONFIG_FILE" && grep -A1 "https:" "$CONFIG_FILE" | grep -q "enabled: true"; then
    check_pass "HTTPS 已启用"
    
    # 检查证书文件
    CERT_FILE=$(grep -A5 "https:" "$CONFIG_FILE" | grep "cert_file:" | awk '{print $2}' | tr -d '"')
    if [ -n "$CERT_FILE" ] && [ -f "$CERT_FILE" ]; then
        check_pass "证书文件存在: $CERT_FILE"
        
        # 检查证书有效期
        if openssl x509 -checkend 2592000 -noout -in "$CERT_FILE" 2>/dev/null; then
            check_pass "证书有效期 > 30天"
        else
            check_warn "证书即将过期 (< 30天)"
        fi
        
        # 检查私钥文件
        KEY_FILE=$(grep -A5 "https:" "$CONFIG_FILE" | grep "key_file:" | awk '{print $2}' | tr -d '"')
        if [ -n "$KEY_FILE" ] && [ -f "$KEY_FILE" ]; then
            check_pass "私钥文件存在: $KEY_FILE"
            
            # 检查私钥权限
            KEY_PERMS=$(stat -c "%a" "$KEY_FILE" 2>/dev/null || stat -f "%Lp" "$KEY_FILE" 2>/dev/null)
            if [ "$KEY_PERMS" == "600" ]; then
                check_pass "私钥权限正确 (600)"
            else
                check_fail "私钥权限不安全: $KEY_PERMS (应为 600)"
            fi
        else
            check_fail "私钥文件不存在: $KEY_FILE"
        fi
    else
        check_fail "证书文件不存在或未配置: $CERT_FILE"
    fi
    
    # 检查 TLS 最低版本
    if grep -A5 "https:" "$CONFIG_FILE" | grep -q "min_version:.*1\.[23]"; then
        check_pass "TLS 最低版本 >= 1.2"
    else
        check_warn "TLS 最低版本未配置或低于 1.2"
    fi
else
    check_fail "HTTPS 未启用"
fi

# ========================================
# 检查 2: 认证配置
# ========================================
section "2. 认证配置"

if grep -q "auth:" "$CONFIG_FILE" && grep -A1 "auth:" "$CONFIG_FILE" | grep -q "enabled: true"; then
    check_pass "API 认证已启用"
    
    # 检查 API Key 强度
    if grep -q "api_keys:" "$CONFIG_FILE"; then
        # 检查是否有硬编码的 Key (不安全)
        if grep -A10 "api_keys:" "$CONFIG_FILE" | grep -q "key:.*sk_live_"; then
            check_warn "检测到硬编码的 API Key (推荐使用环境变量)"
        else
            check_pass "API Key 未硬编码"
        fi
    fi
else
    check_fail "API 认证未启用"
fi

# ========================================
# 检查 3: 密钥管理
# ========================================
section "3. 密钥管理"

if grep -q "secrets:" "$CONFIG_FILE"; then
    PROVIDER=$(grep -A1 "secrets:" "$CONFIG_FILE" | grep "provider:" | awk '{print $2}')
    
    if [ "$PROVIDER" == "vault" ]; then
        check_pass "使用 Vault 作为密钥提供者"
        
        # 检查 Vault 连接
        if command -v vault &> /dev/null; then
            if vault status >/dev/null 2>&1; then
                check_pass "Vault 可访问"
            else
                check_warn "Vault 不可访问 (检查 VAULT_ADDR 和 VAULT_TOKEN)"
            fi
        else
            check_warn "vault CLI 未安装 (无法验证 Vault 连接)"
        fi
    elif [ "$PROVIDER" == "env" ]; then
        check_warn "使用环境变量模式 (不推荐用于生产环境)"
    else
        check_warn "未配置密钥提供者"
    fi
else
    check_warn "未配置密钥管理"
fi

# ========================================
# 检查 4: 审计日志
# ========================================
section "4. 审计日志"

if grep -q "audit:" "$CONFIG_FILE" && grep -A1 "audit:" "$CONFIG_FILE" | grep -q "enabled: true"; then
    check_pass "审计日志已启用"
    
    # 检查日志目录
    LOG_PATH=$(grep -A10 "audit:" "$CONFIG_FILE" | grep "path:" | awk '{print $2}' | tr -d '"')
    if [ -n "$LOG_PATH" ] && [ -d "$LOG_PATH" ]; then
        check_pass "日志目录存在: $LOG_PATH"
        
        # 检查日志目录可写
        if [ -w "$LOG_PATH" ]; then
            check_pass "日志目录可写"
        else
            check_fail "日志目录不可写: $LOG_PATH"
        fi
    else
        check_fail "日志目录不存在: $LOG_PATH"
    fi
    
    # 检查日志保留策略
    MAX_AGE=$(grep -A10 "audit:" "$CONFIG_FILE" | grep "max_age:" | awk '{print $2}')
    if [ -n "$MAX_AGE" ] && [ "$MAX_AGE" -ge 90 ]; then
        check_pass "日志保留期 >= 90天 (符合 SOC2 要求)"
    else
        check_warn "日志保留期 < 90天 (不符合 SOC2 要求)"
    fi
else
    check_fail "审计日志未启用"
fi

# ========================================
# 检查 5: 防火墙
# ========================================
section "5. 防火墙配置"

if command -v ufw &> /dev/null; then
    if ufw status | grep -q "Status: active"; then
        check_pass "防火墙 (ufw) 已启用"
        
        # 检查 HTTPS 端口
        if ufw status | grep -q "8443/tcp"; then
            check_pass "HTTPS 端口 (8443) 已配置"
        else
            check_warn "HTTPS 端口 (8443) 未在防火墙中配置"
        fi
    else
        check_warn "防火墙 (ufw) 未启用"
    fi
elif command -v iptables &> /dev/null; then
    if iptables -L | grep -q "Chain INPUT"; then
        check_pass "防火墙 (iptables) 已配置"
    else
        check_warn "防火墙 (iptables) 未配置"
    fi
else
    check_warn "未检测到防火墙工具 (ufw/iptables)"
fi

# ========================================
# 检查 6: 备份配置
# ========================================
section "6. 备份配置"

# 检查备份脚本
if [ -x "scripts/backup-config.sh" ] || [ -x "/usr/local/bin/backup-config.sh" ]; then
    check_pass "备份脚本存在且可执行"
else
    check_warn "备份脚本不存在或不可执行"
fi

# 检查备份目录
BACKUP_DIR=/backup/waterflow
if [ -d "$BACKUP_DIR" ]; then
    check_pass "备份目录存在: $BACKUP_DIR"
    
    # 检查最近的备份
    LATEST_BACKUP=$(find "$BACKUP_DIR" -name "*.tar.gz" -mtime -7 2>/dev/null | head -1)
    if [ -n "$LATEST_BACKUP" ]; then
        check_pass "最近7天有备份文件"
    else
        check_warn "最近7天没有备份文件"
    fi
else
    check_warn "备份目录不存在: $BACKUP_DIR"
fi

# 检查定时任务
if crontab -l 2>/dev/null | grep -q backup; then
    check_pass "定时备份已配置"
else
    check_warn "定时备份未配置"
fi

# ========================================
# 检查 7: 服务健康
# ========================================
section "7. 服务健康检查"

# 检查 Server 是否运行
if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
    check_pass "Waterflow Server 健康 (HTTP)"
elif curl -sfk https://localhost:8443/health >/dev/null 2>&1; then
    check_pass "Waterflow Server 健康 (HTTPS)"
else
    check_warn "Waterflow Server 健康检查失败 (服务可能未运行)"
fi

# 检查 Temporal 连接
TEMPORAL_HOST=$(grep -A5 "temporal:" "$CONFIG_FILE" | grep "host:" | awk '{print $2}' | tr -d '"' | cut -d: -f1)
TEMPORAL_PORT=$(grep -A5 "temporal:" "$CONFIG_FILE" | grep "port:" | awk '{print $2}' | tr -d '"')
if [ -z "$TEMPORAL_PORT" ]; then
    TEMPORAL_PORT=7233
fi

if [ -n "$TEMPORAL_HOST" ]; then
    if timeout 2 bash -c "echo > /dev/tcp/$TEMPORAL_HOST/$TEMPORAL_PORT" 2>/dev/null; then
        check_pass "Temporal Server 可访问 ($TEMPORAL_HOST:$TEMPORAL_PORT)"
    else
        check_warn "Temporal Server 不可访问 ($TEMPORAL_HOST:$TEMPORAL_PORT)"
    fi
fi

# ========================================
# 检查 8: Git 敏感信息
# ========================================
section "8. Git 敏感信息检查"

if [ -d ".git" ]; then
    # 检查是否有 API Key 泄露
    if git grep -q "sk_live_\|agent_" 2>/dev/null; then
        check_fail "检测到 API Key/Token 泄露到 Git"
    else
        check_pass "未检测到 API Key/Token 泄露"
    fi
    
    # 检查是否有密码泄露
    if git grep -i "password.*=.*['\"]" | grep -v "example\|template\|placeholder" >/dev/null 2>&1; then
        check_warn "可能检测到密码泄露"
    else
        check_pass "未检测到明显的密码泄露"
    fi
else
    check_warn "非 Git 仓库,跳过 Git 检查"
fi

# ========================================
# 总结
# ========================================
section "检查总结"

echo ""
if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}✅ 所有检查通过!${NC}"
    echo ""
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}⚠️  发现 $WARNINGS 个警告${NC}"
    echo ""
    echo "建议查看并解决警告项以提高安全性"
    exit 0
else
    echo -e "${RED}❌ 发现 $ERRORS 个错误, $WARNINGS 个警告${NC}"
    echo ""
    echo "必须修复错误后才能安全运行 Waterflow"
    exit 1
fi

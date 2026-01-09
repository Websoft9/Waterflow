#!/bin/bash
#
# Waterflow Configuration Restore Script
# 恢复配置文件

set -euo pipefail

# 配置
BACKUP_FILE="${1:-}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示使用说明
usage() {
    cat << EOF
使用方法: $0 <backup-file>

参数:
  <backup-file>    配置备份文件路径 (例如: /data/waterflow/backups/waterflow_configs_20260109_120000.tar.gz)

示例:
  $0 /data/waterflow/backups/waterflow_configs_20260109_120000.tar.gz
EOF
    exit 1
}

# 检查参数
check_args() {
    if [ -z "${BACKUP_FILE}" ]; then
        log_error "请指定备份文件"
        usage
    fi
    
    if [ ! -f "${BACKUP_FILE}" ]; then
        log_error "备份文件不存在: ${BACKUP_FILE}"
        exit 1
    fi
}

# 验证备份文件
verify_backup() {
    log_info "验证备份文件完整性..."
    
    if tar -tzf "${BACKUP_FILE}" > /dev/null 2>&1; then
        log_info "备份文件完整性验证通过"
    else
        log_error "备份文件损坏"
        exit 1
    fi
}

# 列出备份内容
list_backup_contents() {
    log_info "备份文件内容:"
    tar -tzf "${BACKUP_FILE}" | sed 's/^/  /'
}

# 执行恢复
restore_configs() {
    log_warn "========================================="
    log_warn "警告: 即将恢复配置文件"
    log_warn "当前配置文件将被覆盖!"
    log_warn "========================================="
    
    read -p "确认继续? (yes/no): " -r
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log_info "恢复已取消"
        exit 0
    fi
    
    log_info "开始恢复配置文件..."
    
    # 解压配置文件到根目录
    if tar -xzf "${BACKUP_FILE}" -C / 2>/dev/null; then
        log_info "配置文件恢复成功"
    else
        log_error "配置文件恢复失败"
        exit 1
    fi
}

# 验证恢复结果
verify_restore() {
    log_info "验证恢复结果..."
    
    # 检查关键配置文件
    local critical_files=(
        "/etc/waterflow/config.yaml"
        "/opt/waterflow/deployments/.env"
    )
    
    local verified=0
    for file in "${critical_files[@]}"; do
        if [ -f "${file}" ]; then
            log_info "✓ ${file}"
            verified=$((verified + 1))
        else
            log_warn "✗ ${file} (不存在)"
        fi
    done
    
    if [ ${verified} -eq 0 ]; then
        log_error "关键配置文件验证失败"
        exit 1
    else
        log_info "配置文件恢复验证通过 (${verified}/${#critical_files[@]})"
    fi
}

# 主函数
main() {
    log_info "========================================="
    log_info "Waterflow Configuration Restore"
    log_info "========================================="
    
    check_args
    verify_backup
    list_backup_contents
    restore_configs
    verify_restore
    
    log_info "========================================="
    log_info "恢复完成!"
    log_info "请重启 Waterflow 服务以应用新配置"
    log_info "========================================="
}

# 执行主函数
main "$@"

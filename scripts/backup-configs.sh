#!/bin/bash
#
# Waterflow Configuration Backup Script
# 备份配置文件和环境变量

set -euo pipefail

# 配置
BACKUP_DIR="${BACKUP_DIR:-/data/waterflow/backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/waterflow_configs_${TIMESTAMP}.tar.gz"

# 要备份的配置目录和文件
CONFIG_PATHS=(
    "/etc/waterflow"
    "/opt/waterflow/deployments/.env"
    "/opt/waterflow/deployments/docker-compose.yaml"
)

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

# 创建备份目录
prepare_backup_dir() {
    if [ ! -d "${BACKUP_DIR}" ]; then
        log_info "创建备份目录: ${BACKUP_DIR}"
        mkdir -p "${BACKUP_DIR}"
    fi
}

# 检查配置文件是否存在
check_configs() {
    local missing=0
    
    for path in "${CONFIG_PATHS[@]}"; do
        if [ ! -e "${path}" ]; then
            log_warn "配置路径不存在: ${path}"
            missing=$((missing + 1))
        fi
    done
    
    if [ ${missing} -eq ${#CONFIG_PATHS[@]} ]; then
        log_error "所有配置路径都不存在"
        exit 1
    fi
}

# 执行备份
backup_configs() {
    log_info "开始备份配置文件..."
    log_info "备份文件: ${BACKUP_FILE}"
    
    # 构建 tar 命令参数
    local tar_args=()
    for path in "${CONFIG_PATHS[@]}"; do
        if [ -e "${path}" ]; then
            tar_args+=("${path}")
        fi
    done
    
    # 打包配置文件
    if tar -czf "${BACKUP_FILE}" "${tar_args[@]}" 2>/dev/null; then
        log_info "配置文件备份成功"
        
        # 显示备份文件大小
        BACKUP_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
        log_info "备份文件大小: ${BACKUP_SIZE}"
    else
        log_error "配置文件备份失败"
        rm -f "${BACKUP_FILE}"
        exit 1
    fi
}

# 列出备份内容
list_backup_contents() {
    log_info "备份内容:"
    tar -tzf "${BACKUP_FILE}" | sed 's/^/  /'
}

# 验证备份完整性
verify_backup() {
    log_info "验证备份完整性..."
    
    if tar -tzf "${BACKUP_FILE}" > /dev/null 2>&1; then
        log_info "备份文件完整性验证通过"
    else
        log_error "备份文件损坏"
        exit 1
    fi
}

# 主函数
main() {
    log_info "========================================="
    log_info "Waterflow Configuration Backup"
    log_info "========================================="
    
    check_configs
    prepare_backup_dir
    backup_configs
    verify_backup
    list_backup_contents
    
    log_info "========================================="
    log_info "备份完成!"
    log_info "备份文件: ${BACKUP_FILE}"
    log_info "========================================="
}

# 执行主函数
main "$@"

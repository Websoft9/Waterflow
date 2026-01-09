#!/bin/bash
#
# Waterflow Backup Cleanup Script
# 清理旧备份文件

set -euo pipefail

# 配置
BACKUP_DIR="${BACKUP_DIR:-/data/waterflow/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
DRY_RUN="${DRY_RUN:-false}"

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
使用方法: $0 [OPTIONS]

选项:
  -d, --days DAYS        保留天数 (默认: 7)
  -p, --path PATH        备份目录路径 (默认: /data/waterflow/backups)
  -n, --dry-run          模拟运行，不实际删除文件
  -h, --help             显示帮助信息

环境变量:
  BACKUP_DIR             备份目录
  RETENTION_DAYS         保留天数
  DRY_RUN                模拟运行 (true/false)

示例:
  # 清理 7 天前的备份
  $0

  # 清理 30 天前的备份
  $0 --days 30

  # 模拟运行（不实际删除）
  $0 --dry-run
EOF
    exit 1
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -d|--days)
                RETENTION_DAYS="$2"
                shift 2
                ;;
            -p|--path)
                BACKUP_DIR="$2"
                shift 2
                ;;
            -n|--dry-run)
                DRY_RUN="true"
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                log_error "未知参数: $1"
                usage
                ;;
        esac
    done
}

# 检查备份目录
check_backup_dir() {
    if [ ! -d "${BACKUP_DIR}" ]; then
        log_error "备份目录不存在: ${BACKUP_DIR}"
        exit 1
    fi
}

# 查找旧备份文件
find_old_backups() {
    log_info "查找 ${RETENTION_DAYS} 天前的备份文件..."
    
    # 数据库备份
    DB_BACKUPS=$(find "${BACKUP_DIR}" -name "waterflow_db_*.sql.gz" -type f -mtime +${RETENTION_DAYS} 2>/dev/null || true)
    
    # 配置备份
    CONFIG_BACKUPS=$(find "${BACKUP_DIR}" -name "waterflow_configs_*.tar.gz" -type f -mtime +${RETENTION_DAYS} 2>/dev/null || true)
    
    ALL_BACKUPS="${DB_BACKUPS}${DB_BACKUPS:+$'\n'}${CONFIG_BACKUPS}"
}

# 显示待删除文件
show_backups() {
    if [ -z "${ALL_BACKUPS}" ]; then
        log_info "没有找到需要清理的备份文件"
        return 0
    fi
    
    log_info "待清理的备份文件:"
    echo "${ALL_BACKUPS}" | while read -r file; do
        if [ -n "${file}" ]; then
            FILE_SIZE=$(du -h "${file}" | cut -f1)
            FILE_DATE=$(stat -c %y "${file}" | cut -d' ' -f1)
            echo "  - ${file} (${FILE_SIZE}, ${FILE_DATE})"
        fi
    done
}

# 删除旧备份
delete_backups() {
    if [ -z "${ALL_BACKUPS}" ]; then
        return 0
    fi
    
    local total_size=0
    local count=0
    
    echo "${ALL_BACKUPS}" | while read -r file; do
        if [ -n "${file}" ]; then
            FILE_SIZE_BYTES=$(stat -c %s "${file}")
            total_size=$((total_size + FILE_SIZE_BYTES))
            count=$((count + 1))
            
            if [ "${DRY_RUN}" = "true" ]; then
                log_warn "[DRY RUN] 将删除: ${file}"
            else
                rm -f "${file}"
                log_info "已删除: ${file}"
            fi
        fi
    done
    
    # 计算释放的空间
    if [ ${count} -gt 0 ]; then
        FREED_SPACE=$(echo "${total_size}" | awk '{printf "%.2f MB", $1/1024/1024}')
        
        if [ "${DRY_RUN}" = "true" ]; then
            log_info "[DRY RUN] 将删除 ${count} 个文件，释放约 ${FREED_SPACE} 空间"
        else
            log_info "已删除 ${count} 个文件，释放 ${FREED_SPACE} 空间"
        fi
    fi
}

# 显示备份统计
show_statistics() {
    log_info "========================================="
    log_info "备份目录统计:"
    
    # 数据库备份数量和大小
    DB_COUNT=$(find "${BACKUP_DIR}" -name "waterflow_db_*.sql.gz" -type f 2>/dev/null | wc -l)
    DB_SIZE=$(du -sh "${BACKUP_DIR}"/waterflow_db_*.sql.gz 2>/dev/null | awk '{sum+=$1} END {print sum}' || echo "0")
    log_info "数据库备份: ${DB_COUNT} 个文件"
    
    # 配置备份数量和大小
    CONFIG_COUNT=$(find "${BACKUP_DIR}" -name "waterflow_configs_*.tar.gz" -type f 2>/dev/null | wc -l)
    log_info "配置备份: ${CONFIG_COUNT} 个文件"
    
    # 总大小
    TOTAL_SIZE=$(du -sh "${BACKUP_DIR}" 2>/dev/null | cut -f1 || echo "0")
    log_info "备份目录总大小: ${TOTAL_SIZE}"
    log_info "========================================="
}

# 主函数
main() {
    parse_args "$@"
    
    log_info "========================================="
    log_info "Waterflow Backup Cleanup"
    log_info "========================================="
    log_info "备份目录: ${BACKUP_DIR}"
    log_info "保留天数: ${RETENTION_DAYS}"
    
    if [ "${DRY_RUN}" = "true" ]; then
        log_warn "模拟运行模式 (不会实际删除文件)"
    fi
    
    log_info "========================================="
    
    check_backup_dir
    find_old_backups
    show_backups
    delete_backups
    show_statistics
    
    log_info "清理完成!"
}

# 执行主函数
main "$@"

#!/bin/bash
#
# Waterflow Database Backup Script
# 备份 Temporal PostgreSQL 数据库

set -euo pipefail

# 配置
BACKUP_DIR="${BACKUP_DIR:-/data/waterflow/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/waterflow_db_${TIMESTAMP}.sql.gz"

# Docker 配置
COMPOSE_FILE="${COMPOSE_FILE:-/opt/waterflow/deployments/docker-compose.yaml}"
DB_CONTAINER="postgresql"
DB_USER="temporal"
DB_NAME="temporal"

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

# 检查 Docker 环境
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装"
        exit 1
    fi
}

# 检查数据库容器状态
check_database() {
    if ! docker-compose -f "${COMPOSE_FILE}" ps "${DB_CONTAINER}" | grep -q "Up"; then
        log_error "数据库容器 ${DB_CONTAINER} 未运行"
        exit 1
    fi
    log_info "数据库容器运行正常"
}

# 创建备份目录
prepare_backup_dir() {
    if [ ! -d "${BACKUP_DIR}" ]; then
        log_info "创建备份目录: ${BACKUP_DIR}"
        mkdir -p "${BACKUP_DIR}"
    fi
}

# 执行备份
backup_database() {
    log_info "开始备份数据库..."
    log_info "备份文件: ${BACKUP_FILE}"
    
    # 使用 pg_dump 导出数据库
    if docker-compose -f "${COMPOSE_FILE}" exec -T "${DB_CONTAINER}" \
        pg_dump -U "${DB_USER}" -d "${DB_NAME}" --clean --if-exists | gzip > "${BACKUP_FILE}"; then
        log_info "数据库备份成功"
        
        # 显示备份文件大小
        BACKUP_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
        log_info "备份文件大小: ${BACKUP_SIZE}"
    else
        log_error "数据库备份失败"
        rm -f "${BACKUP_FILE}"
        exit 1
    fi
}

# 清理旧备份
cleanup_old_backups() {
    log_info "清理 ${RETENTION_DAYS} 天前的备份文件..."
    
    DELETED_COUNT=$(find "${BACKUP_DIR}" -name "waterflow_db_*.sql.gz" -type f -mtime +${RETENTION_DAYS} -delete -print | wc -l)
    
    if [ "${DELETED_COUNT}" -gt 0 ]; then
        log_info "已删除 ${DELETED_COUNT} 个旧备份文件"
    else
        log_info "无需清理旧备份"
    fi
}

# 验证备份完整性
verify_backup() {
    log_info "验证备份完整性..."
    
    if gzip -t "${BACKUP_FILE}" 2>/dev/null; then
        log_info "备份文件完整性验证通过"
    else
        log_error "备份文件损坏"
        exit 1
    fi
}

# 主函数
main() {
    log_info "========================================="
    log_info "Waterflow Database Backup"
    log_info "========================================="
    
    check_docker
    check_database
    prepare_backup_dir
    backup_database
    verify_backup
    cleanup_old_backups
    
    log_info "========================================="
    log_info "备份完成!"
    log_info "备份文件: ${BACKUP_FILE}"
    log_info "========================================="
}

# 执行主函数
main "$@"

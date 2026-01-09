#!/bin/bash
#
# Waterflow Database Restore Script
# 恢复 Temporal PostgreSQL 数据库

set -euo pipefail

# 配置
BACKUP_DIR="${BACKUP_DIR:-/data/waterflow/backups}"
BACKUP_FILE="${1:-}"

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

# 显示使用说明
usage() {
    cat << EOF
使用方法: $0 <backup-file>

参数:
  <backup-file>    备份文件路径 (例如: /data/waterflow/backups/waterflow_db_20260109_120000.sql.gz)

示例:
  $0 /data/waterflow/backups/waterflow_db_20260109_120000.sql.gz

环境变量:
  BACKUP_DIR       备份目录 (默认: /data/waterflow/backups)
  COMPOSE_FILE     docker-compose 文件路径
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

# 验证备份文件
verify_backup() {
    log_info "验证备份文件完整性..."
    
    if gzip -t "${BACKUP_FILE}" 2>/dev/null; then
        log_info "备份文件完整性验证通过"
    else
        log_error "备份文件损坏"
        exit 1
    fi
}

# 停止 Waterflow 和 Temporal 服务
stop_services() {
    log_warn "停止 Waterflow 和 Temporal 服务..."
    docker-compose -f "${COMPOSE_FILE}" stop waterflow temporal agent-linux-1 2>/dev/null || true
    sleep 5
}

# 启动服务
start_services() {
    log_info "启动 Waterflow 和 Temporal 服务..."
    docker-compose -f "${COMPOSE_FILE}" start waterflow temporal agent-linux-1
    
    # 等待服务启动
    log_info "等待服务启动 (30 秒)..."
    sleep 30
}

# 执行恢复
restore_database() {
    log_warn "========================================="
    log_warn "警告: 即将恢复数据库"
    log_warn "当前数据库数据将被覆盖!"
    log_warn "========================================="
    log_warn "备份文件: ${BACKUP_FILE}"
    
    read -p "确认继续? (yes/no): " -r
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log_info "恢复已取消"
        exit 0
    fi
    
    log_info "开始恢复数据库..."
    
    # 解压并导入数据库
    if gunzip -c "${BACKUP_FILE}" | docker-compose -f "${COMPOSE_FILE}" exec -T "${DB_CONTAINER}" \
        psql -U "${DB_USER}" -d "${DB_NAME}"; then
        log_info "数据库恢复成功"
    else
        log_error "数据库恢复失败"
        exit 1
    fi
}

# 验证恢复结果
verify_restore() {
    log_info "验证恢复结果..."
    
    # 检查表是否存在
    TABLE_COUNT=$(docker-compose -f "${COMPOSE_FILE}" exec -T "${DB_CONTAINER}" \
        psql -U "${DB_USER}" -d "${DB_NAME}" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public'" | tr -d ' ')
    
    if [ "${TABLE_COUNT}" -gt 0 ]; then
        log_info "数据库表数量: ${TABLE_COUNT}"
        log_info "恢复验证通过"
    else
        log_error "数据库验证失败: 无表存在"
        exit 1
    fi
}

# 主函数
main() {
    log_info "========================================="
    log_info "Waterflow Database Restore"
    log_info "========================================="
    
    check_args
    check_docker
    check_database
    verify_backup
    stop_services
    restore_database
    start_services
    verify_restore
    
    log_info "========================================="
    log_info "恢复完成!"
    log_info "========================================="
}

# 执行主函数
main "$@"

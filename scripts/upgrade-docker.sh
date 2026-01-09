#!/bin/bash
#
# Waterflow Docker Upgrade Script
# 升级 Docker Compose 部署的 Waterflow

set -euo pipefail

# 配置
DEPLOY_DIR="${DEPLOY_DIR:-/opt/waterflow/deployments}"
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.yaml"
BACKUP_DIR="${BACKUP_DIR:-/data/waterflow/backups}"

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
    
    log_info "Docker 环境检查通过"
}

# 检查部署目录
check_deploy_dir() {
    if [ ! -d "${DEPLOY_DIR}" ]; then
        log_error "部署目录不存在: ${DEPLOY_DIR}"
        exit 1
    fi
    
    if [ ! -f "${COMPOSE_FILE}" ]; then
        log_error "docker-compose.yaml 不存在: ${COMPOSE_FILE}"
        exit 1
    fi
    
    log_info "部署目录检查通过"
}

# 获取当前版本
get_current_version() {
    log_info "获取当前版本..."
    
    CURRENT_VERSION=$(docker-compose -f "${COMPOSE_FILE}" exec -T waterflow /app/server --version 2>/dev/null || echo "unknown")
    log_info "当前版本: ${CURRENT_VERSION}"
}

# 创建备份
create_backup() {
    log_info "创建升级前备份..."
    
    # 备份数据库
    if command -v backup-database.sh &> /dev/null; then
        backup-database.sh
    else
        log_warn "backup-database.sh 未找到，跳过数据库备份"
    fi
    
    # 备份配置
    if command -v backup-configs.sh &> /dev/null; then
        backup-configs.sh
    else
        log_warn "backup-configs.sh 未找到，跳过配置备份"
    fi
}

# 停止服务
stop_services() {
    log_info "停止 Waterflow 服务..."
    cd "${DEPLOY_DIR}"
    docker-compose stop waterflow agent-linux-1
    log_info "服务已停止"
}

# 拉取最新镜像
pull_images() {
    log_info "拉取最新 Docker 镜像..."
    cd "${DEPLOY_DIR}"
    
    if docker-compose pull waterflow agent; then
        log_info "镜像拉取成功"
    else
        log_error "镜像拉取失败"
        exit 1
    fi
}

# 启动服务
start_services() {
    log_info "启动 Waterflow 服务..."
    cd "${DEPLOY_DIR}"
    docker-compose up -d waterflow agent-linux-1
    
    log_info "等待服务启动 (30 秒)..."
    sleep 30
}

# 验证升级
verify_upgrade() {
    log_info "验证升级结果..."
    
    # 检查容器状态
    if ! docker-compose -f "${COMPOSE_FILE}" ps waterflow | grep -q "Up"; then
        log_error "Waterflow 容器未运行"
        return 1
    fi
    
    # 检查健康状态
    HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health || echo "000")
    
    if [ "${HEALTH_STATUS}" = "200" ]; then
        log_info "健康检查通过"
    else
        log_error "健康检查失败 (HTTP ${HEALTH_STATUS})"
        return 1
    fi
    
    # 获取新版本
    NEW_VERSION=$(docker-compose -f "${COMPOSE_FILE}" exec -T waterflow /app/server --version 2>/dev/null || echo "unknown")
    log_info "新版本: ${NEW_VERSION}"
    
    log_info "升级验证通过"
    return 0
}

# 回滚
rollback() {
    log_error "升级失败，开始回滚..."
    
    cd "${DEPLOY_DIR}"
    
    # 停止新版本
    docker-compose stop waterflow agent-linux-1
    
    # 恢复旧版本镜像（需要在升级前标记）
    log_warn "请手动恢复到之前的镜像版本"
    log_warn "docker-compose down && docker-compose up -d"
    
    exit 1
}

# 清理旧镜像
cleanup_old_images() {
    log_info "清理未使用的 Docker 镜像..."
    docker image prune -f
    log_info "清理完成"
}

# 主函数
main() {
    log_info "========================================="
    log_info "Waterflow Docker Upgrade"
    log_info "========================================="
    
    check_docker
    check_deploy_dir
    get_current_version
    
    log_warn "========================================="
    log_warn "警告: 即将升级 Waterflow"
    log_warn "服务将短暂中断"
    log_warn "========================================="
    
    read -p "确认继续升级? (yes/no): " -r
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log_info "升级已取消"
        exit 0
    fi
    
    create_backup
    stop_services
    pull_images
    start_services
    
    if verify_upgrade; then
        cleanup_old_images
        
        log_info "========================================="
        log_info "升级成功!"
        log_info "旧版本: ${CURRENT_VERSION}"
        log_info "新版本: ${NEW_VERSION}"
        log_info "========================================="
    else
        rollback
    fi
}

# 执行主函数
main "$@"

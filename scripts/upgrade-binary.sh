#!/bin/bash
#
# Waterflow Binary Upgrade Script
# 升级二进制部署的 Waterflow

set -euo pipefail

# 配置
INSTALL_DIR="${INSTALL_DIR:-/opt/waterflow}"
BIN_DIR="${INSTALL_DIR}/bin"
CONFIG_DIR="${CONFIG_DIR:-/etc/waterflow}"
BACKUP_DIR="${BACKUP_DIR:-/data/waterflow/backups}"
DOWNLOAD_URL="${DOWNLOAD_URL:-https://github.com/websoft9/waterflow/releases/latest/download}"

# 服务名称
SERVER_SERVICE="waterflow-server"
AGENT_SERVICE="waterflow-agent"

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

# 检查权限
check_permission() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 root 或 sudo 运行此脚本"
        exit 1
    fi
}

# 获取系统架构
get_arch() {
    ARCH=$(uname -m)
    case ${ARCH} in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64)
            ARCH="arm64"
            ;;
        *)
            log_error "不支持的架构: ${ARCH}"
            exit 1
            ;;
    esac
    log_info "系统架构: ${ARCH}"
}

# 获取当前版本
get_current_version() {
    if [ -f "${BIN_DIR}/server" ]; then
        CURRENT_VERSION=$(${BIN_DIR}/server --version 2>/dev/null || echo "unknown")
        log_info "当前版本: ${CURRENT_VERSION}"
    else
        log_warn "未找到现有安装"
        CURRENT_VERSION="none"
    fi
}

# 创建备份
create_backup() {
    log_info "备份当前二进制文件..."
    
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_BIN_DIR="${BACKUP_DIR}/binaries_${TIMESTAMP}"
    
    mkdir -p "${BACKUP_BIN_DIR}"
    
    if [ -f "${BIN_DIR}/server" ]; then
        cp "${BIN_DIR}/server" "${BACKUP_BIN_DIR}/"
        log_info "已备份: server"
    fi
    
    if [ -f "${BIN_DIR}/agent" ]; then
        cp "${BIN_DIR}/agent" "${BACKUP_BIN_DIR}/"
        log_info "已备份: agent"
    fi
    
    if [ -f "${BIN_DIR}/waterflow" ]; then
        cp "${BIN_DIR}/waterflow" "${BACKUP_BIN_DIR}/"
        log_info "已备份: waterflow"
    fi
    
    log_info "备份保存到: ${BACKUP_BIN_DIR}"
}

# 停止服务
stop_services() {
    log_info "停止 Waterflow 服务..."
    
    systemctl stop ${SERVER_SERVICE} 2>/dev/null || log_warn "Server 服务未运行"
    systemctl stop ${AGENT_SERVICE} 2>/dev/null || log_warn "Agent 服务未运行"
    
    log_info "服务已停止"
}

# 下载新版本
download_binaries() {
    log_info "下载最新版本..."
    
    TEMP_DIR=$(mktemp -d)
    cd "${TEMP_DIR}"
    
    # 下载 Server
    log_info "下载 waterflow-server-linux-${ARCH}..."
    if curl -L -o server "${DOWNLOAD_URL}/waterflow-server-linux-${ARCH}"; then
        chmod +x server
        log_info "Server 下载成功"
    else
        log_error "Server 下载失败"
        rm -rf "${TEMP_DIR}"
        exit 1
    fi
    
    # 下载 Agent
    log_info "下载 waterflow-agent-linux-${ARCH}..."
    if curl -L -o agent "${DOWNLOAD_URL}/waterflow-agent-linux-${ARCH}"; then
        chmod +x agent
        log_info "Agent 下载成功"
    else
        log_error "Agent 下载失败"
        rm -rf "${TEMP_DIR}"
        exit 1
    fi
    
    # 下载 CLI
    log_info "下载 waterflow-cli-linux-${ARCH}..."
    if curl -L -o waterflow "${DOWNLOAD_URL}/waterflow-cli-linux-${ARCH}"; then
        chmod +x waterflow
        log_info "CLI 下载成功"
    else
        log_warn "CLI 下载失败（非关键）"
    fi
    
    log_info "所有二进制文件下载完成"
}

# 安装新版本
install_binaries() {
    log_info "安装新版本..."
    
    # 安装二进制文件
    cp server "${BIN_DIR}/"
    cp agent "${BIN_DIR}/"
    [ -f waterflow ] && cp waterflow "${BIN_DIR}/"
    
    # 设置权限
    chown root:root "${BIN_DIR}/server" "${BIN_DIR}/agent"
    chmod 755 "${BIN_DIR}/server" "${BIN_DIR}/agent"
    
    if [ -f "${BIN_DIR}/waterflow" ]; then
        chown root:root "${BIN_DIR}/waterflow"
        chmod 755 "${BIN_DIR}/waterflow"
    fi
    
    # 清理临时文件
    cd - > /dev/null
    rm -rf "${TEMP_DIR}"
    
    log_info "新版本安装完成"
}

# 启动服务
start_services() {
    log_info "启动 Waterflow 服务..."
    
    systemctl start ${SERVER_SERVICE}
    systemctl start ${AGENT_SERVICE}
    
    log_info "等待服务启动 (15 秒)..."
    sleep 15
}

# 验证升级
verify_upgrade() {
    log_info "验证升级结果..."
    
    # 检查服务状态
    if ! systemctl is-active --quiet ${SERVER_SERVICE}; then
        log_error "Server 服务未运行"
        return 1
    fi
    
    if ! systemctl is-active --quiet ${AGENT_SERVICE}; then
        log_error "Agent 服务未运行"
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
    NEW_VERSION=$(${BIN_DIR}/server --version 2>/dev/null || echo "unknown")
    log_info "新版本: ${NEW_VERSION}"
    
    log_info "升级验证通过"
    return 0
}

# 回滚
rollback() {
    log_error "升级失败，开始回滚..."
    
    # 停止服务
    systemctl stop ${SERVER_SERVICE} ${AGENT_SERVICE}
    
    # 恢复备份
    if [ -d "${BACKUP_BIN_DIR}" ]; then
        log_info "恢复备份..."
        cp "${BACKUP_BIN_DIR}"/* "${BIN_DIR}/"
        
        # 启动服务
        systemctl start ${SERVER_SERVICE} ${AGENT_SERVICE}
        
        log_info "回滚完成"
    else
        log_error "备份目录不存在，无法回滚"
    fi
    
    exit 1
}

# 主函数
main() {
    log_info "========================================="
    log_info "Waterflow Binary Upgrade"
    log_info "========================================="
    
    check_permission
    get_arch
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
    download_binaries
    install_binaries
    start_services
    
    if verify_upgrade; then
        log_info "========================================="
        log_info "升级成功!"
        log_info "旧版本: ${CURRENT_VERSION}"
        log_info "新版本: ${NEW_VERSION}"
        log_info "备份位置: ${BACKUP_BIN_DIR}"
        log_info "========================================="
    else
        rollback
    fi
}

# 执行主函数
main "$@"

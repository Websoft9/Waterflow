#!/bin/bash
#
# Waterflow Server Installation Script
# 自动化安装 Waterflow Server (二进制部署)

set -euo pipefail

# 配置
INSTALL_DIR="${INSTALL_DIR:-/opt/waterflow}"
BIN_DIR="${INSTALL_DIR}/bin"
CONFIG_DIR="${CONFIG_DIR:-/etc/waterflow}"
LOG_DIR="${LOG_DIR:-/var/log/waterflow}"
DATA_DIR="${DATA_DIR:-/var/lib/waterflow}"
DOWNLOAD_URL="${DOWNLOAD_URL:-https://github.com/websoft9/waterflow/releases/latest/download}"

# 用户和组
WATERFLOW_USER="waterflow"
WATERFLOW_GROUP="waterflow"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 显示 Banner
show_banner() {
    cat << 'EOF'
 _    _       _            __ _                
| |  | |     | |          / _| |               
| |  | | __ _| |_ ___ _ _| |_| | _____      __ 
| |/\| |/ _` | __/ _ \ '__|  _| |/ _ \ \ /\ / / 
\  /\  / (_| | ||  __/ |  | | | | (_) \ V  V /  
 \/  \/ \__,_|\__\___|_|  |_| |_|\___/ \_/\_/   

            Server Installation Script
EOF
}

# 检查权限
check_permission() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 root 或 sudo 运行此脚本"
        exit 1
    fi
}

# 获取系统信息
get_system_info() {
    log_step "检测系统信息..."
    
    OS=$(uname -s)
    ARCH=$(uname -m)
    
    case ${OS} in
        Linux)
            OS="linux"
            ;;
        *)
            log_error "不支持的操作系统: ${OS}"
            exit 1
            ;;
    esac
    
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
    
    log_info "操作系统: ${OS}"
    log_info "架构: ${ARCH}"
}

# 检查依赖
check_dependencies() {
    log_step "检查系统依赖..."
    
    local missing_deps=()
    
    # 必需的命令
    for cmd in curl tar systemctl; do
        if ! command -v ${cmd} &> /dev/null; then
            missing_deps+=("${cmd}")
        fi
    done
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_error "缺少依赖: ${missing_deps[*]}"
        log_info "请先安装缺失的依赖"
        exit 1
    fi
    
    log_info "所有依赖已满足"
}

# 创建用户和组
create_user() {
    log_step "创建 waterflow 用户..."
    
    if id "${WATERFLOW_USER}" &>/dev/null; then
        log_info "用户 ${WATERFLOW_USER} 已存在"
    else
        useradd -r -s /bin/false -M ${WATERFLOW_USER}
        log_info "已创建用户: ${WATERFLOW_USER}"
    fi
}

# 创建目录结构
create_directories() {
    log_step "创建目录结构..."
    
    # 安装目录
    mkdir -p "${BIN_DIR}"
    log_info "创建目录: ${BIN_DIR}"
    
    # 配置目录
    mkdir -p "${CONFIG_DIR}"
    log_info "创建目录: ${CONFIG_DIR}"
    
    # 日志目录
    mkdir -p "${LOG_DIR}"
    chown ${WATERFLOW_USER}:${WATERFLOW_GROUP} "${LOG_DIR}"
    log_info "创建目录: ${LOG_DIR}"
    
    # 数据目录
    mkdir -p "${DATA_DIR}"
    chown ${WATERFLOW_USER}:${WATERFLOW_GROUP} "${DATA_DIR}"
    log_info "创建目录: ${DATA_DIR}"
}

# 下载二进制文件
download_binaries() {
    log_step "下载 Waterflow Server 二进制文件..."
    
    TEMP_DIR=$(mktemp -d)
    cd "${TEMP_DIR}"
    
    # 下载 Server
    BINARY_NAME="waterflow-server-${OS}-${ARCH}"
    log_info "下载: ${BINARY_NAME}..."
    
    if curl -L -o server "${DOWNLOAD_URL}/${BINARY_NAME}"; then
        chmod +x server
        log_info "下载成功"
    else
        log_error "下载失败"
        rm -rf "${TEMP_DIR}"
        exit 1
    fi
    
    # 移动到安装目录
    mv server "${BIN_DIR}/"
    chown root:root "${BIN_DIR}/server"
    chmod 755 "${BIN_DIR}/server"
    
    # 清理
    cd - > /dev/null
    rm -rf "${TEMP_DIR}"
    
    log_info "二进制文件已安装到: ${BIN_DIR}/server"
}

# 创建配置文件
create_config() {
    log_step "创建配置文件..."
    
    CONFIG_FILE="${CONFIG_DIR}/config.yaml"
    
    if [ -f "${CONFIG_FILE}" ]; then
        log_warn "配置文件已存在: ${CONFIG_FILE}"
        log_warn "跳过配置文件创建"
        return
    fi
    
    cat > "${CONFIG_FILE}" << 'EOF'
# Waterflow Server 配置文件

server:
  port: 8080
  host: "0.0.0.0"

temporal:
  host: "localhost:7233"
  namespace: "default"

database:
  enabled: true
  host: "localhost"
  port: 5432
  database: "temporal"
  username: "temporal"
  password: "temporal"

log:
  level: "info"
  format: "json"
  output: "stdout"

metrics:
  enabled: true
  port: 9100

health:
  enabled: true
  timeout: 10s
  temporal_timeout: 5s
  db_timeout: 3s
EOF
    
    chown root:${WATERFLOW_GROUP} "${CONFIG_FILE}"
    chmod 640 "${CONFIG_FILE}"
    
    log_info "配置文件已创建: ${CONFIG_FILE}"
}

# 安装 systemd 服务
install_systemd_service() {
    log_step "安装 systemd 服务..."
    
    SERVICE_FILE="/etc/systemd/system/waterflow-server.service"
    
    # 检查是否有 systemd 服务文件模板
    if [ -f "${INSTALL_DIR}/deployments/systemd/waterflow-server.service" ]; then
        cp "${INSTALL_DIR}/deployments/systemd/waterflow-server.service" "${SERVICE_FILE}"
        log_info "从模板创建服务文件"
    else
        # 创建基本的服务文件
        cat > "${SERVICE_FILE}" << EOF
[Unit]
Description=Waterflow Server - Workflow Orchestration Engine
Documentation=https://github.com/websoft9/waterflow
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${WATERFLOW_USER}
Group=${WATERFLOW_GROUP}
WorkingDirectory=${INSTALL_DIR}

Environment="WATERFLOW_SERVER_PORT=8080"
Environment="WATERFLOW_TEMPORAL_HOST=localhost:7233"
Environment="WATERFLOW_LOG_LEVEL=info"
EnvironmentFile=-/etc/waterflow/server.env

ExecStart=${BIN_DIR}/server --config ${CONFIG_DIR}/config.yaml

Restart=always
RestartSec=10s

LimitNOFILE=65536
LimitNPROC=4096

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${LOG_DIR} ${DATA_DIR}

StandardOutput=journal
StandardError=journal
SyslogIdentifier=waterflow-server

TimeoutStopSec=30s
KillMode=process
KillSignal=SIGTERM

[Install]
WantedBy=multi-user.target
EOF
        log_info "创建服务文件: ${SERVICE_FILE}"
    fi
    
    # 重新加载 systemd
    systemctl daemon-reload
    log_info "systemd 已重新加载"
}

# 启用并启动服务
enable_service() {
    log_step "启用并启动服务..."
    
    # 启用服务（开机自启）
    systemctl enable waterflow-server
    log_info "已启用开机自启"
    
    # 启动服务
    systemctl start waterflow-server
    log_info "服务已启动"
    
    # 等待服务启动
    sleep 5
    
    # 检查服务状态
    if systemctl is-active --quiet waterflow-server; then
        log_info "服务运行正常"
    else
        log_error "服务启动失败"
        log_info "查看日志: journalctl -u waterflow-server -n 50"
        exit 1
    fi
}

# 验证安装
verify_installation() {
    log_step "验证安装..."
    
    # 检查二进制文件
    if [ -x "${BIN_DIR}/server" ]; then
        VERSION=$(${BIN_DIR}/server --version 2>/dev/null || echo "unknown")
        log_info "版本: ${VERSION}"
    else
        log_error "二进制文件不存在或无法执行"
        exit 1
    fi
    
    # 检查服务状态
    if systemctl is-active --quiet waterflow-server; then
        log_info "服务状态: 运行中"
    else
        log_warn "服务状态: 未运行"
    fi
    
    # 检查健康端点
    sleep 10
    HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health 2>/dev/null || echo "000")
    
    if [ "${HEALTH_STATUS}" = "200" ]; then
        log_info "健康检查: 通过"
    else
        log_warn "健康检查: 失败 (HTTP ${HEALTH_STATUS})"
        log_warn "可能需要配置 Temporal 连接"
    fi
}

# 显示安装后信息
show_post_install() {
    log_step "安装完成!"
    
    echo ""
    echo "========================================="
    echo "Waterflow Server 安装信息"
    echo "========================================="
    echo "安装目录:    ${INSTALL_DIR}"
    echo "二进制文件:  ${BIN_DIR}/server"
    echo "配置文件:    ${CONFIG_DIR}/config.yaml"
    echo "日志目录:    ${LOG_DIR}"
    echo "数据目录:    ${DATA_DIR}"
    echo "服务名称:    waterflow-server"
    echo ""
    echo "常用命令:"
    echo "  启动服务:    sudo systemctl start waterflow-server"
    echo "  停止服务:    sudo systemctl stop waterflow-server"
    echo "  重启服务:    sudo systemctl restart waterflow-server"
    echo "  查看状态:    sudo systemctl status waterflow-server"
    echo "  查看日志:    sudo journalctl -u waterflow-server -f"
    echo ""
    echo "健康检查:"
    echo "  curl http://localhost:8080/health"
    echo ""
    echo "下一步:"
    echo "  1. 配置 Temporal 连接: vi ${CONFIG_DIR}/config.yaml"
    echo "  2. 重启服务: sudo systemctl restart waterflow-server"
    echo "  3. 查看文档: https://github.com/websoft9/waterflow"
    echo "========================================="
}

# 主函数
main() {
    show_banner
    echo ""
    
    check_permission
    get_system_info
    check_dependencies
    create_user
    create_directories
    download_binaries
    create_config
    install_systemd_service
    enable_service
    verify_installation
    
    echo ""
    show_post_install
}

# 执行主函数
main "$@"

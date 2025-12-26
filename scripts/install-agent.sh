#!/bin/bash
# Usage: sudo ./install-agent.sh
# Install Waterflow Agent with systemd service

set -e

# 错误清理函数
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        echo "❌ 安装失败 (退出码: $exit_code)"
        echo "提示: 如需清理，请手动执行:"
        echo "  sudo userdel -r waterflow 2>/dev/null"
        echo "  sudo rm -rf /opt/waterflow /etc/waterflow/agent.yaml 2>/dev/null"
    fi
}
trap cleanup EXIT

echo "Installing Waterflow Agent..."

# 1. 创建用户和组
if ! id -u waterflow &>/dev/null; then
    echo "Creating waterflow user..."
    sudo useradd -r -s /bin/false -d /opt/waterflow waterflow
fi

# 2. 创建目录结构
echo "Creating directories..."
sudo mkdir -p /opt/waterflow/{bin,plugins,data}
sudo mkdir -p /etc/waterflow
sudo mkdir -p /var/log/waterflow

# 3. 复制二进制文件
echo "Installing agent binary..."
sudo cp bin/agent /opt/waterflow/bin/
sudo chmod +x /opt/waterflow/bin/agent

# 4. 复制配置文件
if [ ! -f /etc/waterflow/agent.yaml ]; then
    echo "Installing default config..."
    sudo cp config.agent.example.yaml /etc/waterflow/agent.yaml
    echo "⚠️  Please edit /etc/waterflow/agent.yaml to configure Task Queues"
fi

# 5. 设置权限
sudo chown -R waterflow:waterflow /opt/waterflow
sudo chown -R waterflow:waterflow /var/log/waterflow
sudo chmod 640 /etc/waterflow/agent.yaml

# 6. 安装 systemd service
echo "Installing systemd service..."
sudo cp deployments/systemd/waterflow-agent.service /etc/systemd/system/
sudo systemctl daemon-reload

# 7. 启用并启动服务
echo "Starting Waterflow Agent..."
sudo systemctl enable waterflow-agent
sudo systemctl start waterflow-agent

# 8. 检查状态
sleep 2
sudo systemctl status waterflow-agent --no-pager

echo ""
echo "✅ Waterflow Agent installed successfully!"
echo ""
echo "Next steps:"
echo "  1. Edit config: sudo nano /etc/waterflow/agent.yaml"
echo "  2. Restart service: sudo systemctl restart waterflow-agent"
echo "  3. View logs: sudo journalctl -u waterflow-agent -f"

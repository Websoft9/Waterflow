#!/bin/bash
# scripts/deploy-greeter.sh
# 部署 Greeter 插件到 Waterflow Agent

set -e

echo "=== Deploying Greeter Plugin ==="

# 1. 检查编译环境
if ! command -v go &> /dev/null; then
    echo "❌ Go not found. Please install Go 1.22+"
    exit 1
fi

# 2. 编译插件
cd "$(dirname "$0")/../examples/plugins/greeter"
export CGO_ENABLED=1
make build

# 3. 安装到 Agent
if [ -d /opt/waterflow/plugins ]; then
    make install
else
    echo "⚠️  /opt/waterflow/plugins not found, skipping installation"
    echo "   Plugin is at: $(pwd)/greeter.so"
fi

# 4. 重启 Agent (如果服务存在)
if systemctl list-units --type=service | grep -q waterflow-agent; then
    echo "Restarting waterflow-agent..."
    if sudo systemctl restart waterflow-agent; then
        echo "✅ Agent restarted successfully"
        sleep 3
    else
        echo "❌ Agent restart failed, checking logs..."
        sudo journalctl -u waterflow-agent -n 20 --no-pager
        exit 1
    fi
fi

# 5. 验证插件加载
if command -v curl &> /dev/null && [ -f /opt/waterflow/plugins/custom/greeter.so ]; then
    echo "Checking plugin status..."
    if curl -s http://localhost:9091/health 2>/dev/null | grep -q greeter; then
        echo "✅ Greeter plugin loaded successfully"
    else
        echo "⚠️  Plugin may not be loaded yet (check Agent logs)"
        echo "   Run: sudo journalctl -u waterflow-agent | grep greeter"
    fi
fi

echo "✅ Greeter plugin deployment complete"

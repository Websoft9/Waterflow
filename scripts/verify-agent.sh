#!/bin/bash
# Usage: ./verify-agent.sh
# Verify Waterflow Agent installation and health

set -e

# 检查依赖
command -v docker >/dev/null 2>&1 || { echo "❌ 需要 docker 但未安装"; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "⚠️  建议安装 jq 以便解析 JSON (可选)"; }
command -v curl >/dev/null 2>&1 || { echo "❌ 需要 curl 但未安装"; exit 1; }

echo "🔍 验证 Agent 安装..."

# 1. 检查容器状态
if docker ps | grep -q waterflow-agent; then
    echo "✅ Agent 容器运行中"
else
    echo "❌ Agent 容器未运行"
    exit 1
fi

# 2. 检查日志
if docker logs waterflow-agent 2>&1 | grep -q "Worker started successfully"; then
    echo "✅ Worker 启动成功"
else
    echo "❌ Worker 启动失败"
    exit 1
fi

# 3. 检查心跳
if curl -s http://localhost:8080/v1/agents | jq -e '.total > 0' > /dev/null 2>&1; then
    echo "✅ Agent 已注册"
else
    echo "⚠️  Agent 未注册 (可能未配置 SERVER_URL)"
fi

echo "🎉 验证完成!"

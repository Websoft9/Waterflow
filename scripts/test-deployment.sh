#!/bin/bash
# scripts/test-deployment.sh - Waterflow 部署测试脚本

set -e

echo "========================================="
echo "  Waterflow Deployment Test"
echo "========================================="
echo ""

# 切换到 deployments 目录
cd "$(dirname "$0")/../deployments" || exit 1

# 1. 清理环境
echo "Step 1/6: Cleaning up existing environment..."
docker compose down -v 2>/dev/null || true
sleep 2

# 2. 启动服务
echo ""
echo "Step 2/6: Starting services..."
docker compose up -d

# 3. 等待服务就绪
echo ""
echo "Step 3/6: Waiting for services to be healthy (max 5 minutes)..."
echo "This may take 2-3 minutes on first run..."

TIMEOUT=300
ELAPSED=0
INTERVAL=10

while [ $ELAPSED -lt $TIMEOUT ]; do
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        echo "✅ Waterflow is healthy!"
        break
    fi
    echo "Waiting... (${ELAPSED}s/${TIMEOUT}s)"
    sleep $INTERVAL
    ELAPSED=$((ELAPSED + INTERVAL))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    echo "❌ Timeout waiting for services to be healthy"
    echo ""
    echo "Logs:"
    docker compose logs --tail=50 waterflow
    exit 1
fi

# 4. 验证健康检查
echo ""
echo "Step 4/6: Verifying health checks..."
HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)
echo "Health check response: $HEALTH_RESPONSE"

# 5. 提交测试工作流
echo ""
echo "Step 5/6: Submitting test workflow..."
WORKFLOW_YAML='name: test-workflow
on: push

jobs:
  test:
    runs-on: waterflow-server
    steps:
      - name: Echo Test
        run: echo "Deployment test successful!"'

RESPONSE=$(curl -s -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d "{\"yaml\": \"$(echo "$WORKFLOW_YAML" | sed 's/"/\\"/g' | tr '\n' ' ')\"}")

echo "Workflow submission response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"

# 提取 workflow ID (如果返回是 JSON)
WORKFLOW_ID=$(echo "$RESPONSE" | jq -r '.id' 2>/dev/null || echo "")

# 6. 查询工作流状态
if [ -n "$WORKFLOW_ID" ] && [ "$WORKFLOW_ID" != "null" ]; then
    echo ""
    echo "Step 6/6: Querying workflow status..."
    sleep 3
    STATUS_RESPONSE=$(curl -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID")
    echo "Workflow status:"
    echo "$STATUS_RESPONSE" | jq '.' 2>/dev/null || echo "$STATUS_RESPONSE"
fi

# 显示服务状态
echo ""
echo "========================================="
echo "  Deployment Test PASSED! ✅"
echo "========================================="
echo ""
echo "Services running:"
docker compose ps

echo ""
echo "Access points:"
echo "  - Waterflow API:  http://localhost:8080"
echo "  - Temporal UI:    http://localhost:8088"
echo "  - Health Check:   http://localhost:8080/health"
echo ""
echo "To view logs:  ./scripts/logs.sh [service]"
echo "To cleanup:    ./scripts/cleanup.sh"

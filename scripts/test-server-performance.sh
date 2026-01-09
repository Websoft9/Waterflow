#!/bin/bash
set -e

# 检查依赖
if ! command -v bc &> /dev/null; then
  echo "错误: bc 命令未安装，请先安装 (apt install bc 或 yum install bc)"
  exit 1
fi

echo "=== Waterflow Server 性能测试 ==="
echo ""

IMAGE_NAME="${1:-waterflow/server:latest}"
CONTAINER_NAME="waterflow-perf-test"

# 清理旧容器
docker rm -f ${CONTAINER_NAME} 2>/dev/null || true

echo "1. 测试启动时间..."
START_TIME=$(date +%s.%N)

docker run -d \
  --name ${CONTAINER_NAME} \
  -p 18080:8080 \
  -e WATERFLOW_TEMPORAL_HOST=temporal-mock:7233 \
  ${IMAGE_NAME}

# 等待健康检查通过
MAX_WAIT=30
ELAPSED=0
while [ $ELAPSED -lt $MAX_WAIT ]; do
  HEALTH=$(docker inspect ${CONTAINER_NAME} --format='{{.State.Health.Status}}' 2>/dev/null || echo "starting")
  if [ "$HEALTH" = "healthy" ]; then
    break
  fi
  sleep 1
  ELAPSED=$((ELAPSED + 1))
done

END_TIME=$(date +%s.%N)
STARTUP_TIME=$(echo "$END_TIME - $START_TIME" | bc)

echo "   启动时间: ${STARTUP_TIME} 秒"
if (( $(echo "$STARTUP_TIME < 5.0" | bc -l) )); then
  echo "   ✅ PASS (< 5s)"
else
  echo "   ❌ FAIL (>= 5s)"
fi
echo ""

echo "2. 测试空闲内存占用..."
sleep 5  # 等待稳定

MEM_USAGE=$(docker stats ${CONTAINER_NAME} --no-stream --format "{{.MemUsage}}" | awk '{print $1}')
MEM_VALUE=$(echo $MEM_USAGE | sed 's/MiB//')

echo "   内存占用: ${MEM_USAGE}"
if (( $(echo "$MEM_VALUE < 100" | bc -l) )); then
  echo "   ✅ PASS (< 100MB)"
else
  echo "   ❌ FAIL (>= 100MB)"
fi
echo ""

echo "3. 测试镜像大小..."
IMAGE_SIZE=$(docker images ${IMAGE_NAME} --format "{{.Size}}")
echo "   镜像大小: ${IMAGE_SIZE}"
echo ""

echo "4. 清理测试容器..."
docker rm -f ${CONTAINER_NAME}

echo ""
echo "=== 测试完成 ==="

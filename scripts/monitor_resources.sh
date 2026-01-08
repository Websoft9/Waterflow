#!/bin/bash
# Resource monitoring script
# Monitors CPU, memory, connections, and goroutines

RESULTS_DIR=${1:-"."}
INTERVAL=${2:-5}

echo "Resource monitoring started (interval: ${INTERVAL}s)"
echo "Results dir: $RESULTS_DIR"

# 确保结果目录存在
mkdir -p "$RESULTS_DIR"

# 初始化日志文件
echo "# Timestamp,CPU(%)" > "$RESULTS_DIR/cpu.log"
echo "# Timestamp,Memory(MB)" > "$RESULTS_DIR/memory.log"
echo "# Timestamp,Connections" > "$RESULTS_DIR/connections.log"
echo "# Timestamp,Goroutines" > "$RESULTS_DIR/goroutines.log"

while true; do
    TIMESTAMP=$(date +%s)
    
    # CPU 使用率 (所有核心平均)
    if command -v top > /dev/null; then
        CPU=$(top -bn1 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | awk '{print 100 - $1}')
        if [ -n "$CPU" ]; then
            echo "$CPU" >> "$RESULTS_DIR/cpu.log"
        fi
    fi
    
    # 内存使用 (MB)
    if command -v free > /dev/null; then
        MEM=$(free -m | grep Mem | awk '{print $3}')
        echo "$MEM" >> "$RESULTS_DIR/memory.log"
    fi
    
    # TCP 连接数
    if command -v ss > /dev/null; then
        CONNS=$(ss -tan | grep ESTAB | wc -l)
        echo "$CONNS" >> "$RESULTS_DIR/connections.log"
    elif command -v netstat > /dev/null; then
        CONNS=$(netstat -an | grep ESTABLISHED | wc -l)
        echo "$CONNS" >> "$RESULTS_DIR/connections.log"
    fi
    
    # Goroutine 数量 (从 pprof)
    if command -v curl > /dev/null; then
        GOROUTINES=$(curl -s http://localhost:8080/debug/pprof/goroutine?debug=1 2>/dev/null \
            | grep "^goroutine profile:" | awk '{print $3}' || echo "")
        if [ -n "$GOROUTINES" ]; then
            echo "$GOROUTINES" >> "$RESULTS_DIR/goroutines.log"
        fi
    fi
    
    sleep $INTERVAL
done

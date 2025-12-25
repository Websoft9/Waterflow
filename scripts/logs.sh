#!/bin/bash
# scripts/logs.sh - Waterflow 日志查看脚本

SERVICE="$1"

if [ -z "$SERVICE" ]; then
    echo "Usage: ./scripts/logs.sh [service]"
    echo ""
    echo "Available services:"
    echo "  waterflow       - Waterflow Server"
    echo "  temporal        - Temporal Server"
    echo "  temporal-ui     - Temporal Web UI"
    echo "  postgresql      - PostgreSQL Database"
    echo "  all             - All services"
    echo ""
    echo "Examples:"
    echo "  ./scripts/logs.sh waterflow"
    echo "  ./scripts/logs.sh all"
    exit 1
fi

cd "$(dirname "$0")/../deployments" || exit 1

if [ "$SERVICE" = "all" ]; then
    echo "Showing logs for all services..."
    docker compose logs -f
else
    echo "Showing logs for $SERVICE..."
    docker compose logs -f "$SERVICE"
fi

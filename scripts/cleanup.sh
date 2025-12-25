#!/bin/bash
# scripts/cleanup.sh - Waterflow 环境清理脚本

set -e

echo "========================================="
echo "  Waterflow Environment Cleanup"
echo "========================================="
echo ""
echo "This will:"
echo "  1. Stop all services"
echo "  2. Remove containers"
echo "  3. Remove volumes (ALL DATA WILL BE LOST)"
echo "  4. Remove images"
echo ""

read -p "Are you sure you want to continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled."
    exit 0
fi

echo ""
echo "Stopping all services..."
cd "$(dirname "$0")/../deployments" || exit 1
docker compose down

echo "Removing volumes..."
docker compose down -v

echo "Removing Waterflow image..."
docker rmi waterflow-waterflow:latest 2>/dev/null || echo "Image not found, skipping..."

echo ""
echo "========================================="
echo "  Cleanup Complete!"
echo "========================================="
echo ""
echo "To redeploy, run:"
echo "  cd deployments && docker compose up -d"

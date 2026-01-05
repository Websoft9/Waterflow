#!/bin/bash
# Simple test for waterflow node list command (Story 5.6)

set +e

CLI="./bin/waterflow"

echo "================================================================"
echo "Waterflow CLI - Node List Command Basic Test (Story 5.6)"
echo "================================================================"

echo -e "\nTest 1: Show help"
$CLI node list --help | grep -q "List all available"
if [ $? -eq 0 ]; then
    echo "✓ Help显示正常"
else
    echo "✗ Help失败"
fi

echo -e "\nTest 2: Test command structure"
$CLI node --help | grep -q "list"
if [ $? -eq 0 ]; then
    echo "✓ node子命令注册成功"
else
    echo "✗ node子命令注册失败"
fi

echo -e "\n================================================================"
echo "注意: Server API集成测试需要Server运行"
echo "Story 5.6 CLI框架已完成,Server API待集成测试"
echo "================================================================"

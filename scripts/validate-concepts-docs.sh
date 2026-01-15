#!/bin/bash

# Story 10-6: 核心架构概念文档验证脚本
# 验证所有概念文档是否满足质量标准

DOCS_DIR="docs/concepts"
PASS=0
FAIL=0

echo "========================================="
echo "📋 核心架构概念文档验证"
echo "========================================="
echo ""

# 检查文档是否存在
echo "🔍 检查文档是否存在..."
REQUIRED_DOCS=(
    "event-sourcing-execution-model.md"
    "single-node-execution-pattern.md"
    "plugin-node-system.md"
    "task-queue-routing.md"
    "integration-interfaces.md"
    "README.md"
)

for doc in "${REQUIRED_DOCS[@]}"; do
    if [ -f "$DOCS_DIR/$doc" ]; then
        echo "  ✅ $doc 存在"
        PASS=$((PASS + 1))
    else
        echo "  ❌ $doc 缺失"
        FAIL=$((FAIL + 1))
    fi
done

echo ""

# 检查每个文档的字数 (800+ 词)
echo "📝 检查文档字数 (要求: 800+ 词)..."
for doc in "${REQUIRED_DOCS[@]}"; do
    if [ "$doc" == "README.md" ]; then
        continue
    fi
    
    if [ -f "$DOCS_DIR/$doc" ]; then
        WORD_COUNT=$(wc -w < "$DOCS_DIR/$doc")
        if [ "$WORD_COUNT" -ge 800 ]; then
            echo "  ✅ $doc: $WORD_COUNT 词 (达标)"
            PASS=$((PASS + 1))
        else
            echo "  ❌ $doc: $WORD_COUNT 词 (未达标,需要800+词)"
            FAIL=$((FAIL + 1))
        fi
    fi
done

echo ""

# 检查每个文档的Mermaid图表数量 (2+)
echo "📊 检查Mermaid图表数量 (要求: 2+)..."
for doc in "${REQUIRED_DOCS[@]}"; do
    if [ "$doc" == "README.md" ]; then
        continue
    fi
    
    if [ -f "$DOCS_DIR/$doc" ]; then
        DIAGRAM_COUNT=$(grep -c '```mermaid' "$DOCS_DIR/$doc" || echo "0")
        if [ "$DIAGRAM_COUNT" -ge 2 ]; then
            echo "  ✅ $doc: $DIAGRAM_COUNT 个图表 (达标)"
            PASS=$((PASS + 1))
        else
            echo "  ⚠️  $doc: $DIAGRAM_COUNT 个图表 (建议2+)"
            PASS=$((PASS + 1))
        fi
    fi
done

echo ""

# 检查每个文档的代码示例 (1+)
echo "💻 检查代码示例数量 (要求: 1+)..."
for doc in "${REQUIRED_DOCS[@]}"; do
    if [ "$doc" == "README.md" ]; then
        continue
    fi
    
    if [ -f "$DOCS_DIR/$doc" ]; then
        CODE_COUNT=$(grep -c '```\(yaml\|go\|json\|bash\)' "$DOCS_DIR/$doc" || echo "0")
        if [ "$CODE_COUNT" -ge 1 ]; then
            echo "  ✅ $doc: $CODE_COUNT 个代码示例 (达标)"
            PASS=$((PASS + 1))
        else
            echo "  ❌ $doc: $CODE_COUNT 个代码示例 (未达标,需要1+示例)"
            FAIL=$((FAIL + 1))
        fi
    fi
done

echo ""

# 检查每个文档的ADR引用 (2+)
echo "🔗 检查ADR交叉引用 (要求: 2+)..."
for doc in "${REQUIRED_DOCS[@]}"; do
    if [ "$doc" == "README.md" ]; then
        continue
    fi
    
    if [ -f "$DOCS_DIR/$doc" ]; then
        ADR_COUNT=$(grep -c 'ADR-' "$DOCS_DIR/$doc" || echo "0")
        if [ "$ADR_COUNT" -ge 2 ]; then
            echo "  ✅ $doc: $ADR_COUNT 个ADR引用 (达标)"
            PASS=$((PASS + 1))
        else
            echo "  ⚠️  $doc: $ADR_COUNT 个ADR引用 (建议2+)"
            PASS=$((PASS + 1))
        fi
    fi
done

echo ""

# 检查README.md索引内容
echo "📚 检查concepts/README.md索引质量..."
if [ -f "$DOCS_DIR/README.md" ]; then
    if grep -q "学习路径\|Learning Path" "$DOCS_DIR/README.md"; then
        echo "  ✅ 包含学习路径"
        PASS=$((PASS + 1))
    else
        echo "  ❌ 缺少学习路径"
        FAIL=$((FAIL + 1))
    fi
    
    if grep -q "ADR 映射\|ADR Mapping" "$DOCS_DIR/README.md"; then
        echo "  ✅ 包含ADR映射"
        PASS=$((PASS + 1))
    else
        echo "  ⚠️  建议添加ADR映射"
        PASS=$((PASS + 1))
    fi
    
    if grep -q "event-sourcing-execution-model\|single-node-execution-pattern" "$DOCS_DIR/README.md"; then
        echo "  ✅ 包含文档摘要"
        PASS=$((PASS + 1))
    else
        echo "  ❌ 缺少文档摘要"
        FAIL=$((FAIL + 1))
    fi
fi

echo ""

# 检查主README.md是否引用concepts
echo "🏠 检查主README.md是否引用concepts..."
if grep -q "docs/concepts\|Core Concepts" "README.md"; then
    echo "  ✅ README.md已引用concepts"
    PASS=$((PASS + 1))
else
    echo "  ❌ README.md未引用concepts"
    FAIL=$((FAIL + 1))
fi

echo ""
echo "========================================="
echo "📊 验证结果"
echo "========================================="
echo "✅ 通过: $PASS"
echo "❌ 失败: $FAIL"

if [ "$FAIL" -eq 0 ]; then
    echo ""
    echo "🎉 所有验证通过！文档质量符合标准。"
    exit 0
else
    echo ""
    echo "⚠️  发现 $FAIL 个问题,请修复后重新验证。"
    exit 1
fi

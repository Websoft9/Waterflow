#!/bin/bash
# Waterflow 审计报告生成脚本
#
# 生成月度审计报告用于合规
#
# 使用方法:
#   ./audit-report.sh [output_file]

set -e

OUTPUT_FILE=${1:-"audit-report-$(date +%Y-%m).md"}
START_DATE=$(date -d "1 month ago" +%Y-%m-%d)
END_DATE=$(date +%Y-%m-%d)
TEMP_FILE="/tmp/audit-data-$(date +%Y%m%d%H%M%S).json"

echo "生成审计报告..."
echo "报告期间: $START_DATE 至 $END_DATE"
echo "输出文件: $OUTPUT_FILE"
echo ""

# 导出审计日志
waterflow audit query \
  --start-time="$START_DATE" \
  --end-time="$END_DATE" \
  --format=json > "$TEMP_FILE"

# 生成报告
cat > "$OUTPUT_FILE" <<EOF
# Waterflow 审计报告

**报告期间:** $START_DATE 至 $END_DATE  
**生成日期:** $(date)

## 1. 审计日志统计

### 总事件数
\`\`\`
$(jq '. | length' "$TEMP_FILE") 条记录
\`\`\`

### 按类别分布
\`\`\`
$(jq -r '.[] | .event_category' "$TEMP_FILE" | sort | uniq -c | sort -rn)
\`\`\`

### 按结果分布
\`\`\`
$(jq -r '.[] | .result' "$TEMP_FILE" | sort | uniq -c)
\`\`\`

## 2. 认证事件

### 认证成功/失败
\`\`\`
$(jq -r '.[] | select(.event_category == "auth") | .result' "$TEMP_FILE" | sort | uniq -c)
\`\`\`

### 失败的认证尝试
\`\`\`
$(jq -r '.[] | select(.event_category == "auth" and .result == "failure") | .user.ip' "$TEMP_FILE" | sort | uniq -c | sort -rn | head -10)
\`\`\`

## 3. 密钥访问

### 密钥访问统计
\`\`\`
$(jq -r '.[] | select(.event_category == "secret") | .resource.resource_id' "$TEMP_FILE" | sort | uniq -c | sort -rn)
\`\`\`

### 密钥访问拒绝
\`\`\`
$(jq -r '.[] | select(.event_category == "secret" and .result == "permission_denied")' "$TEMP_FILE" | jq -s '. | length') 次
\`\`\`

## 4. 工作流操作

### 工作流提交数
\`\`\`
$(jq -r '.[] | select(.event_type == "workflow.submit")' "$TEMP_FILE" | jq -s '. | length') 次
\`\`\`

### 工作流取消数
\`\`\`
$(jq -r '.[] | select(.event_type == "workflow.cancel")' "$TEMP_FILE" | jq -s '. | length') 次
\`\`\`

EOF

# 清理临时文件
rm -f "$TEMP_FILE"

echo "✅ 审计报告已生成: $OUTPUT_FILE"
cat "$OUTPUT_FILE"

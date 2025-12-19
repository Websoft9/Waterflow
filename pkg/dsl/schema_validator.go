package dsl

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed schema/workflow-schema.json
var schemaJSON []byte

// SchemaValidator JSON Schema 验证器
type SchemaValidator struct {
	schema *gojsonschema.Schema
}

// NewSchemaValidator 创建 Schema 验证器
func NewSchemaValidator() (*SchemaValidator, error) {
	// 从嵌入文件加载 schema
	schemaLoader := gojsonschema.NewBytesLoader(schemaJSON)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	return &SchemaValidator{schema: schema}, nil
}

// Validate 验证 Workflow 结构
// 注意：这里接受原始 YAML 内容而不是解析后的结构体，以避免 Go tag 转换问题
func (v *SchemaValidator) ValidateYAML(content []byte, workflow *Workflow) error {
	// 将 YAML 转换为通用 map 结构
	var data map[string]interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// 使用 map 进行验证（保留原始字段名）
	documentLoader := gojsonschema.NewGoLoader(data)

	result, err := v.schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	if !result.Valid() {
		return v.convertSchemaErrors(result.Errors(), workflow, content)
	}

	return nil
}

// convertSchemaErrors 转换 schema 错误为 ValidationError
func (v *SchemaValidator) convertSchemaErrors(errs []gojsonschema.ResultError, workflow *Workflow, content []byte) error {
	fieldErrors := make([]FieldError, 0, len(errs))

	for _, err := range errs {
		fieldErr := FieldError{
			Field:      err.Field(),
			Error:      err.Description(),
			Suggestion: v.generateSuggestion(err),
		}

		// 尝试从 LineMap 获取行号
		if workflow.LineMap != nil {
			if line, ok := workflow.LineMap[err.Field()]; ok {
				fieldErr.Line = line
				// 添加代码片段
				fieldErr.Snippet = extractCodeSnippet(content, line, 2)
			}
		}

		fieldErrors = append(fieldErrors, fieldErr)
	}

	return &ValidationError{
		Type:   "schema_validation_error",
		Detail: fmt.Sprintf("Found %d schema validation errors", len(fieldErrors)),
		Errors: fieldErrors,
	}
}

// generateSuggestion 根据错误类型生成修复建议
func (v *SchemaValidator) generateSuggestion(err gojsonschema.ResultError) string {
	errType := err.Type()
	desc := strings.ToLower(err.Description())

	switch errType {
	case "required":
		return fmt.Sprintf("Add the required field. Example: %s: <value>", err.Field())

	case "invalid_type":
		return "Check the field type matches the expected type in the schema."

	case "string_gte":
		return "String length must be greater than or equal to the minimum length."

	case "string_lte":
		return "String length must be less than or equal to the maximum length."

	case "pattern":
		return "Value must match the required pattern format."

	case "number_gte":
		return "Number must be greater than or equal to the minimum value."

	case "number_lte":
		return "Number must be less than or equal to the maximum value."

	case "array_min_items":
		return "Array must contain at least the minimum number of items."

	case "additional_property_not_allowed":
		field := err.Field()
		return fmt.Sprintf("Field '%s' is not allowed. Check for typos or refer to the schema documentation.", field)

	default:
		if strings.Contains(desc, "does not match pattern") {
			return "Value format is invalid. Check the expected format in the documentation."
		}
		return "Check the field value matches the schema requirements."
	}
}

// extractCodeSnippet 提取代码片段 (包含上下文)
func extractCodeSnippet(content []byte, lineNum int, contextLines int) string {
	lines := bytes.Split(content, []byte("\n"))
	if lineNum <= 0 || lineNum > len(lines) {
		return ""
	}

	// 计算起始和结束行
	start := lineNum - contextLines - 1
	end := lineNum + contextLines
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	var buf strings.Builder
	for i := start; i < end; i++ {
		// 标记错误行
		marker := "  "
		if i == lineNum-1 {
			marker = "→ "
		}
		_, _ = buf.WriteString(fmt.Sprintf("%s%3d | %s\n", marker, i+1, lines[i]))
	}

	return buf.String()
}

package dsl

import (
	"fmt"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/dsl/node/builtin"
	"go.uber.org/zap"
)

// Validator 验证器门面
type Validator struct {
	parser            *Parser
	schemaValidator   *SchemaValidator
	semanticValidator *SemanticValidator
	logger            *zap.Logger
}

// NewValidator 创建验证器
func NewValidator(logger *zap.Logger) (*Validator, error) {
	// 创建节点注册表并注册内置节点
	nodeRegistry := node.NewRegistry()

	if err := nodeRegistry.Register(&builtin.CheckoutNode{}); err != nil {
		return nil, fmt.Errorf("failed to register checkout node: %w", err)
	}
	if err := nodeRegistry.Register(&builtin.RunNode{}); err != nil {
		return nil, fmt.Errorf("failed to register run node: %w", err)
	}

	// 创建各个验证器
	schemaValidator, err := NewSchemaValidator()
	if err != nil {
		return nil, fmt.Errorf("failed to create schema validator: %w", err)
	}

	return &Validator{
		parser:            NewParser(logger),
		schemaValidator:   schemaValidator,
		semanticValidator: NewSemanticValidator(nodeRegistry),
		logger:            logger,
	}, nil
}

// ValidateYAML 完整验证流程
func (v *Validator) ValidateYAML(content []byte) (*Workflow, error) {
	// 0. 检查 YAML 文件大小 (防护 YAML Bomb)
	const maxYAMLSize = 10 * 1024 * 1024 // 10MB
	if len(content) > maxYAMLSize {
		return nil, &ValidationError{
			Type:   "validation_error",
			Detail: "YAML file size exceeds limit",
			Errors: []FieldError{{
				Error:      fmt.Sprintf("file size %d bytes exceeds limit %d bytes (10MB)", len(content), maxYAMLSize),
				Suggestion: "Reduce YAML file size or split into multiple workflows",
			}},
		}
	}

	// 1. YAML 语法解析
	workflow, err := v.parser.Parse(content)
	if err != nil {
		return nil, err // 语法错误时直接返回
	}

	var allErrors []FieldError

	// 2. JSON Schema 结构验证
	if err := v.schemaValidator.ValidateYAML(content, workflow); err != nil {
		if validationErr, ok := err.(*ValidationError); ok {
			allErrors = append(allErrors, validationErr.Errors...)
		}
	}

	// 3. 语义验证
	if err := v.semanticValidator.Validate(workflow, content); err != nil {
		if validationErr, ok := err.(*ValidationError); ok {
			allErrors = append(allErrors, validationErr.Errors...)
		}
	}

	// 4. 返回收集的错误
	if len(allErrors) > 0 {
		// 限制错误数量
		if len(allErrors) > 20 {
			allErrors = allErrors[:20]
		}

		return nil, &ValidationError{
			Type:   "validation_error",
			Detail: fmt.Sprintf("Found %d validation errors", len(allErrors)),
			Errors: allErrors,
		}
	}

	v.logger.Info("Workflow validated successfully",
		zap.String("workflow", workflow.Name),
		zap.Int("jobs", len(workflow.Jobs)),
	)

	return workflow, nil
}

// GetSchemaJSON returns the embedded JSON schema
func (v *Validator) GetSchemaJSON() ([]byte, error) {
	return v.schemaValidator.GetSchemaJSON()
}

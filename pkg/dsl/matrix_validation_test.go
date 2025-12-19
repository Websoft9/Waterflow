package dsl

import (
	"strings"
	"testing"

	"github.com/Websoft9/waterflow/pkg/node"
	"github.com/Websoft9/waterflow/pkg/node/builtin"
	"github.com/stretchr/testify/assert"
)

// setupTestRegistry 设置测试用registry
func setupTestRegistry() *node.Registry {
	registry := node.NewRegistry()
	// 注册基本节点用于测试
	_ = registry.Register(&builtin.RunNode{})
	_ = registry.Register(&builtin.CheckoutNode{})
	return registry
}

// TestSemanticValidator_ValidMatrix 测试有效Matrix验证
func TestSemanticValidator_ValidMatrix(t *testing.T) {
	registry := setupTestRegistry()
	validator := NewSemanticValidator(registry)

	workflow := &Workflow{
		Name: "test",
		Jobs: map[string]*Job{
			"deploy": {
				Name:    "deploy",
				RunsOn:  "linux-amd64",
				LineNum: 5,
				Strategy: &Strategy{
					Matrix: map[string][]interface{}{
						"server": {"web1", "web2", "web3"},
					},
				},
				Steps: []*Step{
					{
						Uses: "run@v1",
						With: map[string]interface{}{
							"command": "echo test",
						},
					},
				},
			},
		},
	}

	err := validator.Validate(workflow, []byte(""))
	if err != nil {
		t.Logf("Error details: %v", err)
		valErr, ok := err.(*ValidationError)
		if ok {
			t.Logf("Validation errors: %+v", valErr.Errors)
		}
	}
	assert.NoError(t, err)
}

// TestSemanticValidator_EmptyMatrixDimension 测试空维度错误
func TestSemanticValidator_EmptyMatrixDimension(t *testing.T) {
	registry := setupTestRegistry()
	validator := NewSemanticValidator(registry)

	workflow := &Workflow{
		Name: "test",
		Jobs: map[string]*Job{
			"deploy": {
				Name:    "deploy",
				RunsOn:  "linux-amd64",
				LineNum: 5,
				Strategy: &Strategy{
					Matrix: map[string][]interface{}{
						"server": {}, // 空数组
					},
				},
				Steps: []*Step{
					{
						Uses: "run@v1",
						With: map[string]interface{}{
							"command": "echo test",
						},
					},
				},
			},
		},
	}

	err := validator.Validate(workflow, []byte(""))
	if err != nil {
		t.Logf("Error details: %v", err)
		valErr, ok := err.(*ValidationError)
		if ok {
			t.Logf("Validation errors: %+v", valErr.Errors)
		}
	}
	assert.Error(t, err)

	valErr, ok := err.(*ValidationError)
	assert.True(t, ok)
	if ok && len(valErr.Errors) > 0 {
		found := false
		for _, e := range valErr.Errors {
			if strings.Contains(e.Error, "matrix dimension is empty") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should contain 'matrix dimension is empty' error")
	}
}

// TestSemanticValidator_MatrixCombinationsExceedLimit 测试组合数超限
func TestSemanticValidator_MatrixCombinationsExceedLimit(t *testing.T) {
	registry := setupTestRegistry()
	validator := NewSemanticValidator(registry)

	// 创建超过256组合的矩阵: 10 * 10 * 3 = 300
	matrix := make(map[string][]interface{})

	matrix["a"] = make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		matrix["a"][i] = i
	}

	matrix["b"] = make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		matrix["b"][i] = i
	}

	matrix["c"] = []interface{}{1, 2, 3}

	workflow := &Workflow{
		Name: "test",
		Jobs: map[string]*Job{
			"test": {
				Name:    "test",
				RunsOn:  "linux-amd64",
				LineNum: 5,
				Strategy: &Strategy{
					Matrix: matrix,
				},
				Steps: []*Step{
					{
						Uses: "run@v1",
						With: map[string]interface{}{
							"command": "echo test",
						},
					},
				},
			},
		},
	}

	err := validator.Validate(workflow, []byte(""))
	if err != nil {
		t.Logf("Error details: %v", err)
		valErr, ok := err.(*ValidationError)
		if ok {
			t.Logf("Validation errors: %+v", valErr.Errors)
		}
	}
	assert.Error(t, err)

	valErr, ok := err.(*ValidationError)
	assert.True(t, ok)
	if ok && len(valErr.Errors) > 0 {
		found := false
		for _, e := range valErr.Errors {
			if strings.Contains(e.Error, "exceed limit 256") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should contain 'exceed limit 256' error")
	}
}

// TestSemanticValidator_IncludeNotSupported 测试include不支持错误
func TestSemanticValidator_IncludeNotSupported(t *testing.T) {
	registry := setupTestRegistry()
	validator := NewSemanticValidator(registry)

	workflow := &Workflow{
		Name: "test",
		Jobs: map[string]*Job{
			"test": {
				Name:    "test",
				RunsOn:  "linux-amd64",
				LineNum: 5,
				Strategy: &Strategy{
					Matrix: map[string][]interface{}{
						"os": {"ubuntu", "debian"},
					},
					Include: []map[string]interface{}{
						{"os": "alpine"},
					},
				},
				Steps: []*Step{
					{
						Uses: "run@v1",
						With: map[string]interface{}{
							"command": "echo test",
						},
					},
				},
			},
		},
	}

	err := validator.Validate(workflow, []byte(""))
	assert.Error(t, err)

	valErr, ok := err.(*ValidationError)
	assert.True(t, ok)
	if ok && len(valErr.Errors) > 0 {
		found := false
		for _, e := range valErr.Errors {
			if strings.Contains(e.Error, "include not supported in MVP") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should contain 'include not supported in MVP' error")
	}
}

// TestSemanticValidator_ExcludeNotSupported 测试exclude不支持错误
func TestSemanticValidator_ExcludeNotSupported(t *testing.T) {
	registry := setupTestRegistry()
	validator := NewSemanticValidator(registry)

	workflow := &Workflow{
		Name: "test",
		Jobs: map[string]*Job{
			"test": {
				Name:    "test",
				RunsOn:  "linux-amd64",
				LineNum: 5,
				Strategy: &Strategy{
					Matrix: map[string][]interface{}{
						"os": {"ubuntu", "debian"},
					},
					Exclude: []map[string]interface{}{
						{"os": "debian"},
					},
				},
				Steps: []*Step{
					{
						Uses: "run@v1",
						With: map[string]interface{}{
							"command": "echo test",
						},
					},
				},
			},
		},
	}

	err := validator.Validate(workflow, []byte(""))
	assert.Error(t, err)

	valErr, ok := err.(*ValidationError)
	assert.True(t, ok)
	if ok && len(valErr.Errors) > 0 {
		found := false
		for _, e := range valErr.Errors {
			if strings.Contains(e.Error, "exclude not supported in MVP") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should contain 'exclude not supported in MVP' error")
	}
}

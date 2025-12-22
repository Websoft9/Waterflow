package builtin

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/stretchr/testify/assert"
)

func TestCheckoutNode_Name(t *testing.T) {
	n := &CheckoutNode{}
	assert.Equal(t, "checkout", n.Name())
}

func TestCheckoutNode_Version(t *testing.T) {
	n := &CheckoutNode{}
	assert.Equal(t, "v1", n.Version())
}

func TestCheckoutNode_Params(t *testing.T) {
	n := &CheckoutNode{}
	params := n.Params()

	assert.Len(t, params, 2, "checkout@v1 should have 2 parameters")

	// 验证 repository 参数
	repository, exists := params["repository"]
	assert.True(t, exists, "repository parameter should exist")
	assert.Equal(t, "string", repository.Type)
	assert.True(t, repository.Required, "repository should be required")
	assert.Equal(t, "Git repository URL", repository.Description)

	// 验证 branch 参数
	branch, exists := params["branch"]
	assert.True(t, exists, "branch parameter should exist")
	assert.Equal(t, "string", branch.Type)
	assert.False(t, branch.Required, "branch should be optional")
	assert.Equal(t, "Branch name", branch.Description)
	assert.Equal(t, "main", branch.Default, "branch default should be 'main'")
}

func TestRunNode_Name(t *testing.T) {
	n := &RunNode{}
	assert.Equal(t, "run", n.Name())
}

func TestRunNode_Version(t *testing.T) {
	n := &RunNode{}
	assert.Equal(t, "v1", n.Version())
}

func TestRunNode_Params(t *testing.T) {
	n := &RunNode{}
	params := n.Params()

	assert.Len(t, params, 1, "run@v1 should have 1 parameter")

	// 验证 command 参数
	command, exists := params["command"]
	assert.True(t, exists, "command parameter should exist")
	assert.Equal(t, "string", command.Type)
	assert.True(t, command.Required, "command should be required")
	assert.Equal(t, "Shell command", command.Description)
}

func TestCheckoutNode_Integration(t *testing.T) {
	// 验证 CheckoutNode 实现 Node 接口
	var _ node.Node = &CheckoutNode{}

	n := &CheckoutNode{}

	// 验证完整的节点标识符
	key := n.Name() + "@" + n.Version()
	assert.Equal(t, "checkout@v1", key)

	// 验证参数规范可用于验证
	params := n.Params()

	// 模拟验证场景：检查必填参数
	requiredParams := []string{}
	for name, spec := range params {
		if spec.Required {
			requiredParams = append(requiredParams, name)
		}
	}
	assert.Contains(t, requiredParams, "repository")
	assert.NotContains(t, requiredParams, "branch")
}

func TestRunNode_Integration(t *testing.T) {
	// 验证 RunNode 实现 Node 接口
	var _ node.Node = &RunNode{}

	n := &RunNode{}

	// 验证完整的节点标识符
	key := n.Name() + "@" + n.Version()
	assert.Equal(t, "run@v1", key)

	// 验证参数规范可用于验证
	params := n.Params()

	// 模拟验证场景：检查参数类型
	commandSpec, exists := params["command"]
	assert.True(t, exists)
	assert.Equal(t, "string", commandSpec.Type, "command should be string type")
}

func TestBuiltinNodes_ParamSpecValidation(t *testing.T) {
	tests := []struct {
		name string
		node node.Node
	}{
		{"CheckoutNode", &CheckoutNode{}},
		{"RunNode", &RunNode{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.node.Params()

			// 验证所有参数规范的完整性
			for paramName, spec := range params {
				assert.NotEmpty(t, spec.Type, "param %s should have type", paramName)
				assert.NotEmpty(t, spec.Description, "param %s should have description", paramName)

				// 类型应该是有效的基础类型
				validTypes := []string{"string", "int", "bool", "array", "map"}
				assert.Contains(t, validTypes, spec.Type,
					"param %s has invalid type %s", paramName, spec.Type)
			}
		})
	}
}

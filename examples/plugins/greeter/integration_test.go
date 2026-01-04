//go:build integration
// +build integration

package main

import (
	"context"
	"fmt"
	"os"
	"plugin"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGreeterPlugin_Load(t *testing.T) {
	// 确保插件已编译
	if _, err := os.Stat("greeter.so"); os.IsNotExist(err) {
		t.Skip("greeter.so not found, run 'make build' first")
	}

	// 1. 加载插件
	p, err := plugin.Open("greeter.so")
	require.NoError(t, err, "Failed to open plugin")

	// 2. 查找 Register 函数
	symRegister, err := p.Lookup("Register")
	require.NoError(t, err, "Failed to lookup Register function")

	// 3. 类型断言
	register, ok := symRegister.(func() node.Node)
	require.True(t, ok, "Register is not of type func() node.Node")

	// 4. 调用 Register 获取节点实例
	greeterNode := register()
	require.NotNil(t, greeterNode)

	// 5. 验证节点属性
	assert.Equal(t, "custom/greeter", greeterNode.Name())
	assert.Equal(t, "v1", greeterNode.Version())

	// 6. 验证参数定义
	params := greeterNode.Params()
	assert.Contains(t, params, "name")
	assert.Contains(t, params, "language")

	// 7. 执行节点
	inputs := map[string]interface{}{
		"name":        "Integration Test",
		"language":    "en",
		"time_of_day": "morning",
	}

	result, err := greeterNode.Execute(context.Background(), inputs)
	require.NoError(t, err)

	// 8. 验证输出
	assert.Equal(t, "Good morning, Integration Test!", result.Outputs["greeting"])
	assert.NotEmpty(t, result.Logs)
	assert.GreaterOrEqual(t, result.Duration.Nanoseconds(), int64(0))

	fmt.Println("✅ Plugin loaded and executed successfully")
}

// TestGreeterPlugin_InvalidRegister 测试无效的Register函数签名
func TestGreeterPlugin_InvalidRegister(t *testing.T) {
	// 测试失败场景: Register 函数类型错误
	// 注: 这个测试模拟插件加载失败的场景
	// 实际的 plugin.Open 需要真实的 .so 文件,这里验证类型断言逻辑

	// 模拟错误的 Register 函数签名
	var invalidRegister interface{} = func() string { return "invalid" }

	// 尝试类型断言为正确的签名
	_, ok := invalidRegister.(func() node.Node)
	assert.False(t, ok, "Invalid Register signature should fail type assertion")
}

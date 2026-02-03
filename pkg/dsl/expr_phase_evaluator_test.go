package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPhaseEvaluator_BuildDefinitionContext 测试定义阶段上下文构建
func TestPhaseEvaluator_BuildDefinitionContext(t *testing.T) {
	evaluator := NewPhaseEvaluator(1 * 1000000000) // 1 second

	vars := map[string]interface{}{
		"environment": "production",
		"timeout":     30,
		"servers":     []interface{}{"web1", "web2"},
	}

	ctx := evaluator.BuildDefinitionContext(vars)

	// 验证 vars 存在
	assert.NotNil(t, ctx.Vars)
	assert.Equal(t, "production", ctx.Vars["environment"])
	assert.Equal(t, 30, ctx.Vars["timeout"])

	// 验证运行时上下文不存在
	assert.Nil(t, ctx.Workflow)
	assert.Nil(t, ctx.Job)
	assert.Nil(t, ctx.Steps)
	assert.Nil(t, ctx.Matrix)
	assert.Nil(t, ctx.Runner)
	assert.Nil(t, ctx.Inputs)
	assert.Nil(t, ctx.Needs)

	// 验证内置函数可用
	assert.NotNil(t, ctx.Len)
	assert.NotNil(t, ctx.Upper)
	assert.NotNil(t, ctx.Format)

	// 验证条件函数不可用（定义阶段）
	assert.False(t, ctx.Success())
	assert.False(t, ctx.Failure())
	assert.False(t, ctx.Cancelled())
}

// TestPhaseEvaluator_BuildStepContext 测试 Step 执行阶段上下文构建
func TestPhaseEvaluator_BuildStepContext(t *testing.T) {
	evaluator := NewPhaseEvaluator(1 * 1000000000)

	vars := map[string]interface{}{"env": "prod"}
	workflowInfo := map[string]interface{}{"name": "deploy", "id": "wf_123"}
	jobInfo := map[string]interface{}{"status": "success"}
	stepOutputs := map[string]interface{}{
		"checkout": map[string]interface{}{
			"outputs": map[string]interface{}{"commit": "abc123"},
		},
	}
	matrixVars := map[string]interface{}{"server": "web1"}
	env := map[string]string{"PATH": "/usr/bin"}
	runnerInfo := map[string]interface{}{"os": "linux"}
	inputs := map[string]interface{}{"branch": "main"}
	needsOutputs := map[string]interface{}{}

	ctx := evaluator.BuildStepContext(
		vars, workflowInfo, jobInfo, stepOutputs,
		matrixVars, env, runnerInfo, inputs, needsOutputs,
	)

	// 验证所有上下文存在
	assert.NotNil(t, ctx.Vars)
	assert.NotNil(t, ctx.Workflow)
	assert.NotNil(t, ctx.Job)
	assert.NotNil(t, ctx.Steps)
	assert.NotNil(t, ctx.Matrix)
	assert.NotNil(t, ctx.Env)
	assert.NotNil(t, ctx.Runner)
	assert.NotNil(t, ctx.Inputs)

	// 验证值正确
	assert.Equal(t, "prod", ctx.Vars["env"])
	assert.Equal(t, "deploy", ctx.Workflow["name"])
	assert.Equal(t, "success", ctx.Job["status"])
	assert.Equal(t, "web1", ctx.Matrix["server"])

	// 验证条件函数可用（执行阶段）
	assert.True(t, ctx.Success()) // job.status == "success"
	assert.False(t, ctx.Failure())
	assert.False(t, ctx.Cancelled())
}

// TestPhaseEvaluator_Evaluate 测试表达式求值
func TestPhaseEvaluator_Evaluate(t *testing.T) {
	evaluator := NewPhaseEvaluator(1 * 1000000000)

	ctx := evaluator.BuildDefinitionContext(map[string]interface{}{
		"environment": "production",
		"port":        8080,
	})

	tests := []struct {
		name       string
		expression string
		want       interface{}
		wantErr    bool
	}{
		{
			name:       "simple variable",
			expression: "vars.environment",
			want:       "production",
			wantErr:    false,
		},
		{
			name:       "integer variable",
			expression: "vars.port",
			want:       8080,
			wantErr:    false,
		},
		{
			name:       "string concatenation",
			expression: `vars.environment + "-server"`,
			want:       "production-server",
			wantErr:    false,
		},
		{
			name:       "arithmetic",
			expression: "vars.port + 1",
			want:       8081,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(tt.expression, ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

// TestVarsMerger_Merge 测试三层参数合并
func TestVarsMerger_Merge(t *testing.T) {
	merger := NewVarsMerger()

	yamlVars := map[string]interface{}{
		"env":     "dev",
		"timeout": "30m",
		"debug":   false,
	}

	triggerVars := map[string]interface{}{
		"env": "staging",
		"log": "info",
	}

	executionVars := map[string]interface{}{
		"env":   "prod",
		"debug": true,
	}

	merged := merger.Merge(yamlVars, triggerVars, executionVars)

	// 验证优先级: execution > trigger > yaml
	assert.Equal(t, "prod", merged["env"])    // execution 覆盖
	assert.Equal(t, "30m", merged["timeout"]) // yaml 保留
	assert.Equal(t, true, merged["debug"])    // execution 覆盖
	assert.Equal(t, "info", merged["log"])    // trigger 提供
	assert.Equal(t, 4, len(merged))
}

// TestVarsMerger_MergeWithDefaults 测试带默认值的合并
func TestVarsMerger_MergeWithDefaults(t *testing.T) {
	merger := NewVarsMerger()

	defaults := map[string]interface{}{
		"timeout": "60m",
		"retries": 3,
	}

	yamlVars := map[string]interface{}{
		"env": "dev",
	}

	merged := merger.MergeWithDefaults(yamlVars, nil, nil, defaults)

	assert.Equal(t, "dev", merged["env"])     // yaml 提供
	assert.Equal(t, "60m", merged["timeout"]) // defaults 提供
	assert.Equal(t, 3, merged["retries"])     // defaults 提供
}

// TestDefinitionRenderer_RenderRunsOn 测试 runs-on 表达式求值
func TestDefinitionRenderer_RenderRunsOn(t *testing.T) {
	renderer := NewDefinitionRenderer()

	workflow := &Workflow{
		Vars: map[string]interface{}{
			"target_queue": "web-servers",
		},
		Jobs: map[string]*Job{
			"deploy": {
				RunsOn: "${{ vars.target_queue }}",
			},
		},
	}

	rendered, err := renderer.RenderDefinitionPhase(workflow, nil, nil)
	require.NoError(t, err)

	job := rendered.Jobs["deploy"]
	assert.Equal(t, "web-servers", job.RunsOn)
	assert.Equal(t, "${{ vars.target_queue }}", job.RunsOnExpr) // 保留原表达式
}

// TestDefinitionRenderer_RenderMatrix 测试 matrix 表达式求值
func TestDefinitionRenderer_RenderMatrix(t *testing.T) {
	renderer := NewDefinitionRenderer()

	workflow := &Workflow{
		Vars: map[string]interface{}{
			"servers": []interface{}{"web1", "web2", "web3"},
		},
		Jobs: map[string]*Job{
			"deploy": {
				RunsOn: "default",
				Strategy: &Strategy{
					Matrix: map[string][]interface{}{
						"server": {"${{ vars.servers }}"},
					},
				},
			},
		},
	}

	rendered, err := renderer.RenderDefinitionPhase(workflow, nil, nil)
	require.NoError(t, err)

	job := rendered.Jobs["deploy"]
	require.NotNil(t, job.Strategy)

	// 验证 Matrix 展开
	servers := job.Strategy.Matrix["server"]
	require.Len(t, servers, 3)
	assert.Equal(t, "web1", servers[0])
	assert.Equal(t, "web2", servers[1])
	assert.Equal(t, "web3", servers[2])
}

// TestDefinitionRenderer_ThreeLayerVars 测试三层参数覆盖
func TestDefinitionRenderer_ThreeLayerVars(t *testing.T) {
	renderer := NewDefinitionRenderer()

	workflow := &Workflow{
		Vars: map[string]interface{}{
			"env":     "dev",
			"timeout": 30,
		},
		Jobs: map[string]*Job{
			"deploy": {
				RunsOn: "${{ vars.env }}-servers",
			},
		},
	}

	triggerVars := map[string]interface{}{
		"env": "staging",
	}

	executionVars := map[string]interface{}{
		"env": "prod",
	}

	rendered, err := renderer.RenderDefinitionPhase(workflow, triggerVars, executionVars)
	require.NoError(t, err)

	// 验证 execution 覆盖生效
	job := rendered.Jobs["deploy"]
	assert.Equal(t, "prod-servers", job.RunsOn)

	// 验证 vars 已合并
	assert.Equal(t, "prod", rendered.Vars["env"])
	assert.Equal(t, 30, rendered.Vars["timeout"])
}

// TestDefinitionRenderer_ComplexMatrixExpression 测试复杂 Matrix 表达式
func TestDefinitionRenderer_ComplexMatrixExpression(t *testing.T) {
	renderer := NewDefinitionRenderer()

	workflow := &Workflow{
		Vars: map[string]interface{}{
			"os_list":      []interface{}{"ubuntu", "centos"},
			"version_list": []interface{}{"20.04", "22.04"},
		},
		Jobs: map[string]*Job{
			"test": {
				RunsOn: "ci-runners",
				Strategy: &Strategy{
					Matrix: map[string][]interface{}{
						"os":      {"${{ vars.os_list }}"},
						"version": {"${{ vars.version_list }}"},
					},
				},
			},
		},
	}

	rendered, err := renderer.RenderDefinitionPhase(workflow, nil, nil)
	require.NoError(t, err)

	job := rendered.Jobs["test"]
	matrix := job.Strategy.Matrix

	// 验证两个维度都展开
	assert.Len(t, matrix["os"], 2)
	assert.Len(t, matrix["version"], 2)
	assert.Equal(t, "ubuntu", matrix["os"][0])
	assert.Equal(t, "20.04", matrix["version"][0])
}

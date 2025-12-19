package matrix_test

import (
	"os"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/matrix"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// TestMatrixIntegration_SimpleWorkflow 测试简单Matrix工作流解析和展开
func TestMatrixIntegration_SimpleWorkflow(t *testing.T) {
	// 读取测试文件
	data, err := os.ReadFile("../../testdata/matrix/simple.yaml")
	assert.NoError(t, err)

	var workflow dsl.Workflow
	err = yaml.Unmarshal(data, &workflow)
	assert.NoError(t, err)

	// 验证解析
	assert.Equal(t, "matrix-simple", workflow.Name)
	assert.Contains(t, workflow.Jobs, "deploy")

	job := workflow.Jobs["deploy"]
	assert.NotNil(t, job.Strategy)
	assert.NotNil(t, job.Strategy.Matrix)
	assert.Contains(t, job.Strategy.Matrix, "server")

	// 展开Matrix
	expander := matrix.NewExpander(256)
	instances, err := expander.Expand(job)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(instances))

	// 验证实例
	servers := make([]string, 0)
	for _, inst := range instances {
		servers = append(servers, inst.Matrix["server"].(string))
	}
	assert.Contains(t, servers, "web1")
	assert.Contains(t, servers, "web2")
	assert.Contains(t, servers, "web3")
}

// TestMatrixIntegration_MultiDimensionWorkflow 测试多维Matrix工作流
func TestMatrixIntegration_MultiDimensionWorkflow(t *testing.T) {
	data, err := os.ReadFile("../../testdata/matrix/multi-dimension.yaml")
	assert.NoError(t, err)

	var workflow dsl.Workflow
	err = yaml.Unmarshal(data, &workflow)
	assert.NoError(t, err)

	job := workflow.Jobs["deploy"]
	assert.NotNil(t, job.Strategy)
	assert.Equal(t, 2, job.Strategy.MaxParallel)
	assert.Equal(t, false, *job.Strategy.FailFast)

	// 展开Matrix
	expander := matrix.NewExpander(256)
	instances, err := expander.Expand(job)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(instances)) // 2 servers * 2 envs

	// 验证所有组合
	combinations := make(map[string]bool)
	for _, inst := range instances {
		key := inst.Matrix["server"].(string) + "-" + inst.Matrix["env"].(string)
		combinations[key] = true
	}

	assert.True(t, combinations["web1-prod"])
	assert.True(t, combinations["web1-staging"])
	assert.True(t, combinations["web2-prod"])
	assert.True(t, combinations["web2-staging"])
}

// TestMatrixIntegration_MaxParallelWorkflow 测试max-parallel配置
func TestMatrixIntegration_MaxParallelWorkflow(t *testing.T) {
	data, err := os.ReadFile("../../testdata/matrix/max-parallel.yaml")
	assert.NoError(t, err)

	var workflow dsl.Workflow
	err = yaml.Unmarshal(data, &workflow)
	assert.NoError(t, err)

	job := workflow.Jobs["test"]
	assert.NotNil(t, job.Strategy)
	assert.Equal(t, 2, job.Strategy.MaxParallel)

	// 展开Matrix
	expander := matrix.NewExpander(256)
	instances, err := expander.Expand(job)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(instances))

	// 验证版本
	versions := make([]float64, 0)
	for _, inst := range instances {
		versions = append(versions, inst.Matrix["version"].(float64))
	}
	assert.Contains(t, versions, 1.20)
	assert.Contains(t, versions, 1.21)
	assert.Contains(t, versions, 1.22)
	assert.Contains(t, versions, 1.23)
	assert.Contains(t, versions, 1.24)
}

// TestMatrixIntegration_ContextBuilding 测试Matrix实例上下文构建
func TestMatrixIntegration_ContextBuilding(t *testing.T) {
	workflow := &dsl.Workflow{
		Name: "test",
	}

	job := &dsl.Job{
		Name: "deploy",
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"server": {"web1", "web2"},
				"env":    {"prod", "staging"},
			},
		},
	}

	// 展开Matrix
	expander := matrix.NewExpander(256)
	instances, err := expander.Expand(job)
	assert.NoError(t, err)

	// 为每个实例构建上下文
	for _, inst := range instances {
		ctx := dsl.NewContextBuilder(workflow).
			WithJob(job).
			WithMatrix(inst.Matrix).
			Build()

		assert.NotNil(t, ctx)
		assert.NotNil(t, ctx.Matrix)
		assert.Contains(t, ctx.Matrix, "server")
		assert.Contains(t, ctx.Matrix, "env")
	}
}

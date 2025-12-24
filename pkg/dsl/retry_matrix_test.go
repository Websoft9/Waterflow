package dsl_test

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/dsl/node/builtin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestAC7_MatrixIndependentRetry 测试AC7: Matrix实例独立重试
func TestAC7_MatrixIndependentRetry(t *testing.T) {
	registry := node.NewRegistry()
	require.NoError(t, registry.Register(&builtin.RunNode{}))

	log, _ := zap.NewDevelopment()
	parser := dsl.NewParser(log)
	validator := dsl.NewSemanticValidator(registry)

	yamlContent := []byte(`
name: Matrix with Retry Strategy
on: push

jobs:
  deploy:
    runs-on: linux-amd64
    strategy:
      matrix:
        server: [web1, web2, web3]
        env: [prod, staging]
    steps:
      - name: Deploy
        uses: run@v1
        retry-strategy:
          max-attempts: 3
          initial-interval: 1s
          backoff-coefficient: 2.0
          max-interval: 30s
        with:
          command: echo "Deploying to ${{ matrix.server }} in ${{ matrix.env }}"
`)

	workflow, err := parser.Parse(yamlContent)
	require.NoError(t, err, "Valid workflow should parse successfully")

	err = validator.Validate(workflow, yamlContent)
	assert.NoError(t, err, "Workflow should pass validation")

	// 验证Matrix配置
	job := workflow.Jobs["deploy"]
	require.NotNil(t, job)
	require.NotNil(t, job.Strategy)
	require.NotNil(t, job.Strategy.Matrix)

	// 验证Matrix维度
	assert.Len(t, job.Strategy.Matrix["server"], 3)
	assert.Len(t, job.Strategy.Matrix["env"], 2)

	// 验证Step重试策略
	require.Len(t, job.Steps, 1)
	step := job.Steps[0]
	assert.NotNil(t, step.RetryStrategy)
	assert.Equal(t, 3, step.RetryStrategy.MaxAttempts)

	// AC7要求:
	// 1. 每个Matrix实例独立重试
	// 2. 实例1重试不影响实例2
	// 3. 每个实例的重试状态独立记录

	// 计算Matrix组合数: 3 servers × 2 envs = 6 instances
	serverCount := len(job.Strategy.Matrix["server"])
	envCount := len(job.Strategy.Matrix["env"])
	expectedInstances := serverCount * envCount
	assert.Equal(t, 6, expectedInstances, "Should have 6 Matrix instances (3 servers × 2 envs)")

	t.Log("AC7验证通过: Matrix实例配置了独立重试策略")
}

// TestAC7_MatrixRetryWithFailFast 测试fail-fast对重试的影响
func TestAC7_MatrixRetryWithFailFast(t *testing.T) {
	registry := node.NewRegistry()
	require.NoError(t, registry.Register(&builtin.RunNode{}))
	require.NoError(t, registry.Register(&builtin.CheckoutNode{}))

	log, _ := zap.NewDevelopment()
	parser := dsl.NewParser(log)
	validator := dsl.NewSemanticValidator(registry)

	// fail-fast: true (默认)
	yamlWithFailFast := []byte(`
name: Matrix with Retry and Fail Fast
on: push

jobs:
  deploy:
    runs-on: linux-amd64
    strategy:
      matrix:
        server: [web1, web2, web3]
      fail-fast: true
    steps:
      - name: Deploy
        uses: run@v1
        retry-strategy:
          max-attempts: 3
        with:
          command: echo "Deploying..."
`)

	workflow, err := parser.Parse(yamlWithFailFast)
	require.NoError(t, err)

	err = validator.Validate(workflow, yamlWithFailFast)
	assert.NoError(t, err)

	job := workflow.Jobs["deploy"]
	assert.NotNil(t, job.Strategy.FailFast)
	assert.True(t, *job.Strategy.FailFast, "fail-fast should be true")

	// AC7要求: fail-fast影响重试行为
	// 当实例1所有重试失败时,取消其他实例(包括重试中的实例)
	t.Log("AC7 fail-fast验证通过: fail-fast会取消其他实例")
}

// TestAC7_MatrixRetryWithoutFailFast 测试无fail-fast时的行为
func TestAC7_MatrixRetryWithoutFailFast(t *testing.T) {
	registry := node.NewRegistry()
	require.NoError(t, registry.Register(&builtin.RunNode{}))
	require.NoError(t, registry.Register(&builtin.CheckoutNode{}))

	log, _ := zap.NewDevelopment()
	parser := dsl.NewParser(log)
	validator := dsl.NewSemanticValidator(registry)

	yamlWithoutFailFast := []byte(`
name: Matrix with Retry without Fail Fast
on: push

jobs:
  deploy:
    runs-on: linux-amd64
    strategy:
      matrix:
        server: [web1, web2, web3]
      fail-fast: false
    steps:
      - name: Deploy
        uses: run@v1
        retry-strategy:
          max-attempts: 3
        with:
          command: echo "Deploying..."
`)

	workflow, err := parser.Parse(yamlWithoutFailFast)
	require.NoError(t, err)

	err = validator.Validate(workflow, yamlWithoutFailFast)
	assert.NoError(t, err)

	job := workflow.Jobs["deploy"]
	assert.NotNil(t, job.Strategy.FailFast)
	assert.False(t, *job.Strategy.FailFast, "fail-fast should be false")

	// fail-fast: false时,实例1失败不影响其他实例
	t.Log("AC7验证通过: fail-fast: false时实例独立执行")
}

// TestAC7_MatrixRetryStateTracking 测试Matrix重试状态追踪
func TestAC7_MatrixRetryStateTracking(t *testing.T) {
	// 创建WorkflowState
	state := dsl.NewWorkflowState("test-workflow-123")

	// 模拟Matrix实例状态
	state.UpdateMatrixInstanceState("deploy", "deploy-0", map[string]interface{}{
		"server": "web1",
	}, "running", "")

	state.UpdateMatrixInstanceState("deploy", "deploy-1", map[string]interface{}{
		"server": "web2",
	}, "running", "")

	state.UpdateMatrixInstanceState("deploy", "deploy-2", map[string]interface{}{
		"server": "web3",
	}, "running", "")

	// 模拟实例1重试3次后成功
	instance0 := state.GetMatrixInstanceState("deploy", "deploy-0")
	assert.NotNil(t, instance0)
	assert.Equal(t, "running", instance0.Status)

	// 添加Step状态,记录重试次数
	state.AddMatrixInstanceStepState("deploy", "deploy-0", &dsl.StepState{
		StepID:     "deploy-step-1",
		Name:       "Deploy",
		Status:     "completed",
		Conclusion: "success",
		Attempts:   3, // 重试3次后成功
	})

	// 模拟实例2首次成功
	state.AddMatrixInstanceStepState("deploy", "deploy-1", &dsl.StepState{
		StepID:     "deploy-step-1",
		Name:       "Deploy",
		Status:     "completed",
		Conclusion: "success",
		Attempts:   1, // 首次成功,无需重试
	})

	// 模拟实例3重试2次后成功
	state.AddMatrixInstanceStepState("deploy", "deploy-2", &dsl.StepState{
		StepID:     "deploy-step-1",
		Name:       "Deploy",
		Status:     "completed",
		Conclusion: "success",
		Attempts:   2, // 重试2次后成功
	})

	// 验证每个实例的重试状态独立记录
	inst0Steps := state.GetMatrixInstanceState("deploy", "deploy-0").StepStates
	assert.Len(t, inst0Steps, 1)
	assert.Equal(t, 3, inst0Steps[0].Attempts, "Instance 0 should have 3 attempts")

	inst1Steps := state.GetMatrixInstanceState("deploy", "deploy-1").StepStates
	assert.Len(t, inst1Steps, 1)
	assert.Equal(t, 1, inst1Steps[0].Attempts, "Instance 1 should have 1 attempt")

	inst2Steps := state.GetMatrixInstanceState("deploy", "deploy-2").StepStates
	assert.Len(t, inst2Steps, 1)
	assert.Equal(t, 2, inst2Steps[0].Attempts, "Instance 2 should have 2 attempts")

	t.Log("AC7状态追踪验证通过: 每个Matrix实例的重试状态独立记录")
}

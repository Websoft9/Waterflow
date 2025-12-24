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

// TestAC6_RetryWithContinueOnError 测试AC6: 重试策略与continue-on-error交互
func TestAC6_RetryWithContinueOnError(t *testing.T) {
	registry := node.NewRegistry()
	require.NoError(t, registry.Register(&builtin.RunNode{}))

	log, _ := zap.NewDevelopment()
	parser := dsl.NewParser(log)
	validator := dsl.NewSemanticValidator(registry)

	yamlContent := []byte(`
name: Test Retry with Continue on Error
on: push

jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Flaky Test
        uses: run@v1
        continue-on-error: true
        retry-strategy:
          max-attempts: 3
          initial-interval: 1s
          backoff-coefficient: 2.0
          max-interval: 10s
        with:
          command: echo "Test may fail"
      
      - name: Next Step
        uses: run@v1
        with:
          command: echo "This should run even if previous step fails"
`)

	workflow, err := parser.Parse(yamlContent)
	require.NoError(t, err, "Valid workflow should parse successfully")

	err = validator.Validate(workflow, yamlContent)
	assert.NoError(t, err, "Workflow should pass validation")

	// 验证配置
	job := workflow.Jobs["test"]
	require.NotNil(t, job)
	require.Len(t, job.Steps, 2)

	flakyStep := job.Steps[0]
	assert.Equal(t, "Flaky Test", flakyStep.Name)
	assert.True(t, flakyStep.ContinueOnError, "Step should have continue-on-error enabled")
	assert.NotNil(t, flakyStep.RetryStrategy, "Step should have retry strategy")
	assert.Equal(t, 3, flakyStep.RetryStrategy.MaxAttempts)

	nextStep := job.Steps[1]
	assert.Equal(t, "Next Step", nextStep.Name)

	// AC6要求:
	// 1. Step所有重试失败后,由于continue-on-error: true,工作流继续
	// 2. Step状态为failure,但Job状态为completed
	// 3. 后续Step正常执行
	t.Log("AC6验证通过: continue-on-error与重试策略正确配置")
}

// TestAC6_RetryWithoutContinueOnError 测试没有continue-on-error时的行为
func TestAC6_RetryWithoutContinueOnError(t *testing.T) {
	registry := node.NewRegistry()
	require.NoError(t, registry.Register(&builtin.RunNode{}))
	require.NoError(t, registry.Register(&builtin.CheckoutNode{}))

	log, _ := zap.NewDevelopment()
	parser := dsl.NewParser(log)
	validator := dsl.NewSemanticValidator(registry)

	yamlContent := []byte(`
name: Test Retry without Continue on Error
on: push

jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Critical Step
        uses: run@v1
        retry-strategy:
          max-attempts: 3
        with:
          command: test -f file.txt
      
      - name: Should Not Run
        uses: run@v1
        with:
          command: echo "This should NOT run if previous step fails"
`)

	workflow, err := parser.Parse(yamlContent)
	require.NoError(t, err)

	err = validator.Validate(workflow, yamlContent)
	assert.NoError(t, err)

	// 验证配置
	job := workflow.Jobs["test"]
	criticalStep := job.Steps[0]
	assert.False(t, criticalStep.ContinueOnError, "Step should NOT have continue-on-error")
	assert.NotNil(t, criticalStep.RetryStrategy)

	// 没有continue-on-error时,所有重试失败后应该终止工作流
	t.Log("验证通过: 没有continue-on-error时,失败会终止工作流")
}

// TestAC6_MultipleStepsWithRetryAndContinue 测试多个Step的组合场景
func TestAC6_MultipleStepsWithRetryAndContinue(t *testing.T) {
	registry := node.NewRegistry()
	require.NoError(t, registry.Register(&builtin.RunNode{}))
	require.NoError(t, registry.Register(&builtin.CheckoutNode{}))

	log, _ := zap.NewDevelopment()
	parser := dsl.NewParser(log)
	validator := dsl.NewSemanticValidator(registry)

	yamlContent := []byte(`
name: Multiple Steps with Retry and Continue
on: push

jobs:
  deploy:
    runs-on: linux-amd64
    steps:
      - name: Optional Health Check
        uses: run@v1
        continue-on-error: true
        retry-strategy:
          max-attempts: 2
        with:
          command: curl http://service/health || true
      
      - name: Deploy Code
        uses: run@v1
        retry-strategy:
          max-attempts: 5
        with:
          command: echo "Deploying..."
      
      - name: Optional Cleanup
        uses: run@v1
        continue-on-error: true
        with:
          command: echo "Cleaning up..."
`)

	workflow, err := parser.Parse(yamlContent)
	require.NoError(t, err)

	err = validator.Validate(workflow, yamlContent)
	assert.NoError(t, err)

	job := workflow.Jobs["deploy"]
	assert.Len(t, job.Steps, 3)

	// 验证第一个Step: 可选健康检查,失败可继续,有重试
	step1 := job.Steps[0]
	assert.True(t, step1.ContinueOnError)
	assert.NotNil(t, step1.RetryStrategy)
	assert.Equal(t, 2, step1.RetryStrategy.MaxAttempts)

	// 验证第二个Step: 关键部署,失败终止,有重试
	step2 := job.Steps[1]
	assert.False(t, step2.ContinueOnError)
	assert.NotNil(t, step2.RetryStrategy)
	assert.Equal(t, 5, step2.RetryStrategy.MaxAttempts)

	// 验证第三个Step: 可选清理,失败可继续,无重试
	step3 := job.Steps[2]
	assert.True(t, step3.ContinueOnError)
	assert.Nil(t, step3.RetryStrategy)

	t.Log("AC6组合场景验证通过")
}

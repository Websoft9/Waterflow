package dsl_test

import (
	"os"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/dsl/node/builtin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupIntegrationValidator 设置集成测试验证器
func setupIntegrationValidator() (*dsl.Parser, *dsl.SemanticValidator) {
	registry := node.NewRegistry()
	if err := registry.Register(&builtin.CheckoutNode{}); err != nil {
		panic(err)
	}
	if err := registry.Register(&builtin.RunNode{}); err != nil {
		panic(err)
	}

	log, _ := zap.NewDevelopment()
	parser := dsl.NewParser(log)
	validator := dsl.NewSemanticValidator(registry)
	return parser, validator
}

// TestTimeoutRetryIntegration_ValidWorkflow 测试包含超时和重试配置的有效工作流
func TestTimeoutRetryIntegration_ValidWorkflow(t *testing.T) {
	parser, validator := setupIntegrationValidator()

	yamlContent := []byte(`
name: Build with Timeout and Retry
on: push

jobs:
  build:
    runs-on: linux-amd64
    timeout-minutes: 120
    steps:
      - name: Checkout
        uses: checkout@v1
        timeout-minutes: 10
        with:
          repository: https://github.com/example/repo
          branch: main
        retry-strategy:
          max-attempts: 3
          initial-interval: 2s
          max-interval: 30s
          backoff-coefficient: 2.0

      - name: Run Tests
        uses: run@v1
        timeout-minutes: 60
        with:
          command: go test ./...
        retry-strategy:
          max-attempts: 5
          initial-interval: 5s
          max-interval: 1m
          backoff-coefficient: 1.5
`)

	workflow, err := parser.Parse(yamlContent)
	require.NoError(t, err, "Valid workflow should parse successfully")

	err = validator.Validate(workflow, yamlContent)
	assert.NoError(t, err, "Valid workflow with timeout and retry should pass validation")

	// 验证超时配置
	job := workflow.Jobs["build"]
	assert.Equal(t, 120, job.TimeoutMinutes, "Job timeout should be 120 minutes")
	assert.Equal(t, 10, job.Steps[0].TimeoutMinutes, "Step 0 timeout should be 10 minutes")
	assert.Equal(t, 60, job.Steps[1].TimeoutMinutes, "Step 1 timeout should be 60 minutes")

	// 验证重试策略
	step0Retry := job.Steps[0].RetryStrategy
	assert.NotNil(t, step0Retry, "Step 0 should have retry strategy")
	assert.Equal(t, 3, step0Retry.MaxAttempts, "Step 0 max attempts should be 3")
	assert.Equal(t, "2s", step0Retry.InitialInterval, "Step 0 initial interval should be 2s")
	assert.Equal(t, "30s", step0Retry.MaxInterval, "Step 0 max interval should be 30s")
	assert.Equal(t, 2.0, step0Retry.BackoffCoefficient, "Step 0 backoff coefficient should be 2.0")

	step1Retry := job.Steps[1].RetryStrategy
	assert.NotNil(t, step1Retry, "Step 1 should have retry strategy")
	assert.Equal(t, 5, step1Retry.MaxAttempts, "Step 1 max attempts should be 5")
}

// TestTimeoutRetryIntegration_InvalidTimeout 测试无效的超时配置
func TestTimeoutRetryIntegration_InvalidTimeout(t *testing.T) {
	parser, validator := setupIntegrationValidator()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "Negative job timeout",
			yaml: `
name: Test
on: push
jobs:
  build:
    runs-on: linux
    timeout-minutes: -10
    steps:
      - uses: run@v1
        with:
          command: echo test
`,
		},
		{
			name: "Exceeds maximum timeout",
			yaml: `
name: Test
on: push
jobs:
  build:
    runs-on: linux
    timeout-minutes: 2000
    steps:
      - uses: run@v1
        with:
          command: echo test
`,
		},
		{
			name: "Negative step timeout",
			yaml: `
name: Test
on: push
jobs:
  build:
    runs-on: linux
    steps:
      - uses: run@v1
        timeout-minutes: -5
        with:
          command: echo test
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow, err := parser.Parse([]byte(tt.yaml))
			require.NoError(t, err, "Should parse successfully")

			err = validator.Validate(workflow, []byte(tt.yaml))
			assert.Error(t, err, "Should fail validation")

			validationErr, ok := err.(*dsl.ValidationError)
			require.True(t, ok, "Error should be ValidationError")
			assert.NotEmpty(t, validationErr.Errors, "Should have validation errors")
		})
	}
}

// TestTimeoutRetryIntegration_InvalidRetryStrategy 测试无效的重试策略
func TestTimeoutRetryIntegration_InvalidRetryStrategy(t *testing.T) {
	parser, validator := setupIntegrationValidator()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "max-attempts too low",
			yaml: `
name: Test
on: push
jobs:
  build:
    runs-on: linux
    steps:
      - uses: run@v1
        with:
          command: echo test
        retry-strategy:
          max-attempts: 0
`,
		},
		{
			name: "max-attempts too high",
			yaml: `
name: Test
on: push
jobs:
  build:
    runs-on: linux
    steps:
      - uses: run@v1
        with:
          command: echo test
        retry-strategy:
          max-attempts: 15
`,
		},
		{
			name: "Invalid backoff coefficient",
			yaml: `
name: Test
on: push
jobs:
  build:
    runs-on: linux
    steps:
      - uses: run@v1
        with:
          command: echo test
        retry-strategy:
          max-attempts: 3
          backoff-coefficient: 0.5
`,
		},
		{
			name: "Invalid duration format",
			yaml: `
name: Test
on: push
jobs:
  build:
    runs-on: linux
    steps:
      - uses: run@v1
        with:
          command: echo test
        retry-strategy:
          max-attempts: 3
          initial-interval: invalid
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow, err := parser.Parse([]byte(tt.yaml))
			require.NoError(t, err, "Should parse successfully")

			err = validator.Validate(workflow, []byte(tt.yaml))
			assert.Error(t, err, "Should fail validation")

			validationErr, ok := err.(*dsl.ValidationError)
			require.True(t, ok, "Error should be ValidationError")
			assert.NotEmpty(t, validationErr.Errors, "Should have validation errors")
		})
	}
}

// TestTimeoutRetryIntegration_RealWorldScenario 测试真实场景的工作流
func TestTimeoutRetryIntegration_RealWorldScenario(t *testing.T) {
	parser, validator := setupIntegrationValidator()

	// 模拟一个真实的 CI/CD 工作流
	yamlContent := []byte(`
name: CI/CD Pipeline with Resilience
on: push

jobs:
  test:
    runs-on: linux-amd64
    timeout-minutes: 60
    steps:
      - name: Checkout Code
        uses: checkout@v1
        timeout-minutes: 5
        with:
          repository: https://github.com/example/repo
        retry-strategy:
          max-attempts: 3
          initial-interval: 1s
          max-interval: 10s
          backoff-coefficient: 2.0

      - name: Run Unit Tests
        uses: run@v1
        timeout-minutes: 30
        with:
          command: go test -v ./...
        retry-strategy:
          max-attempts: 2
          initial-interval: 5s
          backoff-coefficient: 1.0

  deploy:
    runs-on: linux-amd64
    timeout-minutes: 30
    needs: [test]
    steps:
      - name: Deploy to Production
        uses: run@v1
        timeout-minutes: 20
        with:
          command: ./deploy.sh
        retry-strategy:
          max-attempts: 5
          initial-interval: 10s
          max-interval: 2m
          backoff-coefficient: 1.5
`)

	workflow, err := parser.Parse(yamlContent)
	require.NoError(t, err, "Real-world workflow should parse successfully")

	err = validator.Validate(workflow, yamlContent)
	assert.NoError(t, err, "Real-world workflow should pass validation")

	// 验证工作流结构
	assert.Len(t, workflow.Jobs, 2, "Should have 2 jobs")

	testJob := workflow.Jobs["test"]
	assert.Equal(t, 60, testJob.TimeoutMinutes, "Test job timeout should be 60 minutes")
	assert.Len(t, testJob.Steps, 2, "Test job should have 2 steps")

	deployJob := workflow.Jobs["deploy"]
	assert.Equal(t, 30, deployJob.TimeoutMinutes, "Deploy job timeout should be 30 minutes")
	assert.Equal(t, []string{"test"}, deployJob.Needs, "Deploy job should depend on test")
}

// TestTimeoutRetryIntegration_FromFile 从测试数据文件加载并验证
func TestTimeoutRetryIntegration_FromFile(t *testing.T) {
	parser, validator := setupIntegrationValidator()

	// 创建临时测试文件
	testYAML := `
name: Timeout Retry Test
on: push
jobs:
  build:
    runs-on: linux-amd64
    timeout-minutes: 90
    steps:
      - name: Test Step
        uses: run@v1
        timeout-minutes: 45
        with:
          command: make test
        retry-strategy:
          max-attempts: 4
          initial-interval: 3s
          max-interval: 45s
          backoff-coefficient: 1.8
`

	tmpFile, err := os.CreateTemp("", "workflow-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(testYAML)
	require.NoError(t, err)
	_ = tmpFile.Close()

	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	err = validator.Validate(workflow, content)
	assert.NoError(t, err, "File-based workflow should pass validation")
}

// TestTimeoutRetryIntegration_ErrorClassification 测试错误分类器集成
func TestTimeoutRetryIntegration_ErrorClassification(t *testing.T) {
	classifier := dsl.NewErrorClassifier()

	tests := []struct {
		name        string
		errType     string
		shouldRetry bool
	}{
		{
			name:        "Validation error should not retry",
			errType:     "validation_error",
			shouldRetry: false,
		},
		{
			name:        "Network timeout should retry",
			errType:     "network_timeout",
			shouldRetry: true,
		},
		{
			name:        "Permission denied should not retry",
			errType:     "permission_denied",
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isRetryable := classifier.IsRetryable(tt.errType)
			assert.Equal(t, tt.shouldRetry, isRetryable)
		})
	}
}

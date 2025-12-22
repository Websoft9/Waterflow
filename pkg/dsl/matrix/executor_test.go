package matrix

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/assert"
)

// MockStepExecutor 模拟 Step 执行器
type MockStepExecutor struct {
	executeFunc func(ctx context.Context, step *dsl.Step, evalCtx *dsl.EvalContext) (*dsl.StepResult, error)
}

func (m *MockStepExecutor) Execute(ctx context.Context, step *dsl.Step, evalCtx *dsl.EvalContext) (*dsl.StepResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, step, evalCtx)
	}
	return &dsl.StepResult{
		Status:     "completed",
		Conclusion: "success",
	}, nil
}

// TestMatrixExecutor_BasicExecution 测试基本执行
func TestMatrixExecutor_BasicExecution(t *testing.T) {
	workflow := &dsl.Workflow{Name: "test"}
	job := &dsl.Job{
		Name:   "deploy",
		RunsOn: "linux-amd64",
		Steps: []*dsl.Step{
			{Name: "Deploy", Uses: "run@v1"},
		},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(&dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"server": {"web1", "web2", "web3"},
			},
		},
	})
	assert.NoError(t, err)

	stepExecutor := &MockStepExecutor{}
	executor := NewMatrixExecutor(10, true, stepExecutor)

	ctx := context.Background()
	results := executor.Execute(ctx, workflow, job, instances)

	assert.Equal(t, 3, len(results))
	for _, result := range results {
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, "success", result.Conclusion)
	}
}

// TestMatrixExecutor_MaxParallel 测试并发控制
func TestMatrixExecutor_MaxParallel(t *testing.T) {
	workflow := &dsl.Workflow{Name: "test"}
	job := &dsl.Job{
		Name:   "test",
		RunsOn: "linux-amd64",
		Steps:  []*dsl.Step{{Name: "Test", Uses: "run@v1"}},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(&dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"version": {1, 2, 3, 4, 5},
			},
		},
	})
	assert.NoError(t, err)

	// 记录同时执行的实例数
	var mu = struct {
		sync.Mutex
		max     int
		current int
	}{}

	stepExecutor := &MockStepExecutor{
		executeFunc: func(ctx context.Context, step *dsl.Step, evalCtx *dsl.EvalContext) (*dsl.StepResult, error) {
			mu.Lock()
			mu.current++
			if mu.current > mu.max {
				mu.max = mu.current
			}
			mu.Unlock()

			time.Sleep(50 * time.Millisecond) // 模拟执行时间

			mu.Lock()
			mu.current--
			mu.Unlock()

			return &dsl.StepResult{Status: "completed", Conclusion: "success"}, nil
		},
	}

	executor := NewMatrixExecutor(2, true, stepExecutor) // max-parallel: 2

	ctx := context.Background()
	results := executor.Execute(ctx, workflow, job, instances)

	assert.Equal(t, 5, len(results))
	assert.LessOrEqual(t, mu.max, 2, "Should not exceed max-parallel")
}

// TestMatrixExecutor_FailFast 测试 fail-fast 策略
func TestMatrixExecutor_FailFast(t *testing.T) {
	workflow := &dsl.Workflow{Name: "test"}
	job := &dsl.Job{
		Name:   "test",
		RunsOn: "linux-amd64",
		Steps:  []*dsl.Step{{Name: "Test", Uses: "run@v1"}},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(&dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"version": {1, 2, 3, 4, 5},
			},
		},
	})
	assert.NoError(t, err)

	executionCount := 0
	stepExecutor := &MockStepExecutor{
		executeFunc: func(ctx context.Context, step *dsl.Step, evalCtx *dsl.EvalContext) (*dsl.StepResult, error) {
			executionCount++
			version := evalCtx.Matrix["version"].(int)

			// 版本 2 失败
			if version == 2 {
				return &dsl.StepResult{Status: "completed", Conclusion: "failure"}, nil
			}

			time.Sleep(100 * time.Millisecond)
			return &dsl.StepResult{Status: "completed", Conclusion: "success"}, nil
		},
	}

	executor := NewMatrixExecutor(10, true, stepExecutor) // fail-fast: true

	ctx := context.Background()
	results := executor.Execute(ctx, workflow, job, instances)

	// 验证有失败和取消的实例
	failureCount := 0
	cancelledCount := 0
	successCount := 0

	for _, result := range results {
		switch result.Conclusion {
		case "failure":
			failureCount++
		case "cancelled":
			cancelledCount++
		case "success":
			successCount++
		}
	}

	assert.Greater(t, failureCount, 0, "Should have at least one failure")
	assert.Greater(t, cancelledCount, 0, "Should have cancelled instances due to fail-fast")
	assert.Less(t, executionCount, 5, "Should not execute all instances due to fail-fast")
}

// TestMatrixExecutor_NoFailFast 测试 fail-fast=false
func TestMatrixExecutor_NoFailFast(t *testing.T) {
	workflow := &dsl.Workflow{Name: "test"}
	job := &dsl.Job{
		Name:   "test",
		RunsOn: "linux-amd64",
		Steps:  []*dsl.Step{{Name: "Test", Uses: "run@v1"}},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(&dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"version": {1, 2, 3, 4, 5},
			},
		},
	})
	assert.NoError(t, err)

	stepExecutor := &MockStepExecutor{
		executeFunc: func(ctx context.Context, step *dsl.Step, evalCtx *dsl.EvalContext) (*dsl.StepResult, error) {
			version := evalCtx.Matrix["version"].(int)

			// 版本 2 和 4 失败
			if version == 2 || version == 4 {
				return &dsl.StepResult{Status: "completed", Conclusion: "failure"}, nil
			}

			return &dsl.StepResult{Status: "completed", Conclusion: "success"}, nil
		},
	}

	executor := NewMatrixExecutor(10, false, stepExecutor) // fail-fast: false

	ctx := context.Background()
	results := executor.Execute(ctx, workflow, job, instances)

	// 所有实例都应该执行完成
	assert.Equal(t, 5, len(results))

	failureCount := 0
	successCount := 0

	for _, result := range results {
		assert.NotEqual(t, "cancelled", result.Conclusion, "No instances should be cancelled")
		if result.Conclusion == "failure" {
			failureCount++
		} else if result.Conclusion == "success" {
			successCount++
		}
	}

	assert.Equal(t, 2, failureCount, "Should have 2 failures")
	assert.Equal(t, 3, successCount, "Should have 3 successes")
}

// TestMatrixExecutor_ContextCancellation 测试上下文取消
func TestMatrixExecutor_ContextCancellation(t *testing.T) {
	workflow := &dsl.Workflow{Name: "test"}
	job := &dsl.Job{
		Name:   "test",
		RunsOn: "linux-amd64",
		Steps:  []*dsl.Step{{Name: "Test", Uses: "run@v1"}},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(&dsl.Job{
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"version": {1, 2, 3, 4, 5},
			},
		},
	})
	assert.NoError(t, err)

	stepExecutor := &MockStepExecutor{
		executeFunc: func(ctx context.Context, step *dsl.Step, evalCtx *dsl.EvalContext) (*dsl.StepResult, error) {
			// 立即检查上下文
			select {
			case <-ctx.Done():
				return &dsl.StepResult{Status: "cancelled", Conclusion: "cancelled"}, ctx.Err()
			default:
			}

			// 模拟长时间执行
			select {
			case <-ctx.Done():
				return &dsl.StepResult{Status: "cancelled", Conclusion: "cancelled"}, ctx.Err()
			case <-time.After(1 * time.Second):
				return &dsl.StepResult{Status: "completed", Conclusion: "success"}, nil
			}
		},
	}

	executor := NewMatrixExecutor(10, true, stepExecutor)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	results := executor.Execute(ctx, workflow, job, instances)

	// 应该有完成或取消的实例，但不应该全部成功
	successCount := 0
	for _, result := range results {
		if result.Conclusion == "success" {
			successCount++
		}
	}

	assert.Less(t, successCount, len(instances), "Should not complete all instances due to timeout")
}

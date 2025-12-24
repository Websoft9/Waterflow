package dsl

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockMatrixStepExecutor 模拟 Step 执行器
type MockMatrixStepExecutor struct {
	executeFunc func(ctx context.Context, step *Step, evalCtx *EvalContext) (*StepResult, error)
}

func (m *MockMatrixStepExecutor) Execute(ctx context.Context, step *Step, evalCtx *EvalContext) (*StepResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, step, evalCtx)
	}
	return &StepResult{
		Status:     "completed",
		Conclusion: "success",
	}, nil
}

// TestMatrixExecutor_BasicExecution 测试基本执行
func TestMatrixExecutor_BasicExecution(t *testing.T) {
	workflow := &Workflow{Name: "test"}
	job := &Job{
		Name:   "deploy",
		RunsOn: "linux-amd64",
		Steps: []*Step{
			{Name: "Deploy", Uses: "run@v1"},
		},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(&Job{
		Strategy: &Strategy{
			Matrix: map[string][]interface{}{
				"server": {"web1", "web2", "web3"},
			},
		},
	})
	assert.NoError(t, err)

	stepExecutor := &MockMatrixStepExecutor{}
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
	workflow := &Workflow{Name: "test"}
	job := &Job{
		Name:   "test",
		RunsOn: "linux-amd64",
		Steps:  []*Step{{Name: "Test", Uses: "run@v1"}},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(&Job{
		Strategy: &Strategy{
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

	stepExecutor := &MockMatrixStepExecutor{
		executeFunc: func(ctx context.Context, step *Step, evalCtx *EvalContext) (*StepResult, error) {
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

			return &StepResult{Status: "completed", Conclusion: "success"}, nil
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
	workflow := &Workflow{Name: "test"}
	job := &Job{
		Name:   "test",
		RunsOn: "linux-amd64",
		Steps:  []*Step{{Name: "Test", Uses: "run@v1"}},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(&Job{
		Strategy: &Strategy{
			Matrix: map[string][]interface{}{
				"version": {1, 2, 3},
			},
		},
	})
	assert.NoError(t, err)

	startTime := time.Now()

	stepExecutor := &MockMatrixStepExecutor{
		executeFunc: func(ctx context.Context, step *Step, evalCtx *EvalContext) (*StepResult, error) {
			version := evalCtx.Matrix["version"].(int)

			// 版本 2 在 100ms 后失败
			if version == 2 {
				time.Sleep(100 * time.Millisecond)
				return &StepResult{Status: "completed", Conclusion: "failure"}, nil
			}

			// 其他版本需要更长时间
			time.Sleep(500 * time.Millisecond)
			return &StepResult{Status: "completed", Conclusion: "success"}, nil
		},
	}

	executor := NewMatrixExecutor(10, true, stepExecutor) // fail-fast: true

	ctx := context.Background()
	results := executor.Execute(ctx, workflow, job, instances)

	// 验证 fail-fast 在 1 秒内取消
	elapsed := time.Since(startTime)
	assert.Less(t, elapsed, 1*time.Second, "fail-fast should cancel other instances within 1 second")

	// 验证结果
	failureCount := 0
	successCount := 0

	for _, result := range results {
		switch result.Conclusion {
		case "failure":
			failureCount++
		case "success":
			successCount++
		}
	}

	// fail-fast 为 true 时，应该有一个失败
	assert.Equal(t, 1, failureCount, "Should have exactly one failure")
	// 其他实例的结果取决于并发执行时序，但总数应该是 3
	assert.Equal(t, 3, len(results), "Should have all 3 results")
}

// TestMatrixExecutor_NoFailFast 测试 fail-fast=false
func TestMatrixExecutor_NoFailFast(t *testing.T) {
	workflow := &Workflow{Name: "test"}
	job := &Job{
		Name:   "test",
		RunsOn: "linux-amd64",
		Steps:  []*Step{{Name: "Test", Uses: "run@v1"}},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(&Job{
		Strategy: &Strategy{
			Matrix: map[string][]interface{}{
				"version": {1, 2, 3, 4, 5},
			},
		},
	})
	assert.NoError(t, err)

	stepExecutor := &MockMatrixStepExecutor{
		executeFunc: func(ctx context.Context, step *Step, evalCtx *EvalContext) (*StepResult, error) {
			version := evalCtx.Matrix["version"].(int)

			// 版本 2 和 4 失败
			if version == 2 || version == 4 {
				return &StepResult{Status: "completed", Conclusion: "failure"}, nil
			}

			return &StepResult{Status: "completed", Conclusion: "success"}, nil
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
	workflow := &Workflow{Name: "test"}
	job := &Job{
		Name:   "test",
		RunsOn: "linux-amd64",
		Steps:  []*Step{{Name: "Test", Uses: "run@v1"}},
	}

	expander := NewExpander(256)
	instances, err := expander.Expand(&Job{
		Strategy: &Strategy{
			Matrix: map[string][]interface{}{
				"version": {1, 2, 3, 4, 5},
			},
		},
	})
	assert.NoError(t, err)

	stepExecutor := &MockMatrixStepExecutor{
		executeFunc: func(ctx context.Context, step *Step, evalCtx *EvalContext) (*StepResult, error) {
			// 立即检查上下文
			select {
			case <-ctx.Done():
				return &StepResult{Status: "cancelled", Conclusion: "cancelled"}, ctx.Err()
			default:
			}

			// 模拟长时间执行
			select {
			case <-ctx.Done():
				return &StepResult{Status: "cancelled", Conclusion: "cancelled"}, ctx.Err()
			case <-time.After(1 * time.Second):
				return &StepResult{Status: "completed", Conclusion: "success"}, nil
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

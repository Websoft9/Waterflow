package dsl

import (
	"context"
	"sync"
)

// MatrixStepExecutor Step 执行器接口（用于 Matrix）
type MatrixStepExecutor interface {
	Execute(ctx context.Context, step *Step, evalCtx *EvalContext) (*StepResult, error)
}

// MatrixExecutor Matrix 实例执行器
type MatrixExecutor struct {
	maxParallel  int
	failFast     bool
	stepExecutor MatrixStepExecutor
}

// NewMatrixExecutor 创建 Matrix 执行器
func NewMatrixExecutor(maxParallel int, failFast bool, stepExecutor MatrixStepExecutor) *MatrixExecutor {
	return &MatrixExecutor{
		maxParallel:  maxParallel,
		failFast:     failFast,
		stepExecutor: stepExecutor,
	}
}

// Execute 执行所有 Matrix 实例
func (e *MatrixExecutor) Execute(
	ctx context.Context,
	workflow *Workflow,
	job *Job,
	instances []*MatrixInstance,
) []*MatrixResult {
	results := make([]*MatrixResult, len(instances))
	resultChan := make(chan *MatrixResult, len(instances))

	// 确定实际并发数：如果 maxParallel <= 0，默认全部并行
	maxParallel := e.maxParallel
	if maxParallel <= 0 {
		maxParallel = len(instances)
	}

	// 使用 semaphore 控制并发
	sem := make(chan struct{}, maxParallel)

	var wg sync.WaitGroup
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i, instance := range instances {
		wg.Add(1)

		go func(idx int, inst *MatrixInstance) {
			defer wg.Done()

			// 获取信号量
			select {
			case sem <- struct{}{}:
			case <-cancelCtx.Done():
				resultChan <- &MatrixResult{
					Index:      idx,
					Status:     "cancelled",
					Conclusion: "cancelled",
				}
				return
			}
			defer func() { <-sem }()

			// 检查是否已取消 (fail-fast)
			select {
			case <-cancelCtx.Done():
				resultChan <- &MatrixResult{
					Index:      idx,
					Status:     "cancelled",
					Conclusion: "cancelled",
				}
				return
			default:
			}

			// 执行实例
			result := e.executeInstance(cancelCtx, workflow, job, inst)
			result.Index = idx

			// fail-fast: 失败时取消其他实例
			if e.failFast && result.Conclusion == "failure" {
				cancel()
			}

			resultChan <- result
		}(i, instance)
	}

	// 等待所有实例完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	for result := range resultChan {
		results[result.Index] = result
	}

	return results
}

// executeInstance 执行单个 Matrix 实例
func (e *MatrixExecutor) executeInstance(
	ctx context.Context,
	workflow *Workflow,
	job *Job,
	instance *MatrixInstance,
) *MatrixResult {
	// 构建上下文 (包含 matrix 变量)
	evalCtx := NewContextBuilder(workflow).
		WithJob(job).
		WithMatrix(instance.Matrix).
		Build()

	// 执行 Steps
	for _, step := range job.Steps {
		select {
		case <-ctx.Done():
			return &MatrixResult{
				Status:     "cancelled",
				Conclusion: "cancelled",
			}
		default:
		}

		stepResult, err := e.stepExecutor.Execute(ctx, step, evalCtx)
		if err != nil {
			return &MatrixResult{
				Status:     "completed",
				Conclusion: "failure",
				Error:      err.Error(),
			}
		}

		if stepResult.Conclusion == "failure" && !step.ContinueOnError {
			return &MatrixResult{
				Status:     "completed",
				Conclusion: "failure",
			}
		}
	}

	return &MatrixResult{
		Status:     "completed",
		Conclusion: "success",
	}
}

// MatrixResult Matrix 实例执行结果
type MatrixResult struct {
	Index      int
	Status     string // completed, cancelled
	Conclusion string // success, failure, cancelled
	Error      string
}

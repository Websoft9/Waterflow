package matrix_test

import (
	"fmt"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/dsl/matrix"
	"github.com/stretchr/testify/assert"
)

// TestMatrixIntegration_StateTracking 测试 Matrix 状态追踪集成
func TestMatrixIntegration_StateTracking(t *testing.T) {
	// 创建带有 Matrix 的 Job
	job := &dsl.Job{
		Name: "test-matrix-job",
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"os":      {"ubuntu", "macos"},
				"version": {"20.04", "22.04"},
			},
			MaxParallel: 2,
			FailFast:    boolPtr(false),
		},
		Steps: []*dsl.Step{
			{
				ID:   "step1",
				Uses: "run",
			},
		},
	}

	// 扩展 Matrix
	expander := matrix.NewExpander(256)
	instances, err := expander.Expand(job)
	assert.NoError(t, err)
	assert.Len(t, instances, 4) // 2 x 2 = 4

	// 创建 Workflow 状态
	ws := dsl.NewWorkflowState("test-workflow")

	// 模拟执行 Matrix 实例并追踪状态
	for _, instance := range instances {
		matrixID := fmt.Sprintf("matrix-%d", instance.Index)

		// 更新实例状态为 in_progress
		ws.UpdateMatrixInstanceState(job.Name, matrixID, instance.Matrix, "in_progress", "")

		// 模拟步骤执行
		for _, step := range job.Steps {
			stepState := &dsl.StepState{
				StepID:     step.ID,
				Status:     "completed",
				Conclusion: "success",
			}
			ws.AddMatrixInstanceStepState(job.Name, matrixID, stepState)
		}

		// 更新实例状态为 completed
		ws.UpdateMatrixInstanceState(job.Name, matrixID, nil, "completed", "success")
	}

	// 验证状态追踪
	jobState := ws.GetJobState(job.Name)
	assert.NotNil(t, jobState)
	assert.True(t, jobState.IsMatrix)
	assert.Len(t, jobState.MatrixInstances, 4)

	// 验证每个实例
	for i := 0; i < 4; i++ {
		matrixID := fmt.Sprintf("matrix-%d", i)
		inst := ws.GetMatrixInstanceState(job.Name, matrixID)
		assert.NotNil(t, inst)
		assert.Equal(t, "completed", inst.Status, "Instance %d should be completed", i)
		assert.Equal(t, "success", inst.Conclusion, "Instance %d should be success", i)
		assert.Len(t, inst.StepStates, 1, "Instance %d should have 1 step", i)
		assert.Equal(t, "step1", inst.StepStates[0].StepID)
		assert.Equal(t, "success", inst.StepStates[0].Conclusion)
	}

	// 验证 Matrix 变量
	instance0 := ws.GetMatrixInstanceState(job.Name, "matrix-0")
	assert.NotNil(t, instance0)
	assert.Contains(t, []interface{}{"ubuntu", "macos"}, instance0.Matrix["os"])
	assert.Contains(t, []interface{}{"20.04", "22.04"}, instance0.Matrix["version"])

	instance3 := ws.GetMatrixInstanceState(job.Name, "matrix-3")
	assert.NotNil(t, instance3)
	assert.Contains(t, []interface{}{"ubuntu", "macos"}, instance3.Matrix["os"])
	assert.Contains(t, []interface{}{"20.04", "22.04"}, instance3.Matrix["version"])
}

// TestMatrixIntegration_FailFastStateTracking 测试 fail-fast 模式的状态追踪
func TestMatrixIntegration_FailFastStateTracking(t *testing.T) {
	job := &dsl.Job{
		Name: "test-fail-fast",
		Strategy: &dsl.Strategy{
			Matrix: map[string][]interface{}{
				"test": {1, 2, 3, 4},
			},
			FailFast: boolPtr(true),
		},
		Steps: []*dsl.Step{{ID: "step1", Uses: "run"}},
	}

	expander := matrix.NewExpander(256)
	instances, err := expander.Expand(job)
	assert.NoError(t, err)

	ws := dsl.NewWorkflowState("test-workflow")

	failAtInstance := 2
	hasFailure := false

	// 手动执行实例
	for _, instance := range instances {
		matrixID := fmt.Sprintf("matrix-%d", instance.Index)
		ws.UpdateMatrixInstanceState(job.Name, matrixID, instance.Matrix, "in_progress", "")

		// 第 2 个实例失败
		testNum := instance.Matrix["test"].(int)
		if testNum == failAtInstance {
			ws.UpdateMatrixInstanceState(job.Name, matrixID, nil, "completed", "failure")
			hasFailure = true
			// 在 fail-fast 模式下，这里应该停止执行
			if job.Strategy.FailFast != nil && *job.Strategy.FailFast {
				break
			}
		} else {
			ws.UpdateMatrixInstanceState(job.Name, matrixID, nil, "completed", "success")
		}
	}

	// 验证状态：由于 fail-fast，不是所有实例都完成
	jobState := ws.GetJobState(job.Name)
	assert.NotNil(t, jobState)
	assert.True(t, hasFailure, "Should have at least one failed instance")

	// 验证至少有一个失败
	foundFailure := false
	for _, inst := range jobState.MatrixInstances {
		if inst.Conclusion == "failure" {
			foundFailure = true
			break
		}
	}
	assert.True(t, foundFailure, "Should have at least one failed instance")

	// 由于 fail-fast，实例数应该少于总数
	assert.Less(t, len(jobState.MatrixInstances), 4, "Should have fewer instances due to fail-fast")
}

func boolPtr(b bool) *bool {
	return &b
}

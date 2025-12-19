package dsl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUpdateMatrixInstanceState tests updating Matrix instance state
func TestUpdateMatrixInstanceState(t *testing.T) {
	ws := NewWorkflowState("test-workflow")

	// 更新第一个 Matrix 实例
	ws.UpdateMatrixInstanceState("job1", "matrix-0", map[string]interface{}{
		"os":      "ubuntu",
		"version": "20.04",
	}, "in_progress", "")

	// 验证 Job 状态创建
	jobState := ws.GetJobState("job1")
	assert.NotNil(t, jobState)
	assert.True(t, jobState.IsMatrix)
	assert.Len(t, jobState.MatrixInstances, 1)

	// 验证 Matrix 实例状态
	instance := jobState.MatrixInstances[0]
	assert.Equal(t, "matrix-0", instance.MatrixID)
	assert.Equal(t, "in_progress", instance.Status)
	assert.Equal(t, "", instance.Conclusion)
	assert.Equal(t, "ubuntu", instance.Matrix["os"])
	assert.Equal(t, "20.04", instance.Matrix["version"])

	// 更新同一个实例的状态
	ws.UpdateMatrixInstanceState("job1", "matrix-0", nil, "completed", "success")
	instance = ws.GetJobState("job1").MatrixInstances[0]
	assert.Equal(t, "completed", instance.Status)
	assert.Equal(t, "success", instance.Conclusion)

	// 添加第二个 Matrix 实例
	ws.UpdateMatrixInstanceState("job1", "matrix-1", map[string]interface{}{
		"os":      "macos",
		"version": "12",
	}, "queued", "")

	jobState = ws.GetJobState("job1")
	assert.Len(t, jobState.MatrixInstances, 2)
	assert.Equal(t, "matrix-1", jobState.MatrixInstances[1].MatrixID)
}

// TestAddMatrixInstanceStepState tests adding step states to Matrix instances
func TestAddMatrixInstanceStepState(t *testing.T) {
	ws := NewWorkflowState("test-workflow")

	// 创建 Matrix 实例
	ws.UpdateMatrixInstanceState("job1", "matrix-0", map[string]interface{}{
		"os": "ubuntu",
	}, "in_progress", "")

	// 添加步骤状态
	step1 := &StepState{
		StepID:     "step1",
		Status:     "completed",
		Conclusion: "success",
	}
	ws.AddMatrixInstanceStepState("job1", "matrix-0", step1)

	// 验证步骤状态
	instance := ws.GetMatrixInstanceState("job1", "matrix-0")
	assert.NotNil(t, instance)
	assert.Len(t, instance.StepStates, 1)
	assert.Equal(t, "step1", instance.StepStates[0].StepID)
	assert.Equal(t, "success", instance.StepStates[0].Conclusion)

	// 添加第二个步骤
	step2 := &StepState{
		StepID:     "step2",
		Status:     "completed",
		Conclusion: "failure",
	}
	ws.AddMatrixInstanceStepState("job1", "matrix-0", step2)

	instance = ws.GetMatrixInstanceState("job1", "matrix-0")
	assert.Len(t, instance.StepStates, 2)
	assert.Equal(t, "step2", instance.StepStates[1].StepID)
}

// TestGetMatrixInstanceState tests retrieving Matrix instance state
func TestGetMatrixInstanceState(t *testing.T) {
	ws := NewWorkflowState("test-workflow")

	// 测试不存在的 Job
	instance := ws.GetMatrixInstanceState("nonexistent", "matrix-0")
	assert.Nil(t, instance)

	// 创建 Matrix 实例
	ws.UpdateMatrixInstanceState("job1", "matrix-0", map[string]interface{}{
		"os": "ubuntu",
	}, "in_progress", "")

	// 测试存在的 Matrix 实例
	instance = ws.GetMatrixInstanceState("job1", "matrix-0")
	assert.NotNil(t, instance)
	assert.Equal(t, "matrix-0", instance.MatrixID)

	// 测试不存在的 Matrix 实例
	instance = ws.GetMatrixInstanceState("job1", "matrix-99")
	assert.Nil(t, instance)
}

// TestMatrixInstanceConcurrentAccess tests concurrent access to Matrix instance state
func TestMatrixInstanceConcurrentAccess(t *testing.T) {
	ws := NewWorkflowState("test-workflow")

	// 并发创建 Matrix 实例
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		matrixID := fmt.Sprintf("matrix-%d", i)
		go func(id string) {
			ws.UpdateMatrixInstanceState("job1", id, map[string]interface{}{
				"index": id,
			}, "queued", "")
			done <- true
		}(matrixID)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证所有实例都已创建
	jobState := ws.GetJobState("job1")
	assert.Len(t, jobState.MatrixInstances, 10)

	// 并发读取
	for i := 0; i < 10; i++ {
		matrixID := fmt.Sprintf("matrix-%d", i)
		go func(id string) {
			instance := ws.GetMatrixInstanceState("job1", id)
			assert.NotNil(t, instance)
			done <- true
		}(matrixID)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

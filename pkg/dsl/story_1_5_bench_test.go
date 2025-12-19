package dsl

import (
	"context"
	"sync"
	"testing"
)

// Benchmark OutputParser concurrent access
func BenchmarkOutputParser_ConcurrentParse(b *testing.B) {
	parser := NewOutputParser()
	output := `::set-output name=version::1.0.0
::set-output name=image::myapp:latest
::set-output name=artifact::app.tar.gz`

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			parser.ParseOutput(output)
		}
	})
}

// Benchmark WorkflowState concurrent access
func BenchmarkWorkflowState_ConcurrentUpdate(b *testing.B) {
	state := NewWorkflowState("wf-bench")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			jobID := "job-1"
			state.UpdateJobState(jobID, "running", "pending", nil)
			_ = state.GetJobState(jobID)
			i++
		}
	})
}

// Benchmark DependencyGraph with complex dependencies
func BenchmarkDependencyGraph_ComplexChain(b *testing.B) {
	workflow := &Workflow{
		Name: "Benchmark",
		Jobs: make(map[string]*Job),
	}

	// Create a chain of 100 jobs
	for i := 0; i < 100; i++ {
		jobName := string(rune('a' + i))
		job := &Job{Name: jobName}
		if i > 0 {
			prevJob := string(rune('a' + i - 1))
			job.Needs = []string{prevJob}
		}
		workflow.Jobs[jobName] = job
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		graph := NewDependencyGraph(workflow)
		_ = graph.ValidateDependencies()
	}
}

// Benchmark JobOrchestrator context building
func BenchmarkJobOrchestrator_BuildContext(b *testing.B) {
	workflow := &Workflow{
		Name: "Test",
		Vars: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
		Jobs: map[string]*Job{
			"build": {
				Name: "build",
				Outputs: map[string]string{
					"version": "1.0.0",
					"image":   "app:latest",
				},
			},
			"deploy": {
				Name:  "deploy",
				Needs: []string{"build"},
			},
		},
	}

	executor := &MockJobExecutor{}
	orch := NewJobOrchestrator(workflow, executor)
	orch.graph.MarkCompleted("build", map[string]string{
		"version": "1.0.0",
		"image":   "app:latest",
	})

	baseCtx := NewContextBuilder(workflow).Build()
	job := workflow.Jobs["deploy"]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = orch.buildJobContext(baseCtx, job)
	}
}

// Benchmark parallel job execution
func BenchmarkJobOrchestrator_ParallelExecution(b *testing.B) {
	workflow := &Workflow{
		Name: "Parallel",
		Jobs: map[string]*Job{
			"job1": {Name: "job1", Steps: []*Step{{Name: "step1"}}},
			"job2": {Name: "job2", Steps: []*Step{{Name: "step1"}}},
			"job3": {Name: "job3", Steps: []*Step{{Name: "step1"}}},
			"job4": {Name: "job4", Steps: []*Step{{Name: "step1"}}},
			"job5": {Name: "job5", Steps: []*Step{{Name: "step1"}}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor := &MockJobExecutor{}
		orch := NewJobOrchestrator(workflow, executor)
		_ = orch.Execute(context.Background(), workflow)
	}
}

// Benchmark StepExecutor with if conditions
func BenchmarkStepExecutor_WithCondition(b *testing.B) {
	executor := NewStepExecutor()
	workflow := &Workflow{
		Name: "Test",
		Vars: map[string]interface{}{
			"env": "production",
		},
	}
	step := &Step{
		Name: "Deploy",
		Uses: "deploy@v1",
		If:   "vars.env == 'production'",
	}
	evalCtx := NewContextBuilder(workflow).Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(context.Background(), step, evalCtx)
	}
}

// Benchmark concurrent step execution
func BenchmarkStepExecutor_Concurrent(b *testing.B) {
	workflow := &Workflow{
		Name: "Test",
		Vars: map[string]interface{}{
			"env": "production",
		},
	}
	step := &Step{
		ID:   "build",
		Name: "Build",
		Uses: "build@v1",
	}
	evalCtx := NewContextBuilder(workflow).Build()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		executor := NewStepExecutor()
		for pb.Next() {
			_, _ = executor.Execute(context.Background(), step, evalCtx)
		}
	})
}

// Compare old vs new buildJobContext approach
func BenchmarkJobContext_StructCopy(b *testing.B) {
	baseCtx := &EvalContext{
		Vars:    map[string]interface{}{"key": "value"},
		Secrets: map[string]string{"secret": "value"},
		Inputs:  map[string]interface{}{"input": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Struct copy (new approach)
		ctx := *baseCtx
		ctx.Needs = map[string]interface{}{"job1": map[string]interface{}{"outputs": map[string]interface{}{}}}
		_ = ctx
	}
}

func BenchmarkJobContext_ManualCopy(b *testing.B) {
	baseCtx := &EvalContext{
		Vars:    map[string]interface{}{"key": "value"},
		Secrets: map[string]string{"secret": "value"},
		Inputs:  map[string]interface{}{"input": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Manual copy (old approach - simulate)
		ctx := &EvalContext{
			Vars:    baseCtx.Vars,
			Secrets: baseCtx.Secrets,
			Inputs:  baseCtx.Inputs,
			Steps:   baseCtx.Steps,
			Env:     baseCtx.Env,
		}
		ctx.Needs = map[string]interface{}{"job1": map[string]interface{}{"outputs": map[string]interface{}{}}}
		_ = ctx
	}
}

// Memory allocation benchmark
func BenchmarkOutputParser_Allocations(b *testing.B) {
	parser := NewOutputParser()
	output := "::set-output name=test::value\n::set-output name=test2::value2"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.ParseOutput(output)
	}
}

// Test lock contention
func BenchmarkWorkflowState_LockContention(b *testing.B) {
	state := NewWorkflowState("wf-bench")
	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate high contention
		for j := 0; j < 10; j++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				state.UpdateJobState("job1", "running", "pending", nil)
			}()
			go func() {
				defer wg.Done()
				_ = state.GetJobState("job1")
			}()
		}
		wg.Wait()
	}
}

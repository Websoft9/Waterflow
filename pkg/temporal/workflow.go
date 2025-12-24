package temporal

import (
	"fmt"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"go.temporal.io/sdk/workflow"
)

// RunWorkflowExecutor is the main workflow orchestrator that executes a YAML workflow.
// It builds dependency graph and executes jobs based on their dependencies.
func RunWorkflowExecutor(ctx workflow.Context, wf *dsl.Workflow) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting workflow", "name", wf.Name)

	// 1. Build job dependency graph (using Story 1.5 DependencyGraph)
	depGraph := dsl.NewDependencyGraph(wf)

	// 2. Validate dependencies (checks circular dependencies)
	if err := depGraph.ValidateDependencies(); err != nil {
		return fmt.Errorf("invalid job dependencies: %w", err)
	}

	// 3. Execute jobs based on dependency graph
	for !depGraph.AllJobsCompleted() {
		// Get jobs ready to execute
		readyJobs := depGraph.GetReadyJobs()

		if len(readyJobs) == 0 && !depGraph.AllJobsCompleted() {
			return fmt.Errorf("deadlock detected: no jobs ready but workflow not complete")
		}

		// Execute ready jobs
		for _, jobName := range readyJobs {
			job := wf.Jobs[jobName]
			job.Name = jobName // Set job name for reference

			logger.Info("Executing job", "job", jobName)
			depGraph.MarkRunning(jobName)

			// Execute job (with matrix support)
			if err := executeJob(ctx, wf, job); err != nil {
				logger.Error("Job failed", "job", jobName, "error", err)
				depGraph.MarkFailed(jobName)
				return fmt.Errorf("job %s failed: %w", jobName, err)
			}

			logger.Info("Job completed successfully", "job", jobName)
			depGraph.MarkCompleted(jobName, nil)
		}
	}

	logger.Info("Workflow completed successfully", "name", wf.Name)
	return nil
}

// executeJob executes a single job (supports matrix expansion).
func executeJob(ctx workflow.Context, wf *dsl.Workflow, job *dsl.Job) error {
	logger := workflow.GetLogger(ctx)

	// 1. Check job-level if condition
	if job.If != "" {
		evalCtx := buildEvalContext(wf, job, nil)
		engine := dsl.NewEngine(30 * time.Second)
		condEval := dsl.NewConditionEvaluator(engine)
		shouldRun, err := condEval.Evaluate(job.If, evalCtx)
		if err != nil {
			return fmt.Errorf("failed to evaluate job if condition: %w", err)
		}
		if !shouldRun {
			logger.Info("Job skipped due to if condition", "job", job.Name)
			return nil
		}
	}

	// 2. Expand matrix (if strategy defined)
	expander := dsl.NewExpander(256) // Max 256 combinations
	instances, err := expander.Expand(job)
	if err != nil {
		return fmt.Errorf("failed to expand matrix: %w", err)
	}

	logger.Info("Matrix expansion completed", "job", job.Name, "instances", len(instances))

	// 3. Determine max parallel execution
	maxParallel := getMaxParallel(job, len(instances))
	failFast := getFailFast(job)
	logger.Info("Matrix execution config", "job", job.Name, "maxParallel", maxParallel, "failFast", failFast)

	// 4. Execute matrix instances with concurrency control
	if maxParallel >= len(instances) {
		// Full parallel execution
		return executeMatrixInstancesParallel(ctx, wf, job, instances, failFast)
	}

	// Limited parallel execution using semaphore pattern
	return executeMatrixInstancesWithLimit(ctx, wf, job, instances, maxParallel, failFast)
}

// executeMatrixInstancesParallel executes all instances in parallel
func executeMatrixInstancesParallel(ctx workflow.Context, wf *dsl.Workflow, job *dsl.Job, instances []*dsl.MatrixInstance, failFast bool) error {
	logger := workflow.GetLogger(ctx)
	futures := make([]workflow.Future, len(instances))
	for i, instance := range instances {
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			TaskQueue: job.RunsOn,
		})
		logger.Info("Starting matrix instance", "job", job.Name, "instance", i, "matrix", instance.Matrix)
		futures[i] = workflow.ExecuteChildWorkflow(childCtx, executeJobInstance, wf, job, instance)
	}

	// Wait for all instances
	for i, future := range futures {
		if err := future.Get(ctx, nil); err != nil {
			if failFast {
				return fmt.Errorf("matrix instance %d failed (fail-fast enabled): %w", i, err)
			}
			logger.Error("Matrix instance failed (continuing)", "instance", i, "error", err)
		}
	}
	return nil
}

// executeMatrixInstancesWithLimit executes instances with max-parallel limit
func executeMatrixInstancesWithLimit(ctx workflow.Context, wf *dsl.Workflow, job *dsl.Job, instances []*dsl.MatrixInstance, maxParallel int, failFast bool) error {
	logger := workflow.GetLogger(ctx)
	semaphore := make(chan struct{}, maxParallel)
	errChan := make(chan error, len(instances))

	for i, instance := range instances {
		// Acquire semaphore
		semaphore <- struct{}{}

		go func(idx int, inst *dsl.MatrixInstance) {
			defer func() { <-semaphore }()

			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				TaskQueue: job.RunsOn,
			})
			logger.Info("Starting matrix instance", "job", job.Name, "instance", idx, "matrix", inst.Matrix)

			future := workflow.ExecuteChildWorkflow(childCtx, executeJobInstance, wf, job, inst)
			if err := future.Get(ctx, nil); err != nil {
				errChan <- fmt.Errorf("matrix instance %d failed: %w", idx, err)
			} else {
				errChan <- nil
			}
		}(i, instance)
	}

	// Wait for all instances and collect errors
	var firstErr error
	for i := 0; i < len(instances); i++ {
		if err := <-errChan; err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if failFast {
				logger.Error("Matrix instance failed (fail-fast enabled)", "error", err)
				return firstErr
			}
		}
	}

	return firstErr
}

// getMaxParallel determines max parallel execution count
func getMaxParallel(job *dsl.Job, totalInstances int) int {
	if job.Strategy == nil || job.Strategy.MaxParallel <= 0 {
		return totalInstances // Default: all parallel
	}
	return job.Strategy.MaxParallel
}

// getFailFast determines fail-fast behavior
func getFailFast(job *dsl.Job) bool {
	if job.Strategy == nil || job.Strategy.FailFast == nil {
		return true // Default: fail-fast enabled
	}
	return *job.Strategy.FailFast
}

// executeJobInstance executes a single job instance (matrix or regular job).
func executeJobInstance(ctx workflow.Context, wf *dsl.Workflow, job *dsl.Job, instance *dsl.MatrixInstance) error {
	logger := workflow.GetLogger(ctx)

	// Build evaluation context (includes matrix variables)
	evalCtx := buildEvalContext(wf, job, instance)

	// Execute steps in order
	for _, step := range job.Steps {
		// Resolve timeout (using Story 1.7 TimeoutResolver)
		timeoutResolver := dsl.NewTimeoutResolver()
		timeout := timeoutResolver.ResolveStepTimeout(step, job)

		// Resolve retry policy (using Story 1.7 RetryPolicyResolver)
		retryResolver := dsl.NewRetryPolicyResolver()
		retryPolicy, _ := retryResolver.Resolve(step.RetryStrategy)

		// Configure activity options
		activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			TaskQueue:           job.RunsOn, // Route to specified task queue
			StartToCloseTimeout: timeout,
			RetryPolicy:         retryPolicy.ToTemporalRetryPolicy(),
		})

		// Execute step activity (ADR-0002: single-node execution pattern)
		var stepResult StepResult
		err := workflow.ExecuteActivity(activityCtx, "ExecuteStepActivity", ExecuteStepInput{
			Workflow: wf,
			Job:      job,
			Step:     step,
			Context:  evalCtx,
		}).Get(activityCtx, &stepResult)

		if err != nil {
			logger.Error("Step failed", "step", step.Name, "error", err)

			// continue-on-error: continue execution
			if step.ContinueOnError {
				logger.Warn("Step failed but continue-on-error enabled", "step", step.Name)
				continue
			}

			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		logger.Info("Step completed", "step", step.Name, "status", stepResult.Status)

		// Update context with step outputs
		if evalCtx.Steps == nil {
			evalCtx.Steps = make(map[string]interface{})
		}
		if step.ID != "" {
			evalCtx.Steps[step.ID] = stepResult.Outputs
		} else if step.Name != "" {
			evalCtx.Steps[step.Name] = stepResult.Outputs
		}
	}

	return nil
}

// buildEvalContext constructs evaluation context from workflow, job, and matrix instance.
func buildEvalContext(wf *dsl.Workflow, job *dsl.Job, instance *dsl.MatrixInstance) *dsl.EvalContext {
	// Use ContextBuilder from Story 1.4
	builder := dsl.NewContextBuilder(wf).WithJob(job)

	// Add matrix variables if instance provided
	if instance != nil {
		builder = builder.WithMatrix(instance.Matrix)
	}

	// Build the context
	ctx := builder.Build()

	// Initialize empty collections if needed
	if ctx.Steps == nil {
		ctx.Steps = make(map[string]interface{})
	}
	if ctx.Needs == nil {
		ctx.Needs = make(map[string]interface{})
	}

	return ctx
}

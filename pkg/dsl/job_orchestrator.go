package dsl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Job orchestrator errors
var (
	ErrJobExecutionFailed = errors.New("job execution failed")
)

// Default timeout for job condition evaluation
const DefaultJobConditionTimeout = 1 * time.Second

// JobResult represents the result of job execution
type JobResult struct {
	JobID      string
	Status     string            // completed, skipped, failed
	Conclusion string            // success, failure, skipped, cancelled
	Outputs    map[string]string // job outputs
}

// JobExecutor is the interface for executing jobs
type JobExecutor interface {
	Execute(ctx context.Context, job *Job, evalCtx *EvalContext) (*JobResult, error)
}

// JobOrchestrator orchestrates job execution with dependency management
type JobOrchestrator struct {
	graph      *DependencyGraph
	executor   JobExecutor
	results    map[string]*JobResult
	resultsMux sync.RWMutex
}

// NewJobOrchestrator creates a new job orchestrator
func NewJobOrchestrator(workflow *Workflow, executor JobExecutor) *JobOrchestrator {
	graph := NewDependencyGraph(workflow)

	return &JobOrchestrator{
		graph:    graph,
		executor: executor,
		results:  make(map[string]*JobResult),
	}
}

// Execute runs all jobs respecting dependencies
func (o *JobOrchestrator) Execute(ctx context.Context, workflow *Workflow) error {
	evalCtx := NewContextBuilder(workflow).Build()

	for {
		// Get jobs ready to execute
		readyJobIDs := o.graph.GetReadyJobs()
		if len(readyJobIDs) == 0 {
			break
		}

		// Execute ready jobs in parallel with context cancellation
		var wg sync.WaitGroup
		errChan := make(chan error, len(readyJobIDs))
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		for _, jobID := range readyJobIDs {
			wg.Add(1)
			go func(jID string) {
				defer wg.Done()

				// Check if context is cancelled
				select {
				case <-ctx.Done():
					return
				default:
				}

				job := workflow.Jobs[jID]

				// Build job context with needs outputs
				jobEvalCtx := o.buildJobContext(evalCtx, job)

				// Evaluate job if condition
				if job.If != "" {
					engine := NewEngine(DefaultJobConditionTimeout)
					evaluator := NewConditionEvaluator(engine)
					shouldRun, err := evaluator.Evaluate(job.If, jobEvalCtx)
					if err != nil {
						errChan <- fmt.Errorf("evaluate job %s if condition: %w", jID, err)
						cancel() // Cancel other jobs
						return
					}

					if !shouldRun {
						// Mark job as skipped
						o.storeResult(&JobResult{
							JobID:      jID,
							Status:     "completed",
							Conclusion: "skipped",
							Outputs:    make(map[string]string),
						})
						o.graph.MarkCompleted(jID, make(map[string]string))
						return
					}
				}

				// Execute job
				result, err := o.executor.Execute(ctx, job, jobEvalCtx)
				if err != nil {
					errChan <- fmt.Errorf("execute job %s: %w", jID, err)
					cancel() // Cancel other jobs
					return
				}

				// Store result and mark completed
				o.storeResult(result)
				o.graph.MarkCompleted(jID, result.Outputs)
			}(jobID)
		}

		wg.Wait()
		close(errChan)

		// Collect all errors
		var errs []error
		for err := range errChan {
			if err != nil {
				errs = append(errs, err)
			}
		}

		// Return first error if any
		if len(errs) > 0 {
			return errs[0]
		}
	}

	return nil
}

// buildJobContext creates evaluation context with needs outputs
// It performs a shallow copy of the base context and adds job-specific needs.
func (o *JobOrchestrator) buildJobContext(baseCtx *EvalContext, job *Job) *EvalContext {
	// Shallow copy the entire context
	jobCtx := *baseCtx

	// Build needs context for this job
	needs := make(map[string]interface{})
	if len(job.Needs) > 0 {
		for _, neededJobID := range job.Needs {
			outputs := o.graph.GetJobOutputs(neededJobID)
			if outputs != nil {
				outputsMap := make(map[string]interface{}, len(outputs))
				for k, v := range outputs {
					outputsMap[k] = v
				}
				needs[neededJobID] = map[string]interface{}{
					"outputs": outputsMap,
				}
			}
		}
	}
	jobCtx.Needs = needs

	return &jobCtx
}

// storeResult stores job execution result
func (o *JobOrchestrator) storeResult(result *JobResult) {
	o.resultsMux.Lock()
	defer o.resultsMux.Unlock()
	o.results[result.JobID] = result
}

// GetResults returns all job results
func (o *JobOrchestrator) GetResults() map[string]*JobResult {
	o.resultsMux.RLock()
	defer o.resultsMux.RUnlock()

	results := make(map[string]*JobResult)
	for k, v := range o.results {
		results[k] = v
	}
	return results
}

// GetResult returns a specific job result
func (o *JobOrchestrator) GetResult(jobID string) *JobResult {
	o.resultsMux.RLock()
	defer o.resultsMux.RUnlock()
	return o.results[jobID]
}

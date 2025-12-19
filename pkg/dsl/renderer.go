package dsl

import (
	"fmt"
	"time"
)

// WorkflowRenderer renders workflows by evaluating expressions
type WorkflowRenderer struct {
	engine        *Engine
	replacer      *ExpressionReplacer
	envMerger     *EnvMerger
	condEvaluator *ConditionEvaluator
}

// NewWorkflowRenderer creates a new workflow renderer
func NewWorkflowRenderer() *WorkflowRenderer {
	engine := NewEngine(1 * time.Second)
	return &WorkflowRenderer{
		engine:        engine,
		replacer:      NewExpressionReplacer(engine),
		envMerger:     NewEnvMerger(engine),
		condEvaluator: NewConditionEvaluator(engine),
	}
}

// RenderWorkflow renders an entire workflow with expression evaluation
func (r *WorkflowRenderer) RenderWorkflow(workflow *Workflow) (*Workflow, error) {
	// Build base context
	ctx := NewContextBuilder(workflow).Build()

	// Render workflow-level env
	renderedWorkflowEnv, err := r.renderEnvMap(workflow.Env, ctx)
	if err != nil {
		return nil, fmt.Errorf("render workflow env: %w", err)
	}

	rendered := &Workflow{
		Name:       workflow.Name,
		On:         workflow.On,
		Vars:       workflow.Vars,
		Env:        renderedWorkflowEnv,
		Jobs:       make(map[string]*Job),
		SourceFile: workflow.SourceFile,
		LineMap:    workflow.LineMap,
	}

	// Render each job
	for jobName, job := range workflow.Jobs {
		renderedJob, err := r.RenderJob(workflow, job, ctx)
		if err != nil {
			return nil, fmt.Errorf("render job %s: %w", jobName, err)
		}
		rendered.Jobs[jobName] = renderedJob
	}

	return rendered, nil
}

// RenderJob renders a job with expression evaluation
func (r *WorkflowRenderer) RenderJob(workflow *Workflow, job *Job, baseCtx *EvalContext) (*Job, error) {
	// Update context with job info
	ctx := NewContextBuilder(workflow).WithJob(job).Build()
	ctx.Steps = baseCtx.Steps // Preserve steps outputs

	// Render job-level env
	renderedJobEnv, err := r.renderEnvMap(job.Env, ctx)
	if err != nil {
		return nil, fmt.Errorf("render job env: %w", err)
	}

	rendered := &Job{
		Name:            job.Name,
		RunsOn:          job.RunsOn,
		TimeoutMinutes:  job.TimeoutMinutes,
		Needs:           job.Needs,
		Env:             renderedJobEnv,
		Steps:           make([]*Step, 0),
		ContinueOnError: job.ContinueOnError,
		LineNum:         job.LineNum,
	}

	// Render each step
	for _, step := range job.Steps {
		renderedStep, err := r.RenderStep(workflow, job, step, ctx)
		if err != nil {
			return nil, err
		}

		// Step may be skipped due to if condition
		if renderedStep != nil {
			rendered.Steps = append(rendered.Steps, renderedStep)
		}
	}

	return rendered, nil
}

// RenderStep renders a step with expression evaluation and if condition
func (r *WorkflowRenderer) RenderStep(workflow *Workflow, job *Job, step *Step, ctx *EvalContext) (*Step, error) {
	// 1. Evaluate if condition
	if step.If != "" {
		shouldRun, err := r.condEvaluator.Evaluate(step.If, ctx)
		if err != nil {
			return nil, fmt.Errorf("evaluate if condition for step %s: %w", step.Name, err)
		}

		if !shouldRun {
			// Skip this step
			return nil, nil
		}
	}

	// 2. Render step.With parameters
	renderedWith, err := r.replacer.ReplaceInMap(step.With, ctx)
	if err != nil {
		return nil, fmt.Errorf("render step.with for %s: %w", step.Name, err)
	}

	// 3. Merge step-level environment variables
	renderedEnv, err := r.envMerger.MergeStepEnv(workflow, job, step, ctx)
	if err != nil {
		return nil, fmt.Errorf("merge step env for %s: %w", step.Name, err)
	}

	return &Step{
		Name:            step.Name,
		Uses:            step.Uses,
		With:            renderedWith,
		TimeoutMinutes:  step.TimeoutMinutes,
		ContinueOnError: step.ContinueOnError,
		If:              step.If,
		Env:             renderedEnv,
		Index:           step.Index,
		LineNum:         step.LineNum,
	}, nil
}

// renderEnvMap renders expressions in environment variable map
func (r *WorkflowRenderer) renderEnvMap(env map[string]string, ctx *EvalContext) (map[string]string, error) {
	if env == nil {
		return nil, nil
	}

	result := make(map[string]string)
	for k, v := range env {
		rendered, err := r.replacer.Replace(v, ctx)
		if err != nil {
			return nil, fmt.Errorf("render env %s: %w", k, err)
		}
		result[k] = rendered
	}

	return result, nil
}

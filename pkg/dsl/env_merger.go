package dsl

import (
	"fmt"
)

// EnvMerger handles three-level environment variable merging
type EnvMerger struct {
	engine *Engine
}

// NewEnvMerger creates a new environment merger
func NewEnvMerger(engine *Engine) *EnvMerger {
	return &EnvMerger{engine: engine}
}

// MergeStepEnv merges environment variables for a step (step > job > workflow)
func (m *EnvMerger) MergeStepEnv(
	workflow *Workflow,
	job *Job,
	step *Step,
	ctx *EvalContext,
) (map[string]string, error) {
	env := make(map[string]string)

	// 1. Workflow level
	if workflow.Env != nil {
		for k, v := range workflow.Env {
			rendered, err := m.renderEnvValue(v, ctx)
			if err != nil {
				return nil, fmt.Errorf("render workflow env %s: %w", k, err)
			}
			env[k] = rendered
		}
	}

	// 2. Job level (overrides workflow)
	if job.Env != nil {
		for k, v := range job.Env {
			rendered, err := m.renderEnvValue(v, ctx)
			if err != nil {
				return nil, fmt.Errorf("render job env %s: %w", k, err)
			}
			env[k] = rendered
		}
	}

	// 3. Step level (overrides job)
	if step.Env != nil {
		for k, v := range step.Env {
			rendered, err := m.renderEnvValue(v, ctx)
			if err != nil {
				return nil, fmt.Errorf("render step env %s: %w", k, err)
			}
			env[k] = rendered
		}
	}

	return env, nil
}

// renderEnvValue renders expressions in environment variable values
func (m *EnvMerger) renderEnvValue(value string, ctx *EvalContext) (string, error) {
	replacer := NewExpressionReplacer(m.engine)
	return replacer.Replace(value, ctx)
}

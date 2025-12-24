package dsl

import (
	"fmt"
	"time"
)

// JobOutputComputer computes job outputs from expressions
type JobOutputComputer struct {
	engine   *Engine
	replacer *ExpressionReplacer
}

// NewJobOutputComputer creates a new job output computer
func NewJobOutputComputer() *JobOutputComputer {
	engine := NewEngine(1 * time.Second)
	return &JobOutputComputer{
		engine:   engine,
		replacer: NewExpressionReplacer(engine),
	}
}

// Compute evaluates all job output expressions and returns the results
func (c *JobOutputComputer) Compute(job *Job, evalCtx *EvalContext) (map[string]string, error) {
	outputs := make(map[string]string)

	for key, valueExpr := range job.Outputs {
		// Render expression (replace ${{ ... }})
		value, err := c.replacer.Replace(valueExpr, evalCtx)
		if err != nil {
			return nil, fmt.Errorf("compute job output %s: %w", key, err)
		}

		outputs[key] = value
	}

	return outputs, nil
}

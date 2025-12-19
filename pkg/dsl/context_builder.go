package dsl

// ContextBuilder builds an EvalContext from workflow components
type ContextBuilder struct {
	workflow *Workflow
	job      *Job
	matrix   map[string]interface{}
	runner   map[string]interface{}
	inputs   map[string]interface{}
	secrets  map[string]string
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(workflow *Workflow) *ContextBuilder {
	return &ContextBuilder{
		workflow: workflow,
		runner:   make(map[string]interface{}),
		inputs:   make(map[string]interface{}),
		secrets:  make(map[string]string),
	}
}

// WithJob sets the job context
func (b *ContextBuilder) WithJob(job *Job) *ContextBuilder {
	b.job = job
	return b
}

// WithMatrix sets the matrix context
func (b *ContextBuilder) WithMatrix(matrix map[string]interface{}) *ContextBuilder {
	b.matrix = matrix
	return b
}

// WithRunner sets the runner context
func (b *ContextBuilder) WithRunner(runner map[string]interface{}) *ContextBuilder {
	b.runner = runner
	return b
}

// WithInputs sets the inputs context
func (b *ContextBuilder) WithInputs(inputs map[string]interface{}) *ContextBuilder {
	b.inputs = inputs
	return b
}

// WithSecrets sets the secrets context
func (b *ContextBuilder) WithSecrets(secrets map[string]string) *ContextBuilder {
	b.secrets = secrets
	return b
}

// Build constructs the final EvalContext
func (b *ContextBuilder) Build() *EvalContext {
	ctx := &EvalContext{
		Workflow: make(map[string]interface{}),
		Vars:     make(map[string]interface{}),
		Env:      b.mergeEnv(),
		Matrix:   b.matrix, // Story 1.6: Matrix变量
		Runner:   b.runner,
		Steps:    make(map[string]interface{}),
		Inputs:   b.inputs,
		Secrets:  b.secrets,
	}

	// Register built-in functions
	funcs := GetBuiltinFunctions()
	ctx.Len = funcs["len"].(func(interface{}) (int, error))
	ctx.Upper = funcs["upper"].(func(string) string)
	ctx.Lower = funcs["lower"].(func(string) string)
	ctx.Trim = funcs["trim"].(func(string) string)
	ctx.Split = funcs["split"].(func(string, string) []string)
	ctx.Join = funcs["join"].(func([]string, string) string)
	ctx.Format = funcs["format"].(func(string, ...interface{}) string)
	ctx.Contains = funcs["contains"].(func(string, string) bool)
	ctx.StartsWith = funcs["startsWith"].(func(string, string) bool)
	ctx.EndsWith = funcs["endsWith"].(func(string, string) bool)
	ctx.ToJSON = funcs["toJSON"].(func(interface{}) (string, error))
	ctx.FromJSON = funcs["fromJSON"].(func(string) (interface{}, error))
	ctx.Always = funcs["always"].(func() bool)

	// Build workflow context
	ctx.Workflow["name"] = b.workflow.Name
	// Note: id, run_id, run_number will be populated at runtime by Temporal

	// Copy vars
	if b.workflow.Vars != nil {
		for k, v := range b.workflow.Vars {
			ctx.Vars[k] = v
		}
	}

	// Build job context if available
	if b.job != nil {
		ctx.Job = map[string]interface{}{
			"id":   b.job.Name,
			"name": b.job.Name,
			// status will be updated at runtime
		}
	}

	// Story 1.5: 初始化条件函数 (默认状态为success)
	jobStatus := "success"
	ctx.Success = MakeSuccessFunc(jobStatus)
	ctx.Failure = MakeFailureFunc(jobStatus)
	ctx.Cancelled = MakeCancelledFunc(jobStatus)

	return ctx
}

// mergeEnv merges environment variables from workflow and job levels
func (b *ContextBuilder) mergeEnv() map[string]string {
	env := make(map[string]string)

	// 1. Workflow level
	if b.workflow.Env != nil {
		for k, v := range b.workflow.Env {
			env[k] = v
		}
	}

	// 2. Job level (overrides workflow)
	if b.job != nil && b.job.Env != nil {
		for k, v := range b.job.Env {
			env[k] = v
		}
	}

	// 3. Step level will be merged at execution time

	return env
}

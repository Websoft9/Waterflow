package dsl

// SerializableEvalContext is a JSON-serializable version of EvalContext
// for Temporal workflow/activity parameter passing.
// It contains only data fields, without function references.
type SerializableEvalContext struct {
	Workflow map[string]interface{} `json:"workflow"`
	Job      map[string]interface{} `json:"job"`
	Steps    map[string]interface{} `json:"steps"`
	Vars     map[string]interface{} `json:"vars"`
	Env      map[string]string      `json:"env"`
	Matrix   map[string]interface{} `json:"matrix"`
	Runner   map[string]interface{} `json:"runner"`
	Inputs   map[string]interface{} `json:"inputs"`
	Secrets  map[string]string      `json:"secrets"`
	Needs    map[string]interface{} `json:"needs"`
}

// ToSerializable converts EvalContext to SerializableEvalContext
func (ctx *EvalContext) ToSerializable() *SerializableEvalContext {
	return &SerializableEvalContext{
		Workflow: ctx.Workflow,
		Job:      ctx.Job,
		Steps:    ctx.Steps,
		Vars:     ctx.Vars,
		Env:      ctx.Env,
		Matrix:   ctx.Matrix,
		Runner:   ctx.Runner,
		Inputs:   ctx.Inputs,
		Secrets:  ctx.Secrets,
		Needs:    ctx.Needs,
	}
}

// ToEvalContext converts SerializableEvalContext back to EvalContext with functions
func (s *SerializableEvalContext) ToEvalContext() *EvalContext {
	ctx := &EvalContext{
		Workflow: s.Workflow,
		Job:      s.Job,
		Steps:    s.Steps,
		Vars:     s.Vars,
		Env:      s.Env,
		Matrix:   s.Matrix,
		Runner:   s.Runner,
		Inputs:   s.Inputs,
		Secrets:  s.Secrets,
		Needs:    s.Needs,
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

	// Initialize context-dependent functions with job status
	var jobStatus string
	if ctx.Job != nil {
		if status, ok := ctx.Job["status"].(string); ok {
			jobStatus = status
		}
	}
	ctx.Success = MakeSuccessFunc(jobStatus)
	ctx.Failure = MakeFailureFunc(jobStatus)
	ctx.Cancelled = MakeCancelledFunc(jobStatus)

	return ctx
}

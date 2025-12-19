package dsl

// EvalContext represents the execution context for expression evaluation
type EvalContext struct {
	Workflow map[string]interface{} `expr:"workflow"`
	Job      map[string]interface{} `expr:"job"`
	Steps    map[string]interface{} `expr:"steps"`
	Vars     map[string]interface{} `expr:"vars"`
	Env      map[string]string      `expr:"env"`
	Matrix   map[string]interface{} `expr:"matrix"` // Story 1.6: Matrix变量
	Runner   map[string]interface{} `expr:"runner"`
	Inputs   map[string]interface{} `expr:"inputs"`
	Secrets  map[string]string      `expr:"secrets"`
	Needs    map[string]interface{} `expr:"needs"` // Story 1.5: Job依赖输出

	// Built-in functions
	Len        func(interface{}) (int, error)      `expr:"len"`
	Upper      func(string) string                 `expr:"upper"`
	Lower      func(string) string                 `expr:"lower"`
	Trim       func(string) string                 `expr:"trim"`
	Split      func(string, string) []string       `expr:"split"`
	Join       func([]string, string) string       `expr:"join"`
	Format     func(string, ...interface{}) string `expr:"format"`
	Contains   func(string, string) bool           `expr:"contains"`
	StartsWith func(string, string) bool           `expr:"startsWith"`
	EndsWith   func(string, string) bool           `expr:"endsWith"`
	ToJSON     func(interface{}) (string, error)   `expr:"toJSON"`
	FromJSON   func(string) (interface{}, error)   `expr:"fromJSON"`
	Always     func() bool                         `expr:"always"`

	// Story 1.5: 条件函数 (上下文相关)
	Success   func() bool `expr:"success"`
	Failure   func() bool `expr:"failure"`
	Cancelled func() bool `expr:"cancelled"`
}

// UpdateJobStatus updates job status and re-initializes condition functions
func (ctx *EvalContext) UpdateJobStatus(status string) {
	if ctx.Job == nil {
		ctx.Job = make(map[string]interface{})
	}
	ctx.Job["status"] = status

	// Re-create condition functions with new status
	ctx.Success = MakeSuccessFunc(status)
	ctx.Failure = MakeFailureFunc(status)
	ctx.Cancelled = MakeCancelledFunc(status)
}

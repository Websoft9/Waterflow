package dsl

// mockContextBuilder creates a minimal EvalContext for testing
func mockContextBuilder() *EvalContext {
	ctx := &EvalContext{
		Workflow: map[string]interface{}{"name": "test"},
		Job:      make(map[string]interface{}),
		Steps:    make(map[string]interface{}),
		Vars:     make(map[string]interface{}),
		Env:      make(map[string]string),
		Runner:   make(map[string]interface{}),
		Inputs:   make(map[string]interface{}),
		Secrets:  make(map[string]string),
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

	return ctx
}

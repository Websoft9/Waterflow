package dsl

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEngine_SandboxSecurity tests that expressions cannot access files or system
func TestEngine_SandboxSecurity(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()

	tests := []struct {
		name       string
		expression string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "cannot access undefined functions",
			expression: `exec("ls")`,
			wantErr:    true,
			errMsg:     "cannot fetch",
		},
		{
			name:       "cannot access system functions",
			expression: `system("cat /etc/passwd")`,
			wantErr:    true,
			errMsg:     "cannot fetch",
		},
		{
			name:       "cannot use eval",
			expression: `eval("malicious code")`,
			wantErr:    true,
			errMsg:     "cannot fetch",
		},
		{
			name:       "cannot access file operations",
			expression: `readFile("/etc/passwd")`,
			wantErr:    true,
			errMsg:     "cannot fetch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.Evaluate(tt.expression, ctx)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestEngine_AllowedBuiltinFunctions tests only whitelisted functions are available
func TestEngine_AllowedBuiltinFunctions(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"text": "hello",
	}

	// Allowed functions should work
	allowedTests := []struct {
		name       string
		expression string
	}{
		{"len", `len("test")`},
		{"upper", `upper(vars.text)`},
		{"lower", `lower("HELLO")`},
		{"trim", `trim("  hello  ")`},
		{"format", `format("Hello {0}", "World")`},
	}

	for _, tt := range allowedTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.Evaluate(tt.expression, ctx)
			assert.NoError(t, err, "Allowed function should work: %s", tt.name)
		})
	}
}

// TestEngine_NoEnvironmentAccess tests expressions cannot access process environment
func TestEngine_NoEnvironmentAccess(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()

	// Set a real environment variable
	require.NoError(t, os.Setenv("SECRET_TEST_VAR", "secret_value"))
	defer func() { _ = os.Unsetenv("SECRET_TEST_VAR") }()

	// Expression should not be able to access it unless in ctx.Env
	_, err := engine.Evaluate(`getenv("SECRET_TEST_VAR")`, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot fetch")
}

// TestEngine_ContextIsolation tests that context is properly isolated
func TestEngine_ContextIsolation(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	ctx1 := mockContextBuilder()
	ctx1.Vars = map[string]interface{}{
		"value": "context1",
	}

	ctx2 := mockContextBuilder()
	ctx2.Vars = map[string]interface{}{
		"value": "context2",
	}

	// Evaluate with ctx1
	result1, err := engine.Evaluate("vars.value", ctx1)
	require.NoError(t, err)
	assert.Equal(t, "context1", result1)

	// Evaluate with ctx2
	result2, err := engine.Evaluate("vars.value", ctx2)
	require.NoError(t, err)
	assert.Equal(t, "context2", result2)

	// Contexts should not interfere
	assert.NotEqual(t, result1, result2)
}

// TestEngine_NoCodeInjection tests protection against code injection
func TestEngine_NoCodeInjection(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"userInput": "; rm -rf /",
	}

	// User input should be treated as data, not code
	result, err := engine.Evaluate(`vars.userInput`, ctx)
	require.NoError(t, err)
	assert.Equal(t, "; rm -rf /", result)

	// Attempt to inject code via string concatenation
	result2, err := engine.Evaluate(`"echo " + vars.userInput`, ctx)
	require.NoError(t, err)
	assert.Equal(t, "echo ; rm -rf /", result2)
	// Should be a string, not executed
}

// TestEngine_MemorySafety tests expressions don't cause memory issues
func TestEngine_MemorySafety(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()

	// Large string operations should be safe
	largeString := make([]byte, 10000)
	for i := range largeString {
		largeString[i] = 'a'
	}

	ctx.Vars = map[string]interface{}{
		"large": string(largeString),
	}

	result, err := engine.Evaluate(`len(vars.large)`, ctx)
	require.NoError(t, err)
	assert.Equal(t, 10000, result)
}

// TestExpressionReplacer_SecretsMasking tests secrets are properly handled
func TestExpressionReplacer_SecretsMasking(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()
	ctx.Secrets = map[string]string{
		"api_key": "super_secret_key_12345",
	}

	// Secrets should be accessible via expression
	result, err := replacer.Replace("API Key: ${{ secrets.api_key }}", ctx)
	require.NoError(t, err)
	assert.Equal(t, "API Key: super_secret_key_12345", result)

	// Note: Actual masking in logs happens at logger level, not here
	// This test verifies the value can be retrieved
}

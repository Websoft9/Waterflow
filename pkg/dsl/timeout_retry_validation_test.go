package dsl_test

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/dsl/node/builtin"
	"github.com/stretchr/testify/assert"
)

// setupValidatorForTimeoutRetry 设置测试验证器
func setupValidatorForTimeoutRetry() *dsl.SemanticValidator {
	registry := node.NewRegistry()
	if err := registry.Register(&builtin.CheckoutNode{}); err != nil {
		panic(err)
	}
	if err := registry.Register(&builtin.RunNode{}); err != nil {
		panic(err)
	}
	return dsl.NewSemanticValidator(registry)
}

func TestSemanticValidator_ValidateJobTimeout(t *testing.T) {
	validator := setupValidatorForTimeoutRetry()

	tests := []struct {
		name        string
		workflow    *dsl.Workflow
		expectError bool
		errorField  string
	}{
		{
			name: "Valid job timeout",
			workflow: &dsl.Workflow{
				Name: "test",
				Jobs: map[string]*dsl.Job{
					"build": {
						Name:           "build",
						TimeoutMinutes: 120,
						RunsOn:         "linux",
						Steps: []*dsl.Step{{
							Uses: "run@v1",
							With: map[string]interface{}{"command": "echo test"},
						}},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Negative job timeout",
			workflow: &dsl.Workflow{
				Name: "test",
				Jobs: map[string]*dsl.Job{
					"build": {
						Name:           "build",
						TimeoutMinutes: -10,
						RunsOn:         "linux",
						Steps: []*dsl.Step{{
							Uses: "run@v1",
							With: map[string]interface{}{"command": "echo test"},
						}},
					},
				},
			},
			expectError: true,
			errorField:  "jobs.build.timeout-minutes",
		},
		{
			name: "Timeout exceeds 24 hours",
			workflow: &dsl.Workflow{
				Name: "test",
				Jobs: map[string]*dsl.Job{
					"build": {
						Name:           "build",
						TimeoutMinutes: 1500,
						RunsOn:         "linux",
						Steps: []*dsl.Step{{
							Uses: "run@v1",
							With: map[string]interface{}{"command": "echo test"},
						}},
					},
				},
			},
			expectError: true,
			errorField:  "jobs.build.timeout-minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.workflow, []byte{})

			if tt.expectError {
				assert.Error(t, err)
				validationErr, ok := err.(*dsl.ValidationError)
				assert.True(t, ok)
				assert.NotEmpty(t, validationErr.Errors)
				found := false
				for _, e := range validationErr.Errors {
					if e.Field == tt.errorField {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected error field %s not found", tt.errorField)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSemanticValidator_ValidateStepTimeout(t *testing.T) {
	validator := setupValidatorForTimeoutRetry()

	tests := []struct {
		name        string
		workflow    *dsl.Workflow
		expectError bool
		errorField  string
	}{
		{
			name: "Valid step timeout",
			workflow: &dsl.Workflow{
				Name: "test",
				Jobs: map[string]*dsl.Job{
					"build": {
						Name:   "build",
						RunsOn: "linux",
						Steps: []*dsl.Step{
							{
								Uses:           "run@v1",
								TimeoutMinutes: 30,
								With:           map[string]interface{}{"command": "echo test"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Negative step timeout",
			workflow: &dsl.Workflow{
				Name: "test",
				Jobs: map[string]*dsl.Job{
					"build": {
						Name:   "build",
						RunsOn: "linux",
						Steps: []*dsl.Step{
							{
								Uses:           "run@v1",
								TimeoutMinutes: -5,
								With:           map[string]interface{}{"command": "echo test"},
							},
						},
					},
				},
			},
			expectError: true,
			errorField:  "jobs.build.steps[0].timeout-minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.workflow, []byte{})

			if tt.expectError {
				assert.Error(t, err)
				validationErr, ok := err.(*dsl.ValidationError)
				assert.True(t, ok)
				assert.NotEmpty(t, validationErr.Errors)
				found := false
				for _, e := range validationErr.Errors {
					if e.Field == tt.errorField {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected error field %s not found", tt.errorField)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSemanticValidator_ValidateRetryStrategy(t *testing.T) {
	validator := setupValidatorForTimeoutRetry()

	tests := []struct {
		name        string
		workflow    *dsl.Workflow
		expectError bool
		errorField  string
	}{
		{
			name: "Valid retry strategy",
			workflow: &dsl.Workflow{
				Name: "test",
				Jobs: map[string]*dsl.Job{
					"build": {
						Name:   "build",
						RunsOn: "linux",
						Steps: []*dsl.Step{
							{
								Uses: "run@v1",
								With: map[string]interface{}{"command": "echo test"},
								RetryStrategy: &dsl.RetryStrategy{
									MaxAttempts:        5,
									InitialInterval:    "2s",
									BackoffCoefficient: 1.5,
									MaxInterval:        "30s",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "max-attempts too low",
			workflow: &dsl.Workflow{
				Name: "test",
				Jobs: map[string]*dsl.Job{
					"build": {
						Name:   "build",
						RunsOn: "linux",
						Steps: []*dsl.Step{
							{
								Uses: "run@v1",
								With: map[string]interface{}{"command": "echo test"},
								RetryStrategy: &dsl.RetryStrategy{
									MaxAttempts: 0,
								},
							},
						},
					},
				},
			},
			expectError: true,
			errorField:  "jobs.build.steps[0].retry-strategy.max-attempts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.workflow, []byte{})

			if tt.expectError {
				assert.Error(t, err)
				validationErr, ok := err.(*dsl.ValidationError)
				assert.True(t, ok)
				assert.NotEmpty(t, validationErr.Errors)
				found := false
				for _, e := range validationErr.Errors {
					if e.Field == tt.errorField {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected error field %s not found", tt.errorField)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

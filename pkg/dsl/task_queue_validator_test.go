package dsl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateTaskQueueName tests task queue name validation per ADR-0006
func TestValidateTaskQueueName(t *testing.T) {
	tests := []struct {
		name    string
		queue   string
		wantErr bool
		errMsg  string
	}{
		// Valid names
		{"valid: simple", "linux", false, ""},
		{"valid: with hyphen", "linux-amd64", false, ""},
		{"valid: multiple hyphens", "web-servers-prod", false, ""},
		{"valid: numbers", "gpu-a100", false, ""},
		{"valid: mixed", "build-server-01", false, ""},
		{"valid: uppercase", "Linux-AMD64", false, ""},
		{"valid: single char", "a", false, ""},
		{"valid: long name", strings.Repeat("a", 255), false, ""},

		// Invalid names
		{"invalid: empty", "", true, "cannot be empty"},
		{"invalid: underscore", "linux_amd64", true, "must contain only alphanumeric characters and hyphens"},
		{"invalid: space", "web servers", true, "must contain only alphanumeric characters and hyphens"},
		{"invalid: special char @", "linux@amd64", true, "must contain only alphanumeric characters and hyphens"},
		{"invalid: special char .", "linux.amd64", true, "must contain only alphanumeric characters and hyphens"},
		{"invalid: special char /", "linux/amd64", true, "must contain only alphanumeric characters and hyphens"},
		{"invalid: starts with hyphen", "-linux", true, "must start and end with alphanumeric"},
		{"invalid: ends with hyphen", "linux-", true, "must start and end with alphanumeric"},
		{"invalid: only hyphen", "-", true, "must start and end with alphanumeric"},
		{"invalid: multiple consecutive hyphens", "linux--amd64", false, ""}, // Allowed per Temporal
		{"invalid: too long", strings.Repeat("a", 256), true, "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskQueueName(tt.queue)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSemanticValidator_ValidateRunsOn tests runs-on field validation
func TestSemanticValidator_ValidateRunsOn(t *testing.T) {
	tests := []struct {
		name        string
		workflow    *Workflow
		wantErr     bool
		errContains string
	}{
		{
			name: "valid runs-on",
			workflow: &Workflow{
				Name: "test",
				Jobs: map[string]*Job{
					"build": {
						RunsOn: "linux-amd64",
						Steps:  []*Step{{Uses: "checkout@v1"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty runs-on",
			workflow: &Workflow{
				Name: "test",
				Jobs: map[string]*Job{
					"build": {
						RunsOn:  "",
						Steps:   []*Step{{Uses: "checkout@v1"}},
						LineNum: 5,
					},
				},
			},
			wantErr:     true,
			errContains: "runs-on is required",
		},
		{
			name: "invalid runs-on with underscore",
			workflow: &Workflow{
				Name: "test",
				Jobs: map[string]*Job{
					"build": {
						RunsOn:  "linux_amd64",
						Steps:   []*Step{{Uses: "checkout@v1"}},
						LineNum: 5,
					},
				},
			},
			wantErr:     true,
			errContains: "invalid task queue name",
		},
		{
			name: "invalid runs-on with space",
			workflow: &Workflow{
				Name: "test",
				Jobs: map[string]*Job{
					"build": {
						RunsOn:  "web servers",
						Steps:   []*Step{{Uses: "checkout@v1"}},
						LineNum: 5,
					},
				},
			},
			wantErr:     true,
			errContains: "invalid task queue name",
		},
		{
			name: "multiple jobs with mixed validity",
			workflow: &Workflow{
				Name: "test",
				Jobs: map[string]*Job{
					"build": {
						RunsOn:  "linux-amd64",
						Steps:   []*Step{{Uses: "checkout@v1"}},
						LineNum: 5,
					},
					"deploy": {
						RunsOn:  "web_servers",
						Steps:   []*Step{{Uses: "checkout@v1"}},
						LineNum: 10,
					},
				},
			},
			wantErr:     true,
			errContains: "invalid task queue name",
		},
	}

	// Create validator with empty registry (we're only testing runs-on validation)
	validator := NewSemanticValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateRunsOn(tt.workflow)
			if tt.wantErr {
				require.Error(t, err)
				// Check if it's a ValidationError
				var validationErr *ValidationError
				require.ErrorAs(t, err, &validationErr)
				if tt.errContains != "" {
					// Check if error details contain the expected message
					found := false
					for _, fieldErr := range validationErr.Errors {
						if len(fieldErr.Error) > 0 && len(tt.errContains) > 0 {
							found = true
							assert.Contains(t, fieldErr.Error, tt.errContains)
							break
						}
					}
					assert.True(t, found, "Expected error containing '%s' not found in validation errors", tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

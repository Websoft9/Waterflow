package integration

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParameterValidation_RealNodeScenarios tests validation with realistic node configurations
func TestParameterValidation_RealNodeScenarios(t *testing.T) {
	tests := []struct {
		name       string
		nodeSpecs  map[string]node.ParamSpec
		inputs     map[string]interface{}
		wantErr    bool
		errType    string
		errCount   int
		errMessage string
	}{
		{
			name: "shell node - valid command",
			nodeSpecs: map[string]node.ParamSpec{
				"command": {Type: "string", Required: true},
				"workdir": {Type: "string", Required: false, Default: "/tmp"},
			},
			inputs: map[string]interface{}{
				"command": "ls -la",
			},
			wantErr: false,
		},
		{
			name: "shell node - missing required command",
			nodeSpecs: map[string]node.ParamSpec{
				"command": {Type: "string", Required: true},
			},
			inputs:     map[string]interface{}{},
			wantErr:    true,
			errType:    "Missing",
			errCount:   1,
			errMessage: "command",
		},
		{
			name: "http node - valid request",
			nodeSpecs: map[string]node.ParamSpec{
				"url":    {Type: "string", Required: true},
				"method": {Type: "string", Enum: []interface{}{"GET", "POST", "PUT", "DELETE"}},
				"timeout": {
					Type:     "int",
					MinValue: ptrFloat64(1),
					MaxValue: ptrFloat64(300),
				},
			},
			inputs: map[string]interface{}{
				"url":     "https://api.example.com",
				"method":  "GET",
				"timeout": 30,
			},
			wantErr: false,
		},
		{
			name: "http node - invalid method",
			nodeSpecs: map[string]node.ParamSpec{
				"url":    {Type: "string", Required: true},
				"method": {Type: "string", Enum: []interface{}{"GET", "POST", "PUT", "DELETE"}},
			},
			inputs: map[string]interface{}{
				"url":    "https://api.example.com",
				"method": "PATCH",
			},
			wantErr:    true,
			errType:    "EnumViolation",
			errCount:   1,
			errMessage: "PATCH",
		},
		{
			name: "http node - timeout out of range",
			nodeSpecs: map[string]node.ParamSpec{
				"url": {Type: "string", Required: true},
				"timeout": {
					Type:     "int",
					MinValue: ptrFloat64(1),
					MaxValue: ptrFloat64(300),
				},
			},
			inputs: map[string]interface{}{
				"url":     "https://api.example.com",
				"timeout": 500,
			},
			wantErr:    true,
			errType:    "RangeViolation",
			errCount:   1,
			errMessage: "maximum",
		},
		{
			name: "docker node - multiple validation errors",
			nodeSpecs: map[string]node.ParamSpec{
				"image": {
					Type:     "string",
					Required: true,
					Pattern:  `^[a-z0-9-_/:.]+$`,
				},
				"replicas": {
					Type:     "int",
					Required: true,
					MinValue: ptrFloat64(1),
					MaxValue: ptrFloat64(100),
				},
				"network_mode": {
					Type: "string",
					Enum: []interface{}{"bridge", "host", "none"},
				},
			},
			inputs: map[string]interface{}{
				// image: missing
				"replicas":     0,         // below minimum
				"network_mode": "invalid", // not in enum
			},
			wantErr:  true,
			errCount: 3, // missing, range violation, enum violation
		},
		{
			name: "file node - path pattern validation",
			nodeSpecs: map[string]node.ParamSpec{
				"path": {
					Type:     "string",
					Required: true,
					Pattern:  `^/[a-zA-Z0-9_/.-]+$`, // Unix absolute path
				},
			},
			inputs: map[string]interface{}{
				"path": "/opt/waterflow/config.yaml",
			},
			wantErr: false,
		},
		{
			name: "file node - invalid path pattern",
			nodeSpecs: map[string]node.ParamSpec{
				"path": {
					Type:     "string",
					Required: true,
					Pattern:  `^/[a-zA-Z0-9_/.-]+$`,
				},
			},
			inputs: map[string]interface{}{
				"path": "relative/path", // not absolute
			},
			wantErr:    true,
			errType:    "PatternMismatch",
			errCount:   1,
			errMessage: "pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := node.ValidateInputs(tt.inputs, tt.nodeSpecs)

			if tt.wantErr {
				require.Error(t, err)

				validationErr, ok := err.(*node.InputValidationError)
				assert.True(t, ok, "error should be InputValidationError")
				assert.True(t, validationErr.NonRetryable())

				if tt.errCount > 0 {
					assert.Len(t, validationErr.Errors, tt.errCount)
				}

				if tt.errType != "" {
					assert.Equal(t, tt.errType, validationErr.Errors[0].ErrorType)
				}

				if tt.errMessage != "" {
					assert.Contains(t, validationErr.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ptrFloat64 is a helper to create *float64 for test cases
func ptrFloat64(v float64) *float64 {
	return &v
}

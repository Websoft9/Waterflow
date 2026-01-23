package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVars(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "single string variable",
			args: []string{"env=production"},
			want: map[string]interface{}{
				"env": "production",
			},
			wantErr: false,
		},
		{
			name: "multiple variables with different types",
			args: []string{
				"env=production",
				"timeout=300",
				"debug=true",
				"replicas=3",
			},
			want: map[string]interface{}{
				"env":      "production",
				"timeout":  float64(300), // JSON unmarshaling produces float64
				"debug":    true,
				"replicas": float64(3), // JSON unmarshaling produces float64
			},
			wantErr: false,
		},
		{
			name: "float values",
			args: []string{"timeout=30.5"},
			want: map[string]interface{}{
				"timeout": 30.5,
			},
			wantErr: false,
		},
		{
			name: "boolean values",
			args: []string{
				"enabled=true",
				"disabled=false",
			},
			want: map[string]interface{}{
				"enabled":  true,
				"disabled": false,
			},
			wantErr: false,
		},
		{
			name: "JSON object",
			args: []string{`config={"timeout":300,"retries":3}`},
			want: map[string]interface{}{
				"config": map[string]interface{}{
					"timeout": float64(300),
					"retries": float64(3),
				},
			},
			wantErr: false,
		},
		{
			name: "JSON array",
			args: []string{`regions=["us-east-1","eu-west-1"]`},
			want: map[string]interface{}{
				"regions": []interface{}{"us-east-1", "eu-west-1"},
			},
			wantErr: false,
		},
		{
			name:    "empty args",
			args:    []string{},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid format - missing equals",
			args:    []string{"invalid"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid format - empty key",
			args:    []string{"=value"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVars(tt.args)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseVarArg(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		wantKey   string
		wantValue interface{}
		wantErr   bool
	}{
		{
			name:      "string value",
			arg:       "name=test",
			wantKey:   "name",
			wantValue: "test",
			wantErr:   false,
		},
		{
			name:      "integer value",
			arg:       "count=42",
			wantKey:   "count",
			wantValue: float64(42), // JSON unmarshaling produces float64
			wantErr:   false,
		},
		{
			name:      "boolean true",
			arg:       "enabled=true",
			wantKey:   "enabled",
			wantValue: true,
			wantErr:   false,
		},
		{
			name:      "boolean false",
			arg:       "enabled=false",
			wantKey:   "enabled",
			wantValue: false,
			wantErr:   false,
		},
		{
			name:      "float value",
			arg:       "ratio=0.75",
			wantKey:   "ratio",
			wantValue: 0.75,
			wantErr:   false,
		},
		{
			name:      "value with equals sign",
			arg:       "query=a=b",
			wantKey:   "query",
			wantValue: "a=b",
			wantErr:   false,
		},
		{
			name:      "whitespace handling",
			arg:       "  key  =  value  ",
			wantKey:   "key",
			wantValue: "value",
			wantErr:   false,
		},
		{
			name:    "missing equals",
			arg:     "invalid",
			wantErr: true,
		},
		{
			name:    "empty key",
			arg:     "=value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, err := parseVarArg(tt.arg)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantKey, key)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  interface{}
	}{
		{
			name:  "string value",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "integer",
			input: "42",
			want:  float64(42), // JSON unmarshaling produces float64
		},
		{
			name:  "negative integer",
			input: "-10",
			want:  float64(-10), // JSON unmarshaling produces float64
		},
		{
			name:  "float",
			input: "3.14",
			want:  3.14,
		},
		{
			name:  "boolean true",
			input: "true",
			want:  true,
		},
		{
			name:  "boolean false",
			input: "false",
			want:  false,
		},
		{
			name:  "JSON object",
			input: `{"key":"value"}`,
			want: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:  "JSON array",
			input: `[1,2,3]`,
			want:  []interface{}{float64(1), float64(2), float64(3)},
		},
		{
			name:  "JSON null",
			input: "null",
			want:  nil,
		},
		{
			name:  "complex string that looks like number",
			input: "123abc",
			want:  "123abc",
		},
		{
			name:  "invalid JSON object",
			input: `{"broken":`,
			want:  `{"broken":`,
		},
		{
			name:  "invalid JSON array",
			input: `[1,2,`,
			want:  `[1,2,`,
		},
		{
			name:  "quoted string",
			input: `"hello world"`,
			want:  "hello world",
		},
		{
			name:  "negative float",
			input: "-3.14",
			want:  -3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseValue(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidWorkflowID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "valid UUID",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			want: true,
		},
		{
			name: "another valid UUID",
			id:   "123e4567-e89b-12d3-a456-426614174000",
			want: true,
		},
		{
			name: "too short",
			id:   "550e8400-e29b-41d4-a716",
			want: false,
		},
		{
			name: "missing hyphens",
			id:   "550e8400e29b41d4a716446655440000",
			want: false,
		},
		{
			name: "wrong hyphen positions",
			id:   "550e840-0e29b-41d4-a716-446655440000",
			want: false,
		},
		{
			name: "empty string",
			id:   "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidWorkflowID(tt.id)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidWorkflowID_HexValidation(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "invalid characters - all x",
			id:   "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			want: false,
		},
		{
			name: "invalid characters - contains g",
			id:   "550e8400-e29b-41d4-a716-44665544000g",
			want: false,
		},
		{
			name: "valid lowercase hex",
			id:   "abcdef12-3456-7890-abcd-ef1234567890",
			want: true,
		},
		{
			name: "valid uppercase hex",
			id:   "ABCDEF12-3456-7890-ABCD-EF1234567890",
			want: true,
		},
		{
			name: "valid mixed case",
			id:   "AbCdEf12-3456-7890-aBcD-eF1234567890",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidWorkflowID(tt.id)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsTerminalStatusInSubmit(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{
			name:   "completed",
			status: "completed",
			want:   true,
		},
		{
			name:   "failed",
			status: "failed",
			want:   true,
		},
		{
			name:   "cancelled",
			status: "cancelled",
			want:   true,
		},
		{
			name:   "running",
			status: "running",
			want:   false,
		},
		{
			name:   "pending",
			status: "pending",
			want:   false,
		},
		{
			name:   "queued",
			status: "queued",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTerminalStatusInSubmit(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestExitError tests the ExitError type
func TestExitError(t *testing.T) {
	err := &ExitError{Code: 1}
	assert.Contains(t, err.Error(), "1")

	err2 := &ExitError{Code: 42}
	assert.Contains(t, err2.Error(), "42")
}

// TestNewSubmitCmd tests submit command creation
func TestNewSubmitCmd(t *testing.T) {
	cmd := newSubmitCmd()
	require.NotNil(t, cmd)

	assert.Equal(t, "submit <workflow-file>", cmd.Use)
	assert.Contains(t, cmd.Short, "Submit")

	// Check flags exist
	assert.NotNil(t, cmd.Flags().Lookup("wait"))
	assert.NotNil(t, cmd.Flags().Lookup("follow"))
	assert.NotNil(t, cmd.Flags().Lookup("validate"))
	assert.NotNil(t, cmd.Flags().Lookup("quiet"))
	assert.NotNil(t, cmd.Flags().Lookup("format"))
	assert.NotNil(t, cmd.Flags().Lookup("var"))
}

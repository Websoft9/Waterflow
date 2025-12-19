package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinLen(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    int
		wantErr bool
	}{
		{
			name:    "string length",
			input:   "hello",
			want:    5,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: false,
		},
		{
			name:    "array length",
			input:   []interface{}{"a", "b", "c"},
			want:    3,
			wantErr: false,
		},
		{
			name:    "string array length",
			input:   []string{"a", "b"},
			want:    2,
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   123,
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinLen(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBuiltinFormat(t *testing.T) {
	tests := []struct {
		name     string
		template string
		args     []interface{}
		want     string
	}{
		{
			name:     "single argument",
			template: "Hello {0}",
			args:     []interface{}{"World"},
			want:     "Hello World",
		},
		{
			name:     "multiple arguments",
			template: "{0} v{1}",
			args:     []interface{}{"App", "1.2.3"},
			want:     "App v1.2.3",
		},
		{
			name:     "no arguments",
			template: "Hello",
			args:     []interface{}{},
			want:     "Hello",
		},
		{
			name:     "reordered placeholders",
			template: "{1} {0}",
			args:     []interface{}{"World", "Hello"},
			want:     "Hello World",
		},
		{
			name:     "numeric argument",
			template: "Count: {0}",
			args:     []interface{}{42},
			want:     "Count: 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builtinFormat(tt.template, tt.args...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuiltinToJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:    "simple object",
			input:   map[string]interface{}{"key": "value"},
			want:    `{"key":"value"}`,
			wantErr: false,
		},
		{
			name:    "array",
			input:   []interface{}{"a", "b", "c"},
			want:    `["a","b","c"]`,
			wantErr: false,
		},
		{
			name:    "string",
			input:   "hello",
			want:    `"hello"`,
			wantErr: false,
		},
		{
			name:    "number",
			input:   42,
			want:    `42`,
			wantErr: false,
		},
		{
			name:    "boolean",
			input:   true,
			want:    `true`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinToJSON(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBuiltinFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    interface{}
		wantErr bool
	}{
		{
			name:    "simple object",
			input:   `{"key":"value"}`,
			want:    map[string]interface{}{"key": "value"},
			wantErr: false,
		},
		{
			name:    "array",
			input:   `["a","b","c"]`,
			want:    []interface{}{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "string",
			input:   `"hello"`,
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "number",
			input:   `42`,
			want:    float64(42), // JSON numbers parse as float64
			wantErr: false,
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builtinFromJSON(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBuiltinAlways(t *testing.T) {
	got := builtinAlways()
	assert.True(t, got)
}

func TestContextualFunctions(t *testing.T) {
	tests := []struct {
		name      string
		jobStatus string
		testFunc  func(*ContextualFunctions) bool
		want      bool
	}{
		{
			name:      "success when status is success",
			jobStatus: "success",
			testFunc:  func(f *ContextualFunctions) bool { return f.Success() },
			want:      true,
		},
		{
			name:      "success when status is failure",
			jobStatus: "failure",
			testFunc:  func(f *ContextualFunctions) bool { return f.Success() },
			want:      false,
		},
		{
			name:      "failure when status is failure",
			jobStatus: "failure",
			testFunc:  func(f *ContextualFunctions) bool { return f.Failure() },
			want:      true,
		},
		{
			name:      "failure when status is success",
			jobStatus: "success",
			testFunc:  func(f *ContextualFunctions) bool { return f.Failure() },
			want:      false,
		},
		{
			name:      "cancelled when status is cancelled",
			jobStatus: "cancelled",
			testFunc:  func(f *ContextualFunctions) bool { return f.Cancelled() },
			want:      true,
		},
		{
			name:      "cancelled when status is success",
			jobStatus: "success",
			testFunc:  func(f *ContextualFunctions) bool { return f.Cancelled() },
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := &ContextualFunctions{JobStatus: tt.jobStatus}
			got := tt.testFunc(cf)
			assert.Equal(t, tt.want, got)
		})
	}
}

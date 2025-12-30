package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNode_ValidNode(t *testing.T) {
	node := &MockNode{
		NameValue:    "exec/shell",
		VersionValue: "v1.2.3",
		ParamsValue: map[string]ParamSpec{
			"command": {Type: "string", Required: true},
		},
		MetadataValue: NodeMetadata{
			Description: "Executes shell commands",
			Category:    "exec",
			InputSchema: map[string]ParamSpec{
				"command": {Type: "string", Required: true},
			},
			OutputSchema: map[string]interface{}{
				"stdout": "string",
			},
		},
	}

	err := ValidateNode(node)
	assert.NoError(t, err)
}

func TestValidateNode_InvalidName(t *testing.T) {
	tests := []struct {
		name      string
		nodeName  string
		wantError string
	}{
		{
			name:      "empty name",
			nodeName:  "",
			wantError: "Name() returned empty string",
		},
		{
			name:      "missing category",
			nodeName:  "shell",
			wantError: "Name() format invalid",
		},
		{
			name:      "invalid characters",
			nodeName:  "exec/shell@v1",
			wantError: "Name() format invalid",
		},
		{
			name:      "too many slashes",
			nodeName:  "exec/docker/shell",
			wantError: "Name() format invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &MockNode{
				NameValue:    tt.nodeName,
				VersionValue: "v1",
				MetadataValue: NodeMetadata{
					Description:  "Test",
					Category:     "exec",
					InputSchema:  map[string]ParamSpec{},
					OutputSchema: map[string]interface{}{},
				},
			}

			err := ValidateNode(node)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestValidateNode_InvalidVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"empty version", ""},
		{"missing v prefix", "1.2.3"},
		{"invalid format", "version1"},
		{"non-numeric", "vABC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &MockNode{
				NameValue:    "exec/shell",
				VersionValue: tt.version,
				MetadataValue: NodeMetadata{
					Description:  "Test",
					Category:     "exec",
					InputSchema:  map[string]ParamSpec{},
					OutputSchema: map[string]interface{}{},
				},
			}

			err := ValidateNode(node)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Version()")
		})
	}
}

func TestValidateNode_ValidVersionFormats(t *testing.T) {
	validVersions := []string{"v1", "v1.0", "v1.2.3", "v2", "v10.20.30"}

	for _, version := range validVersions {
		t.Run(version, func(t *testing.T) {
			node := &MockNode{
				NameValue:    "exec/shell",
				VersionValue: version,
				MetadataValue: NodeMetadata{
					Description:  "Test",
					Category:     "exec",
					InputSchema:  map[string]ParamSpec{},
					OutputSchema: map[string]interface{}{},
				},
			}

			err := ValidateNode(node)
			assert.NoError(t, err)
		})
	}
}

func TestValidateNode_InvalidMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata NodeMetadata
		wantErr  string
	}{
		{
			name: "empty description",
			metadata: NodeMetadata{
				Description:  "",
				Category:     "exec",
				InputSchema:  map[string]ParamSpec{},
				OutputSchema: map[string]interface{}{},
			},
			wantErr: "Description is empty",
		},
		{
			name: "empty category",
			metadata: NodeMetadata{
				Description:  "Test",
				Category:     "",
				InputSchema:  map[string]ParamSpec{},
				OutputSchema: map[string]interface{}{},
			},
			wantErr: "Category is empty",
		},
		{
			name: "invalid category",
			metadata: NodeMetadata{
				Description:  "Test",
				Category:     "unknown",
				InputSchema:  map[string]ParamSpec{},
				OutputSchema: map[string]interface{}{},
			},
			wantErr: "Category invalid",
		},
		{
			name: "nil input schema",
			metadata: NodeMetadata{
				Description:  "Test",
				Category:     "exec",
				InputSchema:  nil,
				OutputSchema: map[string]interface{}{},
			},
			wantErr: "InputSchema is nil",
		},
		{
			name: "nil output schema",
			metadata: NodeMetadata{
				Description:  "Test",
				Category:     "exec",
				InputSchema:  map[string]ParamSpec{},
				OutputSchema: nil,
			},
			wantErr: "OutputSchema is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &MockNode{
				NameValue:     "exec/shell",
				VersionValue:  "v1",
				MetadataValue: tt.metadata,
			}

			err := ValidateNode(node)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestValidateParamSpec_ValidTypes(t *testing.T) {
	validSpecs := []ParamSpec{
		{Type: "string", Required: true},
		{Type: "int", Required: false},
		{Type: "float"},
		{Type: "bool"},
		{Type: "object"},
		{Type: "array"},
	}

	for _, spec := range validSpecs {
		t.Run(spec.Type, func(t *testing.T) {
			err := validateParamSpec("test", spec)
			assert.NoError(t, err)
		})
	}
}

func TestValidateParamSpec_InvalidType(t *testing.T) {
	spec := ParamSpec{Type: "unknown"}
	err := validateParamSpec("test", spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

func TestValidateParamSpec_InvalidPattern(t *testing.T) {
	spec := ParamSpec{
		Type:    "string",
		Pattern: "[invalid(regex",
	}
	err := validateParamSpec("test", spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex pattern")
}

func TestValidateParamSpec_EnumValidation(t *testing.T) {
	tests := []struct {
		name    string
		spec    ParamSpec
		wantErr bool
	}{
		{
			name: "valid enum",
			spec: ParamSpec{
				Type: "string",
				Enum: []interface{}{"debug", "info", "warn", "error"},
			},
			wantErr: false,
		},
		{
			name: "enum with nil value",
			spec: ParamSpec{
				Type: "string",
				Enum: []interface{}{"valid", nil, "values"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParamSpec("test", tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateParamSpec_NumericRanges(t *testing.T) {
	min := 1.0
	max := 100.0
	invalidMin := 200.0

	tests := []struct {
		name    string
		spec    ParamSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid int range",
			spec: ParamSpec{
				Type:     "int",
				MinValue: &min,
				MaxValue: &max,
			},
			wantErr: false,
		},
		{
			name: "valid float range",
			spec: ParamSpec{
				Type:     "float",
				MinValue: &min,
				MaxValue: &max,
			},
			wantErr: false,
		},
		{
			name: "min > max",
			spec: ParamSpec{
				Type:     "int",
				MinValue: &invalidMin,
				MaxValue: &max,
			},
			wantErr: true,
			errMsg:  "MinValue",
		},
		{
			name: "range on non-numeric type",
			spec: ParamSpec{
				Type:     "string",
				MinValue: &min,
			},
			wantErr: true,
			errMsg:  "only valid for int/float",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParamSpec("test", tt.spec)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateInputs_RequiredParameters(t *testing.T) {
	specs := map[string]ParamSpec{
		"command": {Type: "string", Required: true},
		"timeout": {Type: "int", Required: false, Default: 60},
	}

	tests := []struct {
		name    string
		inputs  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "all required provided",
			inputs:  map[string]interface{}{"command": "echo hello"},
			wantErr: false,
		},
		{
			name:    "required missing",
			inputs:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name:    "all provided",
			inputs:  map[string]interface{}{"command": "ls", "timeout": 30},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputs(tt.inputs, specs)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateInputs_EnumConstraints(t *testing.T) {
	specs := map[string]ParamSpec{
		"log_level": {
			Type:     "string",
			Required: true,
			Enum:     []interface{}{"debug", "info", "warn", "error"},
		},
	}

	tests := []struct {
		name    string
		inputs  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "valid enum value",
			inputs:  map[string]interface{}{"log_level": "debug"},
			wantErr: false,
		},
		{
			name:    "invalid enum value",
			inputs:  map[string]interface{}{"log_level": "trace"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputs(tt.inputs, specs)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "not in enum")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateInputs_NumericRanges(t *testing.T) {
	min := 1.0
	max := 100.0

	specs := map[string]ParamSpec{
		"timeout": {
			Type:     "int",
			Required: true,
			MinValue: &min,
			MaxValue: &max,
		},
	}

	tests := []struct {
		name    string
		inputs  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "within range",
			inputs:  map[string]interface{}{"timeout": 50},
			wantErr: false,
		},
		{
			name:    "below minimum",
			inputs:  map[string]interface{}{"timeout": 0},
			wantErr: true,
			errMsg:  "< minimum",
		},
		{
			name:    "above maximum",
			inputs:  map[string]interface{}{"timeout": 150},
			wantErr: true,
			errMsg:  "> maximum",
		},
		{
			name:    "at minimum boundary",
			inputs:  map[string]interface{}{"timeout": 1},
			wantErr: false,
		},
		{
			name:    "at maximum boundary",
			inputs:  map[string]interface{}{"timeout": 100},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputs(tt.inputs, specs)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateInputs_StringPatterns(t *testing.T) {
	specs := map[string]ParamSpec{
		"email": {
			Type:     "string",
			Required: true,
			Pattern:  `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		},
	}

	tests := []struct {
		name    string
		inputs  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "valid email",
			inputs:  map[string]interface{}{"email": "test@example.com"},
			wantErr: false,
		},
		{
			name:    "invalid email",
			inputs:  map[string]interface{}{"email": "invalid-email"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputs(tt.inputs, specs)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "does not match pattern")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateInputs_TypeValidation(t *testing.T) {
	specs := map[string]ParamSpec{
		"count": {Type: "int", Required: true},
	}

	tests := []struct {
		name    string
		inputs  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "int type",
			inputs:  map[string]interface{}{"count": 42},
			wantErr: false,
		},
		{
			name:    "int64 type",
			inputs:  map[string]interface{}{"count": int64(42)},
			wantErr: false,
		},
		{
			name:    "float64 type",
			inputs:  map[string]interface{}{"count": float64(42)},
			wantErr: false,
		},
		{
			name:    "string type (invalid)",
			inputs:  map[string]interface{}{"count": "42"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputs(tt.inputs, specs)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNodeValidationError_Error(t *testing.T) {
	err := &NodeValidationError{
		NodeName: "exec/shell",
		Errors:   []string{"error1", "error2"},
	}

	errMsg := err.Error()
	assert.Contains(t, errMsg, "exec/shell")
	assert.Contains(t, errMsg, "error1")
	assert.Contains(t, errMsg, "error2")
}

func TestIsValidNodeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid simple", "exec/shell", true},
		{"valid with dash", "exec/shell-v2", true},
		{"valid with underscore", "docker/compose_up", true},
		{"missing category", "shell", false},
		{"too many parts", "a/b/c", false},
		{"empty category", "/shell", false},
		{"empty name", "exec/", false},
		{"uppercase", "Exec/Shell", false},
		{"special chars", "exec/shell@v1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidNodeName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"v1", "v1", true},
		{"v1.0", "v1.0", true},
		{"v1.2.3", "v1.2.3", true},
		{"v10.20.30", "v10.20.30", true},
		{"missing v", "1.2.3", false},
		{"uppercase V", "V1.0", false},
		{"non-numeric", "vABC", false},
		{"trailing dot", "v1.", false},
		{"four parts", "v1.2.3.4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidVersion(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidCategory(t *testing.T) {
	validCategories := []string{"exec", "docker", "http", "file", "flow"}
	for _, cat := range validCategories {
		t.Run(cat, func(t *testing.T) {
			assert.True(t, isValidCategory(cat))
		})
	}

	invalidCategories := []string{"unknown", "Exec", "DOCKER", ""}
	for _, cat := range invalidCategories {
		t.Run(cat, func(t *testing.T) {
			assert.False(t, isValidCategory(cat))
		})
	}
}

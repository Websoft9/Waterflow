package cmd

import (
	"testing"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
)

// Test displayNodeListGrouped
func TestDisplayNodeListGrouped(t *testing.T) {
	nodes := []client.NodeInfo{
		{
			Name:        "exec/shell",
			Version:     "v1",
			Category:    "exec",
			Description: "Execute shell commands",
		},
		{
			Name:        "docker/exec",
			Version:     "v1",
			Category:    "docker",
			Description: "Execute Docker commands",
		},
	}

	// This is a display function, just ensure it doesn't panic
	displayNodeListGrouped(nodes)
}

// Test displayNodeDetail - found
func TestDisplayNodeDetail_Found(t *testing.T) {
	nodes := []client.NodeInfo{
		{
			Name:        "exec/shell",
			Version:     "v1",
			Category:    "exec",
			Description: "Execute shell commands",
			InputSchema: map[string]map[string]interface{}{
				"command": {
					"type":     "string",
					"required": true,
				},
			},
			OutputSchema: map[string]interface{}{
				"stdout": "string",
			},
		},
	}

	// Test with full name
	err := displayNodeDetail("exec/shell@v1", nodes)
	if err != nil {
		t.Errorf("displayNodeDetail() failed: %v", err)
	}

	// Test with short name
	err = displayNodeDetail("exec/shell", nodes)
	if err != nil {
		t.Errorf("displayNodeDetail() with short name failed: %v", err)
	}
}

// Test getExampleValue
func TestGetExampleValue(t *testing.T) {
	tests := []struct {
		name     string
		param    map[string]interface{}
		expected string
	}{
		{
			name: "with default value",
			param: map[string]interface{}{
				"type":    "string",
				"default": "/bin/bash",
			},
			expected: "/bin/bash",
		},
		{
			name: "with enum",
			param: map[string]interface{}{
				"type": "string",
				"enum": []interface{}{"bash", "sh", "zsh"},
			},
			expected: "bash",
		},
		{
			name: "string type",
			param: map[string]interface{}{
				"type": "string",
			},
			expected: `"example"`,
		},
		{
			name: "int type",
			param: map[string]interface{}{
				"type": "int",
			},
			expected: "0",
		},
		{
			name: "bool type",
			param: map[string]interface{}{
				"type": "bool",
			},
			expected: "true",
		},
		{
			name: "map type",
			param: map[string]interface{}{
				"type": "map",
			},
			expected: `{"key": "value"}`,
		},
		{
			name: "array type",
			param: map[string]interface{}{
				"type": "array",
			},
			expected: `["item1", "item2"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getExampleValue(tt.param)
			if result != tt.expected {
				t.Errorf("getExampleValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test displayNodeList with different formats
func TestDisplayNodeList_Formats(t *testing.T) {
	nodes := []client.NodeInfo{
		{
			Name:        "exec/shell",
			Version:     "v1",
			Category:    "exec",
			Description: "Execute shell commands",
		},
	}

	tests := []struct {
		format  string
		noGroup bool
	}{
		{"text", false},
		{"text", true},
		{"json", false},
		{"yaml", false},
		{"simple", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			err := displayNodeList(nodes, tt.format, tt.noGroup)
			if err != nil {
				t.Errorf("displayNodeList() format=%s failed: %v", tt.format, err)
			}
		})
	}
}

// Test displayNodeList with empty nodes
func TestDisplayNodeList_Empty(t *testing.T) {
	nodes := []client.NodeInfo{}

	err := displayNodeList(nodes, "text", false)
	if err != nil {
		t.Errorf("displayNodeList() with empty nodes failed: %v", err)
	}
}

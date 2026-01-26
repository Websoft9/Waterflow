package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsYAMLFile tests YAML file extension detection
func TestIsYAMLFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"yaml extension", "workflow.yaml", true},
		{"yml extension", "workflow.yml", true},
		{"uppercase YAML", "workflow.YAML", true},
		{"uppercase YML", "workflow.YML", true},
		{"mixed case", "workflow.Yaml", true},
		{"json extension", "workflow.json", false},
		{"txt extension", "workflow.txt", false},
		{"no extension", "workflow", false},
		{"path with yaml", "/path/to/workflow.yaml", true},
		{"path with yml", "/path/to/workflow.yml", true},
		{"hidden yaml file", ".workflow.yaml", true},
		{"yaml in path but not extension", "/path.yaml/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isYAMLFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPrintValidationText tests text output formatting
func TestPrintValidationText(t *testing.T) {
	tests := []struct {
		name    string
		results *validator.ValidationResults
		verbose bool
		wantErr bool
	}{
		{
			name: "single valid file",
			results: &validator.ValidationResults{
				Total:  1,
				Passed: 1,
				Failed: 0,
				Files: []*validator.ValidationResult{
					{
						File:         "test.yaml",
						Valid:        true,
						WorkflowName: "test-workflow",
						JobsCount:    2,
						StepsCount:   5,
						Remote:       false,
					},
				},
			},
			verbose: false,
			wantErr: false,
		},
		{
			name: "single valid file verbose",
			results: &validator.ValidationResults{
				Total:  1,
				Passed: 1,
				Failed: 0,
				Files: []*validator.ValidationResult{
					{
						File:         "test.yaml",
						Valid:        true,
						WorkflowName: "test-workflow",
						JobsCount:    2,
						StepsCount:   5,
						Remote:       true,
					},
				},
			},
			verbose: true,
			wantErr: false,
		},
		{
			name: "single invalid file",
			results: &validator.ValidationResults{
				Total:  1,
				Passed: 0,
				Failed: 1,
				Files: []*validator.ValidationResult{
					{
						File:  "test.yaml",
						Valid: false,
						Errors: []validator.ValidationError{
							{
								Type:       "syntax",
								Message:    "invalid YAML syntax",
								Line:       10,
								Field:      "steps",
								Suggestion: "Check YAML formatting",
							},
						},
					},
				},
			},
			verbose: false,
			wantErr: false,
		},
		{
			name: "multiple files summary",
			results: &validator.ValidationResults{
				Total:  3,
				Passed: 2,
				Failed: 1,
				Files: []*validator.ValidationResult{
					{File: "a.yaml", Valid: true, WorkflowName: "a"},
					{File: "b.yaml", Valid: true, WorkflowName: "b"},
					{File: "c.yaml", Valid: false, Errors: []validator.ValidationError{{Message: "error"}}},
				},
			},
			verbose: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := printValidationText(tt.results, tt.verbose)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPrintValidationErrors tests error display formatting
func TestPrintValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		result *validator.ValidationResult
	}{
		{
			name: "single error with line",
			result: &validator.ValidationResult{
				File:  "test.yaml",
				Valid: false,
				Errors: []validator.ValidationError{
					{
						Type:       "syntax",
						Message:    "unexpected token",
						Line:       5,
						Suggestion: "Check syntax",
					},
				},
			},
		},
		{
			name: "single error without line",
			result: &validator.ValidationResult{
				File:  "test.yaml",
				Valid: false,
				Errors: []validator.ValidationError{
					{
						Type:    "schema",
						Message: "missing required field",
					},
				},
			},
		},
		{
			name: "multiple errors",
			result: &validator.ValidationResult{
				File:  "test.yaml",
				Valid: false,
				Errors: []validator.ValidationError{
					{
						Type:    "syntax",
						Message: "error 1",
						Line:    1,
					},
					{
						Type:       "schema",
						Message:    "error 2",
						Line:       10,
						Field:      "name",
						Suggestion: "Fix the field",
					},
					{
						Type:    "logic",
						Message: "error 3",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			printValidationErrors(tt.result)
		})
	}
}

// TestPrintValidationJSON tests JSON output formatting
func TestPrintValidationJSON(t *testing.T) {
	tests := []struct {
		name    string
		results *validator.ValidationResults
		wantErr bool
	}{
		{
			name: "single file",
			results: &validator.ValidationResults{
				Total:  1,
				Passed: 1,
				Failed: 0,
				Files: []*validator.ValidationResult{
					{
						File:         "test.yaml",
						Valid:        true,
						WorkflowName: "test",
						JobsCount:    1,
						StepsCount:   2,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple files",
			results: &validator.ValidationResults{
				Total:  2,
				Passed: 1,
				Failed: 1,
				Files: []*validator.ValidationResult{
					{File: "a.yaml", Valid: true, WorkflowName: "a"},
					{File: "b.yaml", Valid: false, Errors: []validator.ValidationError{{Message: "err"}}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := printValidationJSON(tt.results)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPrintValidationYAML tests YAML output formatting
func TestPrintValidationYAML(t *testing.T) {
	tests := []struct {
		name    string
		results *validator.ValidationResults
		wantErr bool
	}{
		{
			name: "single file",
			results: &validator.ValidationResults{
				Total:  1,
				Passed: 1,
				Failed: 0,
				Files: []*validator.ValidationResult{
					{
						File:         "test.yaml",
						Valid:        true,
						WorkflowName: "test",
						JobsCount:    1,
						StepsCount:   2,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple files",
			results: &validator.ValidationResults{
				Total:  2,
				Passed: 1,
				Failed: 1,
				Files: []*validator.ValidationResult{
					{File: "a.yaml", Valid: true, WorkflowName: "a"},
					{File: "b.yaml", Valid: false, Errors: []validator.ValidationError{{Message: "err"}}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := printValidationYAML(tt.results)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPrintValidationResults tests the result dispatcher
func TestPrintValidationResults(t *testing.T) {
	results := &validator.ValidationResults{
		Total:  1,
		Passed: 1,
		Failed: 0,
		Files: []*validator.ValidationResult{
			{
				File:         "test.yaml",
				Valid:        true,
				WorkflowName: "test",
			},
		},
	}

	tests := []struct {
		name    string
		format  string
		verbose bool
		wantErr bool
	}{
		{"text format", "text", false, false},
		{"text verbose", "text", true, false},
		{"json format", "json", false, false},
		{"yaml format", "yaml", false, false},
		{"default format", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := printValidationResults(results, tt.format, tt.verbose)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestNewValidateCmd tests command creation
func TestNewValidateCmd(t *testing.T) {
	cmd := newValidateCmd()
	require.NotNil(t, cmd)

	assert.Equal(t, "validate <workflow-file>...", cmd.Use)
	assert.Contains(t, cmd.Short, "Validate")

	// Check flags exist
	assert.NotNil(t, cmd.Flags().Lookup("remote"))
	assert.NotNil(t, cmd.Flags().Lookup("verbose"))
	assert.NotNil(t, cmd.Flags().Lookup("format"))
	assert.NotNil(t, cmd.Flags().Lookup("recursive"))
}

// TestCaptureOutput helper to capture stdout for testing print functions
//
//nolint:unused // Reserved for future print function tests
func captureOutput(f func()) string {
	var buf bytes.Buffer
	// Note: In real tests, we would need to redirect os.Stdout
	// This is a simplified version
	f()
	return buf.String()
}

// TestCollectValidationFiles tests file collection with real files
func TestCollectValidationFiles(t *testing.T) {
	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "validate_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test YAML files
	yamlFile1 := filepath.Join(tmpDir, "test1.yaml")
	yamlFile2 := filepath.Join(tmpDir, "test2.yml")
	txtFile := filepath.Join(tmpDir, "test.txt")

	require.NoError(t, os.WriteFile(yamlFile1, []byte("name: test1"), 0o600)) //nolint:gosec // Test file
	require.NoError(t, os.WriteFile(yamlFile2, []byte("name: test2"), 0o600)) //nolint:gosec // Test file
	require.NoError(t, os.WriteFile(txtFile, []byte("not yaml"), 0o600))      //nolint:gosec // Test file

	tests := []struct {
		name      string
		args      []string
		recursive bool
		wantCount int
		wantErr   bool
	}{
		{
			name:      "single file",
			args:      []string{yamlFile1},
			recursive: false,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "multiple files",
			args:      []string{yamlFile1, yamlFile2},
			recursive: false,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "directory recursive",
			args:      []string{tmpDir},
			recursive: true,
			wantCount: 2, // Only YAML files
			wantErr:   false,
		},
		{
			name:      "non-existent file",
			args:      []string{"/nonexistent/file.yaml"},
			recursive: false,
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := collectValidationFiles(tt.args, tt.recursive)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, files, tt.wantCount)
			}
		})
	}
}

// TestScanDirectory tests directory scanning
func TestScanDirectory(t *testing.T) {
	// Create temp directory with nested structure
	tmpDir, err := os.MkdirTemp("", "scan_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create nested directories
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o750)) //nolint:gosec // Test directory

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte("name: root"), 0o600))     //nolint:gosec // Test file
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "nested.yaml"), []byte("name: nested"), 0o600)) //nolint:gosec // Test file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "ignored.txt"), []byte("not yaml"), 0o600))     //nolint:gosec // Test file

	files, err := scanDirectory(tmpDir)
	require.NoError(t, err)

	// Should find 2 YAML files
	assert.Len(t, files, 2)

	// All files should be YAML
	for _, f := range files {
		assert.True(t, isYAMLFile(f))
	}
}

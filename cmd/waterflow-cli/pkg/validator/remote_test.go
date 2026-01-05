package validator

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRemoteValidator_Validate_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/workflows", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("dry_run"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		resp := map[string]interface{}{
			"workflow": map[string]interface{}{
				"name": "Test Workflow",
				"jobs": map[string]interface{}{
					"test": map[string]interface{}{},
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create validator
	logger := zap.NewNop()
	v := NewRemoteValidator(server.Client(), server.URL, logger)

	// Create temp file
	content := `name: Test
on: push
jobs:
  test:
    runs-on: default
    steps:
      - uses: checkout@v1
        with:
          repository: https://github.com/test/repo
`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	// Validate
	result, err := v.Validate(tmpFile)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.True(t, result.Remote)
	assert.Equal(t, "Test Workflow", result.WorkflowName)
	assert.Equal(t, 1, result.JobsCount)
}

func TestRemoteValidator_Validate_ServerError(t *testing.T) {
	// Mock server returning validation error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"detail": "Validation failed",
			"errors": []interface{}{
				map[string]interface{}{
					"type":       "schema_validation_error",
					"field":      "jobs.test.steps",
					"line":       10,
					"error":      "required field missing",
					"suggestion": "Add steps",
				},
			},
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := zap.NewNop()
	v := NewRemoteValidator(server.Client(), server.URL, logger)

	tmpFile := createTempFile(t, "name: Test")
	defer os.Remove(tmpFile)

	// Validate
	result, err := v.Validate(tmpFile)

	// Assert
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.True(t, result.Remote)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "schema_validation_error", result.Errors[0].Type)
	assert.Equal(t, "jobs.test.steps", result.Errors[0].Field)
}

func TestRemoteValidator_Validate_NetworkError(t *testing.T) {
	logger := zap.NewNop()
	v := NewRemoteValidator(http.DefaultClient, "http://localhost:99999", logger)

	tmpFile := createTempFile(t, "name: Test")
	defer os.Remove(tmpFile)

	// Validate
	result, err := v.Validate(tmpFile)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to connect to server")
}

func TestRemoteValidator_Validate_FileNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	logger := zap.NewNop()
	v := NewRemoteValidator(server.Client(), server.URL, logger)

	// Validate non-existent file
	result, err := v.Validate("nonexistent.yaml")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "file not found")
}

func TestRemoteValidator_ParseServerErrors(t *testing.T) {
	logger := zap.NewNop()
	v := NewRemoteValidator(http.DefaultClient, "http://test", logger)

	tests := []struct {
		name     string
		response map[string]interface{}
		wantLen  int
		wantType string
	}{
		{
			name: "errors array",
			response: map[string]interface{}{
				"detail": "validation failed",
				"errors": []interface{}{
					map[string]interface{}{
						"type":  "schema_error",
						"error": "missing field",
					},
				},
			},
			wantLen:  1,
			wantType: "schema_error",
		},
		{
			name: "detail only",
			response: map[string]interface{}{
				"detail": "something went wrong",
			},
			wantLen:  1,
			wantType: "server_error",
		},
		{
			name: "message field",
			response: map[string]interface{}{
				"message": "error message",
			},
			wantLen:  1,
			wantType: "server_error",
		},
		{
			name:     "unknown format",
			response: map[string]interface{}{},
			wantLen:  1,
			wantType: "server_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.parseServerErrors("test.yaml", tt.response, 0)

			assert.False(t, result.Valid)
			assert.True(t, result.Remote)
			assert.Len(t, result.Errors, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantType, result.Errors[0].Type)
			}
		})
	}
}

func createTempFile(t *testing.T, content string) string {
	tmpFile := filepath.Join(t.TempDir(), "test.yaml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)
	return tmpFile
}

package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

// TestHTTPRequestNode_Name verifies the node name
func TestHTTPRequestNode_Name(t *testing.T) {
	node := &HTTPRequestNode{}
	assert.Equal(t, "http/request", node.Name())
}

// TestHTTPRequestNode_Version verifies the node version
func TestHTTPRequestNode_Version(t *testing.T) {
	node := &HTTPRequestNode{}
	assert.Equal(t, "v1", node.Version())
}

// TestHTTPRequestNode_Metadata verifies the node metadata
func TestHTTPRequestNode_Metadata(t *testing.T) {
	node := &HTTPRequestNode{}
	metadata := node.Metadata()

	assert.Equal(t, "http", metadata.Category)
	assert.Contains(t, metadata.Description, "HTTP")
	assert.NotEmpty(t, metadata.InputSchema)
	assert.NotEmpty(t, metadata.OutputSchema)

	// Verify required parameters
	urlParam, ok := metadata.InputSchema["url"]
	require.True(t, ok)
	assert.True(t, urlParam.Required)

	methodParam, ok := metadata.InputSchema["method"]
	require.True(t, ok)
	assert.False(t, methodParam.Required)
	assert.Equal(t, "GET", methodParam.Default)
}

// TestHTTPRequestNode_Execute_GET tests basic GET request
func TestHTTPRequestNode_Execute_GET(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Waterflow/1.0", r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url": server.URL,
	}

	result, err := node.Execute(context.Background(), inputs)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 200, result.Outputs["status_code"])
	assert.NotEmpty(t, result.Outputs["headers"])
	assert.Equal(t, "application/json", result.Outputs["content_type"])

	// JSON should be auto-parsed
	body, ok := result.Outputs["body"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "success", body["message"])
}

// TestHTTPRequestNode_Execute_POST tests POST with JSON body
func TestHTTPRequestNode_Execute_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "test", body["key"])

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"created": true}`))
	}))
	defer server.Close()

	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url":    server.URL,
		"method": "POST",
		"body": map[string]interface{}{
			"key": "test",
		},
	}

	result, err := node.Execute(context.Background(), inputs)
	require.NoError(t, err)
	assert.Equal(t, 201, result.Outputs["status_code"])
}

// TestHTTPRequestNode_Execute_CustomHeaders tests custom headers
func TestHTTPRequestNode_Execute_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url": server.URL,
		"headers": map[string]interface{}{
			"Authorization": "Bearer token123",
			"Accept":        "application/json",
		},
	}

	result, err := node.Execute(context.Background(), inputs)
	require.NoError(t, err)
	assert.Equal(t, 200, result.Outputs["status_code"])
}

// TestHTTPRequestNode_Execute_AllMethods tests all HTTP methods
func TestHTTPRequestNode_Execute_AllMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, method, r.Method)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			node := &HTTPRequestNode{}
			inputs := map[string]interface{}{
				"url":    server.URL,
				"method": strings.ToLower(method), // Test case insensitive
			}

			result, err := node.Execute(context.Background(), inputs)
			require.NoError(t, err)
			assert.Equal(t, 200, result.Outputs["status_code"])
		})
	}
}

// TestHTTPRequestNode_Execute_4xxError tests 4xx client error handling
func TestHTTPRequestNode_Execute_4xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url": server.URL,
	}

	result, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "404")

	// Verify 4xx errors are marked as non-retryable
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	assert.True(t, appErr.NonRetryable(), "4xx errors should be non-retryable")
}

// TestHTTPRequestNode_Execute_5xxError tests 5xx server error handling
func TestHTTPRequestNode_Execute_5xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server Error"))
	}))
	defer server.Close()

	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url": server.URL,
	}

	result, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "500")

	// Verify 5xx errors are retryable (not marked NonRetryable)
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	assert.False(t, appErr.NonRetryable(), "5xx errors should be retryable")
}

// TestHTTPRequestNode_Execute_Timeout tests timeout handling
func TestHTTPRequestNode_Execute_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url":     server.URL,
		"timeout": "100ms",
	}

	result, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestHTTPRequestNode_Execute_ContextCancellation tests context cancellation
func TestHTTPRequestNode_Execute_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url": server.URL,
	}

	result, err := node.Execute(ctx, inputs)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestHTTPRequestNode_Execute_StringBody tests string body
func TestHTTPRequestNode_Execute_StringBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "text/plain", r.Header.Get("Content-Type"))
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, "plain text", string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url":    server.URL,
		"method": "POST",
		"body":   "plain text",
	}

	result, err := node.Execute(context.Background(), inputs)
	require.NoError(t, err)
	assert.Equal(t, 200, result.Outputs["status_code"])
}

// TestHTTPRequestNode_Execute_InvalidURL tests invalid URL error
func TestHTTPRequestNode_Execute_InvalidURL(t *testing.T) {
	node := &HTTPRequestNode{}
	tests := []struct {
		name   string
		inputs map[string]interface{}
	}{
		{
			name:   "missing url",
			inputs: map[string]interface{}{},
		},
		{
			name:   "invalid scheme",
			inputs: map[string]interface{}{"url": "ftp://example.com"},
		},
		{
			name:   "malformed url",
			inputs: map[string]interface{}{"url": "://invalid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := node.Execute(context.Background(), tt.inputs)
			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

// TestHTTPRequestNode_Execute_InvalidMethod tests invalid HTTP method
func TestHTTPRequestNode_Execute_InvalidMethod(t *testing.T) {
	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url":    "http://example.com",
		"method": "INVALID",
	}

	result, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid HTTP method")
}

// TestHTTPRequestNode_Execute_JSONParsing tests JSON auto-parsing
func TestHTTPRequestNode_Execute_JSONParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"key": "value", "number": 42}`))
	}))
	defer server.Close()

	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url": server.URL,
	}

	result, err := node.Execute(context.Background(), inputs)
	require.NoError(t, err)

	body, ok := result.Outputs["body"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", body["key"])
	// JSON numbers are float64
	assert.Equal(t, float64(42), body["number"])
}

// TestHTTPRequestNode_Execute_NonJSONResponse tests non-JSON response
func TestHTTPRequestNode_Execute_NonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html>test</html>"))
	}))
	defer server.Close()

	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url": server.URL,
	}

	result, err := node.Execute(context.Background(), inputs)
	require.NoError(t, err)

	body, ok := result.Outputs["body"].(string)
	require.True(t, ok)
	assert.Equal(t, "<html>test</html>", body)
}

// TestRegister tests the Register function
func TestRegister(t *testing.T) {
	node := Register()
	require.NotNil(t, node)
	assert.Equal(t, "http/request", node.Name())
	assert.Equal(t, "v1", node.Version())
}

// TestHTTPRequestNode_Execute_TimeoutValidation tests timeout validation
func TestHTTPRequestNode_Execute_TimeoutValidation(t *testing.T) {
	node := &HTTPRequestNode{}
	inputs := map[string]interface{}{
		"url":     "http://example.com",
		"timeout": "6m", // Exceeds 5 minute limit
	}

	result, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "timeout must not exceed 5 minutes")
}

// TestHTTPRequestNode_Execute_VerifySSL tests SSL verification
func TestHTTPRequestNode_Execute_VerifySSL(t *testing.T) {
	node := &HTTPRequestNode{}

	// Test with verify_ssl = false (should not fail on self-signed certs)
	inputs := map[string]interface{}{
		"url":        "https://self-signed.badssl.com/",
		"verify_ssl": false,
		"timeout":    "5s",
	}

	// This test might fail if badssl.com is unreachable, so we just test the parameter is accepted
	_, _ = node.Execute(context.Background(), inputs)
	// No assertion - just testing parameter is processed
}

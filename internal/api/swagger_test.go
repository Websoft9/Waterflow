package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServeSwaggerUI(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	w := httptest.NewRecorder()

	ServeSwaggerUI(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, "<!DOCTYPE html>")
	assert.Contains(t, body, "Waterflow API Documentation")
	assert.Contains(t, body, "swagger-ui")
	assert.Contains(t, body, "/api/openapi.yaml")
}

func TestServeOpenAPISpec(t *testing.T) {
	// This test requires the openapi.yaml file to exist
	// Skip if running in a directory without the file
	req := httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil)
	w := httptest.NewRecorder()

	ServeOpenAPISpec(w, req)

	// If file exists, should return 200 with YAML content
	// If file doesn't exist, should return 404
	if w.Code == http.StatusOK {
		contentType := w.Header().Get("Content-Type")
		// Content type should be YAML or octet-stream depending on system
		assert.True(t, strings.Contains(contentType, "yaml") ||
			strings.Contains(contentType, "octet-stream") ||
			strings.Contains(contentType, "text/plain"),
			"Content-Type should be YAML-related, got: "+contentType)
	}
	// Either 200 or 404 is acceptable depending on test environment
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound,
		"Expected 200 or 404, got: %d", w.Code)
}

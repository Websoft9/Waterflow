package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Context().Value(RequestIDKey)
		assert.NotNil(t, requestID)

		// Should be a string
		id, ok := requestID.(string)
		assert.True(t, ok)
		assert.NotEmpty(t, id)

		// Should be valid UUID format
		assert.Len(t, id, 36)

		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestID(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Should set X-Request-ID response header
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestIDMiddlewareWithExistingHeader(t *testing.T) {
	existingID := "existing-request-id"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Context().Value(RequestIDKey)
		assert.Equal(t, existingID, requestID)
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestID(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", existingID)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Should preserve existing ID
	assert.Equal(t, existingID, w.Header().Get("X-Request-ID"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetRequestID(t *testing.T) {
	// Test with request ID in context
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-id-123")
	id := GetRequestID(ctx)
	assert.Equal(t, "test-id-123", id)

	// Test without request ID
	ctx = context.Background()
	id = GetRequestID(ctx)
	assert.Equal(t, "", id)
}

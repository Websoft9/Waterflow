package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	versionHandler := Version("1.0.0")(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	versionHandler.ServeHTTP(w, req)

	assert.Equal(t, "1.0.0", w.Header().Get("X-Server-Version"))
}

func TestVersionMiddlewareWithDev(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	versionHandler := Version("dev")(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	versionHandler.ServeHTTP(w, req)

	assert.Equal(t, "dev", w.Header().Get("X-Server-Version"))
}

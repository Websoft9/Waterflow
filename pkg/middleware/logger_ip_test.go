package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1")
	req.RemoteAddr = "127.0.0.1:12345"

	ip := extractClientIP(req)

	// Should extract first IP from X-Forwarded-For
	assert.Equal(t, "192.168.1.100", ip)
}

func TestExtractClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.200")
	req.RemoteAddr = "127.0.0.1:12345"

	ip := extractClientIP(req)

	// Should use X-Real-IP
	assert.Equal(t, "192.168.1.200", ip)
}

func TestExtractClientIP_XForwardedForPriority(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.Header.Set("X-Real-IP", "192.168.1.200")
	req.RemoteAddr = "127.0.0.1:12345"

	ip := extractClientIP(req)

	// X-Forwarded-For takes priority
	assert.Equal(t, "192.168.1.100", ip)
}

func TestExtractClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.300:54321"

	ip := extractClientIP(req)

	// Should extract IP from RemoteAddr (strip port)
	assert.Equal(t, "192.168.1.300", ip)
}

func TestExtractClientIP_RemoteAddrNoPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.400"

	ip := extractClientIP(req)

	// Should return as-is if no port
	assert.Equal(t, "192.168.1.400", ip)
}

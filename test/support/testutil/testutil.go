// Package testutil provides common test utilities for Waterflow tests.
package testutil

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestEnvironment holds common test dependencies and utilities.
type TestEnvironment struct {
	T       *testing.T
	Logger  *zap.Logger
	Context context.Context
	Cancel  context.CancelFunc
	Cleanup []func()
}

// NewTestEnvironment creates a new test environment with standard setup.
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	logger := zaptest.NewLogger(t)
	env := &TestEnvironment{
		T:       t,
		Logger:  logger,
		Context: ctx,
		Cancel:  cancel,
		Cleanup: make([]func(), 0),
	}
	t.Cleanup(func() {
		for i := len(env.Cleanup) - 1; i >= 0; i-- {
			env.Cleanup[i]()
		}
		cancel()
	})
	return env
}

// AddCleanup adds a cleanup function to be called at test end.
func (e *TestEnvironment) AddCleanup(fn func()) {
	e.Cleanup = append(e.Cleanup, fn)
}

// SkipIfShort skips the test if running in short mode.
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// SkipIfCI skips the test if running in CI environment.
func SkipIfCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("Skipping test in CI environment")
	}
}

// Note: RequireEnv is defined in require.go

// TempDir creates a temporary directory for the test.
func TempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "waterflow-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// TempFile creates a temporary file with the given content.
func TempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := TempDir(t)
	path := dir + "/" + name
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return path
}

// MockServer creates a test HTTP server with the given handler.
func MockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// AssertEventually retries assertion until it passes or times out.
func AssertEventually(t *testing.T, condition func() bool, timeout, interval time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("Condition not met within %v: %s", timeout, msg)
}

// MustContain asserts that the string contains all substrings.
func MustContain(t *testing.T, s string, substrings ...string) {
	t.Helper()
	for _, sub := range substrings {
		if !strings.Contains(s, sub) {
			t.Errorf("Expected string to contain %q", sub)
		}
	}
}

// GenerateUniqueID generates a unique ID for test resources.
func GenerateUniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

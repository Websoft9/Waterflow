// Package testutil provides testing utilities for Waterflow tests
// This package helps eliminate hard waits (time.Sleep) in favor of condition-based waiting
package testutil

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// DefaultTimeout is the default timeout for waiting operations
const DefaultTimeout = 10 * time.Second

// DefaultPollInterval is the default polling interval for condition checks
const DefaultPollInterval = 50 * time.Millisecond

// WaitConfig configures wait behavior
type WaitConfig struct {
	Timeout      time.Duration
	PollInterval time.Duration
}

// DefaultWaitConfig returns the default wait configuration
func DefaultWaitConfig() WaitConfig {
	return WaitConfig{
		Timeout:      DefaultTimeout,
		PollInterval: DefaultPollInterval,
	}
}

// Eventually waits for a condition to become true within the timeout
// This is the primary replacement for time.Sleep in tests
//
// Usage:
//
//	err := testutil.Eventually(t, func() bool {
//	   return mockHandler.GetCallCount() > 0
//	}, "handler should be called")
func Eventually(t *testing.T, condition func() bool, msgAndArgs ...interface{}) error {
	t.Helper()
	return EventuallyWithConfig(t, condition, DefaultWaitConfig(), msgAndArgs...)
}

// EventuallyWithTimeout waits for a condition with a custom timeout
func EventuallyWithTimeout(t *testing.T, condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) error {
	t.Helper()
	return EventuallyWithConfig(t, condition, WaitConfig{
		Timeout:      timeout,
		PollInterval: DefaultPollInterval,
	}, msgAndArgs...)
}

// EventuallyWithConfig waits for a condition with full configuration
func EventuallyWithConfig(t *testing.T, condition func() bool, config WaitConfig, msgAndArgs ...interface{}) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	for {
		if condition() {
			return nil
		}

		select {
		case <-ctx.Done():
			if len(msgAndArgs) > 0 {
				t.Fatalf("condition not met within %v: %v", config.Timeout, msgAndArgs[0])
			} else {
				t.Fatalf("condition not met within %v", config.Timeout)
			}
			return ctx.Err()
		case <-ticker.C:
			// Continue polling
		}
	}
}

// RequireEventually is like Eventually but fails the test immediately if condition is not met
func RequireEventually(t *testing.T, condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	err := EventuallyWithTimeout(t, condition, timeout, msgAndArgs...)
	if err != nil {
		t.FailNow()
	}
}

// Never waits and verifies that a condition never becomes true within the timeout
// Useful for testing that certain things should NOT happen
func Never(t *testing.T, condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(DefaultPollInterval)
	defer ticker.Stop()

	for {
		if condition() {
			if len(msgAndArgs) > 0 {
				t.Fatalf("condition became true unexpectedly: %v", msgAndArgs[0])
			} else {
				t.Fatal("condition became true unexpectedly")
			}
			return
		}

		select {
		case <-ctx.Done():
			// Good - condition never became true
			return
		case <-ticker.C:
			// Continue checking
		}
	}
}

// RetryUntilSuccess retries a function until it succeeds or timeout is reached
func RetryUntilSuccess(t *testing.T, fn func() error, timeout time.Duration) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(DefaultPollInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			t.Fatalf("operation did not succeed within %v, last error: %v", timeout, lastErr)
			return lastErr
		case <-ticker.C:
			// Continue retrying
		}
	}
}

// WaitForHTTPReady waits for an HTTP endpoint to become ready
func WaitForHTTPReady(t *testing.T, url string, timeout time.Duration) {
	t.Helper()

	client := &http.Client{Timeout: 2 * time.Second}

	_ = RetryUntilSuccess(t, func() error {
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 {
			return fmt.Errorf("server returned %d", resp.StatusCode)
		}
		return nil
	}, timeout)
}

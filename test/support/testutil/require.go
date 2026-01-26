// Package testutil provides test utilities for the Waterflow project.
// This file contains helper functions for consistent test skipping conditions.
package testutil

import (
	"net"
	"os"
	"testing"
	"time"

	"go.temporal.io/sdk/client"
)

// RequireTemporal returns a Temporal client or skips the test if unavailable.
// It also registers a cleanup function to close the client after the test.
//
// Usage:
//
//	func TestWorkflow(t *testing.T) {
//	    tc := testutil.RequireTemporal(t)
//	    // use tc...
//	}
func RequireTemporal(t *testing.T) client.Client {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	hostPort := getTemporalHostPort()

	tc, err := client.Dial(client.Options{
		HostPort:  hostPort,
		Namespace: "default",
	})
	if err != nil {
		t.Skipf("Temporal not available at %s: %v", hostPort, err)
	}

	t.Cleanup(func() { tc.Close() })
	return tc
}

// RequireDocker skips the test if Docker is not available.
//
// Usage:
//
//	func TestDockerCompose(t *testing.T) {
//	    testutil.RequireDocker(t)
//	    // test Docker functionality...
//	}
func RequireDocker(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	// Check if Docker socket is accessible
	socketPath := "/var/run/docker.sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Skipf("Docker socket not found at %s", socketPath)
	}
}

// RequireIntegration skips the test unless INTEGRATION_TEST=true is set.
//
// Usage:
//
//	func TestExternalService(t *testing.T) {
//	    testutil.RequireIntegration(t)
//	    // test external service...
//	}
func RequireIntegration(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test - set INTEGRATION_TEST=true to run")
	}
}

// RequireNetwork skips the test if the specified host:port is not reachable.
//
// Usage:
//
//	func TestExternalAPI(t *testing.T) {
//	    testutil.RequireNetwork(t, "api.example.com:443")
//	    // test API...
//	}
func RequireNetwork(t *testing.T, hostPort string) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	conn, err := net.DialTimeout("tcp", hostPort, 2*time.Second)
	if err != nil {
		t.Skipf("Network endpoint %s not reachable: %v", hostPort, err)
	}
	_ = conn.Close()
}

// RequireEnv skips the test if the specified environment variable is not set.
//
// Usage:
//
//	func TestWithAPIKey(t *testing.T) {
//	    apiKey := testutil.RequireEnv(t, "API_KEY")
//	    // use apiKey...
//	}
func RequireEnv(t *testing.T, key string) string {
	t.Helper()

	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Environment variable %s not set", key)
	}
	return value
}

// RequireLongRunning skips the test unless LONG_RUNNING_TEST=true is set.
// Use this for tests that take more than 1 minute.
//
// Usage:
//
//	func TestStressScenario(t *testing.T) {
//	    testutil.RequireLongRunning(t)
//	    // stress test...
//	}
func RequireLongRunning(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	if os.Getenv("LONG_RUNNING_TEST") != "true" {
		t.Skip("Skipping long-running test - set LONG_RUNNING_TEST=true to run")
	}
}

// getTemporalHostPort returns the Temporal server address from environment
// or defaults to localhost:7233.
func getTemporalHostPort() string {
	if hostPort := os.Getenv("TEMPORAL_HOST_PORT"); hostPort != "" {
		return hostPort
	}
	return "localhost:7233"
}

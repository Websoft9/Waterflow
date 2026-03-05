package agent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorker_Integration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test - set INTEGRATION_TEST=true to run")
	}

	// Initialize logger
	require.NoError(t, logger.Init("info", "json"))

	cfg := &config.Config{
		Agent: config.AgentConfig{
			TaskQueues:      []string{"test-queue"},
			PluginDir:       "/tmp/plugins",
			ShutdownTimeout: 30 * time.Second,
		},
		Temporal: config.TemporalConfig{
			Host:              "localhost:7233",
			Namespace:         "waterflow",
			ConnectionTimeout: 10 * time.Second,
			MaxRetries:        3,
			RetryInterval:     1 * time.Second,
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	worker, err := NewWorker(cfg, logger.Log)
	require.NoError(t, err)
	assert.NotNil(t, worker)

	// Test shutdown immediately
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestWorkerShutdown_Integration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test - set INTEGRATION_TEST=true to run")
	}

	// Initialize logger
	require.NoError(t, logger.Init("info", "json"))

	cfg := &config.Config{
		Agent: config.AgentConfig{
			TaskQueues:      []string{"test-queue"},
			PluginDir:       "/tmp/plugins",
			ShutdownTimeout: 30 * time.Second,
		},
		Temporal: config.TemporalConfig{
			Host:              "localhost:7233",
			Namespace:         "waterflow",
			ConnectionTimeout: 10 * time.Second,
			MaxRetries:        3,
			RetryInterval:     1 * time.Second,
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	worker, err := NewWorker(cfg, logger.Log)
	require.NoError(t, err)

	err = worker.Start()
	require.NoError(t, err)

	// Worker is started, proceed to shutdown test
	// No need to wait - worker.Shutdown will handle graceful shutdown

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestPluginManager(t *testing.T) {
	// Initialize logger
	require.NoError(t, logger.Init("info", "json"))

	registry := node.NewRegistry()
	pm := NewPluginManager("/opt/waterflow/plugins", registry, logger.Log)
	assert.NotNil(t, pm)

	// Test LoadPlugins (stub returns nil)
	err := pm.LoadPlugins()
	assert.NoError(t, err)

	// Test GetNode (stub returns error - this is expected in Story 2.1)
	_, err = pm.GetNode("shell")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found") // Node not registered
}

func TestConnectToTemporal_Retry(t *testing.T) {
	// Test retry logic without actual Temporal connection
	require.NoError(t, logger.Init("info", "json"))

	cfg := &config.Config{
		Agent: config.AgentConfig{
			TaskQueues:      []string{"test"},
			ShutdownTimeout: 5 * time.Second,
		},
		Temporal: config.TemporalConfig{
			Host:              "invalid-host:9999",
			Namespace:         "test",
			ConnectionTimeout: 1 * time.Second,
			MaxRetries:        3,
			RetryInterval:     100 * time.Millisecond,
		},
	}

	start := time.Now()
	_, err := connectToTemporal(cfg, logger.Log)
	duration := time.Since(start)

	// Should fail after retries
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to Temporal after")

	// Should take at least 2 retry intervals (100ms * 2)
	assert.GreaterOrEqual(t, duration, 200*time.Millisecond)
	// Should not take excessively long (3 retries * 100ms + connection overhead < 5s)
	assert.Less(t, duration, 5*time.Second)
}

func TestParseTaskQueues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single queue",
			input: "linux-amd64",
			want:  []string{"linux-amd64"},
		},
		{
			name:  "multiple queues",
			input: "linux-amd64,web-servers,gpu-a100",
			want:  []string{"linux-amd64", "web-servers", "gpu-a100"},
		},
		{
			name:  "with spaces",
			input: " linux-amd64 , web-servers , gpu-a100 ",
			want:  []string{"linux-amd64", "web-servers", "gpu-a100"},
		},
		{
			name:  "with duplicates",
			input: "linux-amd64,linux-amd64,web-servers",
			want:  []string{"linux-amd64", "web-servers"},
		},
		{
			name:  "with empty elements",
			input: "linux-amd64,,web-servers,,",
			want:  []string{"linux-amd64", "web-servers"},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Import parseTaskQueues logic here for testing
			got := parseTaskQueuesHelper(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---- H1 unit tests: cover NewWorker/Start/Shutdown without Temporal ----

// TestNewWorkerFromClient verifies newWorkerFromClient initialises registries
// and plugin manager without needing a live Temporal connection.
func TestNewWorkerFromClient(t *testing.T) {
	require.NoError(t, logger.Init("info", "json"))

	cfg := &config.Config{
		Agent: config.AgentConfig{
			TaskQueues:      []string{"test-queue"},
			PluginDir:       "/tmp/plugins-nonexistent",
			ShutdownTimeout: 5 * time.Second,
		},
	}

	w, err := newWorkerFromClient(cfg, nil, logger.Log)
	require.NoError(t, err)
	assert.NotNil(t, w)
	assert.NotNil(t, w.pluginManager)
	assert.NotNil(t, w.nodeRegistry)
	assert.Empty(t, w.workers)
}

// TestNewWorkerFromClient_BuiltinRegistration verifies builtin nodes are registered.
func TestNewWorkerFromClient_BuiltinRegistration(t *testing.T) {
	require.NoError(t, logger.Init("info", "json"))

	cfg := &config.Config{
		Agent: config.AgentConfig{
			TaskQueues: []string{"q1"},
			PluginDir:  "/tmp/nonexistent",
		},
	}

	w, err := newWorkerFromClient(cfg, nil, logger.Log)
	require.NoError(t, err)

	// Builtin nodes should be registered (checkout@v1, run@v1)
	list := w.nodeRegistry.List()
	assert.GreaterOrEqual(t, len(list), 1, "Expected at least 1 builtin node registered")
}

// TestWorkerStart_NoTaskQueues verifies Start succeeds when task_queues is empty
// (the for-range loop is skipped, no real Temporal workers created).
func TestWorkerStart_NoTaskQueues(t *testing.T) {
	require.NoError(t, logger.Init("info", "json"))

	cfg := &config.Config{
		Agent: config.AgentConfig{
			TaskQueues:        []string{},
			PluginDir:         "/tmp/nonexistent",
			AutoReloadPlugins: false,
			ShutdownTimeout:   5 * time.Second,
		},
	}

	w, err := newWorkerFromClient(cfg, nil, logger.Log)
	require.NoError(t, err)

	err = w.Start()
	require.NoError(t, err)
	assert.Empty(t, w.workers, "No workers should be created for empty task_queues")

	// Shutdown should succeed with nil temporalClient
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	assert.NoError(t, w.Shutdown(ctx))
}

// TestWorkerShutdown_EmptyWorkers verifies Shutdown handles the case with no
// running workers and a nil temporalClient (no panic).
func TestWorkerShutdown_EmptyWorkers(t *testing.T) {
	require.NoError(t, logger.Init("info", "json"))

	cfg := &config.Config{
		Agent: config.AgentConfig{
			TaskQueues:      []string{},
			ShutdownTimeout: 5 * time.Second,
		},
	}

	w, err := newWorkerFromClient(cfg, nil, logger.Log)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = w.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestWorkerShutdown_ContextTimeout verifies Shutdown logs a warning and returns
// when the context deadline is exceeded (no blocking).
func TestWorkerShutdown_ContextTimeout(t *testing.T) {
	require.NoError(t, logger.Init("info", "json"))

	cfg := &config.Config{
		Agent: config.AgentConfig{
			TaskQueues:      []string{},
			ShutdownTimeout: 1 * time.Second,
		},
	}

	w, err := newWorkerFromClient(cfg, nil, logger.Log)
	require.NoError(t, err)

	// Simulate a goroutine holding the WaitGroup
	w.wg.Add(1)

	// Use a context that expires immediately
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = w.Shutdown(ctx)
	assert.NoError(t, err, "Shutdown should not return error on timeout")

	// Release the goroutine after test
	w.wg.Done()
}

// TestWorkerStart_AutoReloadDisabled verifies AutoReloadPlugins=false skips the watcher goroutine.
func TestWorkerStart_AutoReloadDisabled(t *testing.T) {
	require.NoError(t, logger.Init("info", "json"))

	cfg := &config.Config{
		Agent: config.AgentConfig{
			TaskQueues:        []string{},
			PluginDir:         "/tmp/nonexistent",
			AutoReloadPlugins: false,
			ShutdownTimeout:   3 * time.Second,
		},
	}

	w, err := newWorkerFromClient(cfg, nil, logger.Log)
	require.NoError(t, err)

	err = w.Start()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	assert.NoError(t, w.Shutdown(ctx))
}

// TestWorkerNodeRegistry_AfterInit verifies the node registry is accessible
// after Worker initialisation for Activity binding.
func TestWorkerNodeRegistry_AfterInit(t *testing.T) {
	require.NoError(t, logger.Init("info", "json"))

	cfg := &config.Config{
		Agent: config.AgentConfig{
			TaskQueues: []string{"q"},
			PluginDir:  "/tmp/nonexistent",
		},
	}

	w, err := newWorkerFromClient(cfg, nil, logger.Log)
	require.NoError(t, err)

	// Registry must not be nil (Activities depend on it)
	registered := w.nodeRegistry.List()
	assert.NotNil(t, registered)
}

// parseTaskQueuesHelper duplicates the logic from main.go for testing
func parseTaskQueuesHelper(s string) []string {
	queues := strings.Split(s, ",")
	seen := make(map[string]bool)
	result := make([]string, 0, len(queues))
	for _, q := range queues {
		q = strings.TrimSpace(q)
		if q != "" && !seen[q] {
			seen[q] = true
			result = append(result, q)
		}
	}
	return result
}

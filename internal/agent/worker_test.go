package agent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
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

	// Give worker time to start
	time.Sleep(2 * time.Second)

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestPluginManager(t *testing.T) {
	// Initialize logger
	require.NoError(t, logger.Init("info", "json"))

	pm := NewPluginManager("/opt/waterflow/plugins", logger.Log)
	assert.NotNil(t, pm)

	// Test LoadPlugins (stub returns nil)
	err := pm.LoadPlugins()
	assert.NoError(t, err)

	// Test GetNode (stub returns error - this is expected in Story 2.1)
	_, err = pm.GetNode("shell")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Epic 4")
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

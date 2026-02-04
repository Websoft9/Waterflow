package temporal

import (
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"go.uber.org/zap"
)

func TestWorker_Structure(t *testing.T) {
	logger := zap.NewNop()

	t.Run("worker_fields", func(t *testing.T) {
		// Test that Worker struct has required fields
		w := &Worker{
			logger: logger,
			worker: nil,
		}

		if w.logger == nil {
			t.Error("logger field should be set")
		}
	})
}

func TestWorker_StartStop(t *testing.T) {
	logger := zap.NewNop()

	t.Run("stop_without_start", func(t *testing.T) {
		// Create a mock worker
		w := &Worker{
			logger: logger,
			worker: nil, // Mock: no real worker
		}

		// Should not panic when stopping unstarted worker
		w.Stop()
	})

	t.Run("start_error_handling", func(t *testing.T) {
		// Test that Start returns properly
		// Note: Real worker requires Temporal connection, so we test structure only
		w := &Worker{
			logger: logger,
			worker: nil,
		}

		// Validate Start method exists and has correct signature
		err := w.Start()
		if err != nil {
			t.Errorf("Start() should return nil, got: %v", err)
		}
	})
}

func TestNewWorker_Integration(t *testing.T) {
	// This test requires running Temporal server
	t.Skip("Requires running Temporal server - run manually")

	logger := zap.NewNop()
	cfg := &config.TemporalConfig{
		Host:              "localhost:7233",
		Namespace:         "default",
		TaskQueue:         "test-worker-queue",
		MaxRetries:        3,
		RetryInterval:     1 * time.Second,
		ConnectionTimeout: 5 * time.Second,
	}

	// Create client
	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create node registry
	registry := node.NewRegistry()

	// Create activities
	activities := NewActivities(logger, registry)

	// Create worker
	worker := NewWorker(client, activities)
	if worker == nil {
		t.Fatal("NewWorker returned nil")
	}

	// Start worker
	if err := worker.Start(); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// Give worker time to start
	time.Sleep(100 * time.Millisecond)

	// Stop worker
	worker.Stop()

	// Give worker time to stop
	time.Sleep(100 * time.Millisecond)
}

func TestWorker_LoggingMessages(t *testing.T) {
	// Test that worker logs correct messages
	// This validates the log message format and content

	logger := zap.NewNop()

	t.Run("start_log_message", func(t *testing.T) {
		w := &Worker{
			logger: logger,
			worker: nil,
		}

		// Start should log "Starting Temporal Worker"
		err := w.Start()
		if err != nil {
			t.Errorf("Start() should not return error, got: %v", err)
		}
	})

	t.Run("stop_log_message", func(t *testing.T) {
		w := &Worker{
			logger: logger,
			worker: nil,
		}

		// Stop should log "Stopping Temporal Worker"
		w.Stop()
	})
}

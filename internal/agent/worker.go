package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/google/uuid"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
)

// Worker represents an Agent Worker instance.
type Worker struct {
	config            *config.Config
	logger            *zap.Logger
	temporalClient    *temporal.Client
	workers           []worker.Worker // One worker per task queue
	pluginManager     *PluginManager
	wg                sync.WaitGroup // Wait for worker goroutines
	agentID           string         // Agent ID for heartbeat
	stopCh            chan struct{}  // Stop channel for heartbeat
	heartbeatFailures int            // Consecutive heartbeat failures
	heartbeatMu       sync.Mutex     // Protects heartbeatFailures
}

// NewWorker creates a new Agent Worker and connects to Temporal.
func NewWorker(cfg *config.Config, logger *zap.Logger) (*Worker, error) {
	// Connect to Temporal
	temporalClient, err := connectToTemporal(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Temporal: %w", err)
	}

	// Initialize Plugin Manager (Epic 4 - stub for now)
	pluginManager := NewPluginManager(cfg.Agent.PluginDir, logger)

	w := &Worker{
		config:         cfg,
		logger:         logger,
		temporalClient: temporalClient,
		workers:        make([]worker.Worker, 0, len(cfg.Agent.TaskQueues)),
		pluginManager:  pluginManager,
		stopCh:         make(chan struct{}),
	}

	return w, nil
}

// connectToTemporal creates a Temporal client connection with retries.
func connectToTemporal(cfg *config.Config, logger *zap.Logger) (*temporal.Client, error) {
	for attempt := 1; attempt <= cfg.Temporal.MaxRetries; attempt++ {
		temporalClient, err := temporal.NewClient(&cfg.Temporal, logger)
		if err == nil {
			logger.Info("Connected to Temporal",
				zap.String("host", cfg.Temporal.Host),
				zap.String("namespace", cfg.Temporal.Namespace),
			)
			return temporalClient, nil
		}

		// Use Error for first 5 attempts, then Warn
		if attempt <= 5 {
			logger.Error("Failed to connect to Temporal, retrying",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", cfg.Temporal.MaxRetries),
				zap.Error(err),
			)
		} else {
			logger.Warn("Failed to connect to Temporal, retrying",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", cfg.Temporal.MaxRetries),
				zap.Error(err),
			)
		}

		if attempt < cfg.Temporal.MaxRetries {
			time.Sleep(cfg.Temporal.RetryInterval)
		}
	}

	return nil, fmt.Errorf("failed to connect to Temporal after %d attempts", cfg.Temporal.MaxRetries)
}

// Start starts the Agent Worker and begins polling task queues.
func (w *Worker) Start() error {
	// Load plugins (Epic 4 - stub for now)
	if err := w.pluginManager.LoadPlugins(); err != nil {
		w.logger.Warn("Failed to load plugins", zap.Error(err))
		// Don't fail startup - plugins are optional in Story 2.1
	}

	// Create and start a worker for each task queue
	for _, taskQueue := range w.config.Agent.TaskQueues {
		workerInstance := worker.New(w.temporalClient.GetClient(), taskQueue, worker.Options{
			MaxConcurrentActivityExecutionSize:     100,
			MaxConcurrentWorkflowTaskExecutionSize: 50,
			// Use configured shutdown timeout for worker stop
			WorkerStopTimeout: w.config.Agent.ShutdownTimeout,
		})

		// Register workflows (Workflow executor from Server)
		workerInstance.RegisterWorkflow(temporal.RunWorkflowExecutor)

		// Register activities (Step executor)
		activities := temporal.NewActivities(w.logger)
		workerInstance.RegisterActivity(activities.ExecuteStepActivity)

		w.workers = append(w.workers, workerInstance)

		w.logger.Info("Registered worker for task queue",
			zap.String("task_queue", taskQueue),
		)

		// Start worker in background
		w.wg.Add(1)
		go func(queue string, wk worker.Worker) {
			defer w.wg.Done()
			w.logger.Info("Starting worker", zap.String("task_queue", queue))
			if err := wk.Run(worker.InterruptCh()); err != nil {
				w.logger.Error("Worker stopped with error",
					zap.String("task_queue", queue),
					zap.Error(err),
				)
			}
		}(taskQueue, workerInstance)
	}

	w.logger.Info("All workers started",
		zap.Int("worker_count", len(w.workers)),
		zap.Strings("task_queues", w.config.Agent.TaskQueues),
	)

	// Register to server (if configured)
	if w.config.Agent.ServerURL != "" {
		if err := w.registerToServer(); err != nil {
			w.logger.Warn("Failed to register to server", zap.Error(err))
			// Don't fail startup - registration is optional
		} else {
			// Start heartbeat updater after successful registration
			w.startHeartbeatUpdater()
		}
	}

	return nil
}

// registerToServer registers this agent to the Waterflow server.
func (w *Worker) registerToServer() error {
	hostname, _ := os.Hostname()
	w.agentID = fmt.Sprintf("agent-%s", uuid.New().String()[:8])

	reqBody := map[string]interface{}{
		"agent_id":    w.agentID,
		"hostname":    hostname,
		"ip_address":  getLocalIP(),
		"task_queues": w.config.Agent.TaskQueues,
		"metadata": map[string]string{
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
			"version": "dev",
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(
		w.config.Agent.ServerURL+"/v1/agents/register",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send registration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	w.logger.Info("Registered to server",
		zap.String("agent_id", w.agentID),
		zap.String("server_url", w.config.Agent.ServerURL),
	)

	return nil
}

// getLocalIP returns the local IP address (simplified implementation).
func getLocalIP() string {
	// Simplified: return empty string
	// Production: use net.InterfaceAddrs() to get actual IP
	return ""
}

// startHeartbeatUpdater starts a background goroutine to update heartbeat.
func (w *Worker) startHeartbeatUpdater() {
	if w.config.Agent.ServerURL == "" {
		w.logger.Info("Server URL not configured, skipping heartbeat updates")
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := w.updateHeartbeat(); err != nil {
					w.logger.Warn("Failed to update heartbeat", zap.Error(err))
				}
			case <-w.stopCh:
				return
			}
		}
	}()

	w.logger.Info("Heartbeat updater started")
}

// updateHeartbeat sends a heartbeat to the server with retry logic.
func (w *Worker) updateHeartbeat() error {
	var lastErr error

	// Retry up to 3 times with exponential backoff
	for attempt := 0; attempt < 3; attempt++ {
		if err := w.doHeartbeat(); err != nil {
			lastErr = err
			w.logger.Warn("Heartbeat failed, retrying...",
				zap.Int("attempt", attempt+1),
				zap.Error(err),
			)
			time.Sleep(time.Second * time.Duration(attempt+1))
			continue
		}
		return nil
	}

	return fmt.Errorf("heartbeat failed after 3 attempts: %w", lastErr)
}

// doHeartbeat performs a single heartbeat request.
func (w *Worker) doHeartbeat() error {
	reqBody := map[string]interface{}{
		"agent_id": w.agentID,
		"status":   "healthy",
	}

	jsonData, _ := json.Marshal(reqBody)

	// Use HTTP client with timeout to prevent hanging requests
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Post(
		w.config.Agent.ServerURL+"/v1/agents/heartbeat",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		w.heartbeatMu.Lock()
		w.heartbeatFailures++
		failures := w.heartbeatFailures
		w.heartbeatMu.Unlock()

		// Log critical error if too many consecutive failures
		if failures >= 5 {
			w.logger.Error("Critical: heartbeat failed 5+ times consecutively",
				zap.Int("consecutive_failures", failures),
				zap.Error(err),
			)
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.heartbeatMu.Lock()
		w.heartbeatFailures++
		failures := w.heartbeatFailures
		w.heartbeatMu.Unlock()

		if failures >= 5 {
			w.logger.Error("Critical: heartbeat failed 5+ times consecutively",
				zap.Int("consecutive_failures", failures),
				zap.Int("status_code", resp.StatusCode),
			)
		}
		return fmt.Errorf("heartbeat failed with status %d", resp.StatusCode)
	}

	// Reset failure counter on success
	w.heartbeatMu.Lock()
	w.heartbeatFailures = 0
	w.heartbeatMu.Unlock()

	return nil
}

// Shutdown gracefully stops all workers.
func (w *Worker) Shutdown(ctx context.Context) error {
	w.logger.Info("Shutting down agent workers")

	// Stop heartbeat updater
	close(w.stopCh)

	// Stop all workers
	for i, workerInstance := range w.workers {
		w.logger.Info("Stopping worker", zap.Int("index", i))
		workerInstance.Stop()
	}

	// Wait for all worker goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.logger.Info("All workers stopped gracefully")
	case <-ctx.Done():
		w.logger.Warn("Shutdown timeout exceeded, forcing close")
	}

	// Close Temporal client
	w.temporalClient.Close()

	w.logger.Info("Agent shutdown complete")
	return nil
}

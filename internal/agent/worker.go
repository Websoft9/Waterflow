package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/temporal"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
)

// Worker represents an Agent Worker instance.
type Worker struct {
	config         *config.Config
	logger         *zap.Logger
	temporalClient *temporal.Client
	workers        []worker.Worker // One worker per task queue
	pluginManager  *PluginManager
	wg             sync.WaitGroup // Wait for worker goroutines
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

	return nil
}

// Shutdown gracefully stops all workers.
func (w *Worker) Shutdown(ctx context.Context) error {
	w.logger.Info("Shutting down agent workers")

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

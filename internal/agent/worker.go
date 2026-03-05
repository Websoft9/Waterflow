package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/dsl/node/builtin"
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
	nodeRegistry   *node.Registry // Node registry for Activities
	wg             sync.WaitGroup // Wait for worker goroutines
}

// NewWorker creates a new Agent Worker and connects to Temporal.
func NewWorker(cfg *config.Config, logger *zap.Logger) (*Worker, error) {
	// Connect to Temporal
	temporalClient, err := connectToTemporal(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Temporal: %w", err)
	}
	return newWorkerFromClient(cfg, temporalClient, logger)
}

// newWorkerFromClient creates a Worker from an already-connected Temporal client.
// Used internally and by tests to bypass Temporal network connection.
func newWorkerFromClient(cfg *config.Config, temporalClient *temporal.Client, log *zap.Logger) (*Worker, error) {
	// Initialize NodeRegistry
	nodeRegistry := node.NewRegistry()

	// Register builtin nodes (checkout@v1, run@v1)
	if err := builtin.RegisterBuiltinNodes(nodeRegistry); err != nil {
		return nil, fmt.Errorf("failed to register builtin nodes: %w", err)
	}
	log.Info("Registered builtin nodes", zap.Int("count", 2))

	// Initialize Plugin Manager with registry
	pluginManager := NewPluginManager(cfg.Agent.PluginDir, nodeRegistry, log)

	w := &Worker{
		config:         cfg,
		logger:         log,
		temporalClient: temporalClient,
		workers:        make([]worker.Worker, 0, len(cfg.Agent.TaskQueues)),
		pluginManager:  pluginManager,
		nodeRegistry:   nodeRegistry,
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

		// Final attempt: log as Error with "giving up"; intermediate: Warn with "retrying"
		if attempt == cfg.Temporal.MaxRetries {
			logger.Error("Failed to connect to Temporal, giving up",
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
	// Load plugins
	if err := w.pluginManager.LoadPlugins(); err != nil {
		w.logger.Warn("Failed to load plugins", zap.Error(err))
		// Don't fail startup - plugins are optional
	}

	// Start hot-reload watcher if enabled
	if w.config.Agent.AutoReloadPlugins {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			ctx := context.Background() // TODO: use worker context for cancellation
			if err := w.pluginManager.WatchPlugins(ctx); err != nil {
				w.logger.Error("Plugin watcher failed", zap.Error(err))
			}
		}()
		w.logger.Info("Plugin hot-reload enabled")
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
		// Register job instance workflow for matrix execution
		workerInstance.RegisterWorkflow(temporal.ExecuteJobInstance)

		// Register activities (Step executor)
		activities := temporal.NewActivities(w.logger, w.nodeRegistry)
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
	if w.temporalClient != nil {
		w.temporalClient.Close()
	}

	w.logger.Info("Agent shutdown complete")
	return nil
}

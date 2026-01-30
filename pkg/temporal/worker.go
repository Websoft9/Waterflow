package temporal

import (
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
)

// Worker wraps the Temporal worker with logging.
type Worker struct {
	worker worker.Worker
	logger *zap.Logger
}

// NewWorker creates a new Temporal worker and registers workflows ONLY (no activities).
// Server Worker only orchestrates workflows - Step execution happens on Agent Workers.
func NewWorker(client *Client, activities *Activities) *Worker {
	w := worker.New(client.client, client.config.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     100,
		MaxConcurrentWorkflowTaskExecutionSize: 50,
	})

	// Register workflows (Server orchestrates workflow execution)
	w.RegisterWorkflow(RunWorkflowExecutor)
	w.RegisterWorkflow(ExecuteJobInstance) // Child workflow for job execution

	// ⚠️ DO NOT register activities here!
	// Step activities are executed by Agent Workers on their task queues.
	// Server Worker only handles workflow orchestration.

	client.logger.Info("Temporal Worker created (orchestration only - no activities)",
		zap.String("task_queue", client.config.TaskQueue),
	)

	return &Worker{
		worker: w,
		logger: client.logger,
	}
}

// Start starts the worker in a non-blocking way.
func (w *Worker) Start() error {
	w.logger.Info("Starting Temporal Worker")

	// Start worker in background goroutine
	go func() {
		if err := w.worker.Run(worker.InterruptCh()); err != nil {
			w.logger.Error("Worker stopped with error", zap.Error(err))
		}
	}()

	return nil
}

// Stop stops the worker gracefully.
func (w *Worker) Stop() {
	w.logger.Info("Stopping Temporal Worker")
	w.worker.Stop()
}

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Websoft9/waterflow/internal/agent"
	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/logger"
	"go.uber.org/zap"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	configFile := flag.String("config", "/etc/waterflow/agent.yaml", "config file path")
	taskQueues := flag.String("task-queues", "", "comma-separated task queue names")
	logLevel := flag.String("log-level", "", "log level (overrides config)")
	showVersion := flag.Bool("version", false, "show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Waterflow Agent\n")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Commit:     %s\n", Commit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadAgent(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Override with command-line flags
	if *taskQueues != "" {
		cfg.Agent.TaskQueues = parseTaskQueues(*taskQueues)
	}
	if *logLevel != "" {
		cfg.Log.Level = *logLevel
	}

	// Initialize logger
	if err := logger.Init(cfg.Log.Level, cfg.Log.Format); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync()
	}()

	logger.Log.Info("Waterflow Agent starting",
		zap.String("version", Version),
		zap.String("commit", Commit),
		zap.String("build_time", BuildTime),
	)

	logger.Log.Info("Configuration loaded",
		zap.String("config_file", *configFile),
		zap.Strings("task_queues", cfg.Agent.TaskQueues),
		zap.String("temporal_address", cfg.Temporal.Host),
		zap.String("log_level", cfg.Log.Level),
	)

	// Create and start Agent Worker
	worker, err := agent.NewWorker(cfg, logger.Log)
	if err != nil {
		logger.Log.Error("Failed to create worker", zap.Error(err))
		os.Exit(1)
	}

	// Start worker
	if err := worker.Start(); err != nil {
		logger.Log.Error("Failed to start worker", zap.Error(err))
		os.Exit(1)
	}

	logger.Log.Info("Agent started successfully",
		zap.Strings("task_queues", cfg.Agent.TaskQueues),
	)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutdown signal received")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Agent.ShutdownTimeout)
	defer cancel()

	if err := worker.Shutdown(ctx); err != nil {
		logger.Log.Error("Worker shutdown failed", zap.Error(err))
		os.Exit(2)
	}

	logger.Log.Info("Agent stopped gracefully")
}

func parseTaskQueues(s string) []string {
	// Split by comma and trim spaces
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

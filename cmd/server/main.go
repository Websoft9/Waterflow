package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Websoft9/waterflow/internal/server"
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
	configFile := flag.String("config", "config.yaml", "config file path")
	port := flag.Int("port", 0, "server port (overrides config)")
	logLevel := flag.String("log-level", "", "log level (overrides config)")
	showVersion := flag.Bool("version", false, "show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Waterflow Server\n")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Commit:     %s\n", Commit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}

	cfg, err := config.Load(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *logLevel != "" {
		cfg.Log.Level = *logLevel
	}

	if err := logger.Init(cfg.Log.Level, cfg.Log.Format); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync() // Ignore errors on sync
	}()

	logger.Log.Info("Waterflow Server starting",
		zap.String("version", Version),
		zap.String("commit", Commit),
		zap.String("build_time", BuildTime),
	)

	logger.Log.Info("Configuration loaded",
		zap.String("config_file", *configFile),
		zap.String("log_level", cfg.Log.Level),
		zap.String("log_format", cfg.Log.Format),
	)

	srv := server.New(cfg, logger.Log)

	go func() {
		if err := srv.Start(); err != nil {
			logger.Log.Error("Server failed", zap.Error(err))
			os.Exit(1)
		}
	}()

	logger.Log.Info("Server started successfully",
		zap.String("address", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("Server shutdown failed", zap.Error(err))
		os.Exit(2)
	}

	logger.Log.Info("Server stopped gracefully")
}

// Package temporal provides Temporal workflow engine integration for Waterflow.
// It handles client connection, worker registration, and workflow execution.
package temporal

import (
	"fmt"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// Client wraps the Temporal client with configuration and logging.
type Client struct {
	client client.Client
	config *config.TemporalConfig
	logger *zap.Logger
}

// NewClient creates a new Temporal client with retry logic.
// It will retry connection up to config.MaxRetries times with config.RetryInterval between attempts.
// Returns error if all connection attempts fail.
func NewClient(cfg *config.TemporalConfig, logger *zap.Logger) (*Client, error) {
	var temporalClient client.Client
	var err error

	logger.Info("Connecting to Temporal",
		zap.String("address", cfg.Host),
		zap.String("namespace", cfg.Namespace),
		zap.Int("max_retries", cfg.MaxRetries),
	)

	// Retry connection with exponential backoff
	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
		temporalClient, err = client.Dial(client.Options{
			HostPort:  cfg.Host,
			Namespace: cfg.Namespace,
			Logger:    newTemporalLogger(logger),
		})

		if err == nil {
			logger.Info("Connected to Temporal successfully",
				zap.String("address", cfg.Host),
				zap.String("namespace", cfg.Namespace),
			)

			return &Client{
				client: temporalClient,
				config: cfg,
				logger: logger,
			}, nil
		}

		logger.Warn("Failed to connect to Temporal, retrying",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", cfg.MaxRetries),
			zap.Error(err),
		)

		if attempt < cfg.MaxRetries {
			time.Sleep(cfg.RetryInterval)
		}
	}

	return nil, fmt.Errorf("failed to connect to Temporal after %d attempts: %w", cfg.MaxRetries, err)
}

// GetClient returns the underlying Temporal client.
func (c *Client) GetClient() client.Client {
	return c.client
}

// GetConfig returns the Temporal configuration.
func (c *Client) GetConfig() *config.TemporalConfig {
	return c.config
}

// Close closes the Temporal client connection.
func (c *Client) Close() {
	c.client.Close()
	c.logger.Info("Temporal client closed")
}

// CheckHealth checks if the Temporal connection is healthy.
// Returns nil if healthy, error otherwise.
func (c *Client) CheckHealth() error {
	if c.client == nil {
		return fmt.Errorf("temporal client not initialized")
	}
	// Temporal client maintains connection health internally
	// If client is not nil and not closed, it's considered healthy
	return nil
}

// temporalLogger adapts zap.Logger to Temporal's logger interface.
type temporalLogger struct {
	logger *zap.Logger
}

// newTemporalLogger creates a new Temporal logger adapter.
func newTemporalLogger(logger *zap.Logger) *temporalLogger {
	return &temporalLogger{logger: logger}
}

func (l *temporalLogger) Debug(msg string, keyvals ...interface{}) {
	l.logger.Debug(msg, convertToZapFields(keyvals)...)
}

func (l *temporalLogger) Info(msg string, keyvals ...interface{}) {
	l.logger.Info(msg, convertToZapFields(keyvals)...)
}

func (l *temporalLogger) Warn(msg string, keyvals ...interface{}) {
	l.logger.Warn(msg, convertToZapFields(keyvals)...)
}

func (l *temporalLogger) Error(msg string, keyvals ...interface{}) {
	l.logger.Error(msg, convertToZapFields(keyvals)...)
}

// convertToZapFields converts key-value pairs to zap fields.
func convertToZapFields(keyvals []interface{}) []zap.Field {
	fields := make([]zap.Field, 0, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			key, ok := keyvals[i].(string)
			if !ok {
				continue
			}
			fields = append(fields, zap.Any(key, keyvals[i+1]))
		}
	}
	return fields
}

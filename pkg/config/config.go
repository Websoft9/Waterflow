// Package config provides configuration management for Waterflow server.
// It supports loading configuration from files, environment variables, and default values.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	// Server contains HTTP server configuration.
	Server ServerConfig `mapstructure:"server"`
	// Log contains logging configuration.
	Log LogConfig `mapstructure:"log"`
	// Temporal contains Temporal workflow engine configuration.
	Temporal TemporalConfig `mapstructure:"temporal"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	// Host is the server listening address.
	Host string `mapstructure:"host"`
	// Port is the server listening port (1-65535).
	Port int `mapstructure:"port"`
	// ReadTimeout is the maximum duration for reading request.
	ReadTimeout time.Duration `mapstructure:"read_timeout"`
	// WriteTimeout is the maximum duration for writing response.
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	// ShutdownTimeout is the maximum duration for graceful shutdown.
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	// Level is the logging level: debug, info, warn, error.
	Level string `mapstructure:"level"`
	// Format is the log format: json, text.
	Format string `mapstructure:"format"`
	// Output is the log output destination: stdout, stderr, or file path.
	Output string `mapstructure:"output"`
}

// TemporalConfig holds Temporal workflow engine configuration.
type TemporalConfig struct {
	// Host is the Temporal server address (host:port).
	Host string `mapstructure:"host"`
	// Namespace is the Temporal namespace.
	Namespace string `mapstructure:"namespace"`
	// TaskQueue is the default task queue name.
	TaskQueue string `mapstructure:"task_queue"`
}

// Load loads configuration from file and environment variables.
// Priority: Command line flags > Environment variables > Config file > Defaults
func Load(configFile string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure viper to read environment variables
	v.SetEnvPrefix("WATERFLOW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Load config file if provided
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			// If config file is explicitly specified but not found, return error
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// Warn if config file not found but continue with defaults
			fmt.Fprintf(os.Stderr, "Warning: config file %s not found, using defaults and environment variables\n", configFile)
		}
	}

	// Unmarshal config into struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "30s")

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")

	// Temporal defaults
	v.SetDefault("temporal.host", "localhost:7233")
	v.SetDefault("temporal.namespace", "waterflow")
	v.SetDefault("temporal.task_queue", "waterflow-server")
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}
	if c.Server.ReadTimeout < time.Second {
		return fmt.Errorf("server.read_timeout must be at least 1s, got %v", c.Server.ReadTimeout)
	}
	if c.Server.WriteTimeout < time.Second {
		return fmt.Errorf("server.write_timeout must be at least 1s, got %v", c.Server.WriteTimeout)
	}
	if c.Server.ShutdownTimeout < time.Second {
		return fmt.Errorf("server.shutdown_timeout must be at least 1s, got %v", c.Server.ShutdownTimeout)
	}

	// Validate log config
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.Log.Level] {
		return fmt.Errorf("log.level must be one of [debug, info, warn, error], got %s", c.Log.Level)
	}

	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[c.Log.Format] {
		return fmt.Errorf("log.format must be one of [json, text], got %s", c.Log.Format)
	}

	// Validate Temporal config
	if c.Temporal.Host == "" {
		return fmt.Errorf("temporal.host is required")
	}
	if c.Temporal.Namespace == "" {
		return fmt.Errorf("temporal.namespace is required")
	}
	if c.Temporal.TaskQueue == "" {
		return fmt.Errorf("temporal.task_queue is required")
	}

	return nil
}

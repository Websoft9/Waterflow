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
	// Agent contains Agent worker configuration.
	Agent AgentConfig `mapstructure:"agent"`
	// Log contains logging configuration.
	Log LogConfig `mapstructure:"log"`
	// Temporal contains Temporal workflow engine configuration.
	Temporal TemporalConfig `mapstructure:"temporal"`
	// Events contains event handling configuration.
	Events EventsConfig `mapstructure:"events"`
	// Audit contains audit logging configuration (Story 9-3).
	Audit AuditConfig `mapstructure:"audit"`
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

	// APIKey is the API authentication key (optional).
	// If not set, authentication is disabled.
	APIKey string `mapstructure:"api_key"`

	// MetricsPort is the port for Prometheus metrics endpoint.
	// Default: 9090
	MetricsPort int `mapstructure:"metrics_port"`

	// TLSCertFile is the path to TLS certificate file (optional, backward compatibility).
	// DEPRECATED: Use server.https.cert_file instead.
	// If set, server will use HTTPS.
	TLSCertFile string `mapstructure:"tls_cert_file"`

	// TLSKeyFile is the path to TLS private key file (optional, backward compatibility).
	// DEPRECATED: Use server.https.key_file instead.
	// Required if TLSCertFile is set.
	TLSKeyFile string `mapstructure:"tls_key_file"`

	// HTTP holds HTTP server configuration (Story 9-1).
	HTTP HTTPConfig `mapstructure:"http"`

	// HTTPS holds HTTPS/TLS server configuration (Story 9-1).
	HTTPS HTTPSConfig `mapstructure:"https"`

	// ServerGroupProvider specifies the provider type for server group management.
	// Options: "memory" (default), "file", "custom"
	ServerGroupProvider string `mapstructure:"server_group_provider"`

	// ServerGroupFile is the path to server groups YAML file (if provider=file).
	ServerGroupFile string `mapstructure:"server_group_file"`

	// Health contains health check configuration (Story 8-4 AC6).
	Health HealthConfig `mapstructure:"health"`
}

// HealthConfig holds health check configuration (Story 8-4 AC6).
type HealthConfig struct {
	// Timeout is the overall health check timeout (default: 2s).
	Timeout time.Duration `mapstructure:"timeout"`
	// TemporalTimeout is the Temporal connection check timeout (default: 2s).
	TemporalTimeout time.Duration `mapstructure:"temporal_timeout"`
	// DatabaseTimeout is the database ping timeout (default: 1s).
	DatabaseTimeout time.Duration `mapstructure:"db_timeout"`
}

// AgentConfig holds Agent worker configuration.
type AgentConfig struct {
	// TaskQueues is the list of task queues this agent will poll.
	// Corresponds to `runs-on` values in workflow YAML.
	// Example: ["linux-amd64", "linux-common", "gpu-a100"]
	TaskQueues []string `mapstructure:"task_queues"`

	// ID is the unique identifier for this agent instance.
	// Example: "agent-linux-1"
	// Optional: If empty, the Agent will register with Temporal Worker's default identity.
	ID string `mapstructure:"id"`

	// PluginDir is the directory containing node plugins (.so files).
	// Default: /opt/waterflow/plugins
	PluginDir string `mapstructure:"plugin_dir"`

	// AutoReloadPlugins enables hot-reloading of plugins when files change.
	// Default: false (requires fsnotify, Epic 4)
	AutoReloadPlugins bool `mapstructure:"auto_reload_plugins"`

	// MetricsPort is the port for Prometheus metrics endpoint.
	// Default: 9090
	MetricsPort string `mapstructure:"metrics_port"`

	// ShutdownTimeout is the maximum time to wait for graceful shutdown.
	// Default: 30s
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
	// ConnectionTimeout is the timeout for connecting to Temporal server.
	ConnectionTimeout time.Duration `mapstructure:"connection_timeout"`
	// MaxRetries is the maximum number of connection retry attempts.
	MaxRetries int `mapstructure:"max_retries"`
	// RetryInterval is the interval between connection retry attempts.
	RetryInterval time.Duration `mapstructure:"retry_interval"`
}

// EventsConfig holds event handling configuration.
type EventsConfig struct {
	// HandlerType specifies the event handler implementation.
	// Options: "noop" (default), "webhook", "custom"
	HandlerType string `mapstructure:"handler_type"`

	// Webhook configuration (used when handler_type=webhook)
	Webhook WebhookEventConfig `mapstructure:"webhook"`
}

// AuditConfig holds audit logging configuration (Story 9-3).
type AuditConfig struct {
	// Enabled turns on audit logging (default: true).
	Enabled bool `mapstructure:"enabled"`

	// StoreType specifies the storage backend (file, database).
	// Default: file
	StoreType string `mapstructure:"store_type"`

	// File configuration for file-based audit store.
	File FileAuditConfig `mapstructure:"file"`
}

// FileAuditConfig configures the file-based audit store.
type FileAuditConfig struct {
	// Path is the directory for audit log files.
	// Default: /var/log/waterflow/audit
	Path string `mapstructure:"path"`

	// MaxSize is the maximum size in MB before rotation (default: 100).
	MaxSize int `mapstructure:"max_size"`

	// MaxAge is the maximum days to retain old logs (default: 90).
	MaxAge int `mapstructure:"max_age"`

	// MaxBackups is the maximum number of old log files to retain (default: 30).
	MaxBackups int `mapstructure:"max_backups"`

	// Compress enables gzip compression of rotated files (default: true).
	Compress bool `mapstructure:"compress"`
}

// WebhookEventConfig holds webhook event handler configuration.
type WebhookEventConfig struct {
	// URL is the webhook endpoint
	URL string `mapstructure:"url"`

	// Headers are custom HTTP headers
	Headers map[string]string `mapstructure:"headers"`

	// Timeout is the request timeout
	Timeout time.Duration `mapstructure:"timeout"`
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
		// Support TOML format (AC2)
		if strings.HasSuffix(configFile, ".toml") {
			v.SetConfigType("toml")
		} else if strings.HasSuffix(configFile, ".yaml") || strings.HasSuffix(configFile, ".yml") {
			v.SetConfigType("yaml")
		}
		if err := v.ReadInConfig(); err != nil {
			// Check if it's a "file not found" error
			if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
				// Warn if config file not found but continue with defaults
				fmt.Fprintf(os.Stderr, "Warning: config file %s not found, using defaults and environment variables\n", configFile)
			} else {
				// Other errors (e.g., parse errors) should fail
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
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
	v.SetDefault("server.api_key", "")
	v.SetDefault("server.metrics_port", 9090)
	v.SetDefault("server.tls_cert_file", "")
	v.SetDefault("server.tls_key_file", "")

	// HTTP defaults (Story 9-1)
	v.SetDefault("server.http.enabled", false)
	v.SetDefault("server.http.port", 8080)
	v.SetDefault("server.http.redirect_to_https", false)

	// HTTPS defaults (Story 9-1)
	v.SetDefault("server.https.enabled", false)
	v.SetDefault("server.https.port", 8443)
	v.SetDefault("server.https.cert_file", "")
	v.SetDefault("server.https.key_file", "")
	v.SetDefault("server.https.min_tls_version", "1.2")

	v.SetDefault("server.server_group_provider", "memory")
	v.SetDefault("server.server_group_file", "")

	// Health check defaults (Story 8-4 AC6)
	v.SetDefault("server.health.timeout", "2s")
	v.SetDefault("server.health.temporal_timeout", "2s")
	v.SetDefault("server.health.db_timeout", "1s")

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")

	// Temporal defaults
	v.SetDefault("temporal.host", "localhost:7233")
	v.SetDefault("temporal.namespace", "waterflow")
	v.SetDefault("temporal.task_queue", "waterflow-server")
	v.SetDefault("temporal.connection_timeout", "10s")
	v.SetDefault("temporal.max_retries", 10)
	v.SetDefault("temporal.retry_interval", "5s")

	// Events defaults
	v.SetDefault("events.handler_type", "noop")
	v.SetDefault("events.webhook.timeout", "5s")
}

// Validate validates the Server configuration.
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server.port: %d, must be 1-65535\n"+
			"  Set WATERFLOW_SERVER_PORT=8080 or update server.port in config file", c.Server.Port)
	}
	if c.Server.MetricsPort < 0 || c.Server.MetricsPort > 65535 {
		return fmt.Errorf("invalid server.metrics_port: %d, must be 0-65535\n"+
			"  Set WATERFLOW_SERVER_METRICS_PORT=9090 or update server.metrics_port in config file", c.Server.MetricsPort)
	}
	if c.Server.ReadTimeout < 0 {
		return fmt.Errorf("invalid server.read_timeout: %v, must be >= 0\n"+
			"  Set WATERFLOW_SERVER_READ_TIMEOUT=30s or update server.read_timeout in config file", c.Server.ReadTimeout)
	}
	if c.Server.WriteTimeout < 0 {
		return fmt.Errorf("invalid server.write_timeout: %v, must be >= 0\n"+
			"  Set WATERFLOW_SERVER_WRITE_TIMEOUT=30s or update server.write_timeout in config file", c.Server.WriteTimeout)
	}
	if c.Server.ShutdownTimeout < 0 {
		return fmt.Errorf("invalid server.shutdown_timeout: %v, must be >= 0\n"+
			"  Set WATERFLOW_SERVER_SHUTDOWN_TIMEOUT=30s or update server.shutdown_timeout in config file", c.Server.ShutdownTimeout)
	}

	// Validate TLS configuration (Story 9-1, supports both old and new formats)
	// ValidateTLS handles both old (tls_cert_file) and new (https.cert_file) formats
	if err := c.Server.ValidateTLS(); err != nil {
		return err
	}

	// Validate log config
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, c.Log.Level) {
		return fmt.Errorf("invalid log.level: %s, must be one of: %v\n"+
			"  Set WATERFLOW_LOG_LEVEL=info or update log.level in config file", c.Log.Level, validLogLevels)
	}

	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, c.Log.Format) {
		return fmt.Errorf("invalid log.format: %s, must be one of: %v\n"+
			"  Set WATERFLOW_LOG_FORMAT=json or update log.format in config file", c.Log.Format, validLogFormats)
	}

	// Validate Temporal config
	if c.Temporal.Host == "" {
		return fmt.Errorf("temporal.host is required\n" +
			"  Set WATERFLOW_TEMPORAL_HOST=localhost:7233 or update temporal.host in config file")
	}
	if c.Temporal.Namespace == "" {
		return fmt.Errorf("temporal.namespace is required\n" +
			"  Set WATERFLOW_TEMPORAL_NAMESPACE=waterflow or update temporal.namespace in config file")
	}
	if c.Temporal.TaskQueue == "" {
		return fmt.Errorf("temporal.task_queue is required\n" +
			"  Set WATERFLOW_TEMPORAL_TASK_QUEUE=waterflow-server or update temporal.task_queue in config file")
	}
	if c.Temporal.ConnectionTimeout < 0 {
		return fmt.Errorf("invalid temporal.connection_timeout: %v, must be >= 0\n"+
			"  Set WATERFLOW_TEMPORAL_CONNECTION_TIMEOUT=10s or update temporal.connection_timeout in config file", c.Temporal.ConnectionTimeout)
	}
	if c.Temporal.MaxRetries < 0 {
		return fmt.Errorf("invalid temporal.max_retries: %d, must be >= 0\n"+
			"  Set WATERFLOW_TEMPORAL_MAX_RETRIES=10 or update temporal.max_retries in config file", c.Temporal.MaxRetries)
	}
	if c.Temporal.RetryInterval < 0 {
		return fmt.Errorf("invalid temporal.retry_interval: %v, must be >= 0\n"+
			"  Set WATERFLOW_TEMPORAL_RETRY_INTERVAL=5s or update temporal.retry_interval in config file", c.Temporal.RetryInterval)
	}

	// Validate Events config
	if c.Events.HandlerType == "webhook" {
		if c.Events.Webhook.URL == "" {
			return fmt.Errorf("events.webhook.url is required when handler_type=webhook\n" +
				"  Set WATERFLOW_EVENTS_WEBHOOK_URL=http://example.com/webhook or update events.webhook.url in config file")
		}
	}

	return nil
}

// ValidateAgent validates Agent-specific configuration.
func (c *Config) ValidateAgent() error {
	// 1. Task Queues must not be empty
	if len(c.Agent.TaskQueues) == 0 {
		return fmt.Errorf("agent.task_queues is required, must specify at least one task queue\n" +
			"  Hint: Set WATERFLOW_AGENT_TASK_QUEUES environment variable or use --task-queues flag\n" +
			"  Example: export WATERFLOW_AGENT_TASK_QUEUES=linux-amd64,linux-common")
	}

	// 2. Validate Task Queue names (ADR-0006)
	for _, queue := range c.Agent.TaskQueues {
		if err := validateQueueName(queue); err != nil {
			return fmt.Errorf("invalid task queue name %q: %w\n"+
				"  Task queue names must be alphanumeric with hyphens, max 255 chars", queue, err)
		}
	}

	// 3. Temporal config validation
	if c.Temporal.Host == "" {
		return fmt.Errorf("temporal.host is required\n" +
			"  Set WATERFLOW_TEMPORAL_HOST=localhost:7233 or update temporal.host in config file")
	}
	if c.Temporal.Namespace == "" {
		return fmt.Errorf("temporal.namespace is required\n" +
			"  Set WATERFLOW_TEMPORAL_NAMESPACE=default or update temporal.namespace in config file")
	}

	// 4. Plugin directory validation
	if c.Agent.PluginDir == "" {
		return fmt.Errorf("agent.plugin_dir is required\n" +
			"  Set WATERFLOW_AGENT_PLUGIN_DIR=/opt/waterflow/plugins or update agent.plugin_dir in config file")
	}
	// Check if plugin directory exists and is accessible (warn only, don't fail)
	// This allows tests to run without creating directories
	if stat, err := os.Stat(c.Agent.PluginDir); err != nil {
		if !os.IsNotExist(err) {
			// Only fail on permission errors, not on "not exist"
			return fmt.Errorf("agent.plugin_dir not accessible: %w\n"+
				"  Check permissions for: %s", err, c.Agent.PluginDir)
		}
		// Directory doesn't exist - this is OK, it will be created at runtime if needed
	} else if !stat.IsDir() {
		return fmt.Errorf("agent.plugin_dir is not a directory: %s\n"+
			"  Ensure path points to a directory", c.Agent.PluginDir)
	}

	// 5. Metrics port validation (if set)
	if c.Agent.MetricsPort != "" {
		// MetricsPort is a string to allow "" (disabled) or port number
		// We don't validate the actual port number format here as it will fail on Listen anyway
	}

	// 6. Log config validation
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, c.Log.Level) {
		return fmt.Errorf("invalid log.level: %s, must be one of: %v\n"+
			"  Set WATERFLOW_LOG_LEVEL=info or update log.level in config file", c.Log.Level, validLogLevels)
	}

	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, c.Log.Format) {
		return fmt.Errorf("invalid log.format: %s, must be one of: %v\n"+
			"  Set WATERFLOW_LOG_FORMAT=json or update log.format in config file", c.Log.Format, validLogFormats)
	}

	return nil
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// LoadAgent loads Agent configuration from file and environment variables.
// It validates Agent-specific settings and Task Queue names.
func LoadAgent(configFile string) (*Config, error) {
	v := viper.New()

	// Set defaults for Agent
	// Note: agent.task_queues has no default - it's required via environment or config file
	v.SetDefault("agent.plugin_dir", "/opt/waterflow/plugins")
	v.SetDefault("agent.auto_reload_plugins", false)
	v.SetDefault("agent.shutdown_timeout", 30*time.Second)

	// Same defaults as Server for Temporal and Log
	v.SetDefault("temporal.host", "localhost:7233")
	v.SetDefault("temporal.namespace", "default")
	// Note: Agent does NOT need task_queue config (uses agent.task_queues instead)
	v.SetDefault("temporal.connection_timeout", 10*time.Second)
	v.SetDefault("temporal.max_retries", 10)
	v.SetDefault("temporal.retry_interval", 5*time.Second)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")

	// Load from file if exists
	if configFile != "" {
		v.SetConfigFile(configFile)
		// Support TOML format (AC2)
		if strings.HasSuffix(configFile, ".toml") {
			v.SetConfigType("toml")
		} else if strings.HasSuffix(configFile, ".yaml") || strings.HasSuffix(configFile, ".yml") {
			v.SetConfigType("yaml")
		}
		if err := v.ReadInConfig(); err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// Warn if config file not found but continue with defaults
			fmt.Fprintf(os.Stderr, "Warning: config file %s not found, using defaults and environment variables\n", configFile)
		}
	}

	// Environment variable overrides
	v.SetEnvPrefix("WATERFLOW")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Special handling for array environment variables (Viper limitation)
	// TASK_QUEUES (legacy, without WATERFLOW prefix for backward compatibility)
	if taskQueuesEnv := os.Getenv("TASK_QUEUES"); taskQueuesEnv != "" {
		cfg.Agent.TaskQueues = strings.Split(taskQueuesEnv, ",")
		// Trim spaces from each queue name
		for i, q := range cfg.Agent.TaskQueues {
			cfg.Agent.TaskQueues[i] = strings.TrimSpace(q)
		}
	}
	// WATERFLOW_AGENT_TASK_QUEUES (new standard naming)
	if taskQueuesEnv := os.Getenv("WATERFLOW_AGENT_TASK_QUEUES"); taskQueuesEnv != "" {
		cfg.Agent.TaskQueues = strings.Split(taskQueuesEnv, ",")
		for i, q := range cfg.Agent.TaskQueues {
			cfg.Agent.TaskQueues[i] = strings.TrimSpace(q)
		}
	}

	// Validate Agent config using ValidateAgent method
	if err := cfg.ValidateAgent(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// validateQueueName validates Task Queue naming per ADR-0006.
// Queue names must contain only alphanumeric characters and hyphens, and be less than 256 characters.
func validateQueueName(name string) error {
	// Temporal requirement: alphanumeric and hyphens, length < 256
	if len(name) == 0 {
		return fmt.Errorf("queue name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("queue name too long (max 255 characters)")
	}

	// First and last characters must be alphanumeric
	first := name[0]
	last := name[len(name)-1]
	if !isAlphanumeric(first) || !isAlphanumeric(last) {
		return fmt.Errorf("queue name must start and end with alphanumeric characters")
	}

	// Middle characters can be alphanumeric or hyphens
	for i, ch := range name {
		if !isAlphanumeric(byte(ch)) && ch != '-' {
			return fmt.Errorf("queue name contains invalid character at position %d: %c", i, ch)
		}
	}

	return nil
}

// isAlphanumeric checks if a byte is alphanumeric (a-z, A-Z, 0-9).
func isAlphanumeric(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

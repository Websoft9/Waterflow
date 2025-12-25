package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary config file
	content := `
server:
  host: "127.0.0.1"
  port: 9090
  read_timeout: "60s"
  write_timeout: "60s"
  shutdown_timeout: "60s"

log:
  level: "debug"
  format: "text"
  output: "stdout"

temporal:
  host: "temporal:7233"
  namespace: "test"
  task_queue: "test-queue"
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	// Load config
	cfg, err := Load(tmpFile.Name())
	require.NoError(t, err)

	// Verify values
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, 60*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "text", cfg.Log.Format)
	assert.Equal(t, "temporal:7233", cfg.Temporal.Host)
	assert.Equal(t, "test", cfg.Temporal.Namespace)
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Set environment variables
	require.NoError(t, os.Setenv("WATERFLOW_SERVER_HOST", "10.0.0.1"))
	require.NoError(t, os.Setenv("WATERFLOW_SERVER_PORT", "7070"))
	require.NoError(t, os.Setenv("WATERFLOW_LOG_LEVEL", "error"))
	defer func() {
		_ = os.Unsetenv("WATERFLOW_SERVER_HOST")
		_ = os.Unsetenv("WATERFLOW_SERVER_PORT")
		_ = os.Unsetenv("WATERFLOW_LOG_LEVEL")
	}()

	// Load config without file
	cfg, err := Load("")
	require.NoError(t, err)

	// Verify environment variables override defaults
	assert.Equal(t, "10.0.0.1", cfg.Server.Host)
	assert.Equal(t, 7070, cfg.Server.Port)
	assert.Equal(t, "error", cfg.Log.Level)
	// Verify defaults are still used for unset values
	assert.Equal(t, "json", cfg.Log.Format)
}

func TestConfigPriority(t *testing.T) {
	// Create config file with port 9090
	content := `
server:
  port: 9090
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	// Set environment variable with port 7070
	require.NoError(t, os.Setenv("WATERFLOW_SERVER_PORT", "7070"))
	defer func() {
		_ = os.Unsetenv("WATERFLOW_SERVER_PORT")
	}()

	// Load config
	cfg, err := Load(tmpFile.Name())
	require.NoError(t, err)

	// Environment variable should override file
	assert.Equal(t, 7070, cfg.Server.Port)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Server: ServerConfig{
					Port:            8080,
					ReadTimeout:     30 * time.Second,
					WriteTimeout:    30 * time.Second,
					ShutdownTimeout: 30 * time.Second,
				},
				Log: LogConfig{
					Level:  "info",
					Format: "json",
				},
				Temporal: TemporalConfig{
					Host:              "localhost:7233",
					Namespace:         "waterflow",
					TaskQueue:         "test",
					ConnectionTimeout: 10 * time.Second,
					MaxRetries:        3,
					RetryInterval:     5 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			config: Config{
				Server: ServerConfig{
					Port:            0,
					ReadTimeout:     30 * time.Second,
					WriteTimeout:    30 * time.Second,
					ShutdownTimeout: 30 * time.Second,
				},
				Log: LogConfig{
					Level:  "info",
					Format: "json",
				},
				Temporal: TemporalConfig{
					Host:      "localhost:7233",
					Namespace: "waterflow",
					TaskQueue: "test",
				},
			},
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			config: Config{
				Server: ServerConfig{
					Port:            70000,
					ReadTimeout:     30 * time.Second,
					WriteTimeout:    30 * time.Second,
					ShutdownTimeout: 30 * time.Second,
				},
				Log: LogConfig{
					Level:  "info",
					Format: "json",
				},
				Temporal: TemporalConfig{
					Host:      "localhost:7233",
					Namespace: "waterflow",
					TaskQueue: "test",
				},
			},
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name: "invalid log level",
			config: Config{
				Server: ServerConfig{
					Port:            8080,
					ReadTimeout:     30 * time.Second,
					WriteTimeout:    30 * time.Second,
					ShutdownTimeout: 30 * time.Second,
				},
				Log: LogConfig{
					Level:  "invalid",
					Format: "json",
				},
				Temporal: TemporalConfig{
					Host:      "localhost:7233",
					Namespace: "waterflow",
					TaskQueue: "test",
				},
			},
			wantErr: true,
			errMsg:  "log.level must be one of",
		},
		{
			name: "timeout too short",
			config: Config{
				Server: ServerConfig{
					Port:            8080,
					ReadTimeout:     500 * time.Millisecond,
					WriteTimeout:    30 * time.Second,
					ShutdownTimeout: 30 * time.Second,
				},
				Log: LogConfig{
					Level:  "info",
					Format: "json",
				},
				Temporal: TemporalConfig{
					Host:      "localhost:7233",
					Namespace: "waterflow",
					TaskQueue: "test",
				},
			},
			wantErr: true,
			errMsg:  "read_timeout must be at least 1s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	// Load config without file and without environment variables
	cfg, err := Load("")
	require.NoError(t, err)

	// Verify defaults
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.ShutdownTimeout)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, "stdout", cfg.Log.Output)
	assert.Equal(t, "localhost:7233", cfg.Temporal.Host)
	assert.Equal(t, "waterflow", cfg.Temporal.Namespace)
	assert.Equal(t, "waterflow-server", cfg.Temporal.TaskQueue)
}

func TestLoadAgent(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		errMsg  string
		verify  func(*testing.T, *Config)
	}{
		{
			name: "valid agent config",
			yaml: `
agent:
  task_queues:
    - linux-amd64
    - linux-common
  plugin_dir: /opt/waterflow/plugins
  shutdown_timeout: 30s
temporal:
  host: localhost:7233
  namespace: waterflow
log:
  level: info
  format: json
`,
			wantErr: false,
			verify: func(t *testing.T, cfg *Config) {
				assert.Equal(t, []string{"linux-amd64", "linux-common"}, cfg.Agent.TaskQueues)
				assert.Equal(t, "/opt/waterflow/plugins", cfg.Agent.PluginDir)
				assert.Equal(t, 30*time.Second, cfg.Agent.ShutdownTimeout)
			},
		},
		{
			name: "empty task queues",
			yaml: `
agent:
  task_queues: []
`,
			wantErr: true,
			errMsg:  "task_queues cannot be empty",
		},
		{
			name: "invalid queue name with underscore",
			yaml: `
agent:
  task_queues:
    - invalid_queue_name
`,
			wantErr: true,
			errMsg:  "invalid task queue name",
		},
		{
			name: "invalid queue name with special chars",
			yaml: `
agent:
  task_queues:
    - "queue@123"
`,
			wantErr: true,
			errMsg:  "invalid task queue name",
		},
		{
			name: "queue name starting with hyphen",
			yaml: `
agent:
  task_queues:
    - "-invalid"
`,
			wantErr: true,
			errMsg:  "invalid task queue name",
		},
		{
			name: "queue name ending with hyphen",
			yaml: `
agent:
  task_queues:
    - "invalid-"
`,
			wantErr: true,
			errMsg:  "invalid task queue name",
		},
		{
			name: "valid queue names with hyphens",
			yaml: `
agent:
  task_queues:
    - linux-amd64
    - gpu-a100
    - server-group-1
temporal:
  host: localhost:7233
  namespace: waterflow
`,
			wantErr: false,
			verify: func(t *testing.T, cfg *Config) {
				assert.Equal(t, []string{"linux-amd64", "gpu-a100", "server-group-1"}, cfg.Agent.TaskQueues)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "agent-config-*.yaml")
			require.NoError(t, err)
			defer func() {
				_ = os.Remove(tmpfile.Name())
			}()

			_, err = tmpfile.Write([]byte(tt.yaml))
			require.NoError(t, err)
			_ = tmpfile.Close()

			cfg, err := LoadAgent(tmpfile.Name())
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.verify != nil {
					tt.verify(t, cfg)
				}
			}
		})
	}
}

func TestValidateQueueName(t *testing.T) {
	tests := []struct {
		name    string
		queue   string
		wantErr bool
	}{
		{"valid alphanumeric", "queue123", false},
		{"valid with hyphens", "linux-amd64", false},
		{"valid complex", "server-group-1-amd64", false},
		{"empty string", "", true},
		{"too long", string(make([]byte, 256)), true},
		{"underscore", "invalid_name", true},
		{"special chars", "queue@123", true},
		{"start with hyphen", "-invalid", true},
		{"end with hyphen", "invalid-", true},
		{"space", "invalid name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueueName(tt.queue)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

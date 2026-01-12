// Package secrets provides interfaces and implementations for runtime secret injection.
//
// The SecretProvider interface allows workflows to retrieve secrets from external
// systems (HashiCorp Vault, AWS Secrets Manager, environment variables, etc.) without
// storing them in the Waterflow system itself.
package secrets

import (
	"context"
)

// SecretProvider defines the interface for retrieving secrets at runtime.
//
// Implementations fetch secrets from external systems and should:
// - Support context cancellation and timeouts
// - Return SecretNotFoundError when a secret doesn't exist
// - Be thread-safe for concurrent access
// - Implement proper resource cleanup in Close()
type SecretProvider interface {
	// GetSecret retrieves a single secret value by key.
	//
	// The key format depends on the provider implementation:
	//   - EnvProvider: "api_key" → WATERFLOW_SECRET_API_KEY
	//   - VaultProvider: "db.password" → path=db, field=password
	//
	// Returns SecretNotFoundError if the secret doesn't exist.
	GetSecret(ctx context.Context, key string) (string, error)

	// GetSecretMap retrieves all secrets with a given prefix.
	//
	// This is an optional optimization for batch operations. Implementations
	// may return an error if batch operations are not supported.
	//
	// Example:
	//   - EnvProvider: prefix="db" → {db.host, db.port, db.password}
	//   - VaultProvider: prefix="app" → all fields in the "app" path
	GetSecretMap(ctx context.Context, prefix string) (map[string]string, error)

	// Close cleans up provider resources.
	//
	// This is optional for stateless providers. Providers with connection
	// pools or background goroutines should implement proper cleanup.
	Close() error
}

// SecretProviderConfig holds the configuration for secret providers.
type SecretProviderConfig struct {
	// ProviderType specifies which provider to use (env, vault, file, etc.)
	ProviderType string `mapstructure:"provider_type" yaml:"provider_type"`

	// Env configuration for EnvSecretProvider
	Env *EnvProviderConfig `mapstructure:"env" yaml:"env,omitempty"`

	// Vault configuration for VaultSecretProvider
	Vault *VaultProviderConfig `mapstructure:"vault" yaml:"vault,omitempty"`

	// File configuration for FileSecretProvider (YAML file backend)
	File *FileProviderConfig `mapstructure:"file" yaml:"file,omitempty"`

	// Cache configuration
	Cache *CacheConfig `mapstructure:"cache" yaml:"cache,omitempty"`

	// Redaction configuration for log sanitization
	Redaction *RedactionConfig `mapstructure:"redaction" yaml:"redaction,omitempty"`
}

// CacheConfig controls secret caching behavior.
type CacheConfig struct {
	// Enabled turns on in-memory caching (default: true)
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`

	// TTL is the cache time-to-live in seconds (default: 300)
	TTL int `mapstructure:"ttl" yaml:"ttl"`

	// MaxSize is the maximum number of cached secrets (default: 1000)
	MaxSize int `mapstructure:"max_size" yaml:"max_size"`
}

// RedactionConfig controls secret redaction in logs.
type RedactionConfig struct {
	// Enabled turns on automatic secret redaction (default: true)
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`

	// Mode controls redaction mode: "full" or "partial" (default: "full")
	// - full: Replace with "***REDACTED***"
	// - partial: Show first 3 and last 3 characters
	Mode string `mapstructure:"mode" yaml:"mode"`
}

// EnvProviderConfig configures the environment variable provider.
type EnvProviderConfig struct {
	// Prefix for all secret environment variables (default: WATERFLOW_SECRET_)
	Prefix string `mapstructure:"prefix" yaml:"prefix"`

	// CaseSensitive controls whether key names are case-sensitive (default: false)
	CaseSensitive bool `mapstructure:"case_sensitive" yaml:"case_sensitive"`
}

// VaultProviderConfig configures HashiCorp Vault provider.
type VaultProviderConfig struct {
	// Address is the Vault server URL (e.g., https://vault.example.com:8200)
	Address string `mapstructure:"address" yaml:"address"`

	// Token is the Vault authentication token
	Token string `mapstructure:"token" yaml:"token"`

	// MountPath is the KV secrets engine mount path (default: secret)
	MountPath string `mapstructure:"mount_path" yaml:"mount_path"`

	// SecretPath is the base path for secrets (e.g., waterflow/production)
	SecretPath string `mapstructure:"secret_path" yaml:"secret_path"`

	// KVVersion is the KV engine version (1 or 2, default: 2)
	KVVersion int `mapstructure:"kv_version" yaml:"kv_version"`
}

// FileProviderConfig configures YAML file-based provider.
type FileProviderConfig struct {
	// Path to the YAML file containing secrets
	Path string `mapstructure:"path" yaml:"path"`
}

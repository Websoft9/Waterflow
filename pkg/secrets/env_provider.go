package secrets

import (
	"context"
	"os"
	"strings"
)

// EnvSecretProvider retrieves secrets from environment variables.
//
// Key conversion rules:
// - Dots are replaced with underscores: "db.password" → "DB_PASSWORD"
// - Keys are converted to uppercase by default (unless CaseSensitive is true)
// - Prefix is prepended: "api_key" → "WATERFLOW_SECRET_API_KEY"
type EnvSecretProvider struct {
	config *EnvProviderConfig
}

// NewEnvSecretProvider creates a new environment variable provider.
func NewEnvSecretProvider(config *EnvProviderConfig) *EnvSecretProvider {
	if config == nil {
		config = &EnvProviderConfig{}
	}
	if config.Prefix == "" {
		config.Prefix = "WATERFLOW_SECRET_"
	}
	return &EnvSecretProvider{config: config}
}

// GetSecret retrieves a secret from environment variables.
func (p *EnvSecretProvider) GetSecret(ctx context.Context, key string) (string, error) {
	// Convert key to environment variable name
	envKey := p.keyToEnvVar(key)

	// Get from environment
	value := os.Getenv(envKey)
	if value == "" {
		return "", &SecretNotFoundError{Key: key}
	}

	return value, nil
}

// GetSecretMap retrieves all secrets with a given prefix.
func (p *EnvSecretProvider) GetSecretMap(ctx context.Context, prefix string) (map[string]string, error) {
	secrets := make(map[string]string)
	envPrefix := p.keyToEnvVar(prefix)

	// Scan all environment variables
	for _, env := range os.Environ() {
		// Skip if doesn't have our base prefix
		if !strings.HasPrefix(env, p.config.Prefix) {
			continue
		}

		// Check if matches the requested prefix
		if prefix != "" && !strings.HasPrefix(env, envPrefix) {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			// Convert env var back to key
			key := p.envVarToKey(parts[0])
			secrets[key] = parts[1]
		}
	}

	return secrets, nil
}

// Close is a no-op for environment provider.
func (p *EnvSecretProvider) Close() error {
	return nil
}

// keyToEnvVar converts a secret key to environment variable name.
// Example: "db.password" → "WATERFLOW_SECRET_DB_PASSWORD"
func (p *EnvSecretProvider) keyToEnvVar(key string) string {
	// Replace dots with underscores
	envKey := strings.ReplaceAll(key, ".", "_")

	// Convert to uppercase if not case-sensitive
	if !p.config.CaseSensitive {
		envKey = strings.ToUpper(envKey)
	}

	// Add prefix
	return p.config.Prefix + envKey
}

// envVarToKey converts environment variable name back to key.
func (p *EnvSecretProvider) envVarToKey(envVar string) string {
	// Remove prefix
	key := strings.TrimPrefix(envVar, p.config.Prefix)

	// Convert underscores to dots
	key = strings.ReplaceAll(key, "_", ".")

	// Lowercase if not case-sensitive
	if !p.config.CaseSensitive {
		key = strings.ToLower(key)
	}

	return key
}

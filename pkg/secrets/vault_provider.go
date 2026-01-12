package secrets

import (
	"context"
	"fmt"
	"strings"

	vault "github.com/hashicorp/vault/api"
)

// VaultSecretProvider retrieves secrets from HashiCorp Vault.
//
// Supports both KV v1 and KV v2 secret engines. Key format:
// - "db.password" → read from path "db", field "password"
// - "api_key" → read from root path, field "api_key"
type VaultSecretProvider struct {
	client *vault.Client
	config *VaultProviderConfig
}

// NewVaultSecretProvider creates a new Vault provider.
func NewVaultSecretProvider(config *VaultProviderConfig) (*VaultSecretProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("vault config is required")
	}

	if config.Address == "" {
		return nil, fmt.Errorf("vault address is required")
	}

	if config.Token == "" {
		return nil, fmt.Errorf("vault token is required")
	}

	// Set defaults
	if config.MountPath == "" {
		config.MountPath = "secret"
	}
	if config.KVVersion == 0 {
		config.KVVersion = 2
	}

	// Validate KV version
	if config.KVVersion != 1 && config.KVVersion != 2 {
		return nil, fmt.Errorf("invalid kv_version: %d (must be 1 or 2)", config.KVVersion)
	}

	// Create Vault client
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = config.Address

	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	// Set token
	client.SetToken(config.Token)

	return &VaultSecretProvider{
		client: client,
		config: config,
	}, nil
}

// GetSecret retrieves a secret from Vault.
func (p *VaultSecretProvider) GetSecret(ctx context.Context, key string) (string, error) {
	// Parse key into path and field
	// Example: "db.password" → path="db", field="password"
	path, field := p.parseKey(key)

	// Build full Vault path
	vaultPath := p.buildVaultPath(path)

	// Read from Vault
	secret, err := p.client.Logical().ReadWithContext(ctx, vaultPath)
	if err != nil {
		return "", &SecretProviderError{
			Provider: "vault",
			Key:      key,
			Err:      err,
		}
	}
	if secret == nil {
		return "", &SecretNotFoundError{Key: key}
	}

	// Extract value based on KV version
	var data map[string]interface{}
	if p.config.KVVersion == 2 {
		// KV v2: secret.Data["data"]["field"]
		dataInterface, ok := secret.Data["data"]
		if !ok {
			return "", &SecretNotFoundError{Key: key}
		}
		data, ok = dataInterface.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("invalid secret data format for key: %s", key)
		}
	} else {
		// KV v1: secret.Data["field"]
		data = secret.Data
	}

	// Get field value
	value, ok := data[field]
	if !ok {
		return "", &SecretNotFoundError{Key: key}
	}

	valueStr, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("secret value is not a string: %s", key)
	}

	return valueStr, nil
}

// GetSecretMap retrieves all secrets at a given path.
func (p *VaultSecretProvider) GetSecretMap(ctx context.Context, prefix string) (map[string]string, error) {
	vaultPath := p.buildVaultPath(prefix)

	secret, err := p.client.Logical().ReadWithContext(ctx, vaultPath)
	if err != nil {
		return nil, &SecretProviderError{
			Provider: "vault",
			Key:      prefix,
			Err:      err,
		}
	}
	if secret == nil {
		return make(map[string]string), nil
	}

	// Extract all fields
	var data map[string]interface{}
	if p.config.KVVersion == 2 {
		if dataInterface, ok := secret.Data["data"]; ok {
			data, _ = dataInterface.(map[string]interface{})
		}
	} else {
		data = secret.Data
	}

	secrets := make(map[string]string)
	for k, v := range data {
		if vStr, ok := v.(string); ok {
			// Prefix the key if a path was specified
			if prefix != "" {
				k = prefix + "." + k
			}
			secrets[k] = vStr
		}
	}

	return secrets, nil
}

// Close closes the Vault client.
func (p *VaultSecretProvider) Close() error {
	// Vault client doesn't require explicit cleanup
	return nil
}

// parseKey splits a key into path and field.
// Example: "db.password" → path="db", field="password"
//
//	"api_key" → path="", field="api_key"
func (p *VaultSecretProvider) parseKey(key string) (string, string) {
	// Find the last dot to separate path from field
	idx := strings.LastIndex(key, ".")
	if idx == -1 {
		// No dot, entire key is the field name
		return "", key
	}

	return key[:idx], key[idx+1:]
}

// buildVaultPath constructs the full Vault path.
func (p *VaultSecretProvider) buildVaultPath(path string) string {
	// Build: mount_path/[data/]secret_path/path
	parts := []string{p.config.MountPath}

	// KV v2 requires /data/ in path
	if p.config.KVVersion == 2 {
		parts = append(parts, "data")
	}

	if p.config.SecretPath != "" {
		parts = append(parts, p.config.SecretPath)
	}

	if path != "" {
		parts = append(parts, path)
	}

	return strings.Join(parts, "/")
}

package secrets

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVaultSecretProvider(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := &VaultProviderConfig{
			Address:   "http://localhost:8200",
			Token:     "test-token",
			MountPath: "secret",
		}
		provider, err := NewVaultSecretProvider(config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, 2, provider.config.KVVersion) // Default to v2
	})

	t.Run("nil config", func(t *testing.T) {
		_, err := NewVaultSecretProvider(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config is required")
	})

	t.Run("missing address", func(t *testing.T) {
		config := &VaultProviderConfig{
			Token: "test-token",
		}
		_, err := NewVaultSecretProvider(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "address is required")
	})

	t.Run("missing token", func(t *testing.T) {
		config := &VaultProviderConfig{
			Address: "http://localhost:8200",
		}
		_, err := NewVaultSecretProvider(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token is required")
	})

	t.Run("invalid kv version", func(t *testing.T) {
		config := &VaultProviderConfig{
			Address:   "http://localhost:8200",
			Token:     "test-token",
			KVVersion: 3,
		}
		_, err := NewVaultSecretProvider(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid kv_version")
	})

	t.Run("defaults", func(t *testing.T) {
		config := &VaultProviderConfig{
			Address: "http://localhost:8200",
			Token:   "test-token",
		}
		provider, err := NewVaultSecretProvider(config)
		require.NoError(t, err)
		assert.Equal(t, "secret", provider.config.MountPath)
		assert.Equal(t, 2, provider.config.KVVersion)
	})
}

func TestVaultSecretProvider_ParseKey(t *testing.T) {
	provider := &VaultSecretProvider{
		config: &VaultProviderConfig{},
	}

	tests := []struct {
		name          string
		key           string
		expectedPath  string
		expectedField string
	}{
		{
			name:          "simple key",
			key:           "api_key",
			expectedPath:  "",
			expectedField: "api_key",
		},
		{
			name:          "key with one dot",
			key:           "db.password",
			expectedPath:  "db",
			expectedField: "password",
		},
		{
			name:          "key with multiple dots",
			key:           "app.db.password",
			expectedPath:  "app.db",
			expectedField: "password",
		},
		{
			name:          "key with nested path",
			key:           "production.mysql.root.password",
			expectedPath:  "production.mysql.root",
			expectedField: "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, field := provider.parseKey(tt.key)
			assert.Equal(t, tt.expectedPath, path)
			assert.Equal(t, tt.expectedField, field)
		})
	}
}

func TestVaultSecretProvider_BuildVaultPath(t *testing.T) {
	tests := []struct {
		name         string
		config       *VaultProviderConfig
		path         string
		expectedPath string
	}{
		{
			name: "KV v2 with secret path",
			config: &VaultProviderConfig{
				MountPath:  "secret",
				SecretPath: "waterflow/prod",
				KVVersion:  2,
			},
			path:         "db",
			expectedPath: "secret/data/waterflow/prod/db",
		},
		{
			name: "KV v1 with secret path",
			config: &VaultProviderConfig{
				MountPath:  "secret",
				SecretPath: "waterflow/prod",
				KVVersion:  1,
			},
			path:         "db",
			expectedPath: "secret/waterflow/prod/db",
		},
		{
			name: "KV v2 without secret path",
			config: &VaultProviderConfig{
				MountPath: "secret",
				KVVersion: 2,
			},
			path:         "api",
			expectedPath: "secret/data/api",
		},
		{
			name: "KV v2 without path parameter",
			config: &VaultProviderConfig{
				MountPath:  "secret",
				SecretPath: "app",
				KVVersion:  2,
			},
			path:         "",
			expectedPath: "secret/data/app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &VaultSecretProvider{config: tt.config}
			result := provider.buildVaultPath(tt.path)
			assert.Equal(t, tt.expectedPath, result)
		})
	}
}

func TestVaultSecretProvider_Close(t *testing.T) {
	config := &VaultProviderConfig{
		Address: "http://localhost:8200",
		Token:   "test-token",
	}
	provider, err := NewVaultSecretProvider(config)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

// Note: Integration tests with real Vault server will be in a separate file
// that requires Docker and can be skipped in CI/CD

func TestVaultSecretProvider_GetSecret_Errors(t *testing.T) {
	// Test error handling without real Vault server
	config := &VaultProviderConfig{
		Address:   "http://localhost:9999", // Invalid port
		Token:     "test-token",
		MountPath: "secret",
		KVVersion: 2,
	}
	provider, err := NewVaultSecretProvider(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("connection error wrapped", func(t *testing.T) {
		_, err := provider.GetSecret(ctx, "test_key")
		assert.Error(t, err)
		// Should be wrapped in SecretProviderError
		var providerErr *SecretProviderError
		assert.ErrorAs(t, err, &providerErr)
		assert.Equal(t, "vault", providerErr.Provider)
		assert.Equal(t, "test_key", providerErr.Key)
	})
}

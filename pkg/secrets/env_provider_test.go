package secrets

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvSecretProvider_DefaultConfig(t *testing.T) {
	provider := NewEnvSecretProvider(nil)
	assert.NotNil(t, provider)
	assert.Equal(t, "WATERFLOW_SECRET_", provider.config.Prefix)
	assert.False(t, provider.config.CaseSensitive)
}

func TestEnvSecretProvider_CustomConfig(t *testing.T) {
	config := &EnvProviderConfig{
		Prefix:        "CUSTOM_",
		CaseSensitive: true,
	}
	provider := NewEnvSecretProvider(config)
	assert.Equal(t, "CUSTOM_", provider.config.Prefix)
	assert.True(t, provider.config.CaseSensitive)
}

func TestEnvSecretProvider_GetSecret(t *testing.T) {
	// Setup environment
	_ = os.Setenv("WATERFLOW_SECRET_API_KEY", "test-key-123")
	_ = os.Setenv("WATERFLOW_SECRET_DB_PASSWORD", "pass-456")
	defer func() { _ = os.Unsetenv("WATERFLOW_SECRET_API_KEY") }()
	defer func() { _ = os.Unsetenv("WATERFLOW_SECRET_DB_PASSWORD") }()

	provider := NewEnvSecretProvider(nil)
	ctx := context.Background()

	t.Run("get existing secret", func(t *testing.T) {
		value, err := provider.GetSecret(ctx, "api_key")
		require.NoError(t, err)
		assert.Equal(t, "test-key-123", value)
	})

	t.Run("get secret with dots", func(t *testing.T) {
		value, err := provider.GetSecret(ctx, "db.password")
		require.NoError(t, err)
		assert.Equal(t, "pass-456", value)
	})

	t.Run("secret not found", func(t *testing.T) {
		_, err := provider.GetSecret(ctx, "nonexistent")
		assert.Error(t, err)
		assert.True(t, IsSecretNotFound(err))
		assert.Contains(t, err.Error(), "nonexistent")
	})
}

func TestEnvSecretProvider_GetSecret_CaseSensitive(t *testing.T) {
	_ = os.Setenv("CUSTOM_mySecret", "value1")
	defer func() { _ = os.Unsetenv("CUSTOM_mySecret") }()

	config := &EnvProviderConfig{
		Prefix:        "CUSTOM_",
		CaseSensitive: true,
	}
	provider := NewEnvSecretProvider(config)
	ctx := context.Background()

	value, err := provider.GetSecret(ctx, "mySecret")
	require.NoError(t, err)
	assert.Equal(t, "value1", value)
}

func TestEnvSecretProvider_GetSecretMap(t *testing.T) {
	// Setup environment
	_ = os.Setenv("WATERFLOW_SECRET_DB_HOST", "localhost")
	_ = os.Setenv("WATERFLOW_SECRET_DB_PORT", "5432")
	_ = os.Setenv("WATERFLOW_SECRET_DB_PASSWORD", "secret")
	_ = os.Setenv("WATERFLOW_SECRET_API_KEY", "key123")
	defer func() {
		_ = os.Unsetenv("WATERFLOW_SECRET_DB_HOST")
		_ = os.Unsetenv("WATERFLOW_SECRET_DB_PORT")
		_ = os.Unsetenv("WATERFLOW_SECRET_DB_PASSWORD")
		_ = os.Unsetenv("WATERFLOW_SECRET_API_KEY")
	}()

	provider := NewEnvSecretProvider(nil)
	ctx := context.Background()

	t.Run("get all secrets", func(t *testing.T) {
		secrets, err := provider.GetSecretMap(ctx, "")
		require.NoError(t, err)
		assert.Len(t, secrets, 4)
		assert.Equal(t, "localhost", secrets["db.host"])
		assert.Equal(t, "5432", secrets["db.port"])
		assert.Equal(t, "secret", secrets["db.password"])
		assert.Equal(t, "key123", secrets["api.key"])
	})

	t.Run("get secrets with prefix", func(t *testing.T) {
		secrets, err := provider.GetSecretMap(ctx, "db")
		require.NoError(t, err)
		assert.Len(t, secrets, 3)
		assert.Contains(t, secrets, "db.host")
		assert.Contains(t, secrets, "db.port")
		assert.Contains(t, secrets, "db.password")
		assert.NotContains(t, secrets, "api.key")
	})

	t.Run("prefix with no matches", func(t *testing.T) {
		secrets, err := provider.GetSecretMap(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, secrets)
	})
}

func TestEnvSecretProvider_KeyConversion(t *testing.T) {
	tests := []struct {
		name     string
		config   *EnvProviderConfig
		key      string
		expected string
	}{
		{
			name:     "default config",
			config:   nil,
			key:      "api_key",
			expected: "WATERFLOW_SECRET_API_KEY",
		},
		{
			name:     "with dots",
			config:   nil,
			key:      "db.password",
			expected: "WATERFLOW_SECRET_DB_PASSWORD",
		},
		{
			name:     "custom prefix",
			config:   &EnvProviderConfig{Prefix: "CUSTOM_"},
			key:      "secret",
			expected: "CUSTOM_SECRET",
		},
		{
			name: "case sensitive",
			config: &EnvProviderConfig{
				Prefix:        "TEST_",
				CaseSensitive: true,
			},
			key:      "mySecret",
			expected: "TEST_mySecret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewEnvSecretProvider(tt.config)
			result := provider.keyToEnvVar(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnvSecretProvider_EnvVarToKey(t *testing.T) {
	tests := []struct {
		name     string
		config   *EnvProviderConfig
		envVar   string
		expected string
	}{
		{
			name:     "default config",
			config:   nil,
			envVar:   "WATERFLOW_SECRET_API_KEY",
			expected: "api.key",
		},
		{
			name:     "with underscores",
			config:   nil,
			envVar:   "WATERFLOW_SECRET_DB_PASSWORD",
			expected: "db.password",
		},
		{
			name: "case sensitive",
			config: &EnvProviderConfig{
				Prefix:        "TEST_",
				CaseSensitive: true,
			},
			envVar:   "TEST_mySecret",
			expected: "mySecret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewEnvSecretProvider(tt.config)
			result := provider.envVarToKey(tt.envVar)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnvSecretProvider_Close(t *testing.T) {
	provider := NewEnvSecretProvider(nil)
	err := provider.Close()
	assert.NoError(t, err)
}

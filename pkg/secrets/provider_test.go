package secrets

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockSecretProvider(t *testing.T) {
	provider := NewMockSecretProvider(map[string]string{
		"api_key":     "secret123",
		"db_password": "pass456",
	})

	t.Run("GetSecret found", func(t *testing.T) {
		value, err := provider.GetSecret(context.Background(), "api_key")
		assert.NoError(t, err)
		assert.Equal(t, "secret123", value)
	})

	t.Run("GetSecret not found", func(t *testing.T) {
		_, err := provider.GetSecret(context.Background(), "nonexistent")
		assert.Error(t, err)
		assert.True(t, IsSecretNotFound(err))
	})

	t.Run("GetSecretMap", func(t *testing.T) {
		secrets, err := provider.GetSecretMap(context.Background(), "")
		assert.NoError(t, err)
		assert.Len(t, secrets, 2)
		assert.Equal(t, "secret123", secrets["api_key"])
	})

	t.Run("Close", func(t *testing.T) {
		err := provider.Close()
		assert.NoError(t, err)
		assert.True(t, provider.closed)
	})
}

func TestSecretProviderConfig(t *testing.T) {
	config := &SecretProviderConfig{
		ProviderType: "env",
		Env: &EnvProviderConfig{
			Prefix:        "CUSTOM_",
			CaseSensitive: true,
		},
		Cache: &CacheConfig{
			Enabled: true,
			TTL:     600,
			MaxSize: 500,
		},
	}

	assert.Equal(t, "env", config.ProviderType)
	assert.NotNil(t, config.Env)
	assert.Equal(t, "CUSTOM_", config.Env.Prefix)
	assert.True(t, config.Cache.Enabled)
}

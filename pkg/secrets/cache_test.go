package secrets

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCachedSecretProvider(t *testing.T) {
	mock := NewMockSecretProvider(map[string]string{
		"test": "value",
	})

	t.Run("with config", func(t *testing.T) {
		config := &CacheConfig{
			Enabled: true,
			TTL:     600,
			MaxSize: 500,
		}
		provider := NewCachedSecretProvider(mock, config)
		assert.NotNil(t, provider)
		assert.True(t, provider.config.Enabled)
		assert.Equal(t, 600, provider.config.TTL)
		assert.Equal(t, 500, provider.config.MaxSize)
		defer func() { _ = provider.Close() }()
	})

	t.Run("with nil config", func(t *testing.T) {
		provider := NewCachedSecretProvider(mock, nil)
		assert.NotNil(t, provider)
		assert.True(t, provider.config.Enabled)
		assert.Equal(t, 300, provider.config.TTL)
		assert.Equal(t, 1000, provider.config.MaxSize)
		defer func() { _ = provider.Close() }()
	})

	t.Run("with defaults", func(t *testing.T) {
		config := &CacheConfig{Enabled: true}
		provider := NewCachedSecretProvider(mock, config)
		assert.Equal(t, 300, provider.config.TTL)
		assert.Equal(t, 1000, provider.config.MaxSize)
		defer func() { _ = provider.Close() }()
	})
}

func TestCachedSecretProvider_GetSecret(t *testing.T) {
	mock := NewMockSecretProvider(map[string]string{
		"api_key": "secret123",
	})

	config := &CacheConfig{
		Enabled: true,
		TTL:     2, // 2 seconds
		MaxSize: 10,
	}
	provider := NewCachedSecretProvider(mock, config)
	defer func() { _ = provider.Close() }()

	ctx := context.Background()

	t.Run("first fetch from provider", func(t *testing.T) {
		value, err := provider.GetSecret(ctx, "api_key")
		require.NoError(t, err)
		assert.Equal(t, "secret123", value)
		assert.Equal(t, 1, provider.CacheSize())
	})

	t.Run("second fetch from cache", func(t *testing.T) {
		// Modify mock data to verify cache is used
		mock.secrets["api_key"] = "modified"

		value, err := provider.GetSecret(ctx, "api_key")
		require.NoError(t, err)
		assert.Equal(t, "secret123", value) // Should be cached value
	})

	t.Run("cache expiration", func(t *testing.T) {
		// Wait for TTL to expire using condition-based waiting
		require.Eventually(t, func() bool {
			value, err := provider.GetSecret(ctx, "api_key")
			return err == nil && value == "modified"
		}, 5*time.Second, 100*time.Millisecond, "cache should expire and return new value")
	})
}

func TestCachedSecretProvider_CacheDisabled(t *testing.T) {
	mock := NewMockSecretProvider(map[string]string{
		"test": "value1",
	})

	config := &CacheConfig{
		Enabled: false,
	}
	provider := NewCachedSecretProvider(mock, config)
	defer func() { _ = provider.Close() }()

	ctx := context.Background()

	value, err := provider.GetSecret(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "value1", value)
	assert.Equal(t, 0, provider.CacheSize()) // No caching

	// Change value and fetch again
	mock.secrets["test"] = "value2"
	value, err = provider.GetSecret(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "value2", value) // Always fetch fresh
}

func TestCachedSecretProvider_MaxSize(t *testing.T) {
	mock := NewMockSecretProvider(map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
	})

	config := &CacheConfig{
		Enabled: true,
		TTL:     300,
		MaxSize: 2, // Only cache 2 secrets
	}
	provider := NewCachedSecretProvider(mock, config)
	defer func() { _ = provider.Close() }()

	ctx := context.Background()

	// Fill cache
	_, _ = provider.GetSecret(ctx, "key1")
	_, _ = provider.GetSecret(ctx, "key2")
	assert.Equal(t, 2, provider.CacheSize())

	// Add third key, should evict oldest
	_, _ = provider.GetSecret(ctx, "key3")
	assert.Equal(t, 2, provider.CacheSize())
}

func TestCachedSecretProvider_SecretNotFound(t *testing.T) {
	mock := NewMockSecretProvider(map[string]string{})

	config := &CacheConfig{Enabled: true}
	provider := NewCachedSecretProvider(mock, config)
	defer func() { _ = provider.Close() }()

	ctx := context.Background()

	_, err := provider.GetSecret(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, IsSecretNotFound(err))
	assert.Equal(t, 0, provider.CacheSize()) // Errors not cached
}

func TestCachedSecretProvider_GetSecretMap(t *testing.T) {
	mock := NewMockSecretProvider(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})

	config := &CacheConfig{Enabled: true}
	provider := NewCachedSecretProvider(mock, config)
	defer func() { _ = provider.Close() }()

	ctx := context.Background()

	secrets, err := provider.GetSecretMap(ctx, "")
	require.NoError(t, err)
	assert.Len(t, secrets, 2)
	// GetSecretMap doesn't use cache
	assert.Equal(t, 0, provider.CacheSize())
}

func TestCachedSecretProvider_ClearCache(t *testing.T) {
	mock := NewMockSecretProvider(map[string]string{
		"key1": "value1",
	})

	config := &CacheConfig{Enabled: true}
	provider := NewCachedSecretProvider(mock, config)
	defer func() { _ = provider.Close() }()

	ctx := context.Background()

	_, _ = provider.GetSecret(ctx, "key1")
	assert.Equal(t, 1, provider.CacheSize())

	provider.ClearCache()
	assert.Equal(t, 0, provider.CacheSize())
}

func TestCachedSecretProvider_CleanupLoop(t *testing.T) {
	mock := NewMockSecretProvider(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})

	config := &CacheConfig{
		Enabled: true,
		TTL:     1, // 1 second
		MaxSize: 10,
	}
	provider := NewCachedSecretProvider(mock, config)

	ctx := context.Background()

	// Add secrets
	_, _ = provider.GetSecret(ctx, "key1")
	_, _ = provider.GetSecret(ctx, "key2")
	assert.Equal(t, 2, provider.CacheSize())

	// Wait for cleanup (runs every minute, but TTL is 1 second)
	// We'll manually trigger expiration check
	time.Sleep(2 * time.Second)

	// Access to trigger recheck (cleanupLoop runs in background)
	// After TTL expires, next access will fetch fresh
	value, _ := provider.GetSecret(ctx, "key1")
	assert.Equal(t, "value1", value)

	// Close should stop cleanup loop
	err := provider.Close()
	assert.NoError(t, err)
	assert.True(t, mock.closed)
}

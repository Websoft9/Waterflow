package dsl

import (
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpressionReplacer_SecretsIntegration(t *testing.T) {
	engine := NewEngine(30 * time.Second)
	replacer := NewExpressionReplacer(engine)

	t.Run("static secrets map (backward compatibility)", func(t *testing.T) {
		ctx := &EvalContext{
			Secrets: map[string]string{
				"api_key":  "static-key-123",
				"password": "static-pass-456",
			},
		}

		input := "Token: ${{ secrets.api_key }}, Password: ${{ secrets.password }}"
		result, err := replacer.Replace(input, ctx)
		require.NoError(t, err)
		assert.Equal(t, "Token: static-key-123, Password: static-pass-456", result)
	})

	t.Run("SecretProvider (dynamic)", func(t *testing.T) {
		mockProvider := secrets.NewMockSecretProvider(map[string]string{
			"api_key":     "dynamic-key-789",
			"db_password": "dynamic-pass-xyz",
		})

		ctx := &EvalContext{
			SecretProvider: mockProvider,
		}

		input := "API: ${{ secrets.api_key }}, DB: ${{ secrets.db_password }}"
		result, err := replacer.Replace(input, ctx)
		require.NoError(t, err)
		assert.Equal(t, "API: dynamic-key-789, DB: dynamic-pass-xyz", result)
	})

	t.Run("Provider with fallback to map", func(t *testing.T) {
		mockProvider := secrets.NewMockSecretProvider(map[string]string{
			"provider_key": "from-provider",
		})

		ctx := &EvalContext{
			SecretProvider: mockProvider,
			Secrets: map[string]string{
				"map_key":      "from-map",
				"provider_key": "should-not-use-this",
			},
		}

		// Provider key from provider
		result, err := replacer.Replace("${{ secrets.provider_key }}", ctx)
		require.NoError(t, err)
		assert.Equal(t, "from-provider", result)

		// Map key (not in provider) falls back to map
		result, err = replacer.Replace("${{ secrets.map_key }}", ctx)
		require.NoError(t, err)
		assert.Equal(t, "from-map", result)
	})

	t.Run("secret not found", func(t *testing.T) {
		ctx := &EvalContext{
			Secrets: map[string]string{},
		}

		_, err := replacer.Replace("${{ secrets.nonexistent }}", ctx)
		assert.Error(t, err)
		assert.True(t, secrets.IsSecretNotFound(err))
	})

	t.Run("mixed expressions", func(t *testing.T) {
		mockProvider := secrets.NewMockSecretProvider(map[string]string{
			"token": "secret-token-123",
		})

		ctx := &EvalContext{
			SecretProvider: mockProvider,
			Vars: map[string]interface{}{
				"url": "https://api.example.com",
			},
		}

		input := "curl ${{ vars.url }}/endpoint -H 'Authorization: Bearer ${{ secrets.token }}'"
		result, err := replacer.Replace(input, ctx)
		require.NoError(t, err)
		assert.Equal(t, "curl https://api.example.com/endpoint -H 'Authorization: Bearer secret-token-123'", result)
	})

	t.Run("secrets in nested structures", func(t *testing.T) {
		mockProvider := secrets.NewMockSecretProvider(map[string]string{
			"db_password": "secret-pass",
		})

		ctx := &EvalContext{
			SecretProvider: mockProvider,
		}

		input := map[string]interface{}{
			"database": map[string]interface{}{
				"host":     "localhost",
				"password": "${{ secrets.db_password }}",
			},
			"api": map[string]interface{}{
				"url": "https://api.example.com",
			},
		}

		result, err := replacer.ReplaceInMap(input, ctx)
		require.NoError(t, err)

		dbConfig := result["database"].(map[string]interface{})
		assert.Equal(t, "secret-pass", dbConfig["password"])
		assert.Equal(t, "localhost", dbConfig["host"])
	})
}

func TestEvalContext_WithSecretProvider(t *testing.T) {
	provider := secrets.NewMockSecretProvider(map[string]string{
		"test_secret": "test_value",
	})

	ctx := &EvalContext{
		SecretProvider: provider,
	}

	assert.NotNil(t, ctx.SecretProvider)

	// SecretProvider should not be exposed to expressions (tag: expr:"-")
	// This is tested implicitly by the expression engine
}

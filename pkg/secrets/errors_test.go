package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecretNotFoundError(t *testing.T) {
	err := &SecretNotFoundError{Key: "api_key"}
	assert.Equal(t, "secret not found: api_key", err.Error())
}

func TestIsSecretNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "SecretNotFoundError",
			err:      &SecretNotFoundError{Key: "test"},
			expected: true,
		},
		{
			name:     "wrapped SecretNotFoundError",
			err:      &SecretProviderError{Provider: "env", Key: "test", Err: &SecretNotFoundError{Key: "test"}},
			expected: true,
		},
		{
			name:     "other error",
			err:      assert.AnError,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsSecretNotFound(tt.err))
		})
	}
}

func TestSecretProviderError(t *testing.T) {
	innerErr := assert.AnError
	err := &SecretProviderError{
		Provider: "vault",
		Key:      "db_password",
		Err:      innerErr,
	}

	assert.Contains(t, err.Error(), "vault")
	assert.Contains(t, err.Error(), "db_password")
	assert.Equal(t, innerErr, err.Unwrap())
}

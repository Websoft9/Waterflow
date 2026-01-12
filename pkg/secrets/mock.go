package secrets

import (
	"context"
)

// MockSecretProvider is a mock implementation for testing (exported).
type MockSecretProvider struct {
	secrets map[string]string
	closed  bool
}

// NewMockSecretProvider creates a mock provider (exported for testing).
func NewMockSecretProvider(secrets map[string]string) *MockSecretProvider {
	return &MockSecretProvider{
		secrets: secrets,
		closed:  false,
	}
}

func (m *MockSecretProvider) GetSecret(ctx context.Context, key string) (string, error) {
	value, ok := m.secrets[key]
	if !ok {
		return "", &SecretNotFoundError{Key: key}
	}
	return value, nil
}

func (m *MockSecretProvider) GetSecretMap(ctx context.Context, prefix string) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range m.secrets {
		result[k] = v
	}
	return result, nil
}

func (m *MockSecretProvider) Close() error {
	m.closed = true
	return nil
}

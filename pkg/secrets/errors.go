package secrets

import (
	"errors"
	"fmt"
)

// SecretNotFoundError indicates the requested secret doesn't exist.
type SecretNotFoundError struct {
	Key string
}

func (e *SecretNotFoundError) Error() string {
	return fmt.Sprintf("secret not found: %s", e.Key)
}

// IsSecretNotFound checks if an error is SecretNotFoundError.
func IsSecretNotFound(err error) bool {
	var notFoundErr *SecretNotFoundError
	return errors.As(err, &notFoundErr)
}

// SecretProviderError wraps errors from secret providers with additional context.
type SecretProviderError struct {
	Provider string
	Key      string
	Err      error
}

func (e *SecretProviderError) Error() string {
	return fmt.Sprintf("secret provider %s failed to get %s: %v", e.Provider, e.Key, e.Err)
}

func (e *SecretProviderError) Unwrap() error {
	return e.Err
}

package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRedactor(t *testing.T) {
	t.Run("default mode", func(t *testing.T) {
		r := NewRedactor("")
		assert.NotNil(t, r)
		assert.Equal(t, "full", r.mode)
	})

	t.Run("explicit mode", func(t *testing.T) {
		r := NewRedactor("partial")
		assert.Equal(t, "partial", r.mode)
	})
}

func TestRedactor_AddSecret(t *testing.T) {
	r := NewRedactor("full")

	t.Run("add valid secret", func(t *testing.T) {
		r.AddSecret("myPassword123")
		assert.Equal(t, 1, r.Count())
	})

	t.Run("add empty secret ignored", func(t *testing.T) {
		count := r.Count()
		r.AddSecret("")
		assert.Equal(t, count, r.Count())
	})

	t.Run("add duplicate secret", func(t *testing.T) {
		r.AddSecret("duplicate")
		r.AddSecret("duplicate")
		// Map handles duplicates automatically
		assert.True(t, r.Count() >= 1)
	})
}

func TestRedactor_AddSecrets(t *testing.T) {
	r := NewRedactor("full")

	secrets := []string{"secret1", "secret2", "secret3"}
	r.AddSecrets(secrets)

	assert.Equal(t, 3, r.Count())
}

func TestRedactor_Redact_FullMode(t *testing.T) {
	r := NewRedactor("full")
	r.AddSecret("mySecretPassword")
	r.AddSecret("api-key-12345")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single secret",
			input:    "Password is mySecretPassword",
			expected: "Password is ***REDACTED***",
		},
		{
			name:     "multiple secrets",
			input:    "Use mySecretPassword and api-key-12345",
			expected: "Use ***REDACTED*** and ***REDACTED***",
		},
		{
			name:     "secret in quotes",
			input:    "Connection failed: password 'mySecretPassword' invalid",
			expected: "Connection failed: password '***REDACTED***' invalid",
		},
		{
			name:     "no secrets",
			input:    "This text has no secrets",
			expected: "This text has no secrets",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.Redact(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactor_Redact_PartialMode(t *testing.T) {
	r := NewRedactor("partial")
	r.AddSecret("mySecretPassword123")
	r.AddSecret("short") // Too short for partial

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "long secret partial",
			input:    "Password is mySecretPassword123",
			expected: "Password is myS***123", // First 3: myS, Last 3: 123
		},
		{
			name:     "short secret full",
			input:    "Key: short",
			expected: "Key: ***REDACTED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.Redact(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactor_RedactBytes(t *testing.T) {
	r := NewRedactor("full")
	r.AddSecret("secret123")

	input := []byte("Token: secret123")
	expected := []byte("Token: ***REDACTED***")

	result := r.RedactBytes(input)
	assert.Equal(t, expected, result)
}

func TestRedactor_Clear(t *testing.T) {
	r := NewRedactor("full")
	r.AddSecrets([]string{"secret1", "secret2", "secret3"})
	assert.Equal(t, 3, r.Count())

	r.Clear()
	assert.Equal(t, 0, r.Count())

	// After clear, secrets should not be redacted
	result := r.Redact("secret1 and secret2")
	assert.Equal(t, "secret1 and secret2", result)
}

func TestRedactor_ConcurrentAccess(t *testing.T) {
	r := NewRedactor("full")

	// Add secrets concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			r.AddSecret("secret" + string(rune(n)))
			_ = r.Redact("test secret" + string(rune(n)))
			_ = r.Count()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.True(t, r.Count() > 0)
}

func TestRedactor_RealWorldScenarios(t *testing.T) {
	r := NewRedactor("full")
	r.AddSecret("ghp_1234567890abcdefghijklmnop")
	r.AddSecret("sk-proj-abcdefghijklmnopqrstuvwxyz123456")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "github token in git command",
			input:    "git clone https://ghp_1234567890abcdefghijklmnop@github.com/user/repo.git",
			expected: "git clone https://***REDACTED***@github.com/user/repo.git",
		},
		{
			name:     "api key in curl command",
			input:    "curl -H 'Authorization: Bearer sk-proj-abcdefghijklmnopqrstuvwxyz123456' https://api.example.com",
			expected: "curl -H 'Authorization: Bearer ***REDACTED***' https://api.example.com",
		},
		{
			name:     "multiple occurrences",
			input:    "Token sk-proj-abcdefghijklmnopqrstuvwxyz123456 used in request. Token: sk-proj-abcdefghijklmnopqrstuvwxyz123456",
			expected: "Token ***REDACTED*** used in request. Token: ***REDACTED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.Redact(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

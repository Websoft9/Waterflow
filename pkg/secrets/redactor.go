package secrets

import (
	"strings"
	"sync"
)

// Redactor redacts secret values from text to prevent accidental leakage in logs.
//
// Usage:
//
//	redactor := NewRedactor("full")
//	redactor.AddSecret("mySecretPassword")
//	safe := redactor.Redact("Connection failed: password 'mySecretPassword' invalid")
//	// Result: "Connection failed: password '***REDACTED***' invalid"
type Redactor struct {
	secrets map[string]bool
	mu      sync.RWMutex
	mode    string // "full" or "partial"
}

// NewRedactor creates a new secret redactor.
func NewRedactor(mode string) *Redactor {
	if mode == "" {
		mode = "full"
	}
	return &Redactor{
		secrets: make(map[string]bool),
		mode:    mode,
	}
}

// AddSecret registers a secret value for redaction.
// Empty strings are ignored.
func (r *Redactor) AddSecret(value string) {
	if value == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.secrets[value] = true
}

// AddSecrets registers multiple secret values for redaction.
func (r *Redactor) AddSecrets(values []string) {
	for _, value := range values {
		r.AddSecret(value)
	}
}

// Redact replaces all registered secrets in the text.
func (r *Redactor) Redact(text string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := text
	for secret := range r.secrets {
		replacement := r.redactValue(secret)
		result = strings.ReplaceAll(result, secret, replacement)
	}

	return result
}

// RedactBytes is the same as Redact but operates on byte slices.
func (r *Redactor) RedactBytes(data []byte) []byte {
	redacted := r.Redact(string(data))
	return []byte(redacted)
}

// Clear removes all registered secrets.
func (r *Redactor) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.secrets = make(map[string]bool)
}

// Count returns the number of registered secrets.
func (r *Redactor) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.secrets)
}

// redactValue generates the redacted representation of a secret.
func (r *Redactor) redactValue(secret string) string {
	if r.mode == "partial" && len(secret) > 6 {
		// Show first 3 and last 3 characters
		return secret[:3] + "***" + secret[len(secret)-3:]
	}
	return "***REDACTED***"
}

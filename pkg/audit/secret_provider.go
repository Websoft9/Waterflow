package audit

import (
	"context"
	"time"

	"github.com/Websoft9/waterflow/pkg/secrets"
)

// AuditSecretProvider wraps a SecretProvider with audit logging.
//
// All secret access operations are logged for compliance and security monitoring.
type AuditSecretProvider struct {
	provider secrets.SecretProvider
	logger   AuditLogger
}

// NewAuditSecretProvider creates a new audited secret provider.
func NewAuditSecretProvider(provider secrets.SecretProvider, logger AuditLogger) *AuditSecretProvider {
	return &AuditSecretProvider{
		provider: provider,
		logger:   logger,
	}
}

// GetSecret retrieves a secret and logs the access.
func (p *AuditSecretProvider) GetSecret(ctx context.Context, key string) (string, error) {
	startTime := time.Now()
	value, err := p.provider.GetSecret(ctx, key)
	duration := time.Since(startTime)

	// Audit secret access
	entry := NewAuditLogEntry(EventSecretAccess, CategorySecret).
		WithResource(&ResourceContext{
			Type: "secret",
			Name: key,
		}).
		WithAction("get_secret").
		WithDetail("duration_ms", duration.Milliseconds())

	if err != nil {
		if secrets.IsSecretNotFound(err) {
			entry.WithResult(ResultNotFound).
				WithSeverity(SeverityWarn).
				WithDetail("error", "secret_not_found")
		} else {
			entry.WithResult(ResultError).
				WithSeverity(SeverityError).
				WithDetail("error", err.Error())
		}
	} else {
		entry.WithResult(ResultSuccess)
	}

	_ = p.logger.Log(ctx, entry)

	return value, err
}

// GetSecretMap retrieves multiple secrets and logs the access.
func (p *AuditSecretProvider) GetSecretMap(ctx context.Context, prefix string) (map[string]string, error) {
	startTime := time.Now()
	secrets, err := p.provider.GetSecretMap(ctx, prefix)
	duration := time.Since(startTime)

	// Audit batch secret access
	entry := NewAuditLogEntry(EventSecretList, CategorySecret).
		WithResource(&ResourceContext{
			Type: "secret",
			Name: "batch",
		}).
		WithAction("get_secret_map").
		WithDetail("prefix", prefix).
		WithDetail("retrieved_count", len(secrets)).
		WithDetail("duration_ms", duration.Milliseconds())

	if err != nil {
		entry.WithResult(ResultError).
			WithSeverity(SeverityError).
			WithDetail("error", err.Error())
	} else {
		entry.WithResult(ResultSuccess)
	}

	_ = p.logger.Log(ctx, entry)

	return secrets, err
}

// Close closes the underlying provider.
func (p *AuditSecretProvider) Close() error {
	return p.provider.Close()
}

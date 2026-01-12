# Secrets Management Guide

This guide explains how to securely manage secrets in Waterflow using the SecretProvider interface.

## Overview

Waterflow implements **zero-credential storage** - secrets are never persisted to disk. Instead, they are retrieved at runtime from external secret management systems using the `SecretProvider` interface.

**Key Features:**
- 🔐 **Zero Storage** - No secrets stored in Waterflow
- 🔐 **Runtime Injection** - Fetch from external systems on-demand
- 🔐 **Multi-Backend** - Env vars, HashiCorp Vault, AWS, GCP, Azure
- 🔐 **Automatic Redaction** - Prevent secrets from leaking into logs
- 🔐 **Caching** - Reduce external API calls with configurable TTL

## Quick Start

### 1. Environment Variables (Development)

The simplest way to provide secrets during development:

```bash
# Set secrets as environment variables
export WATERFLOW_SECRET_API_KEY=sk-1234567890
export WATERFLOW_SECRET_DB_PASSWORD=super-secret-password

# Start Waterflow
./bin/server
```

**Configuration:**
```yaml
# config.yaml
secrets:
  provider_type: env
  env:
    prefix: WATERFLOW_SECRET_  # Default prefix
    case_sensitive: false       # Convert to uppercase
```

**Usage in Workflows:**
```yaml
steps:
  - name: Deploy application
    uses: exec/shell@v1
    with:
      command: |
        deploy --api-key ${{ secrets.api_key }} \
               --db-password ${{ secrets.db_password }}
```

### 2. HashiCorp Vault (Production)

For production environments, use HashiCorp Vault:

```yaml
# config.yaml
secrets:
  provider_type: vault
  vault:
    address: https://vault.example.com:8200
    token: ${VAULT_TOKEN}          # Read from environment
    mount_path: secret             # KV mount path
    secret_path: waterflow/prod    # Base path for secrets
    kv_version: 2                  # KV engine version
  cache:
    enabled: true
    ttl: 300                       # 5 minutes
```

**Vault Setup:**
```bash
# 1. Enable KV v2 secrets engine
vault secrets enable -path=secret kv-v2

# 2. Write secrets
vault kv put secret/waterflow/prod \
  api_key=sk-1234567890 \
  db_password=super-secret

# 3. Create policy
vault policy write waterflow-policy - <<EOF
path "secret/data/waterflow/prod/*" {
  capabilities = ["read"]
}
EOF

# 4. Create token
export VAULT_TOKEN=$(vault token create -policy=waterflow-policy -format=json | jq -r '.auth.client_token')
```

## Secret Reference Syntax

Use `${{ secrets.<key> }}` syntax in any workflow parameter:

### Basic References
```yaml
# Simple key
${{ secrets.api_key }}

# Nested path (Vault)
${{ secrets.db.password }}
${{ secrets.aws.access_key }}
```

### Advanced Examples

**HTTP Headers:**
```yaml
- uses: http/request@v1
  with:
    url: https://api.example.com/deploy
    headers:
      Authorization: "Bearer ${{ secrets.api_token }}"
      X-API-Key: "${{ secrets.api_key }}"
```

**Shell Commands:**
```yaml
- uses: exec/shell@v1
  with:
    command: |
      mysql -h ${{ secrets.db.host }} \
            -u ${{ secrets.db.user }} \
            -p${{ secrets.db.password }} \
            < schema.sql
```

**JSON Payloads:**
```yaml
- uses: http/request@v1
  with:
    method: POST
    body: |
      {
        "api_key": "${{ secrets.api_key }}",
        "credentials": {
          "username": "${{ secrets.user }}",
          "password": "${{ secrets.password }}"
        }
      }
```

## Provider Implementations

### Environment Variable Provider

**Key Mapping Rules:**
- Dots → Underscores: `db.password` → `DB_PASSWORD`
- Lowercase → Uppercase (default)
- Prefix added: `api_key` → `WATERFLOW_SECRET_API_KEY`

**Example:**
```bash
export WATERFLOW_SECRET_API_KEY=value1
export WATERFLOW_SECRET_DB_PASSWORD=value2
export WATERFLOW_SECRET_AWS_ACCESS_KEY=value3
```

```yaml
secrets:
  provider_type: env
  env:
    prefix: WATERFLOW_SECRET_
    case_sensitive: false
```

### HashiCorp Vault Provider

**Path Resolution:**
- KV v2: `mount_path/data/secret_path/path`
- KV v1: `mount_path/secret_path/path`

**Example:**
```yaml
# Workflow references: ${{ secrets.db.password }}
# Vault path: secret/data/waterflow/prod/db → field "password"

secrets:
  provider_type: vault
  vault:
    address: https://vault.example.com:8200
    token: ${VAULT_TOKEN}
    mount_path: secret
    secret_path: waterflow/prod
    kv_version: 2
```

**Authentication Methods:**

1. **Token Authentication (Default):**
```yaml
vault:
  token: ${VAULT_TOKEN}
```

2. **AppRole Authentication:**
```bash
# Get role ID and secret ID
export VAULT_ROLE_ID=$(vault read -field=role_id auth/approle/role/waterflow/role-id)
export VAULT_SECRET_ID=$(vault write -field=secret_id -f auth/approle/role/waterflow/secret-id)

# Login and get token
export VAULT_TOKEN=$(vault write -field=token auth/approle/login \
  role_id=$VAULT_ROLE_ID \
  secret_id=$VAULT_SECRET_ID)
```

## Caching

Reduce latency and external API calls with caching:

```yaml
secrets:
  provider_type: vault
  cache:
    enabled: true
    ttl: 300         # Cache for 5 minutes
    max_size: 1000   # Maximum cached secrets
```

**Cache Behavior:**
- **LRU Eviction** - Least recently used secrets are evicted when cache is full
- **TTL Expiration** - Secrets expire after configured TTL
- **Background Cleanup** - Expired entries cleaned every minute
- **Thread-Safe** - Concurrent access supported

**Cache Invalidation:**
- Process restart clears cache
- TTL expiration
- Manual eviction via Close()

## Log Redaction

Secrets are automatically redacted from logs to prevent leakage:

```yaml
secrets:
  redaction:
    enabled: true
    mode: full  # or "partial"
```

**Redaction Modes:**

1. **Full (Default):**
```
Original: "Authorization: Bearer sk-1234567890"
Redacted: "Authorization: Bearer ***REDACTED***"
```

2. **Partial (Debug):**
```
Original: "Authorization: Bearer sk-1234567890"
Redacted: "Authorization: Bearer sk-***-890"
```

## Custom Provider Development

Implement the `SecretProvider` interface to add custom backends:

```go
package secrets

import "context"

type SecretProvider interface {
	GetSecret(ctx context.Context, key string) (string, error)
	GetSecretMap(ctx context.Context, prefix string) (map[string]string, error)
	Close() error
}
```

**Example: AWS Secrets Manager Provider**

```go
package secrets

import (
	"context"
	"encoding/json"
	
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type AWSSecretsManagerProvider struct {
	client     *secretsmanager.SecretsManager
	secretName string
}

func NewAWSSecretsManagerProvider(region, secretName string) (*AWSSecretsManagerProvider, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	
	return &AWSSecretsManagerProvider{
		client:     secretsmanager.New(sess),
		secretName: secretName,
	}, nil
}

func (p *AWSSecretsManagerProvider) GetSecret(ctx context.Context, key string) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(p.secretName),
	}
	
	result, err := p.client.GetSecretValueWithContext(ctx, input)
	if err != nil {
		return "", err
	}
	
	// Parse JSON secret
	var secrets map[string]string
	if err := json.Unmarshal([]byte(*result.SecretString), &secrets); err != nil {
		return "", err
	}
	
	value, ok := secrets[key]
	if !ok {
		return "", &SecretNotFoundError{Key: key}
	}
	
	return value, nil
}

func (p *AWSSecretsManagerProvider) GetSecretMap(ctx context.Context, prefix string) (map[string]string, error) {
	// Retrieve all secrets and filter by prefix
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(p.secretName),
	}
	
	result, err := p.client.GetSecretValueWithContext(ctx, input)
	if err != nil {
		return nil, err
	}
	
	var allSecrets map[string]string
	json.Unmarshal([]byte(*result.SecretString), &allSecrets)
	
	// Filter by prefix
	filtered := make(map[string]string)
	for k, v := range allSecrets {
		if strings.HasPrefix(k, prefix) {
			filtered[k] = v
		}
	}
	
	return filtered, nil
}

func (p *AWSSecretsManagerProvider) Close() error {
	return nil
}
```

## Security Best Practices

### 1. Token Management
- ✅ Store Vault tokens in environment variables, never in config files
- ✅ Use short-lived tokens with automatic renewal
- ✅ Rotate tokens regularly
- ❌ Never commit tokens to version control

### 2. Network Security
- ✅ Use HTTPS for all Vault communication
- ✅ Verify TLS certificates
- ✅ Use private networks for Vault access
- ❌ Don't expose Vault publicly

### 3. Access Control
- ✅ Use least-privilege Vault policies
- ✅ Separate secrets per environment (dev/staging/prod)
- ✅ Audit secret access logs
- ❌ Don't share secrets across environments

### 4. Cache Security
- ✅ Use short TTL in production (300s)
- ✅ Disable cache for highly sensitive secrets
- ✅ Ensure cache is cleared on shutdown
- ❌ Don't cache secrets to disk

### 5. Log Security
- ✅ Enable automatic redaction
- ✅ Use "full" mode in production
- ✅ Review logs before sharing
- ❌ Don't log raw secret values

## Troubleshooting

### Secret Not Found

**Error:**
```
failed to resolve secret 'api_key': secret not found: api_key
```

**Solutions:**
1. Check environment variable name: `WATERFLOW_SECRET_API_KEY`
2. Verify Vault path: `secret/data/waterflow/prod/<path>`
3. Check Vault permissions
4. Verify secret exists in backend

### Vault Connection Failed

**Error:**
```
failed to read secret from Vault: connection refused
```

**Solutions:**
1. Check Vault address in config
2. Verify network connectivity: `curl $VAULT_ADDR/v1/sys/health`
3. Check firewall rules
4. Verify Vault is running

### Token Permission Denied

**Error:**
```
failed to read secret from Vault: permission denied
```

**Solutions:**
1. Check Vault policy allows read access
2. Verify token hasn't expired
3. Test with `vault kv get secret/waterflow/prod/<path>`
4. Review Vault audit logs

### Cache Issues

**Stale secrets after rotation:**

1. Reduce TTL:
```yaml
cache:
  ttl: 60  # 1 minute
```

2. Disable cache:
```yaml
cache:
  enabled: false
```

3. Restart Waterflow to clear cache

## Migration Guide

### From Static Secrets Map

**Before (Old Way):**
```go
ctx := &dsl.EvalContext{
	Secrets: map[string]string{
		"api_key": "hardcoded-value",
	},
}
```

**After (New Way):**
```go
provider := secrets.NewEnvSecretProvider(&secrets.EnvProviderConfig{
	Prefix: "WATERFLOW_SECRET_",
})

ctx := &dsl.EvalContext{
	SecretProvider: provider,
}
```

**Backward Compatible (Both):**
```go
// Old static secrets still work as fallback
ctx := &dsl.EvalContext{
	Secrets: map[string]string{
		"fallback_key": "static-value",
	},
	SecretProvider: provider, // Checked first
}
```

## Examples

See `examples/secrets/` for complete examples:
- `env-example.yaml` - Environment variable secrets
- `vault-example.yaml` - HashiCorp Vault integration
- `custom-provider.go` - Custom provider implementation

## References

- [HashiCorp Vault Documentation](https://www.vaultproject.io/docs)
- [AWS Secrets Manager](https://docs.aws.amazon.com/secretsmanager/)
- [12-Factor App - Config](https://12factor.net/config)
- [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html)

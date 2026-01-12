# HTTPS/TLS Deployment Guide

This guide explains how to deploy Waterflow Server with HTTPS/TLS encryption.

## Quick Start

### 1. Generate Self-Signed Certificate (Development)

For development and testing:

```bash
./scripts/generate-self-signed-cert.sh ./certs localhost 365
```

### 2. Start with Docker Compose

```bash
docker-compose -f deployments/docker-compose-https.yaml up -d
```

### 3. Verify HTTPS

```bash
# Test HTTPS endpoint (skip certificate verification for self-signed)
curl -k https://localhost:443/health

# Test HTTP redirect
curl -L http://localhost:80/health
```

## Production Deployment

### Option 1: Let's Encrypt Certificate

**Install Certbot:**
```bash
# Ubuntu/Debian
sudo apt-get install certbot

# CentOS/RHEL
sudo yum install certbot
```

**Obtain Certificate:**
```bash
sudo certbot certonly --standalone -d waterflow.example.com
```

**Configure Waterflow:**
```yaml
# config.yaml
server:
  https:
    enabled: true
    port: 443
    cert_file: /etc/letsencrypt/live/waterflow.example.com/fullchain.pem
    key_file: /etc/letsencrypt/live/waterflow.example.com/privkey.pem
    min_tls_version: "1.2"
  http:
    enabled: true
    port: 80
    redirect_to_https: true
```

**Auto-Renewal:**
```bash
# Test renewal
sudo certbot renew --dry-run

# Add to crontab for auto-renewal
0 0 * * * certbot renew --post-hook "systemctl reload waterflow-server"
```

### Option 2: Commercial CA Certificate

1. Generate Certificate Signing Request (CSR)
2. Submit CSR to CA (DigiCert, GlobalSign, etc.)
3. Download certificate files
4. Configure as shown above

## Configuration Reference

### YAML Configuration

```yaml
server:
  # HTTP Configuration (Optional)
  http:
    enabled: true
    port: 8080
    redirect_to_https: true  # Redirect HTTP to HTTPS
  
  # HTTPS/TLS Configuration
  https:
    enabled: true
    port: 8443
    cert_file: /etc/waterflow/certs/server.crt
    key_file: /etc/waterflow/certs/server.key
    min_tls_version: "1.2"  # Options: 1.0, 1.1, 1.2, 1.3
```

### Environment Variables

```bash
# HTTPS Configuration
export WATERFLOW_HTTPS_ENABLED=true
export WATERFLOW_HTTPS_PORT=8443
export WATERFLOW_HTTPS_CERT_FILE=/etc/waterflow/certs/server.crt
export WATERFLOW_HTTPS_KEY_FILE=/etc/waterflow/certs/server.key
export WATERFLOW_HTTPS_MIN_TLS_VERSION=1.2

# HTTP Redirect (Optional)
export WATERFLOW_HTTP_ENABLED=true
export WATERFLOW_HTTP_PORT=8080
export WATERFLOW_HTTP_REDIRECT_TO_HTTPS=true
```

### Backward Compatibility

Old configuration format is still supported:

```yaml
server:
  tls_cert_file: /etc/waterflow/certs/server.crt
  tls_key_file: /etc/waterflow/certs/server.key
```

Environment variables:
```bash
export WATERFLOW_SERVER_TLS_CERT_FILE=/etc/waterflow/certs/server.crt
export WATERFLOW_SERVER_TLS_KEY_FILE=/etc/waterflow/certs/server.key
```

**Note:** New configuration format is recommended for better features.

## Security Best Practices

### TLS Version Selection

- **TLS 1.2** (Recommended): Good balance of security and compatibility
- **TLS 1.3**: Maximum security, requires recent clients
- **TLS 1.0/1.1**: Deprecated, use only for legacy systems

### Certificate Management

1. **Private Key Security**
   ```bash
   chmod 600 /etc/waterflow/certs/server.key
   chown waterflow:waterflow /etc/waterflow/certs/server.key
   ```

2. **Certificate Expiration Monitoring**
   - Set up alerts 30 days before expiration
   - Use automated renewal (Let's Encrypt)

3. **Certificate Validation**
   ```bash
   # Check certificate expiration
   openssl x509 -in /etc/waterflow/certs/server.crt -noout -enddate
   
   # Verify certificate and key match
   openssl x509 -in /etc/waterflow/certs/server.crt -noout -modulus | md5sum
   openssl rsa -in /etc/waterflow/certs/server.key -noout -modulus | md5sum
   ```

### Production Checklist

- [ ] Use certificates from trusted CA (not self-signed)
- [ ] Force minimum TLS 1.2 or higher
- [ ] Enable HTTP → HTTPS redirect
- [ ] Configure firewall (allow 443, block 8080 if redirect enabled)
- [ ] Set up certificate auto-renewal
- [ ] Monitor certificate expiration
- [ ] Restrict private key file permissions (600)
- [ ] Back up certificate and key securely

## Troubleshooting

### Certificate Errors

**Problem:** `certificate file not found`
```
Solution: Verify file path and permissions
ls -l /etc/waterflow/certs/server.crt
```

**Problem:** `failed to load TLS certificate: x509: malformed certificate`
```
Solution: Ensure certificate is in PEM format
openssl x509 -in /etc/waterflow/certs/server.crt -text -noout
```

### Connection Errors

**Problem:** `TLS handshake error`
```
Solution 1: Check client TLS version (must be >= server min_tls_version)
Solution 2: Verify certificate is valid and not expired
openssl s_client -connect localhost:8443 -tls1_2
```

**Problem:** HTTP redirect not working
```
Solution: Ensure http.enabled=true and http.redirect_to_https=true
Check HTTP port is accessible (not blocked by firewall)
```

### Testing TLS Configuration

```bash
# Test TLS 1.2 connection
openssl s_client -connect localhost:8443 -tls1_2

# Test TLS 1.3 connection
openssl s_client -connect localhost:8443 -tls1_3

# Test cipher suites
nmap --script ssl-enum-ciphers -p 8443 localhost

# Scan for vulnerabilities
testssl.sh https://localhost:8443
```

## Client Configuration

### CLI

```bash
# HTTPS with valid certificate
waterflow-cli submit workflow.yaml --server https://waterflow.example.com

# HTTPS with self-signed certificate (development)
waterflow-cli submit workflow.yaml --server https://localhost:8443 --insecure
```

### Go SDK

```go
import (
    "crypto/tls"
    "net/http"
    "github.com/Websoft9/waterflow/pkg/sdk"
)

// Production (valid certificate)
client, err := sdk.NewClient("https://waterflow.example.com")

// Development (self-signed certificate)
httpClient := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: true,
        },
    },
}
client, err := sdk.NewClientWithHTTP("https://localhost:8443", httpClient)
```

## Migration Guide

### From HTTP to HTTPS

1. **Generate/Obtain Certificates**
2. **Update Configuration**
   ```yaml
   server:
     https:
       enabled: true
       cert_file: /path/to/cert.pem
       key_file: /path/to/key.pem
     http:
       redirect_to_https: true  # Redirect existing clients
   ```
3. **Restart Server**
4. **Update Clients** (use https:// URLs)
5. **Disable HTTP** (after all clients updated)
   ```yaml
   server:
     http:
       enabled: false
   ```

### From Old Config Format to New

Old:
```yaml
server:
  tls_cert_file: /path/to/cert.pem
  tls_key_file: /path/to/key.pem
```

New (more features):
```yaml
server:
  https:
    enabled: true
    port: 8443
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem
    min_tls_version: "1.2"
  http:
    enabled: true
    redirect_to_https: true
```

**Note:** Old format continues to work for backward compatibility.

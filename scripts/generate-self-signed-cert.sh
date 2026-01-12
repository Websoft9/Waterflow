#!/bin/bash
# Generate self-signed TLS certificate for development/testing
# Usage: ./generate-self-signed-cert.sh [cert_dir] [domain] [days]

set -e

CERT_DIR=${1:-./certs}
DOMAIN=${2:-localhost}
DAYS=${3:-365}

# Create certificate directory
mkdir -p "$CERT_DIR"

echo "========================================="
echo "Generating Self-Signed TLS Certificate"
echo "========================================="
echo "Domain:      $DOMAIN"
echo "Output Dir:  $CERT_DIR"
echo "Valid Days:  $DAYS"
echo ""

# Generate self-signed certificate
openssl req -x509 -newkey rsa:4096 -nodes \
  -keyout "$CERT_DIR/server.key" \
  -out "$CERT_DIR/server.crt" \
  -days "$DAYS" \
  -subj "/CN=$DOMAIN" \
  -addext "subjectAltName=DNS:$DOMAIN,DNS:localhost,DNS:*.localhost,IP:127.0.0.1,IP:::1"

# Set proper permissions
chmod 600 "$CERT_DIR/server.key"
chmod 644 "$CERT_DIR/server.crt"

echo ""
echo "✅ Certificate generated successfully!"
echo ""
echo "Files created:"
echo "  Certificate: $CERT_DIR/server.crt"
echo "  Private Key: $CERT_DIR/server.key"
echo ""
echo "========================================="
echo "Configuration Example"
echo "========================================="
echo ""
echo "YAML Configuration (config.yaml):"
echo "  server:"
echo "    https:"
echo "      enabled: true"
echo "      port: 8443"
echo "      cert_file: $CERT_DIR/server.crt"
echo "      key_file: $CERT_DIR/server.key"
echo "      min_tls_version: \"1.2\""
echo ""
echo "Environment Variables:"
echo "  export WATERFLOW_HTTPS_ENABLED=true"
echo "  export WATERFLOW_HTTPS_PORT=8443"
echo "  export WATERFLOW_HTTPS_CERT_FILE=$CERT_DIR/server.crt"
echo "  export WATERFLOW_HTTPS_KEY_FILE=$CERT_DIR/server.key"
echo ""
echo "========================================="
echo "Testing HTTPS"
echo "========================================="
echo ""
echo "Start the server:"
echo "  ./bin/server --config config.yaml"
echo ""
echo "Test with curl (skip certificate verification):"
echo "  curl -k https://localhost:8443/health"
echo ""
echo "Note: Self-signed certificates will show security warnings"
echo "      in browsers. Use -k/--insecure flag with curl."
echo ""

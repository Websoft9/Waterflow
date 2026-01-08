#!/bin/bash
# Install performance testing tools
# Optional: Only needed for detailed percentile analysis

set -e

echo "=== Installing Performance Testing Tools ==="
echo ""

# hey - HTTP load generator
if ! command -v hey &> /dev/null; then
    echo "Installing hey..."
    go install github.com/rakyll/hey@latest
    echo "✅ hey installed"
else
    echo "✅ hey already installed"
fi

# vegeta - HTTP load testing tool
if ! command -v vegeta &> /dev/null; then
    echo "Installing vegeta..."
    go install github.com/tsenart/vegeta@latest
    echo "✅ vegeta installed"
else
    echo "✅ vegeta already installed"
fi

# Verify
echo ""
echo "=== Verification ==="
hey -h > /dev/null 2>&1 && echo "✅ hey working" || echo "❌ hey not found"
vegeta -h > /dev/null 2>&1 && echo "✅ vegeta working" || echo "❌ vegeta not found"

echo ""
echo "Done! Tools installed to \$(go env GOPATH)/bin"
echo "Make sure \$(go env GOPATH)/bin is in your PATH"

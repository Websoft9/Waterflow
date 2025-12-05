#!/bin/bash

# Waterflow Development Setup Script
# This script sets up the development environment for Waterflow

set -e

echo "🚀 Setting up Waterflow development environment..."

# Check prerequisites
command -v go >/dev/null 2>&1 || { echo "❌ Go is required but not installed. Please install Go 1.21+"; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "❌ Docker is required but not installed. Please install Docker"; exit 1; }

# Verify Go version
GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | cut -d'o' -f2)
REQUIRED_VERSION="1.21"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Go version $GO_VERSION is too old. Please upgrade to Go $REQUIRED_VERSION or later"
    exit 1
fi

echo "✅ Prerequisites check passed"

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod download
go mod tidy

# Install development tools
echo "🔧 Installing development tools..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/cosmtrek/air@latest
go install github.com/go-delve/delve/cmd/dlv@latest

# Install Node.js dependencies (if package.json exists)
if [ -f "package.json" ]; then
    echo "📦 Installing Node.js dependencies..."
    npm install
fi

# Install Python dependencies (if requirements.txt exists)
if [ -f "requirements.txt" ]; then
    echo "📦 Installing Python dependencies..."
    pip install -r requirements.txt
fi

# Create necessary directories
echo "📁 Creating development directories..."
mkdir -p bin/
mkdir -p test/results/
mkdir -p .tmp/

# Set up pre-commit hooks (if git is initialized)
if [ -d ".git" ]; then
    echo "🔗 Setting up git hooks..."
    # Add pre-commit hook for linting
    cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
echo "Running pre-commit checks..."

# Run linting
if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run
fi

# Run tests
go test ./...

echo "Pre-commit checks completed"
EOF
    chmod +x .git/hooks/pre-commit
fi

# Create default configuration
if [ ! -f "config/default.yaml" ]; then
    echo "⚙️ Creating default configuration..."
    mkdir -p config/
    cat > config/default.yaml << 'EOF'
# Default Waterflow Configuration
server:
  host: localhost
  port: 8080
  tls:
    enabled: false

logging:
  level: info
  format: json

database:
  type: sqlite
  path: ~/.waterflow/waterflow.db

workflows:
  directory: ./workflows
  max_concurrent: 10

containers:
  runtime: docker
  default_image: alpine:latest
  pull_policy: if-not-present
EOF
fi

# Build initial binary
echo "🔨 Building initial binary..."
go build -o bin/waterflow ./cmd/waterflow

echo ""
echo "🎉 Development environment setup complete!"
echo ""
echo "Next steps:"
echo "1. Run 'make test' to execute tests"
echo "2. Run 'make lint' to check code quality"
echo "3. Run './bin/waterflow version' to verify installation"
echo "4. Start developing with 'air' for hot reloading:"
echo "   air"
echo ""
echo "Happy coding! 🚀"
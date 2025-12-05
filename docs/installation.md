# Installation Guide

This guide will help you install and set up Waterflow on your system.

## 📋 Prerequisites

Before installing Waterflow, ensure your system meets these requirements:

### System Requirements

- **Operating System**: Linux, macOS, or Windows (via WSL2)
- **CPU**: 2+ cores recommended
- **Memory**: 4GB RAM minimum, 8GB recommended
- **Storage**: 1GB free space for installation and data

### Software Dependencies

- **Docker**: 20.10+ (for containerized workflows)
- **Git**: 2.30+ (for cloning and version control)
- **kubectl**: 1.24+ (for Kubernetes integration, optional)

### Optional Dependencies

- **Helm**: 3.0+ (for Kubernetes deployments)
- **Terraform**: 1.0+ (for infrastructure provisioning)
- **AWS CLI**: For AWS integrations
- **Azure CLI**: For Azure integrations
- **gcloud**: For GCP integrations

## 🚀 Installation Methods

### Method 1: Binary Installation (Recommended)

#### Linux/macOS

```bash
# Download the latest release
curl -L https://github.com/Websoft9/Waterflow/releases/latest/download/waterflow-linux-amd64.tar.gz -o waterflow.tar.gz

# Extract the archive
tar -xzf waterflow.tar.gz

# Move to PATH (optional)
sudo mv waterflow /usr/local/bin/

# Verify installation
waterflow version
```

#### Windows

```powershell
# Download the latest release
Invoke-WebRequest -Uri "https://github.com/Websoft9/Waterflow/releases/latest/download/waterflow-windows-amd64.zip" -OutFile "waterflow.zip"

# Extract the archive
Expand-Archive -Path "waterflow.zip" -DestinationPath "."

# Add to PATH (optional)
# Move waterflow.exe to a directory in your PATH

# Verify installation
waterflow version
```

### Method 2: Docker Installation

```bash
# Pull the official Docker image
docker pull websoft9/waterflow:latest

# Run Waterflow in a container
docker run -it --rm websoft9/waterflow:latest version

# Or mount a volume for persistent data
docker run -it -v $(pwd):/workspace websoft9/waterflow:latest
```

### Method 3: Build from Source

```bash
# Clone the repository
git clone https://github.com/Websoft9/Waterflow.git
cd Waterflow

# Build the project (choose your language)
make build
# or
npm run build
# or
python setup.py build
# or
go build ./cmd/waterflow

# Install locally
make install
# or
npm install -g .
# or
pip install .
# or
go install ./cmd/waterflow
```

## ⚙️ Configuration

### Basic Configuration

Create a configuration file at `~/.waterflow/config.yaml`:

```yaml
# Waterflow Configuration
apiVersion: v1
kind: Config

# Server settings
server:
  host: localhost
  port: 8080
  tls:
    enabled: false

# Database settings (for state persistence)
database:
  type: sqlite  # or postgresql, mysql
  path: ~/.waterflow/waterflow.db

# Logging configuration
logging:
  level: info
  format: json
  output: stderr

# Plugin directories
plugins:
  - ~/.waterflow/plugins
  - /usr/local/lib/waterflow/plugins

# Security settings
security:
  jwt_secret: "your-secret-key"
  rbac_enabled: true
```

### Environment Variables

You can also configure Waterflow using environment variables:

```bash
# Server configuration
export WATERFLOW_SERVER_HOST=0.0.0.0
export WATERFLOW_SERVER_PORT=8080

# Database configuration
export WATERFLOW_DATABASE_URL="postgresql://user:pass@localhost/waterflow"

# Logging
export WATERFLOW_LOG_LEVEL=debug

# Security
export WATERFLOW_JWT_SECRET="your-secret-key"
```

## 🔧 Post-Installation Setup

### 1. Initialize Waterflow

```bash
# Initialize the workspace
waterflow init

# This creates:
# - Configuration directory (~/.waterflow/)
# - Default workspace structure
# - Example workflow files
```

### 2. Verify Installation

```bash
# Check version
waterflow version

# List available commands
waterflow --help

# Test basic functionality
waterflow validate --file examples/hello-world.yaml
```

### 3. Set Up Auto-completion (Optional)

#### Bash

```bash
# Add to ~/.bashrc
echo 'source <(waterflow completion bash)' >> ~/.bashrc
source ~/.bashrc
```

#### Zsh

```bash
# Add to ~/.zshrc
echo 'source <(waterflow completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

#### Fish

```fish
# Add to ~/.config/fish/config.fish
waterflow completion fish > ~/.config/fish/completions/waterflow.fish
```

## 🐳 Docker Compose Setup

For development or testing, use Docker Compose:

```yaml
# docker-compose.yml
version: '3.8'

services:
  waterflow:
    image: websoft9/waterflow:latest
    ports:
      - "8080:8080"
    volumes:
      - ./workflows:/workflows
      - ./config:/config
    environment:
      - WATERFLOW_CONFIG=/config/config.yaml
    command: server

  # Optional: PostgreSQL for persistence
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: waterflow
      POSTGRES_USER: waterflow
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

```bash
# Start the services
docker-compose up -d

# Access Waterflow
curl http://localhost:8080/health
```

## ☸️ Kubernetes Installation

### Using Helm (Recommended)

```bash
# Add the Waterflow Helm repository
helm repo add waterflow https://charts.websoft9.com/
helm repo update

# Install Waterflow
helm install waterflow waterflow/waterflow \
  --namespace waterflow-system \
  --create-namespace \
  --set image.tag=latest

# Verify installation
kubectl get pods -n waterflow-system
```

### Manual Kubernetes Deployment

```bash
# Apply the manifests
kubectl apply -f https://raw.githubusercontent.com/Websoft9/Waterflow/main/deploy/kubernetes/

# Check deployment status
kubectl get deployments -n waterflow-system
kubectl get services -n waterflow-system
```

## 🔄 Upgrading

### Binary Upgrade

```bash
# Download new version
curl -L https://github.com/Websoft9/Waterflow/releases/latest/download/waterflow-linux-amd64.tar.gz -o waterflow-new.tar.gz

# Stop any running instances
waterflow stop  # if running as service

# Backup configuration
cp ~/.waterflow/config.yaml ~/.waterflow/config.yaml.backup

# Install new version
tar -xzf waterflow-new.tar.gz
sudo mv waterflow /usr/local/bin/

# Restart service
waterflow start
```

### Docker Upgrade

```bash
# Pull new image
docker pull websoft9/waterflow:latest

# Restart containers
docker-compose down
docker-compose up -d
```

### Helm Upgrade

```bash
# Update Helm repository
helm repo update

# Upgrade release
helm upgrade waterflow waterflow/waterflow \
  --namespace waterflow-system \
  --set image.tag=latest
```

## 🐛 Troubleshooting

### Common Issues

#### Permission Denied

```bash
# Fix executable permissions
chmod +x /usr/local/bin/waterflow

# Or run with sudo if needed
sudo waterflow
```

#### Port Already in Use

```bash
# Find process using port 8080
lsof -i :8080

# Kill the process or change port
waterflow server --port 8081
```

#### Configuration Errors

```bash
# Validate configuration
waterflow config validate

# Generate default config
waterflow config init
```

### Getting Help

- Check the [troubleshooting guide](troubleshooting.md)
- Search [GitHub Issues](https://github.com/Websoft9/Waterflow/issues)
- Join [GitHub Discussions](https://github.com/Websoft9/Waterflow/discussions)

## 📞 Support

- **Documentation**: [docs.websoft9.com/waterflow](https://docs.websoft9.com/waterflow)
- **Community**: [GitHub Discussions](https://github.com/Websoft9/Waterflow/discussions)
- **Issues**: [GitHub Issues](https://github.com/Websoft9/Waterflow/issues)

---

*Last updated: December 5, 2025*
#!/bin/bash
#
# Waterflow Installation Script
# 
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Websoft9/waterflow/main/scripts/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/Websoft9/waterflow/main/scripts/install.sh | bash -s -- [OPTIONS]
#
# Options:
#   -v, --version VERSION   Install specific version (default: latest)
#   -c, --component COMP    Component to install: server, agent, cli, all (default: cli)
#   -d, --dir PATH          Installation directory (default: /usr/local/bin)
#   -h, --help              Show this help message
#
# Examples:
#   # Install latest CLI
#   curl -fsSL .../install.sh | bash
#
#   # Install specific version
#   curl -fsSL .../install.sh | bash -s -- -v v0.1.0
#
#   # Install server to custom path
#   curl -fsSL .../install.sh | bash -s -- -c server -d /opt/waterflow/bin
#
#   # Install all components
#   curl -fsSL .../install.sh | bash -s -- -c all

set -e

# Configuration
GITHUB_REPO="Websoft9/waterflow"
GITHUB_API="https://api.github.com/repos/${GITHUB_REPO}"
GITHUB_RELEASES="https://github.com/${GITHUB_REPO}/releases"

# Default values
VERSION="latest"
COMPONENT="cli"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Show help
show_help() {
    cat << EOF
Waterflow Installation Script

Usage:
    install.sh [OPTIONS]

Options:
    -v, --version VERSION   Install specific version (default: latest)
    -c, --component COMP    Component to install: server, agent, cli, all (default: cli)
    -d, --dir PATH          Installation directory (default: /usr/local/bin)
    -h, --help              Show this help message

Components:
    cli       Command-line interface tool
    server    Waterflow server (workflow orchestrator)
    agent     Waterflow agent (task executor)
    all       Install all components

Examples:
    # Install latest CLI
    ./install.sh

    # Install specific version
    ./install.sh -v v0.1.0

    # Install server to custom path
    ./install.sh -c server -d /opt/waterflow/bin

    # Install all components
    ./install.sh -c all

EOF
    exit 0
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -c|--component)
                COMPONENT="$2"
                shift 2
                ;;
            -d|--dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                ;;
        esac
    done
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    # Normalize OS
    case "$OS" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        mingw*|msys*|cygwin*)
            OS="windows"
            ;;
        *)
            log_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac
    
    # Normalize architecture
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    log_info "Detected platform: ${OS}/${ARCH}"
    
    # Warn about unsupported platform combinations
    if [[ "$OS" == "windows" && "$ARCH" == "arm64" ]]; then
        log_warn "Windows arm64 binaries are not available. Falling back to amd64."
        log_warn "Consider using WSL2 for better arm64 support."
        ARCH="amd64"
    fi
}

# Get latest version from GitHub
get_latest_version() {
    if [[ "$VERSION" == "latest" ]]; then
        log_info "Fetching latest version..."
        VERSION=$(curl -fsSL "${GITHUB_API}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        
        if [[ -z "$VERSION" ]]; then
            log_error "Failed to fetch latest version"
            exit 1
        fi
        
        log_info "Latest version: ${VERSION}"
    fi
}

# Build download URL
get_download_url() {
    local component=$1
    local binary_name="waterflow-${component}-${OS}-${ARCH}"
    
    # Add .exe extension for Windows
    if [[ "$OS" == "windows" ]]; then
        binary_name="${binary_name}.exe"
    fi
    
    echo "${GITHUB_RELEASES}/download/${VERSION}/${binary_name}"
}

# Download and install a component
install_component() {
    local component=$1
    local url=$(get_download_url "$component")
    local binary_name="waterflow-${component}"
    
    # Add .exe extension for Windows
    if [[ "$OS" == "windows" ]]; then
        binary_name="${binary_name}.exe"
    fi
    
    local target_path="${INSTALL_DIR}/${binary_name}"
    
    log_info "Downloading ${component}..."
    log_info "URL: ${url}"
    
    # Create temp file
    local tmp_file=$(mktemp)
    
    # Download
    if ! curl -fsSL -o "$tmp_file" "$url"; then
        log_error "Failed to download ${component}"
        rm -f "$tmp_file"
        return 1
    fi
    
    # Create install directory if needed
    if [[ ! -d "$INSTALL_DIR" ]]; then
        log_info "Creating directory: ${INSTALL_DIR}"
        sudo mkdir -p "$INSTALL_DIR" 2>/dev/null || mkdir -p "$INSTALL_DIR"
    fi
    
    # Install binary
    log_info "Installing to ${target_path}..."
    
    if [[ -w "$INSTALL_DIR" ]]; then
        mv "$tmp_file" "$target_path"
        chmod +x "$target_path"
    else
        sudo mv "$tmp_file" "$target_path"
        sudo chmod +x "$target_path"
    fi
    
    # Verify installation
    if [[ -x "$target_path" ]]; then
        log_success "${component} installed successfully!"
        
        # Show version if possible
        if [[ "$component" == "cli" ]]; then
            "${target_path}" version 2>/dev/null || true
        else
            "${target_path}" --version 2>/dev/null || true
        fi
    else
        log_error "Installation verification failed for ${component}"
        return 1
    fi
}

# Verify checksums
verify_checksum() {
    local component=$1
    local binary_path=$2
    
    log_info "Verifying checksum for ${component}..."
    
    # Download checksums file
    local checksums_url="${GITHUB_RELEASES}/download/${VERSION}/checksums.txt"
    local checksums=$(curl -fsSL "$checksums_url" 2>/dev/null)
    
    if [[ -z "$checksums" ]]; then
        log_warn "Could not download checksums.txt, skipping verification"
        return 0
    fi
    
    # Get expected checksum
    local binary_name="waterflow-${component}-${OS}-${ARCH}"
    if [[ "$OS" == "windows" ]]; then
        binary_name="${binary_name}.exe"
    fi
    
    local expected=$(echo "$checksums" | grep "$binary_name" | awk '{print $1}')
    
    if [[ -z "$expected" ]]; then
        log_warn "Checksum not found for ${binary_name}, skipping verification"
        return 0
    fi
    
    # Calculate actual checksum
    local actual
    if command -v sha256sum &>/dev/null; then
        actual=$(sha256sum "$binary_path" | awk '{print $1}')
    elif command -v shasum &>/dev/null; then
        actual=$(shasum -a 256 "$binary_path" | awk '{print $1}')
    else
        log_warn "sha256sum/shasum not found, skipping verification"
        return 0
    fi
    
    # Compare
    if [[ "$expected" == "$actual" ]]; then
        log_success "Checksum verified: ${actual}"
        return 0
    else
        log_error "Checksum mismatch!"
        log_error "Expected: ${expected}"
        log_error "Actual:   ${actual}"
        return 1
    fi
}

# Main installation logic
main() {
    echo ""
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║                    Waterflow Installer                        ║"
    echo "║           Declarative Workflow Orchestration Engine           ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo ""
    
    parse_args "$@"
    detect_platform
    get_latest_version
    
    log_info "Version: ${VERSION}"
    log_info "Component: ${COMPONENT}"
    log_info "Install directory: ${INSTALL_DIR}"
    echo ""
    
    # Install requested components
    case "$COMPONENT" in
        cli)
            install_component "cli"
            ;;
        server)
            install_component "server"
            ;;
        agent)
            install_component "agent"
            ;;
        all)
            install_component "server"
            install_component "agent"
            install_component "cli"
            ;;
        *)
            log_error "Unknown component: ${COMPONENT}"
            exit 1
            ;;
    esac
    
    echo ""
    log_success "Installation complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Verify installation: waterflow version (or waterflow-server --version)"
    echo "  2. Read the documentation: https://github.com/${GITHUB_REPO}#readme"
    echo "  3. Quick start: https://github.com/${GITHUB_REPO}/blob/main/docs/quick-start.md"
    echo ""
}

# Run main
main "$@"

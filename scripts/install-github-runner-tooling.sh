#!/bin/bash

# Quiz Application GitHub Runner Tooling Installer
# ================================================
# Minimal installer for GitHub Actions runner environment.
# This script handles any custom tooling that can't be installed via
# standard GitHub Actions setup actions.

set -euo pipefail

# Colors for output
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

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Verify installations
verify_installations() {
    log_info "Verifying installations..."

    # Check Go
    if command_exists go; then
        local go_version=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
        log_success "Go: $(go version)"
        if [[ "$(printf '%s\n' "1.25" "$go_version" | sort -V | head -n1)" != "1.25" ]]; then
            log_warning "Go version $go_version may not meet requirements (>= 1.25)"
        fi
    else
        log_error "Go not found"
    fi

    # Check Node.js
    if command_exists node; then
        log_success "Node.js: $(node --version)"
    else
        log_error "Node.js not found"
    fi

    # Check npm
    if command_exists npm; then
        log_success "npm: $(npm --version)"
    else
        log_error "npm not found"
    fi

    # Check Task
    if command_exists task; then
        log_success "Task: $(task --version)"
    else
        log_error "Task not found"
    fi

    # Check Docker
    if command_exists docker; then
        log_success "Docker: $(docker --version)"
    else
        log_error "Docker not found"
    fi

    # Check Go tools
    local go_tools=("oapi-codegen" "goimports")
    for tool in "${go_tools[@]}"; do
        if command_exists "$tool"; then
            log_success "Go tool $tool: $(command -v "$tool")"
        else
            log_error "Go tool $tool: NOT FOUND"
        fi
    done

    # Check Node.js tools
    local node_tools=("orval" "vite" "tsc")
    for tool in "${node_tools[@]}"; do
        if command_exists "$tool"; then
            log_success "Node.js tool $tool: $(command -v "$tool")"
        else
            log_error "Node.js tool $tool: NOT FOUND"
        fi
    done
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect OS (simplified for GitHub Actions)
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS="linux"
    else
        OS="linux"  # Default for GitHub Actions runners
    fi

    log_info "Detected OS: $OS"
}

# Install Go development tools
install_go_tools() {
    log_info "Installing Go development tools..."

    # Ensure Go bin directory is in PATH for current session
    export PATH=$PATH:$(go env GOPATH)/bin

    # Install oapi-codegen
    if ! command_exists oapi-codegen; then
        log_info "Installing oapi-codegen..."
        go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
        log_success "oapi-codegen installed successfully"
    else
        log_info "oapi-codegen already installed"
    fi

    # Install goimports
    if ! command_exists goimports; then
        log_info "Installing goimports..."
        go install golang.org/x/tools/cmd/goimports@latest
        log_success "goimports installed successfully"
    else
        log_info "goimports already installed"
    fi
}

# Install Node.js development tools
install_node_tools() {
    log_info "Installing Node.js development tools..."

    # Install global npm packages
    local packages=("orval" "vite")

    for package in "${packages[@]}"; do
        if ! command_exists "$package"; then
            log_info "Installing $package..."
            if [[ "$OS" == "linux" ]]; then
                sudo npm install -g "$package"
            else
                # On macOS, try without sudo first, fall back to sudo if needed
                if ! npm install -g "$package" 2>/dev/null; then
                    log_warning "npm install failed without sudo, trying with sudo..."
                    sudo npm install -g "$package"
                fi
            fi
            log_success "$package installed successfully"
        else
            log_info "$package already installed"
        fi
    done

    # Install TypeScript separately (provides tsc command)
    if ! command_exists tsc; then
        log_info "Installing TypeScript (tsc)..."
        if [[ "$OS" == "linux" ]]; then
            sudo npm install -g typescript
        else
            # On macOS, try without sudo first, fall back to sudo if needed
            if ! npm install -g typescript 2>/dev/null; then
                log_warning "npm install failed without sudo, trying with sudo..."
                sudo npm install -g typescript
            fi
        fi
        log_success "TypeScript (tsc) installed successfully"
    else
        log_info "TypeScript (tsc) already installed"
    fi
}

# Install any additional tools if needed
install_additional_tools() {
    # Install Task if not already installed
    if ! command_exists task; then
        log_info "Installing Task..."
        sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
        log_success "Task installed successfully"
    else
        log_info "Task already installed: $(task --version)"
    fi

    # Install Go tools
    install_go_tools

    # Install Node.js tools
    install_node_tools

    # Add any other custom installations here if needed
    # This is where you would put any tools that can't be installed
    # via standard GitHub Actions setup actions
}

# Main installation function
main() {
    log_info "Starting GitHub Runner tooling setup..."

    # Detect OS
    detect_os

    # Verify all expected tools are available
    verify_installations

    # Install any additional custom tools
    install_additional_tools

    log_success "GitHub Runner tooling setup completed!"
}

# Run main function
main "$@"

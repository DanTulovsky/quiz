#!/bin/bash

# Quiz Application Tooling Installer
# ===================================
# This script installs all required tools for the AI-powered quiz application.
# It is idempotent and handles errors gracefully.
# Supports macOS and Linux.
#
# Usage:
#   ./scripts/install-tooling.sh          # Normal installation
#   ./scripts/install-tooling.sh --dry-run # Show what would be installed

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Dry run mode
DRY_RUN=false

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

log_dry_run() {
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}[DRY-RUN]${NC} $1"
    fi
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Fix Go tools compatibility
fix_go_tools() {
    log_info "Fixing Go tools compatibility..."

    if ! command_exists go; then
        log_error "Go is not installed. Please install Go first."
        return 1
    fi

    local go_version=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
    log_info "Current Go version: $go_version"

    # Reinstall all Go tools
    local tools=("oapi-codegen" "deadcode" "golangci-lint" "revive" "goimports")

    for tool in "${tools[@]}"; do
        log_info "Reinstalling $tool..."
        case $tool in
        "oapi-codegen")
            go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
            ;;
        "deadcode")
            go install golang.org/x/tools/cmd/deadcode@latest
            ;;
        "golangci-lint")
            go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.1
            ;;
        "revive")
            go install github.com/mgechev/revive@latest
            ;;
        "goimports")
            go install golang.org/x/tools/cmd/goimports@latest
            ;;
        esac
    done

    log_success "Go tools reinstalled successfully!"
    log_info "All Go tools should now be compatible with Go $go_version"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
    --dry-run)
        DRY_RUN=true
        shift
        ;;
    --fix-go-tools)
        fix_go_tools
        exit 0
        ;;
    --help | -h)
        echo "Usage: $0 [--dry-run|--fix-go-tools]"
        echo ""
        echo "Options:"
        echo "  --dry-run       Show what would be installed without actually installing"
        echo "  --fix-go-tools  Fix Go tools compatibility issues"
        echo "  --help, -h      Show this help message"
        exit 0
        ;;
    *)
        echo "Unknown option: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
    esac
done

# Error handling
handle_error() {
    log_error "An error occurred on line $1"
    exit 1
}

trap 'handle_error $LINENO' ERR

# Detect OS
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
        PACKAGE_MANAGER="brew"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS="linux"
        if command -v apt-get &>/dev/null; then
            PACKAGE_MANAGER="apt"
        elif command -v yum &>/dev/null; then
            PACKAGE_MANAGER="yum"
        elif command -v dnf &>/dev/null; then
            PACKAGE_MANAGER="dnf"
        else
            log_error "Unsupported Linux package manager. Please install apt, yum, or dnf."
            exit 1
        fi
    else
        log_error "Unsupported operating system: $OSTYPE"
        exit 1
    fi

    log_info "Detected OS: $OS with package manager: $PACKAGE_MANAGER"
}

# Install Homebrew (macOS)
install_homebrew() {
    if [[ "$OS" == "macos" ]]; then
        if ! command_exists brew; then
            log_info "Installing Homebrew..."
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
            else
                /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

                # Add Homebrew to PATH if needed
                if [[ -f "/opt/homebrew/bin/brew" ]]; then
                    echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >>~/.zprofile
                    # Also add to fish config if it exists
                    if [[ -f ~/.config/fish/config.fish ]]; then
                        echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >>~/.config/fish/config.fish
                    fi
                    eval "$(/opt/homebrew/bin/brew shellenv)"
                elif [[ -f "/usr/local/bin/brew" ]]; then
                    echo 'eval "$(/usr/local/bin/brew shellenv)"' >>~/.zprofile
                    # Also add to fish config if it exists
                    if [[ -f ~/.config/fish/config.fish ]]; then
                        echo 'eval "$(/usr/local/bin/brew shellenv)"' >>~/.config/fish/config.fish
                    fi
                    eval "$(/usr/local/bin/brew shellenv)"
                fi
            fi
        else
            log_info "Homebrew already installed"
        fi
    fi
}

# Install system dependencies (Linux)
install_system_deps() {
    if [[ "$OS" == "linux" ]]; then
        log_info "Installing system dependencies..."

        case $PACKAGE_MANAGER in
        "apt")
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: sudo apt-get update && sudo apt-get install -y curl wget git make build-essential libssl-dev zlib1g-dev libbz2-dev libreadline-dev libsqlite3-dev liblzma-dev libffi-dev python3 python3-pip python3-venv"
            else
                sudo apt-get update
                sudo apt-get install -y curl wget git make build-essential libssl-dev zlib1g-dev libbz2-dev libreadline-dev libsqlite3-dev liblzma-dev libffi-dev python3 python3-pip python3-venv
            fi
            ;;
        "yum")
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: sudo yum update -y && sudo yum install -y curl wget git make gcc openssl-devel bzip2-devel libffi-devel xz-devel python3 python3-pip"
            else
                sudo yum update -y
                sudo yum install -y curl wget git make gcc openssl-devel bzip2-devel libffi-devel xz-devel python3 python3-pip
            fi
            ;;
        "dnf")
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: sudo dnf update -y && sudo dnf install -y curl wget git make gcc openssl-devel bzip2-devel libffi-devel xz-devel python3 python3-pip"
            else
                sudo dnf update -y
                sudo dnf install -y curl wget git make gcc openssl-devel bzip2-devel libffi-devel xz-devel python3 python3-pip
            fi
            ;;
        esac
    fi
}

# Install system Python (required for pyenv)
install_system_python() {
    log_info "Ensuring system Python is available..."

    if [[ "$OS" == "macos" ]]; then
        # macOS typically comes with Python, but we'll ensure it's available
        if ! command_exists python3; then
            log_info "Installing Python via Homebrew..."
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: brew install python"
            else
                brew install python
            fi
        else
            log_info "System Python already available: $(python3 --version)"
        fi
    else
        # Linux - system Python should be installed via install_system_deps
        if ! command_exists python3; then
            log_error "System Python not found. Please ensure system dependencies are installed."
            return 1
        else
            log_info "System Python already available: $(python3 --version)"
        fi
    fi
}

# Install Go
install_go() {
    # Function to check if Go version is >= 1.25
    check_go_version() {
        if command_exists go; then
            local go_version=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
            local major=$(echo "$go_version" | cut -d. -f1)
            local minor=$(echo "$go_version" | cut -d. -f2)

            if [[ "$major" -gt 1 ]] || ([[ "$major" -eq 1 ]] && [[ "$minor" -ge 25 ]]); then
                return 0 # Version is >= 1.25
            else
                return 1 # Version is < 1.24
            fi
        else
            return 1 # Go not installed
        fi
    }

    # Check if Go is installed and has correct version
    if command_exists go && check_go_version; then
        log_info "Go already installed with version >= 1.25: $(go version)"
        return 0
    fi

    # Go is either not installed or has version < 1.25
    local current_version=""
    if command_exists go; then
        current_version=$(go version)
        log_info "Upgrading Go from $current_version to version >= 1.25..."
    else
        log_info "Installing Go..."
    fi

    if [[ "$OS" == "macos" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: brew install go"
        else
            brew install go
            # Add Go bin directory to PATH for macOS
            echo 'export PATH=$PATH:$(go env GOPATH)/bin' >>~/.zprofile
            # Also add to fish config if it exists
            if [[ -f ~/.config/fish/config.fish ]]; then
                echo 'set -gx PATH $PATH (go env GOPATH)/bin' >>~/.config/fish/config.fish
            fi
            export PATH=$PATH:$(go env GOPATH)/bin
        fi
    else
        # Install Go on Linux
        GO_VERSION="1.25.0"
        GO_ARCH="amd64"
        if [[ $(uname -m) == "aarch64" ]]; then
            GO_ARCH="arm64"
        fi

        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would download: https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
            log_dry_run "Would extract to: /usr/local/go"
            log_dry_run "Would add to PATH: /usr/local/go/bin"
        else
            log_info "Downloading Go ${GO_VERSION}..."
            if ! wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -O /tmp/go.tar.gz; then
                log_error "Failed to download Go ${GO_VERSION} from https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
                return 1
            fi

            log_info "Removing existing Go installation..."
            if ! sudo rm -rf /usr/local/go; then
                log_error "Failed to remove existing Go installation"
                return 1
            fi

            log_info "Extracting Go ${GO_VERSION}..."
            if ! sudo tar -C /usr/local -xzf /tmp/go.tar.gz; then
                log_error "Failed to extract Go ${GO_VERSION}"
                return 1
            fi

            log_info "Updating PATH configuration..."
            if ! echo 'export PATH=$PATH:/usr/local/go/bin' >>~/.bashrc; then
                log_error "Failed to update .bashrc PATH"
                return 1
            fi

            # Also add Go bin directory to PATH
            if ! echo 'export PATH=$PATH:$(go env GOPATH)/bin' >>~/.bashrc; then
                log_error "Failed to update .bashrc GOPATH"
                return 1
            fi

            # Also add to fish config if it exists
            if [[ -f ~/.config/fish/config.fish ]]; then
                if ! echo 'set -gx PATH $PATH /usr/local/go/bin' >>~/.config/fish/config.fish; then
                    log_error "Failed to update fish config PATH"
                    return 1
                fi
                if ! echo 'set -gx PATH $PATH (go env GOPATH)/bin' >>~/.config/fish/config.fish; then
                    log_error "Failed to update fish config GOPATH"
                    return 1
                fi
            fi

            log_info "Updating current session PATH..."
            export PATH=$PATH:/usr/local/go/bin
            export PATH=$PATH:$(go env GOPATH)/bin
        fi
    fi

    # Verify installation
    if [[ "$DRY_RUN" != "true" ]]; then
        if check_go_version; then
            log_success "Go installed successfully: $(go version)"
            # If Go was upgraded, we need to reinstall Go tools
            if command_exists go && [[ -n "$current_version" && "$(go version)" != "$current_version" ]]; then
                log_info "Go was upgraded, Go tools will be reinstalled to ensure compatibility"
                export GO_UPGRADED=true
            fi
        else
            log_error "Go installation failed or version is still < 1.25"
            return 1
        fi
    fi
}

# Install Node.js and npm
install_node() {
    if ! command_exists node; then
        log_info "Installing Node.js..."
        if [[ "$OS" == "macos" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: brew install node"
            else
                brew install node
            fi
        else
            # Install Node.js on Linux using NodeSource
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -"
                log_dry_run "Would run: sudo apt-get install -y nodejs"
            else
                curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
                sudo apt-get install -y nodejs
            fi
        fi
    else
        log_info "Node.js already installed: $(node --version)"
    fi

    if ! command_exists npm; then
        log_error "npm not found after Node.js installation"
        exit 1
    else
        log_info "npm already installed: $(npm --version)"
    fi
}

# Install Docker
install_docker() {
    if ! command_exists docker; then
        log_info "Installing Docker..."
        if [[ "$OS" == "macos" ]]; then
            log_warning "Please install Docker Desktop manually from https://www.docker.com/products/docker-desktop/"
            log_warning "After installation, ensure Docker is running and try again."
            return 1
        else
            # Install Docker on Linux
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: curl -fsSL https://get.docker.com -o get-docker.sh"
                log_dry_run "Would run: sudo sh get-docker.sh"
                log_dry_run "Would run: sudo usermod -aG docker $USER"
            else
                curl -fsSL https://get.docker.com -o get-docker.sh
                sudo sh get-docker.sh
                sudo usermod -aG docker $USER
                log_warning "Docker installed. Please log out and back in for group changes to take effect."
            fi
        fi
    else
        log_info "Docker already installed: $(docker --version)"
    fi
}

# Install yq (YAML processor)
install_yq() {
    # Minimum required version
    local min_version=4
    local yq_version=""

    if command_exists yq; then
        # Extract version from yq output like "yq (https://github.com/mikefarah/yq/) version v4.45.4"
        yq_version=$(yq --version 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | sed 's/^v//' | head -1)
        local major=$(echo "$yq_version" | cut -d. -f1)
        if [[ -n "$major" && "$major" -ge $min_version ]]; then
            log_info "yq already installed: $(yq --version)"
            return 0
        else
            log_warning "yq version $yq_version is too old. Upgrading to >= v4..."
        fi
    else
        log_info "yq not found. Installing..."
    fi

    if [[ "$OS" == "macos" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: brew install yq"
        else
            brew install yq
        fi
    else
        # Linux: Download latest yq binary from GitHub
        local yq_url="https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64"
        if [[ $(uname -m) == "aarch64" ]]; then
            yq_url="https://github.com/mikefarah/yq/releases/latest/download/yq_linux_arm64"
        fi
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would download and install yq from $yq_url to /usr/local/bin/yq"
        else
            sudo curl -L "$yq_url" -o /usr/local/bin/yq
            sudo chmod +x /usr/local/bin/yq
        fi
    fi

    # Verify installation and version
    if command_exists yq; then
        # Extract version from yq output like "yq (https://github.com/mikefarah/yq/) version v4.45.4"
        yq_version=$(yq --version 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | sed 's/^v//' | head -1)
        local major=$(echo "$yq_version" | cut -d. -f1)
        if [[ -n "$major" && "$major" -ge $min_version ]]; then
            log_success "yq installed successfully: $(yq --version)"
        else
            log_error "yq version $yq_version is too old or not installed correctly. Please install yq >= v4 manually."
            exit 1
        fi
    else
        log_error "yq installation failed. Please install yq >= v4 manually."
        exit 1
    fi
}

# Install PostgreSQL client
install_postgres_client() {
    if ! command_exists psql; then
        log_info "Installing PostgreSQL client..."
        if [[ "$OS" == "macos" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: brew install postgresql"
            else
                brew install postgresql
            fi
        else
            # Install PostgreSQL client on Linux
            case $PACKAGE_MANAGER in
            "apt")
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_dry_run "Would run: sudo apt-get install -y postgresql-client"
                else
                    sudo apt-get install -y postgresql-client
                fi
                ;;
            "yum")
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_dry_run "Would run: sudo yum install -y postgresql"
                else
                    sudo yum install -y postgresql
                fi
                ;;
            "dnf")
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_dry_run "Would run: sudo dnf install -y postgresql"
                else
                    sudo dnf install -y postgresql
                fi
                ;;
            esac
        fi
    else
        log_info "PostgreSQL client already installed: $(psql --version)"
    fi
}

# Install nginx (for configuration validation)
install_nginx() {
    if ! command_exists nginx; then
        log_info "Installing nginx (for configuration validation)..."
        if [[ "$OS" == "macos" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: brew install nginx"
            else
                brew install nginx
            fi
        else
            # Install nginx on Linux
            case $PACKAGE_MANAGER in
            "apt")
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_dry_run "Would run: sudo apt-get install -y nginx"
                else
                    sudo apt-get install -y nginx
                fi
                ;;
            "yum")
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_dry_run "Would run: sudo yum install -y nginx"
                else
                    sudo yum install -y nginx
                fi
                ;;
            "dnf")
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_dry_run "Would run: sudo dnf install -y nginx"
                else
                    sudo dnf install -y nginx
                fi
                ;;
            esac
        fi
    else
        log_info "nginx already installed: $(nginx -v 2>&1)"
    fi
}

# Install Artillery
install_artillery() {
    if ! command_exists artillery; then
        log_info "Installing Artillery..."
        if [[ "$DRY_RUN" == "true" ]]; then
            if [[ "$OS" == "linux" ]]; then
                log_dry_run "Would run: sudo npm install -g artillery"
            else
                log_dry_run "Would run: npm install -g artillery"
            fi
        else
            if [[ "$OS" == "linux" ]]; then
                sudo npm install -g artillery
            else
                # On macOS, try without sudo first, fall back to sudo if needed
                if ! npm install -g artillery 2>/dev/null; then
                    log_warning "npm install failed without sudo, trying with sudo..."
                    sudo npm install -g artillery
                fi
            fi
        fi
    else
        log_info "Artillery already installed: $(artillery --version)"
    fi

    # Install Artillery fuzzer plugin
    log_info "Installing Artillery fuzzer plugin..."
    if [[ "$DRY_RUN" == "true" ]]; then
        if [[ "$OS" == "linux" ]]; then
            log_dry_run "Would run: sudo npm install -g artillery-plugin-fuzzer"
        else
            log_dry_run "Would run: npm install -g artillery-plugin-fuzzer"
        fi
    else
        if [[ "$OS" == "linux" ]]; then
            sudo npm install -g artillery-plugin-fuzzer
        else
            # On macOS, try without sudo first, fall back to sudo if needed
            if ! npm install -g artillery-plugin-fuzzer 2>/dev/null; then
                log_warning "npm install failed without sudo, trying with sudo..."
                sudo npm install -g artillery-plugin-fuzzer
            fi
        fi
    fi
}

# Install Task
install_task() {
    if ! command_exists task; then
        log_info "Installing Task..."
        if [[ "$OS" == "macos" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: brew install go-task/tap/go-task"
            else
                brew install go-task/tap/go-task
            fi
        else
            # Install Task on Linux
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: sh -c \"\$(curl --location https://taskfile.dev/install.sh)\" -- -d -b ~/.local/bin"
                log_dry_run "Would add to PATH: ~/.local/bin"
            else
                sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
                echo 'export PATH=$PATH:~/.local/bin' >>~/.bashrc
                # Also add to fish config if it exists
                if [[ -f ~/.config/fish/config.fish ]]; then
                    echo 'set -gx PATH $PATH ~/.local/bin' >>~/.config/fish/config.fish
                fi
                export PATH=$PATH:~/.local/bin
            fi
        fi
    else
        log_info "Task already installed: $(task --version)"
    fi
}


# Note about ZAP security scanning
note_zap() {
    log_info "Note about ZAP security scanning:"
    log_info "ZAP security scans are run via Docker containers and don't require local installation."
    log_info "The Taskfile includes ZAP commands that use:"
    log_info "  - zap (baseline scan)"
    log_info "  - zap-quick (quick scan)"
    log_info "  - zap-authenticated (authenticated scan)"
    log_info "  - zap-api (API scan)"
    log_info "  - zap-all (comprehensive scan)"
    log_info "All ZAP commands use Docker and are ready to use once Docker is installed."
}

# Note about npm permissions
note_npm_permissions() {
    if [[ "$OS" == "linux" ]]; then
        log_info "Note about npm global installations:"
        log_info "Global npm packages are installed with sudo on Linux to avoid permission issues."
        log_info "If you prefer not to use sudo, you can configure npm to use a different directory:"
        log_info "  mkdir ~/.npm-global"
        log_info "  npm config set prefix '~/.npm-global'"
        log_info "  echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc"
        log_info "Then restart your shell and re-run this script."
    fi
}

# Install pyenv
install_pyenv() {
    if ! command_exists pyenv; then
        log_info "Installing pyenv..."
        if [[ "$OS" == "macos" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: brew install pyenv"
            else
                brew install pyenv
            fi
        else
            # Install pyenv on Linux
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: curl https://pyenv.run | bash"
                log_dry_run "Would add pyenv to shell configuration"
            else
                # Check if pyenv is already installed
                if [[ -d "$HOME/.pyenv" ]]; then
                    log_info "pyenv already installed at $HOME/.pyenv"
                else
                    curl https://pyenv.run | bash
                fi

                # Add pyenv to shell configuration (only if not already added)
                if [[ -f ~/.bashrc ]] && ! grep -q "PYENV_ROOT" ~/.bashrc; then
                    echo 'export PYENV_ROOT="$HOME/.pyenv"' >>~/.bashrc
                    echo 'command -v pyenv >/dev/null || export PATH="$PYENV_ROOT/bin:$PATH"' >>~/.bashrc
                    echo 'eval "$(pyenv init -)"' >>~/.bashrc
                fi

                if [[ -f ~/.zshrc ]] && ! grep -q "PYENV_ROOT" ~/.zshrc; then
                    echo 'export PYENV_ROOT="$HOME/.pyenv"' >>~/.zshrc
                    echo 'command -v pyenv >/dev/null || export PATH="$PYENV_ROOT/bin:$PATH"' >>~/.zshrc
                    echo 'eval "$(pyenv init -)"' >>~/.zshrc
                fi

                if [[ -f ~/.config/fish/config.fish ]] && ! grep -q "PYENV_ROOT" ~/.config/fish/config.fish; then
                    echo 'set -gx PYENV_ROOT "$HOME/.pyenv"' >>~/.config/fish/config.fish
                    echo 'if not contains $PYENV_ROOT/bin $fish_user_paths' >>~/.config/fish/config.fish
                    echo '    set -U fish_user_paths $PYENV_ROOT/bin $fish_user_paths' >>~/.config/fish/config.fish
                    echo 'end' >>~/.config/fish/config.fish
                    echo 'pyenv init - | source' >>~/.config/fish/config.fish
                fi

                # Source the configuration for current session
                export PYENV_ROOT="$HOME/.pyenv"
                export PATH="$PYENV_ROOT/bin:$PATH"
                eval "$(pyenv init -)"
            fi
        fi
    else
        log_info "pyenv already installed: $(pyenv --version)"
    fi

    # Install pyenv-virtualenv plugin
    if [[ ! -d "$HOME/.pyenv/plugins/pyenv-virtualenv" ]]; then
        log_info "Installing pyenv-virtualenv plugin..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: git clone https://github.com/pyenv/pyenv-virtualenv.git ~/.pyenv/plugins/pyenv-virtualenv"
        else
            git clone https://github.com/pyenv/pyenv-virtualenv.git ~/.pyenv/plugins/pyenv-virtualenv
        fi
    else
        log_info "pyenv-virtualenv plugin already installed"
    fi
}

# Setup Python virtual environment
setup_python_env() {
    log_info "Setting up Python virtual environment..."

    # Ensure pyenv is available in current session
    if command_exists pyenv; then
        export PYENV_ROOT="$HOME/.pyenv"
        export PATH="$PYENV_ROOT/bin:$PATH"
        eval "$(pyenv init -)"
        eval "$(pyenv virtualenv-init -)"
    else
        log_error "pyenv not found. Please ensure pyenv is installed and available in PATH."
        return 1
    fi

    # Check if Python 3.13 is already installed (exact match, not virtual environments)
    local python_3_13_installed=false
    if pyenv versions | grep -E "^  3\.13\." >/dev/null; then
        python_3_13_installed=true
        local python_version=$(pyenv versions | grep -E "^  3\.13\." | head -1 | tr -d ' ')
        log_info "Python 3.13 already installed: $python_version"
    fi

    # Install Python 3.13 if not available
    if [[ "$python_3_13_installed" == "false" ]]; then
        log_info "Installing Python 3.13..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: pyenv install 3.13"
        else
            pyenv install 3.13
        fi
    fi

    # Create virtual environment if it doesn't exist
    local virtualenvs_output=$(pyenv virtualenvs)
    echo "$virtualenvs_output"
    if echo "$virtualenvs_output" | grep -q "quiz"; then
        log_info "Virtual environment 'quiz' already exists"
    else
        log_info "Virtual environment 'quiz' not found, creating it..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: pyenv virtualenv 3.13 quiz"
        else
            pyenv virtualenv 3.13 quiz
        fi
    fi

    # Set local virtual environment
    if [[ "$DRY_RUN" == "true" ]]; then
        log_dry_run "Would run: pyenv local quiz"
    else
        pyenv local quiz
    fi

    # Install dependencies
    if [[ "$DRY_RUN" == "true" ]]; then
        log_dry_run "Would install requirements.txt in the quiz virtual environment"
    else
        # Upgrade pip
        pip install --upgrade pip

        # Install requirements if they exist
        if [[ -f "requirements.txt" ]]; then
            log_info "Installing Python dependencies from requirements.txt..."
            pip install -r requirements.txt
        else
            log_warning "requirements.txt not found, skipping Python dependencies"
        fi

        log_success "Python virtual environment 'quiz' is ready!"
        log_info "To activate the environment manually, run: pyenv activate quiz"
        log_info "Or simply run commands and pyenv will automatically use the quiz environment"
    fi
}

# Install Python dependencies
install_python_deps() {
    if command_exists python3; then
        log_info "Installing Python dependencies..."
        if [[ -f "requirements.txt" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: pip3 install -r requirements.txt"
            else
                pip3 install -r requirements.txt
            fi
        else
            log_warning "requirements.txt not found, skipping Python dependencies"
        fi
    else
        log_warning "Python3 not found, skipping Python dependencies"
    fi
}

# Install Go tools
install_go_tools() {
    log_info "Installing Go development tools..."

    # Ensure Go bin directory is in PATH for current session
    export PATH=$PATH:$(go env GOPATH)/bin

    # Check if Go was upgraded and we need to reinstall tools
    local force_reinstall=false
    if [[ "${GO_UPGRADED:-false}" == "true" ]]; then
        log_info "Go was upgraded, reinstalling all Go tools for compatibility..."
        force_reinstall=true
    fi

    # Install oapi-codegen
    if ! command_exists oapi-codegen || [[ "$force_reinstall" == "true" ]]; then
        log_info "Installing oapi-codegen..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest"
        else
            go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
        fi
    else
        log_info "oapi-codegen already installed"
    fi

    # Install deadcode
    if ! command_exists deadcode || [[ "$force_reinstall" == "true" ]]; then
        log_info "Installing deadcode..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: go install github.com/remyoudompheng/go-misc/deadcode@latest"
        else
            go install github.com/remyoudompheng/go-misc/deadcode@latest
        fi
    else
        log_info "deadcode already installed"
    fi

    # Install golangci-lint
    if ! command_exists golangci-lint || [[ "$force_reinstall" == "true" ]]; then
        log_info "Installing golangci-lint..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.1"
        else
            go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.1
        fi
    else
        log_info "golangci-lint already installed"
    fi

    # Install revive
    if ! command_exists revive || [[ "$force_reinstall" == "true" ]]; then
        log_info "Installing revive..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: go install github.com/mgechev/revive@latest"
        else
            go install github.com/mgechev/revive@latest
        fi
    else
        log_info "revive already installed"
    fi

    # Install goimports
    if ! command_exists goimports || [[ "$force_reinstall" == "true" ]]; then
        log_info "Installing goimports..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: go install golang.org/x/tools/cmd/goimports@latest"
        else
            go install golang.org/x/tools/cmd/goimports@latest
        fi
    else
        log_info "goimports already installed"
    fi

    # Install gofumpt (modern formatting, enforces 'any' over 'interface{}')
    if ! command_exists gofumpt || [[ "$force_reinstall" == "true" ]]; then
        log_info "Installing gofumpt..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: go install mvdan.cc/gofumpt@latest"
        else
            go install mvdan.cc/gofumpt@latest
        fi
    else
        log_info "gofumpt already installed"
    fi

    # Install staticcheck (modernize linter)
    if ! command_exists staticcheck || [[ "$force_reinstall" == "true" ]]; then
        log_info "Installing staticcheck..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: go install honnef.co/go/tools/cmd/staticcheck@latest"
        else
            go install honnef.co/go/tools/cmd/staticcheck@latest
        fi
    else
        log_info "staticcheck already installed"
    fi

    # Install covreport (coverage reporting tool)
    if ! command_exists covreport || [[ "$force_reinstall" == "true" ]]; then
        log_info "Installing covreport..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would run: go install github.com/cancue/covreport@latest"
        else
            go install github.com/cancue/covreport@latest
        fi
    else
        log_info "covreport already installed"
    fi
}

# Install Node.js tools
install_node_tools() {
    log_info "Installing Node.js development tools..."

    # Install global npm packages
    local packages=("eslint" "prettier" "ts-prune" "orval" "pyright" "vite" "vitest" "@playwright/test")

    for package in "${packages[@]}"; do
        if ! command_exists "$package"; then
            log_info "Installing $package..."
            if [[ "$DRY_RUN" == "true" ]]; then
                if [[ "$OS" == "linux" ]]; then
                    log_dry_run "Would run: sudo npm install -g $package"
                else
                    log_dry_run "Would run: npm install -g $package"
                fi
            else
                if [[ "$OS" == "linux" ]]; then
                    sudo npm install -g "$package"
                else
                    # On macOS, try without sudo first, fall back to sudo if needed
                    if ! npm install -g "$package" 2>/dev/null; then
                        log_warning "npm install failed without sudo, trying with sudo..."
                        sudo npm install -g "$package"
                    fi
                fi
            fi
        else
            log_info "$package already installed"
        fi
    done

    # Install TypeScript separately (provides tsc command)
    if ! command_exists tsc; then
        log_info "Installing TypeScript (tsc)..."
        if [[ "$DRY_RUN" == "true" ]]; then
            if [[ "$OS" == "linux" ]]; then
                log_dry_run "Would run: sudo npm install -g typescript"
            else
                log_dry_run "Would run: npm install -g typescript"
            fi
        else
            if [[ "$OS" == "linux" ]]; then
                sudo npm install -g typescript
            else
                # On macOS, try without sudo first, fall back to sudo if needed
                if ! npm install -g typescript 2>/dev/null; then
                    log_warning "npm install failed without sudo, trying with sudo..."
                    sudo npm install -g typescript
                fi
            fi
        fi
    else
        log_info "TypeScript (tsc) already installed"
    fi

    # Install Playwright browser dependencies
    log_info "Installing Playwright browser dependencies..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log_dry_run "Would run: npx playwright install-deps"
    else
        # Create a temporary directory for Playwright installation
        local temp_dir=$(mktemp -d)
        cd "$temp_dir"

        # Create a minimal package.json for Playwright installation
        echo '{"name":"temp-playwright-install"}' > package.json

        # Install Playwright and its dependencies
        npm install @playwright/test
        npx playwright install-deps

        # Clean up
        cd - > /dev/null
        rm -rf "$temp_dir"

        log_success "Playwright browser dependencies installed successfully!"
    fi
}

# Ensure docker buildx bake support (buildx is bundled with recent Docker; nothing to install in most cases)
ensure_buildx_bake() {
    if command -v docker >/dev/null 2>&1; then
        echo "Docker available: $(docker --version)"
        # buildx bake is available when buildx plugin is present (Docker Desktop includes it)
        if docker buildx bake --help >/dev/null 2>&1; then
            echo "docker buildx bake available"
            return 0
        else
            echo "Warning: docker buildx bake not available. Ensure Docker Buildx plugin is installed and enabled." >&2
            return 1
        fi
    else
        echo "Warning: docker not found; install Docker to use buildx bake" >&2
        return 1
    fi
}

# Install golang-migrate
install_golang_migrate() {
    if ! command_exists migrate; then
        log_info "Installing golang-migrate..."
        if [[ "$OS" == "macos" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would run: brew install golang-migrate"
            else
                brew install golang-migrate
            fi
        else
            # Linux: Download latest migrate binary from GitHub
            local url="https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-amd64.tar.gz"
            if [[ $(uname -m) == "aarch64" ]]; then
                url="https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-arm64.tar.gz"
            fi
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would download and install golang-migrate from $url to /usr/local/bin/migrate"
            else
                curl -L "$url" -o /tmp/migrate.tar.gz
                tar -xzf /tmp/migrate.tar.gz -C /tmp
                sudo mv /tmp/migrate /usr/local/bin/migrate
                sudo chmod +x /usr/local/bin/migrate
            fi
        fi
    else
        log_info "golang-migrate already installed: $(migrate --version)"
    fi
}

# Verify installations
verify_installations() {
    log_info "Verifying installations..."

    local tools=("node" "npm" "docker" "task" "pyenv" "eslint" "prettier" "ts-prune" "orval" "pyright" "tsc" "vite" "vitest" "playwright" "artillery" "oapi-codegen" "deadcode" "golangci-lint" "revive" "goimports" "gofumpt" "staticcheck" "psql" "nginx")
    local missing_tools=()

    # Check Go specifically for version >= 1.24
    if command_exists go; then
        local go_version=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
        local major=$(echo "$go_version" | cut -d. -f1)
        local minor=$(echo "$go_version" | cut -d. -f2)

        if [[ "$major" -gt 1 ]] || ([[ "$major" -eq 1 ]] && [[ "$minor" -ge 24 ]]); then
            log_success "go: $(command -v "go") (version $go_version >= 1.24)"
        else
            log_error "go: $(command -v "go") (version $go_version < 1.24)"
            missing_tools+=("go")
        fi
    else
        log_error "go: NOT FOUND"
        missing_tools+=("go")
    fi

    for tool in "${tools[@]}"; do
        if command_exists "$tool"; then
            log_success "$tool: $(command -v "$tool")"
        else
            log_error "$tool: NOT FOUND"
            missing_tools+=("$tool")
        fi
    done

    # Check system Python
    if command_exists python3; then
        log_success "System Python: $(python3 --version)"
    else
        log_error "System Python: NOT FOUND"
        missing_tools+=("python3")
    fi

    # Check if virtual environment exists
    local verify_virtualenvs_output=$(pyenv virtualenvs)
    if echo "$verify_virtualenvs_output" | grep -q "quiz"; then
        log_success "Python virtual environment: quiz exists"
    else
        log_error "Python virtual environment: quiz NOT FOUND"
        missing_tools+=("python-venv")
    fi

    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing tools: ${missing_tools[*]}"
        return 1
    else
        log_success "All tools verified successfully!"
    fi
}

# Main installation function
main() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "DRY RUN MODE - No actual installations will be performed"
        log_info "This will show what would be installed"
        echo ""
    fi

    log_info "Starting Quiz Application tooling installation..."
    log_info "OS: $OSTYPE"

    # Detect OS and package manager
    detect_os

    # Install core tools
    install_homebrew
    install_system_deps
    install_system_python
    install_go
    install_node
    install_docker
    install_yq
    install_postgres_client
    install_nginx
    install_artillery
    install_task

    # Install development tools
    install_pyenv
    setup_python_env
    install_go_tools
    install_golang_migrate
    install_node_tools

    # Note about ZAP
    note_zap

    # Note about npm permissions
    note_npm_permissions

    # Verify everything (only in non-dry-run mode)
    if [[ "$DRY_RUN" != "true" ]]; then
        verify_installations
    fi

    if [[ "$DRY_RUN" == "true" ]]; then
        log_success "Dry run completed! Review the output above to see what would be installed."
        log_info ""
        log_info "To perform the actual installation, run:"
        log_info "  ./scripts/install-tooling.sh"
    else
        log_success "Installation completed successfully!"
        log_info ""
        log_info "Next steps:"
        log_info "1. Ensure Docker Desktop is running (if on macOS)"
        log_info "2. The Python virtual environment 'quiz' is automatically active in this directory"
        log_info "3. Run 'task start-prod' to start the application"
        log_info "4. Run 'task test' to run all tests"
        log_info "5. Access the application at http://localhost:3000"
        log_info "6. Run 'task zap' for security scanning"
        log_info ""
        log_info "Note: You may need to restart your shell for PATH changes to take effect"
        log_info "Note: The Python virtual environment 'quiz' is managed by pyenv"
        log_info "Note: ZAP security scanning uses Docker and is ready to use"
        log_info "Note: Shell configuration has been added for bash, zsh, and fish shells"
    fi
}

# Run main function
main "$@"

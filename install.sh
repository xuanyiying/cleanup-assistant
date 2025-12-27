#!/bin/bash

# Cleanup CLI Installation Script for macOS
# This script downloads and installs the latest version of cleanup-cli

set -e

VERSION="1.0.0"
BINARY_NAME="cleanup"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.cleanup"
CONFIG_FILE="$HOME/.cleanuprc.yaml"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect architecture
detect_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

# Print colored message
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Check if running on macOS
check_os() {
    if [[ "$OSTYPE" != "darwin"* ]]; then
        print_message "$RED" "Error: This script is for macOS only"
        exit 1
    fi
}

# Check prerequisites
check_prerequisites() {
    print_message "$BLUE" "Checking prerequisites..."
    
    # Check if Ollama is installed
    if ! command -v ollama &> /dev/null; then
        print_message "$YELLOW" "Warning: Ollama is not installed"
        print_message "$YELLOW" "Cleanup CLI requires Ollama for AI features"
        print_message "$YELLOW" "Visit https://ollama.ai to install Ollama"
        echo ""
        read -p "Continue installation anyway? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        print_message "$GREEN" "✓ Ollama is installed"
    fi
}

# Install binary
install_binary() {
    print_message "$BLUE" "Installing Cleanup CLI..."
    
    local arch=$(detect_arch)
    if [[ "$arch" == "unknown" ]]; then
        print_message "$RED" "Error: Unsupported architecture"
        exit 1
    fi
    
    # Check if binary exists in build directory
    if [[ -f "build/cleanup-darwin-$arch" ]]; then
        print_message "$GREEN" "Found local build"
        sudo cp "build/cleanup-darwin-$arch" "$INSTALL_DIR/$BINARY_NAME"
    elif [[ -f "build/cleanup-darwin-universal" ]]; then
        print_message "$GREEN" "Found universal binary"
        sudo cp "build/cleanup-darwin-universal" "$INSTALL_DIR/$BINARY_NAME"
    elif [[ -f "build/cleanup" ]]; then
        print_message "$GREEN" "Found local build"
        sudo cp "build/cleanup" "$INSTALL_DIR/$BINARY_NAME"
    else
        print_message "$RED" "Error: Binary not found"
        print_message "$YELLOW" "Please build the project first:"
        print_message "$YELLOW" "  make build"
        exit 1
    fi
    
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    print_message "$GREEN" "✓ Binary installed to $INSTALL_DIR/$BINARY_NAME"
}

# Setup configuration
setup_config() {
    print_message "$BLUE" "Setting up configuration..."
    
    # Create config directory
    mkdir -p "$CONFIG_DIR"
    
    # Copy example config if user doesn't have one
    if [[ ! -f "$CONFIG_FILE" ]]; then
        if [[ -f ".cleanuprc.yaml" ]]; then
            cp ".cleanuprc.yaml" "$CONFIG_FILE"
            print_message "$GREEN" "✓ Configuration file created at $CONFIG_FILE"
        else
            print_message "$YELLOW" "Warning: Example config not found"
        fi
    else
        print_message "$YELLOW" "Configuration file already exists at $CONFIG_FILE"
    fi
    
    # Create transaction log directory
    mkdir -p "$CONFIG_DIR/transactions"
    mkdir -p "$CONFIG_DIR/trash"
    
    print_message "$GREEN" "✓ Configuration setup complete"
}

# Post-installation instructions
post_install() {
    echo ""
    print_message "$GREEN" "╔════════════════════════════════════════╗"
    print_message "$GREEN" "║   Cleanup CLI Installation Complete   ║"
    print_message "$GREEN" "╚════════════════════════════════════════╝"
    echo ""
    print_message "$BLUE" "Quick Start:"
    echo "  1. Ensure Ollama is running:"
    echo "     $ ollama serve"
    echo ""
    echo "  2. Pull the required model:"
    echo "     $ ollama pull llama3.2"
    echo ""
    echo "  3. Run cleanup:"
    echo "     $ cleanup"
    echo ""
    echo "  4. Or organize a specific directory:"
    echo "     $ cleanup organize ~/Downloads"
    echo ""
    print_message "$BLUE" "Configuration:"
    echo "  Edit: $CONFIG_FILE"
    echo ""
    print_message "$BLUE" "Documentation:"
    echo "  Run: cleanup --help"
    echo ""
}

# Main installation flow
main() {
    echo ""
    print_message "$BLUE" "╔════════════════════════════════════════╗"
    print_message "$BLUE" "║     Cleanup CLI Installer v$VERSION      ║"
    print_message "$BLUE" "╚════════════════════════════════════════╝"
    echo ""
    
    check_os
    check_prerequisites
    install_binary
    setup_config
    post_install
}

# Run main function
main

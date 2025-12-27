#!/bin/bash

# Cleanup CLI Uninstallation Script for macOS

set -e

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

# Print colored message
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Remove binary
remove_binary() {
    if [[ -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
        print_message "$BLUE" "Removing binary..."
        sudo rm -f "$INSTALL_DIR/$BINARY_NAME"
        print_message "$GREEN" "✓ Binary removed"
    else
        print_message "$YELLOW" "Binary not found at $INSTALL_DIR/$BINARY_NAME"
    fi
}

# Remove configuration
remove_config() {
    echo ""
    print_message "$YELLOW" "Do you want to remove configuration and data?"
    print_message "$YELLOW" "This will delete:"
    echo "  - $CONFIG_FILE"
    echo "  - $CONFIG_DIR (including transaction logs and trash)"
    echo ""
    read -p "Remove configuration? (y/n) " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [[ -f "$CONFIG_FILE" ]]; then
            rm -f "$CONFIG_FILE"
            print_message "$GREEN" "✓ Configuration file removed"
        fi
        
        if [[ -d "$CONFIG_DIR" ]]; then
            rm -rf "$CONFIG_DIR"
            print_message "$GREEN" "✓ Data directory removed"
        fi
    else
        print_message "$BLUE" "Configuration and data preserved"
    fi
}

# Main uninstallation flow
main() {
    echo ""
    print_message "$BLUE" "╔════════════════════════════════════════╗"
    print_message "$BLUE" "║     Cleanup CLI Uninstaller            ║"
    print_message "$BLUE" "╚════════════════════════════════════════╝"
    echo ""
    
    remove_binary
    remove_config
    
    echo ""
    print_message "$GREEN" "╔════════════════════════════════════════╗"
    print_message "$GREEN" "║   Cleanup CLI Uninstalled              ║"
    print_message "$GREEN" "╚════════════════════════════════════════╝"
    echo ""
}

# Run main function
main

#!/bin/bash

# Package script for creating macOS distribution packages

set -e

VERSION="1.0.0"
PROJECT_NAME="cleanup-cli"
BUILD_DIR="build"
DIST_DIR="dist"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

print_message() {
    echo -e "${BLUE}$1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Clean previous builds
clean() {
    print_message "Cleaning previous builds..."
    rm -rf "$BUILD_DIR"
    rm -rf "$DIST_DIR"
    mkdir -p "$BUILD_DIR"
    mkdir -p "$DIST_DIR"
    print_success "Clean complete"
}

# Build binaries
build() {
    print_message "Building binaries..."
    make darwin-universal
    print_success "Build complete"
}

# Create tar.gz package
create_tarball() {
    print_message "Creating tar.gz package..."
    
    local package_dir="$BUILD_DIR/$PROJECT_NAME-$VERSION"
    mkdir -p "$package_dir"
    
    # Copy files
    cp "$BUILD_DIR/cleanup-darwin-universal" "$package_dir/cleanup"
    cp .cleanuprc.yaml "$package_dir/cleanuprc.yaml.example"
    cp README.md "$package_dir/"
    cp install.sh "$package_dir/"
    cp uninstall.sh "$package_dir/"
    cp -r examples "$package_dir/" 2>/dev/null || true
    
    # Create archive
    tar -czf "$DIST_DIR/$PROJECT_NAME-$VERSION-darwin.tar.gz" -C "$BUILD_DIR" "$PROJECT_NAME-$VERSION"
    
    # Calculate checksum
    shasum -a 256 "$DIST_DIR/$PROJECT_NAME-$VERSION-darwin.tar.gz" > "$DIST_DIR/$PROJECT_NAME-$VERSION-darwin.tar.gz.sha256"
    
    print_success "Tarball created: $DIST_DIR/$PROJECT_NAME-$VERSION-darwin.tar.gz"
}

# Create .pkg installer
create_pkg() {
    print_message "Creating .pkg installer..."
    
    local pkg_root="$BUILD_DIR/pkg-root"
    local pkg_scripts="$BUILD_DIR/pkg-scripts"
    
    # Create package structure
    mkdir -p "$pkg_root/usr/local/bin"
    mkdir -p "$pkg_root/usr/local/share/cleanup"
    mkdir -p "$pkg_scripts"
    
    # Copy files
    cp "$BUILD_DIR/cleanup-darwin-universal" "$pkg_root/usr/local/bin/cleanup"
    chmod +x "$pkg_root/usr/local/bin/cleanup"
    cp .cleanuprc.yaml "$pkg_root/usr/local/share/cleanup/cleanuprc.yaml.example"
    cp README.md "$pkg_root/usr/local/share/cleanup/"
    
    # Create postinstall script
    cat > "$pkg_scripts/postinstall" << 'EOF'
#!/bin/bash
# Post-installation script

CONFIG_FILE="$HOME/.cleanuprc.yaml"
CONFIG_DIR="$HOME/.cleanup"

# Create config directory
mkdir -p "$CONFIG_DIR"
mkdir -p "$CONFIG_DIR/transactions"
mkdir -p "$CONFIG_DIR/trash"

# Copy example config if user doesn't have one
if [[ ! -f "$CONFIG_FILE" ]]; then
    cp /usr/local/share/cleanup/cleanuprc.yaml.example "$CONFIG_FILE"
    echo "Configuration file created at $CONFIG_FILE"
fi

echo "Cleanup CLI installed successfully!"
echo "Run 'cleanup --help' to get started"

exit 0
EOF
    
    chmod +x "$pkg_scripts/postinstall"
    
    # Build package
    pkgbuild \
        --root "$pkg_root" \
        --scripts "$pkg_scripts" \
        --identifier "com.cleanup.cli" \
        --version "$VERSION" \
        --install-location "/" \
        "$DIST_DIR/$PROJECT_NAME-$VERSION.pkg"
    
    print_success "Package created: $DIST_DIR/$PROJECT_NAME-$VERSION.pkg"
}

# Create Homebrew formula
create_formula() {
    print_message "Updating Homebrew formula..."
    
    local tarball="$DIST_DIR/$PROJECT_NAME-$VERSION-darwin.tar.gz"
    local sha256=$(shasum -a 256 "$tarball" | awk '{print $1}')
    
    # Update formula with actual SHA256
    sed -i.bak "s/sha256 \".*\"/sha256 \"$sha256\"/" Formula/cleanup.rb
    rm -f Formula/cleanup.rb.bak
    
    print_success "Formula updated with SHA256: $sha256"
}

# Create distribution info
create_info() {
    print_message "Creating distribution info..."
    
    cat > "$DIST_DIR/INSTALL.md" << EOF
# Cleanup CLI v$VERSION - Installation Guide

## Installation Methods

### Method 1: Using the installer (.pkg)

1. Download \`$PROJECT_NAME-$VERSION.pkg\`
2. Double-click to run the installer
3. Follow the installation wizard
4. Run \`cleanup --help\` to verify installation

### Method 2: Using tar.gz archive

1. Download \`$PROJECT_NAME-$VERSION-darwin.tar.gz\`
2. Extract the archive:
   \`\`\`bash
   tar -xzf $PROJECT_NAME-$VERSION-darwin.tar.gz
   cd $PROJECT_NAME-$VERSION
   \`\`\`
3. Run the installation script:
   \`\`\`bash
   ./install.sh
   \`\`\`

### Method 3: Using Homebrew (local)

\`\`\`bash
brew install --formula ./Formula/cleanup.rb
\`\`\`

## Prerequisites

- macOS 10.15 or later
- [Ollama](https://ollama.ai) installed and running

## Quick Start

1. Start Ollama:
   \`\`\`bash
   ollama serve
   \`\`\`

2. Pull the required model:
   \`\`\`bash
   ollama pull llama3.2
   \`\`\`

3. Run cleanup:
   \`\`\`bash
   cleanup
   \`\`\`

## Configuration

Edit the configuration file at \`~/.cleanuprc.yaml\`

## Uninstallation

Run the uninstall script:
\`\`\`bash
./uninstall.sh
\`\`\`

Or if installed via Homebrew:
\`\`\`bash
brew uninstall cleanup
\`\`\`

## Support

For issues and questions, visit: https://github.com/xuanyiying/cleanup-cli
EOF
    
    print_success "Installation guide created"
}

# Main packaging flow
main() {
    echo ""
    print_message "╔════════════════════════════════════════╗"
    print_message "║   Cleanup CLI Package Builder          ║"
    print_message "╚════════════════════════════════════════╝"
    echo ""
    
    clean
    build
    create_tarball
    create_pkg
    create_formula
    create_info
    
    echo ""
    print_message "╔════════════════════════════════════════╗"
    print_message "║   Packaging Complete!                  ║"
    print_message "╚════════════════════════════════════════╝"
    echo ""
    print_message "Distribution packages created in: $DIST_DIR/"
    echo ""
    ls -lh "$DIST_DIR/"
    echo ""
}

# Run main function
main

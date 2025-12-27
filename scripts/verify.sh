#!/bin/bash

# Verification script for Cleanup CLI build system

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

echo ""
print_info "╔════════════════════════════════════════╗"
print_info "║   Cleanup CLI Build Verification      ║"
print_info "╚════════════════════════════════════════╝"
echo ""

# Check Go installation
print_info "Checking Go installation..."
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    print_success "Go installed: $GO_VERSION"
else
    print_error "Go not found"
    exit 1
fi

# Check required files
print_info "Checking required files..."
REQUIRED_FILES=(
    "Makefile"
    "install.sh"
    "uninstall.sh"
    "README.md"
    "INSTALL.md"
    "QUICKSTART.md"
    ".cleanuprc.yaml"
    "go.mod"
    "cmd/cleanup/main.go"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        print_success "$file exists"
    else
        print_error "$file missing"
        exit 1
    fi
done

# Check scripts are executable
print_info "Checking script permissions..."
SCRIPTS=(
    "install.sh"
    "uninstall.sh"
    "examples/demo.sh"
    "scripts/package.sh"
)

for script in "${SCRIPTS[@]}"; do
    if [[ -x "$script" ]]; then
        print_success "$script is executable"
    else
        print_error "$script is not executable"
        exit 1
    fi
done

# Run tests
print_info "Running tests..."
if go test ./... > /dev/null 2>&1; then
    print_success "All tests passed"
else
    print_error "Tests failed"
    exit 1
fi

# Test build
print_info "Testing build..."
if make clean > /dev/null 2>&1 && make build > /dev/null 2>&1; then
    print_success "Build successful"
else
    print_error "Build failed"
    exit 1
fi

# Check binary
print_info "Checking binary..."
if [[ -f "build/cleanup" ]]; then
    print_success "Binary created"
    
    # Test version command
    if ./build/cleanup version > /dev/null 2>&1; then
        VERSION=$(./build/cleanup version | head -1)
        print_success "Version command works: $VERSION"
    else
        print_error "Version command failed"
        exit 1
    fi
else
    print_error "Binary not found"
    exit 1
fi

# Check Makefile targets
print_info "Checking Makefile targets..."
TARGETS=(
    "build"
    "darwin"
    "darwin-arm"
    "darwin-universal"
    "linux"
    "windows"
    "test"
    "clean"
    "install"
    "uninstall"
    "package-tar"
)

for target in "${TARGETS[@]}"; do
    if grep -q "^$target:" Makefile; then
        print_success "Target '$target' exists"
    else
        print_error "Target '$target' missing"
        exit 1
    fi
done

# Summary
echo ""
print_info "╔════════════════════════════════════════╗"
print_info "║   Verification Complete!               ║"
print_info "╚════════════════════════════════════════╝"
echo ""
print_success "All checks passed!"
echo ""
print_info "Next steps:"
echo "  1. Build for all platforms: make all-platforms"
echo "  2. Create packages: ./scripts/package.sh"
echo "  3. Test installation: ./install.sh"
echo ""

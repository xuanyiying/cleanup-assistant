# Cleanup CLI Makefile

VERSION := 1.0.0
BINARY_NAME := cleanup
BUILD_DIR := build
INSTALL_DIR := /usr/local/bin

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

.PHONY: all build clean test install uninstall darwin linux windows package help

all: test build

## build: Build the binary for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/cleanup
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## darwin: Build for macOS (Intel)
darwin:
	@echo "Building for macOS (Intel)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/cleanup
	@echo "✓ macOS Intel build complete"

## darwin-arm: Build for macOS (Apple Silicon)
darwin-arm:
	@echo "Building for macOS (Apple Silicon)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/cleanup
	@echo "✓ macOS ARM build complete"

## darwin-universal: Build universal binary for macOS
darwin-universal: darwin darwin-arm
	@echo "Creating universal binary..."
	@mkdir -p $(BUILD_DIR)
	lipo -create -output $(BUILD_DIR)/$(BINARY_NAME)-darwin-universal \
		$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 \
		$(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64
	@echo "✓ Universal binary created: $(BUILD_DIR)/$(BINARY_NAME)-darwin-universal"

## linux: Build for Linux
linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/cleanup
	@echo "✓ Linux build complete"

## windows: Build for Windows
windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/cleanup
	@echo "✓ Windows build complete"

## all-platforms: Build for all platforms
all-platforms: darwin-universal linux windows
	@echo "✓ All platform builds complete"

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...
	@echo "✓ Tests complete"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)
	@echo "✓ Clean complete"

## install: Install binary to system (macOS/Linux)
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ Installed to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo ""
	@echo "Run 'cleanup --help' to get started"

## uninstall: Remove installed binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ Uninstalled"

## package-darwin: Create macOS installer package
package-darwin: darwin-universal
	@echo "Creating macOS package..."
	@mkdir -p $(BUILD_DIR)/package/usr/local/bin
	@mkdir -p $(BUILD_DIR)/package/usr/local/share/cleanup
	@cp $(BUILD_DIR)/$(BINARY_NAME)-darwin-universal $(BUILD_DIR)/package/usr/local/bin/$(BINARY_NAME)
	@cp .cleanuprc.yaml $(BUILD_DIR)/package/usr/local/share/cleanup/cleanuprc.yaml.example
	@cp README.md $(BUILD_DIR)/package/usr/local/share/cleanup/
	@chmod +x $(BUILD_DIR)/package/usr/local/bin/$(BINARY_NAME)
	@echo "✓ Package structure created"
	@echo ""
	@echo "To create .pkg installer, run:"
	@echo "  pkgbuild --root $(BUILD_DIR)/package --identifier com.cleanup.cli --version $(VERSION) --install-location / $(BUILD_DIR)/cleanup-$(VERSION).pkg"

## package-tar: Create tar.gz archive
package-tar: darwin-universal
	@echo "Creating tar.gz package..."
	@mkdir -p $(BUILD_DIR)/cleanup-$(VERSION)
	@cp $(BUILD_DIR)/$(BINARY_NAME)-darwin-universal $(BUILD_DIR)/cleanup-$(VERSION)/$(BINARY_NAME)
	@cp .cleanuprc.yaml $(BUILD_DIR)/cleanup-$(VERSION)/cleanuprc.yaml.example
	@cp README.md $(BUILD_DIR)/cleanup-$(VERSION)/
	@cp examples/demo.sh $(BUILD_DIR)/cleanup-$(VERSION)/
	@tar -czf $(BUILD_DIR)/cleanup-$(VERSION)-darwin.tar.gz -C $(BUILD_DIR) cleanup-$(VERSION)
	@rm -rf $(BUILD_DIR)/cleanup-$(VERSION)
	@echo "✓ Package created: $(BUILD_DIR)/cleanup-$(VERSION)-darwin.tar.gz"

## help: Show this help message
help:
	@echo "Cleanup CLI - Makefile commands:"
	@echo ""
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := help

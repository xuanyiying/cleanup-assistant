#!/bin/bash
set -e

# Go to project root
cd "$(dirname "$0")/.."

echo "Starting release build process..."

# 1. Clean build directory
make clean

# 2. Build binaries for all platforms
echo "Building binaries..."
make all-platforms

# 3. Build macOS Installer
echo "Building macOS installer..."
chmod +x packaging/macos/build_pkg.sh
./packaging/macos/build_pkg.sh

# 4. Instructions for Windows
echo "----------------------------------------------------------------"
echo "Build Complete!"
echo "----------------------------------------------------------------"
echo "Artifacts:"
echo "  - macOS Installer: dist/cleanup-1.0.0.pkg"
echo "  - Windows Binaries: build/cleanup-windows-amd64.exe, build/cleanup-windows-x86.exe"
echo "  - Linux Binary: build/cleanup-linux-amd64"
echo ""
echo "To build the Windows Installer (.exe):"
echo "1. Copy the 'packaging/windows/cleanup.iss' file and the 'build' directory to a Windows machine."
echo "2. Install Inno Setup (https://jrsoftware.org/isinfo.php)."
echo "3. Open 'cleanup.iss' with Inno Setup Compiler."
echo "4. Compile the script."
echo "5. The installer will be generated in the 'dist' folder."
echo "----------------------------------------------------------------"

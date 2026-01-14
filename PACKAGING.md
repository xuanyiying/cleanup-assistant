# Cleanup CLI Packaging Guide

This document describes how to package the Cleanup CLI for Windows and macOS.

## Prerequisites

- **Go 1.21+**: Required to compile the source code.
- **Make**: For running build scripts.
- **Inno Setup 6+** (Windows only): Required to build the Windows installer.
- **macOS**: Required to build the macOS installer package (uses `pkgbuild`).

## Build Process

We provide a unified script to build binaries and the macOS installer.

### 1. Build Binaries and macOS Package

Run the following command from the project root:

```bash
./scripts/build_release.sh
```

This script will:
1. Clean the `build` directory.
2. Compile binaries for:
   - Windows (amd64, 386)
   - macOS (Universal Binary: amd64 + arm64)
   - Linux (amd64)
3. Create the macOS installer package (`dist/cleanup-1.0.0.pkg`).

### 2. Build Windows Installer

Since Inno Setup is a Windows tool, you must perform this step on a Windows machine (or using Wine).

1. Ensure the binaries are built (Step 1).
2. Install [Inno Setup](https://jrsoftware.org/isinfo.php).
3. Open `packaging/windows/cleanup.iss`.
4. Click **Build > Compile**.
5. The installer (`cleanup-setup.exe`) will be generated in the `dist` directory.

## Installer Features

### Windows Installer (.exe)
- **Wizard Interface**: Graphical installation wizard supporting English and Simplified Chinese.
- **Architecture Detection**: Automatically installs the 64-bit binary on 64-bit systems and 32-bit on 32-bit systems.
- **Path Configuration**: Automatically adds the installation directory to the user's `PATH` environment variable.
- **Shortcuts**: Creates shortcuts in the Start Menu and optionally on the Desktop.
- **Uninstaller**: Registers in Control Panel "Programs and Features".

### macOS Installer (.pkg)
- **Standard Installer**: Uses the native macOS Installer interface.
- **System Integration**: Installs the binary to `/usr/local/bin`, which is in the default `PATH`.
- **Universal Binary**: Runs natively on both Intel and Apple Silicon Macs.

## Silent Installation

### Windows
The Windows installer supports standard Inno Setup silent install parameters:

- `/SILENT`: Runs the installer and displays the progress window.
- `/VERYSILENT`: Runs the installer without displaying any window.
- `/SUPPRESSMSGBOXES`: Suppresses message boxes.
- `/NORESTART`: Prevents a restart even if necessary.
- `/DIR="x:\dirname"`: Overrides the default installation directory.

Example:
```cmd
cleanup-setup.exe /VERYSILENT /DIR="C:\Tools\Cleanup"
```

### macOS
You can install the `.pkg` file silently using the `installer` command (requires sudo):

```bash
sudo installer -pkg dist/cleanup-1.0.0.pkg -target /
```

## Verification

After installation, verify the installation by running the following command in a new terminal window:

```bash
cleanup version
```

It should output the version information, confirming that the tool is in your `PATH` and executable.

## Troubleshooting

- **Windows PATH not updating**: The installer updates the `HKCU` environment variable. You may need to restart your terminal or log out and log back in for changes to take effect.
- **macOS install failed**: Ensure you have administrative privileges. The installer writes to `/usr/local/bin`.

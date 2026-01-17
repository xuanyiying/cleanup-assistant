# Cleanup CLI - User Guide

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [Quick Start](#quick-start)
4. [Features](#features)
5. [Configuration](#configuration)
6. [Usage Examples](#usage-examples)
7. [Best Practices](#best-practices)
8. [Troubleshooting](#troubleshooting)

## Introduction

Cleanup CLI is an intelligent file organization tool that helps you automatically organize, rename, and clean up your files using AI-powered suggestions and customizable rules.

### Key Features

- 🤖 AI-powered file naming suggestions
- 📁 Rule-based file organization
- 🔄 Transaction support with rollback
- ⚡ High-performance concurrent processing
- 🔒 Safe file operations with locking
- 📊 Detailed operation reports

## Installation

### Using Homebrew (macOS)

```bash
brew tap yourusername/cleanup
brew install cleanup
```

### Using Install Script

```bash
curl -sSL https://raw.githubusercontent.com/yourusername/cleanup-cli/main/install.sh | bash
```

### From Source

```bash
git clone https://github.com/yourusername/cleanup-cli.git
cd cleanup-cli
go build -o cleanup ./cmd/cleanup
sudo mv cleanup /usr/local/bin/
```

### Verify Installation

```bash
cleanup --version
```

## Quick Start

### Basic Usage

1. **Scan a directory**

   ```bash
   cleanup scan ~/Downloads
   ```

2. **Organize files**

   ```bash
   cleanup organize ~/Downloads
   ```

3. **Preview changes (dry-run)**
   ```bash
   cleanup organize ~/Downloads --dry-run
   ```

### First-Time Setup

Run the setup wizard:

```bash
cleanup setup
```

This will:

- Create configuration file
- Set up default rules
- Configure AI integration (optional)

## Features

### 1. Intelligent File Naming

Cleanup CLI can analyze file content and suggest meaningful names:

```bash
# Rename files with AI suggestions
cleanup rename ~/Downloads --ai
```

**Example:**

- `IMG_1234.jpg` → `sunset_beach_vacation.jpg`
- `document.pdf` → `project_proposal_2024.pdf`

### 2. Rule-Based Organization

Organize files based on customizable rules:

```bash
# Organize by file type
cleanup organize ~/Downloads --by-type

# Organize by date
cleanup organize ~/Downloads --by-date

# Use custom rules
cleanup organize ~/Downloads --config ~/.cleanuprc.yaml
```

### 3. Safe Operations

All operations are transactional and can be undone:

```bash
# Undo last operation
cleanup undo

# View operation history
cleanup history

# Rollback specific transaction
cleanup rollback <transaction-id>
```

### 4. Duplicate Detection

Find and manage duplicate files:

```bash
# Find duplicates
cleanup duplicates ~/Downloads

# Remove duplicates (interactive)
cleanup duplicates ~/Downloads --remove
```

## Configuration

### Configuration File

Create `~/.cleanuprc.yaml`:

```yaml
# General settings
concurrency: 4
dry_run: false

# AI settings
ai:
  enabled: true
  provider: ollama
  model: llama2

# Organization rules
rules:
  - name: "Images by date"
    pattern: "*.{jpg,png,gif}"
    action:
      type: move
      target: "~/Pictures/{year}/{month}"

  - name: "Documents by type"
    pattern: "*.{pdf,doc,docx}"
    action:
      type: move
      target: "~/Documents/{category}"

# Exclusions
exclude:
  extensions: [tmp, log, cache]
  patterns: [".*", "node_modules"]
  directories: [".git", ".svn"]
```

### Environment Variables

```bash
# AI provider
export CLEANUP_AI_PROVIDER=ollama

# Concurrency level
export CLEANUP_CONCURRENCY=8

# Enable debug logging
export CLEANUP_DEBUG=true
```

## Usage Examples

### Example 1: Clean Downloads Folder

```bash
# Preview what will happen
cleanup organize ~/Downloads --dry-run

# Execute organization
cleanup organize ~/Downloads

# View results
cleanup history --last
```

### Example 2: Rename Screenshots

```bash
# Find all screenshots
cleanup scan ~/Desktop --pattern "Screenshot*.png"

# Rename with AI
cleanup rename ~/Desktop --pattern "Screenshot*.png" --ai

# Result: Screenshot 2024-01-17.png → meeting_notes_diagram.png
```

### Example 3: Organize Photos

```bash
# Organize photos by date
cleanup organize ~/Pictures \
  --by-date \
  --format "{year}/{month}/{day}"

# Result: IMG_1234.jpg → 2024/01/17/IMG_1234.jpg
```

### Example 4: Clean Up Project Directory

```bash
# Remove build artifacts
cleanup clean ~/project \
  --pattern "*.{o,pyc,class}" \
  --directories "build,dist,target"

# Move logs to archive
cleanup organize ~/project/logs \
  --target ~/Archives/logs/{year}-{month}
```

## Best Practices

### 1. Always Preview First

Use `--dry-run` to preview changes:

```bash
cleanup organize ~/Downloads --dry-run
```

### 2. Start with Small Directories

Test on a small directory first:

```bash
cleanup organize ~/test-folder
```

### 3. Use Exclusions

Exclude system files and directories:

```yaml
exclude:
  patterns: [".*", "Thumbs.db", "Desktop.ini"]
  directories: [".git", "node_modules"]
```

### 4. Regular Backups

Always maintain backups of important files:

```bash
# Backup before organizing
tar -czf backup-$(date +%Y%m%d).tar.gz ~/Downloads
cleanup organize ~/Downloads
```

### 5. Review History

Regularly review operation history:

```bash
cleanup history --last 10
```

## Troubleshooting

### Common Issues

#### Issue: "Permission denied"

**Solution:**

```bash
# Check file permissions
ls -la /path/to/file

# Run with appropriate permissions
sudo cleanup organize /path/to/directory
```

#### Issue: "AI suggestions not working"

**Solution:**

1. Check AI provider is running:

   ```bash
   ollama list
   ```

2. Verify configuration:

   ```bash
   cleanup config --show
   ```

3. Test AI connection:
   ```bash
   cleanup test-ai
   ```

#### Issue: "Operation failed, files not moved"

**Solution:**

1. Check transaction log:

   ```bash
   cleanup history --failed
   ```

2. Rollback if needed:

   ```bash
   cleanup rollback <transaction-id>
   ```

3. Check disk space:
   ```bash
   df -h
   ```

#### Issue: "Slow performance"

**Solution:**

1. Increase concurrency:

   ```bash
   cleanup organize ~/Downloads --workers 8
   ```

2. Disable hash calculation:

   ```bash
   cleanup organize ~/Downloads --no-hash
   ```

3. Use exclusions to skip unnecessary files

### Getting Help

- **Documentation**: Check `docs/` directory
- **Issues**: Report bugs on GitHub
- **Community**: Join discussions

### Debug Mode

Enable debug logging:

```bash
cleanup organize ~/Downloads --debug
```

Or set environment variable:

```bash
export CLEANUP_DEBUG=true
cleanup organize ~/Downloads
```

## Advanced Usage

### Custom Rules

Create complex organization rules:

```yaml
rules:
  - name: "Work documents"
    conditions:
      - path_contains: "work"
      - extension: [pdf, docx]
      - size_gt: 1MB
    action:
      type: move
      target: "~/Documents/Work/{year}"
```

### Scripting

Use Cleanup CLI in scripts:

```bash
#!/bin/bash

# Organize downloads daily
cleanup organize ~/Downloads --quiet

# Clean up old files
cleanup clean ~/Downloads \
  --older-than 30d \
  --move-to ~/Archives

# Send notification
echo "Cleanup complete" | mail -s "Daily Cleanup" user@example.com
```

### Integration

Integrate with other tools:

```bash
# Find large files and organize
find ~/Downloads -size +100M -exec cleanup organize {} \;

# Watch directory and auto-organize
fswatch ~/Downloads | xargs -n1 cleanup organize
```

## Performance Tips

1. **Use appropriate concurrency**: Default is 4, increase for faster systems
2. **Skip hash calculation**: Use `--no-hash` if not needed
3. **Use exclusions**: Skip unnecessary files and directories
4. **Batch operations**: Process multiple directories in one command

## Safety Features

- ✅ Automatic backups before overwriting
- ✅ Transaction logging for all operations
- ✅ Rollback support
- ✅ Dry-run mode for previewing
- ✅ File locking for concurrent safety

## Next Steps

- Read [API Documentation](./API_DOCUMENTATION.md) for developers
- Check [FAQ](./FAQ.md) for common questions
- See [Examples](./EXAMPLES.md) for more use cases

---

For more information, visit the [project repository](https://github.com/yourusername/cleanup-cli).

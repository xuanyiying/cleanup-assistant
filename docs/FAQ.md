# Cleanup CLI - Frequently Asked Questions

## General Questions

### What is Cleanup CLI?

Cleanup CLI is an intelligent file organization tool that uses AI and customizable rules to automatically organize, rename, and clean up your files.

### Is it safe to use?

Yes! Cleanup CLI includes multiple safety features:

- All operations are transactional and can be rolled back
- Automatic backups before overwriting files
- Dry-run mode to preview changes
- File locking to prevent concurrent conflicts

### What platforms are supported?

- macOS (primary support)
- Linux (tested)
- Windows (experimental)

### Do I need AI to use Cleanup CLI?

No, AI features are optional. You can use rule-based organization without AI.

## Installation & Setup

### How do I install Cleanup CLI?

See the [Installation Guide](./USER_GUIDE.md#installation) for detailed instructions.

### Where is the configuration file located?

Default location: `~/.cleanuprc.yaml`

You can specify a custom location:

```bash
cleanup organize ~/Downloads --config /path/to/config.yaml
```

### How do I set up AI integration?

1. Install Ollama or configure OpenAI
2. Update configuration:
   ```yaml
   ai:
     enabled: true
     provider: ollama
     model: llama2
   ```
3. Test connection:
   ```bash
   cleanup test-ai
   ```

## Usage Questions

### How do I preview changes before applying them?

Use the `--dry-run` flag:

```bash
cleanup organize ~/Downloads --dry-run
```

### Can I undo an operation?

Yes! Use the undo command:

```bash
cleanup undo
```

Or rollback a specific transaction:

```bash
cleanup rollback <transaction-id>
```

### How do I organize files by date?

```bash
cleanup organize ~/Downloads --by-date --format "{year}/{month}"
```

### How do I rename files with AI?

```bash
cleanup rename ~/Downloads --ai
```

### Can I organize multiple directories at once?

Yes:

```bash
cleanup organize ~/Downloads ~/Desktop ~/Documents
```

### How do I exclude certain files?

In your configuration file:

```yaml
exclude:
  extensions: [tmp, log]
  patterns: [".*", "Thumbs.db"]
  directories: [".git", "node_modules"]
```

Or via command line:

```bash
cleanup organize ~/Downloads --exclude "*.tmp,*.log"
```

## Performance Questions

### Why is scanning slow?

Possible reasons:

1. Large number of files
2. Hash calculation enabled (default)
3. Low concurrency setting

Solutions:

```bash
# Increase workers
cleanup organize ~/Downloads --workers 8

# Disable hash calculation
cleanup organize ~/Downloads --no-hash

# Use exclusions
cleanup organize ~/Downloads --exclude "node_modules"
```

### How can I make it faster?

1. Increase concurrency:

   ```bash
   cleanup organize ~/Downloads --workers 8
   ```

2. Skip unnecessary operations:

   ```bash
   cleanup organize ~/Downloads --no-hash --no-ai
   ```

3. Use exclusions to skip files

### Does it use a lot of memory?

Memory usage depends on:

- Number of files being processed
- Whether hash calculation is enabled
- AI cache size

For very large directories (>10,000 files), consider:

- Processing in batches
- Disabling hash calculation
- Using exclusions

## AI Questions

### Which AI providers are supported?

- Ollama (local, recommended)
- OpenAI (API key required)
- Custom providers (via configuration)

### Do I need an internet connection for AI?

Not if using Ollama (local AI). OpenAI requires internet.

### How accurate are AI suggestions?

Accuracy depends on:

- File content quality
- AI model used
- Context available

Always review suggestions before applying.

### Can I customize AI behavior?

Yes, in configuration:

```yaml
ai:
  enabled: true
  provider: ollama
  model: llama2
  temperature: 0.7
  max_tokens: 100
```

### Are AI responses cached?

Yes, responses are cached for 24 hours to improve performance and reduce API calls.

## Rules & Configuration

### How do I create custom rules?

Edit `~/.cleanuprc.yaml`:

```yaml
rules:
  - name: "My Rule"
    pattern: "*.pdf"
    action:
      type: move
      target: "~/Documents/{year}"
```

### Can I use multiple rule files?

Yes:

```bash
cleanup organize ~/Downloads --config rules1.yaml,rules2.yaml
```

### What placeholders are available in rules?

- `{year}` - Current year
- `{month}` - Current month
- `{day}` - Current day
- `{ext}` - File extension
- `{category}` - AI-detected category
- `{name}` - Original filename

### How are rules prioritized?

Rules are applied in order. First matching rule wins.

## Troubleshooting

### "Permission denied" error

Solutions:

1. Check file permissions:

   ```bash
   ls -la /path/to/file
   ```

2. Run with appropriate permissions:

   ```bash
   sudo cleanup organize /path/to/directory
   ```

3. Check directory ownership

### "Transaction failed" error

Solutions:

1. Check transaction log:

   ```bash
   cleanup history --failed
   ```

2. Rollback transaction:

   ```bash
   cleanup rollback <transaction-id>
   ```

3. Check disk space:
   ```bash
   df -h
   ```

### AI not working

Solutions:

1. Verify AI provider is running:

   ```bash
   ollama list  # for Ollama
   ```

2. Check configuration:

   ```bash
   cleanup config --show
   ```

3. Test connection:

   ```bash
   cleanup test-ai
   ```

4. Check logs:
   ```bash
   cleanup --debug organize ~/Downloads
   ```

### Files not being organized

Possible reasons:

1. No matching rules
2. Files excluded by configuration
3. Dry-run mode enabled

Solutions:

1. Check rules:

   ```bash
   cleanup config --show-rules
   ```

2. Run with debug:

   ```bash
   cleanup --debug organize ~/Downloads
   ```

3. Verify not in dry-run mode

## Safety & Recovery

### What if something goes wrong?

1. Check transaction history:

   ```bash
   cleanup history
   ```

2. Rollback last operation:

   ```bash
   cleanup undo
   ```

3. Restore from backup (if you made one)

### Are my files safe?

Yes, with multiple safety layers:

- Transactional operations
- Automatic backups
- Rollback support
- File locking

### Can I recover deleted files?

Files are moved to trash, not permanently deleted:

```bash
# View trash
cleanup trash --list

# Restore from trash
cleanup trash --restore <file>
```

### How long are transactions kept?

Default: 30 days

Configure in `~/.cleanuprc.yaml`:

```yaml
transaction:
  retention_days: 90
```

## Advanced Usage

### Can I use Cleanup CLI in scripts?

Yes:

```bash
#!/bin/bash
cleanup organize ~/Downloads --quiet
```

### Can I schedule automatic cleanup?

Yes, using cron:

```bash
# Edit crontab
crontab -e

# Add daily cleanup at 2 AM
0 2 * * * /usr/local/bin/cleanup organize ~/Downloads --quiet
```

### Can I integrate with other tools?

Yes, Cleanup CLI works well with:

- `find` - Find files to organize
- `fswatch` - Watch directories for changes
- `cron` - Schedule automatic cleanup

### Can I extend Cleanup CLI?

Yes, through:

- Custom rules
- Plugin system (coming soon)
- API integration

## Performance & Limits

### How many files can it handle?

Tested with:

- ✅ 10,000 files - Excellent performance
- ✅ 50,000 files - Good performance
- ⚠️ 100,000+ files - Consider batch processing

### What's the maximum file size?

No hard limit, but:

- Hash calculation slower for large files
- Content preview limited to first 500 chars
- AI analysis works best with < 10MB files

### Can it handle network drives?

Yes, but:

- Performance may be slower
- Ensure proper permissions
- Consider network latency

## Getting Help

### Where can I get help?

- **Documentation**: Check `docs/` directory
- **Issues**: GitHub Issues for bugs
- **Discussions**: GitHub Discussions for questions
- **Email**: support@example.com

### How do I report a bug?

1. Check existing issues
2. Create new issue with:
   - Steps to reproduce
   - Expected vs actual behavior
   - System information
   - Debug logs

### How do I request a feature?

1. Check existing feature requests
2. Create new issue with:
   - Use case description
   - Proposed solution
   - Examples

### Is there a community?

- GitHub Discussions
- Discord (coming soon)
- Twitter: @cleanup_cli

## Licensing & Privacy

### What license is Cleanup CLI under?

MIT License - free and open source.

### Does it collect data?

No. Cleanup CLI:

- Runs entirely locally
- No telemetry or tracking
- No data sent to external servers (except AI API if configured)

### Is my data private?

Yes:

- All processing is local
- AI can be run locally (Ollama)
- No cloud storage required

## Contributing

### Can I contribute?

Yes! See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

### What can I contribute?

- Code improvements
- Bug fixes
- Documentation
- Feature ideas
- Testing

### How do I get started?

1. Fork the repository
2. Read [CONTRIBUTING.md](./CONTRIBUTING.md)
3. Pick an issue or propose a feature
4. Submit a pull request

---

**Still have questions?** Open an issue on GitHub or join our community discussions!

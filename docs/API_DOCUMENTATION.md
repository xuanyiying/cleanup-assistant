# Cleanup CLI - API Documentation

## Overview

This document provides comprehensive API documentation for the Cleanup CLI's public packages and interfaces.

## Core Packages

### pkg/validator

Input validation utilities for filenames and paths.

#### Functions

##### `ValidateFilename(name string) error`

Validates a filename against common restrictions.

**Parameters:**

- `name` - The filename to validate

**Returns:**

- `error` - Validation error, or nil if valid

**Checks:**

- Non-empty filename
- No illegal characters (`/ \ : * ? " < > | \x00`)
- Not a Windows reserved name (CON, PRN, AUX, COM1-9, LPT1-9)
- Not consisting only of dots
- Length ≤ 255 characters

**Example:**

```go
if err := validator.ValidateFilename("my-file.txt"); err != nil {
    log.Fatal(err)
}
```

##### `ValidatePath(path string) error`

Validates a file path for safety.

**Parameters:**

- `path` - The path to validate

**Returns:**

- `error` - Validation error, or nil if valid

**Checks:**

- Non-empty path
- No parent directory references (`..`)

##### `SanitizeFilename(name string) string`

Sanitizes a filename by replacing invalid characters.

**Parameters:**

- `name` - The filename to sanitize

**Returns:**

- `string` - Sanitized filename

**Actions:**

- Replaces invalid characters with underscore
- Trims leading/trailing spaces, dots, and underscores
- Returns "unnamed" if empty after sanitization
- Truncates to 255 characters if too long

### pkg/fileutil

Safe file operation utilities.

#### Functions

##### `SafeRename(src, dst string) error`

Renames a file with automatic backup and rollback.

**Parameters:**

- `src` - Source file path
- `dst` - Destination file path

**Returns:**

- `error` - Operation error, or nil if successful

**Behavior:**

- Creates backup if destination exists
- Rolls back on failure
- Cleans up backup on success

##### `SafeMove(src, dstDir string) error`

Moves a file to a target directory.

**Parameters:**

- `src` - Source file path
- `dstDir` - Destination directory path

**Returns:**

- `error` - Operation error, or nil if successful

**Behavior:**

- Creates destination directory if needed
- Uses SafeRename internally

##### `CopyFile(src, dst string) error`

Copies a file with permission preservation.

**Parameters:**

- `src` - Source file path
- `dst` - Destination file path

**Returns:**

- `error` - Operation error, or nil if successful

**Behavior:**

- Copies file content
- Preserves file permissions
- Syncs to ensure data is written

##### `FileExists(path string) bool`

Checks if a file exists.

##### `DirExists(path string) bool`

Checks if a directory exists.

##### `EnsureDir(path string) error`

Ensures a directory exists, creating it if necessary.

##### `GetFileSize(path string) (int64, error)`

Returns the size of a file in bytes.

##### `IsEmpty(path string) (bool, error)`

Checks if a directory is empty.

### pkg/filelock

Thread-safe file locking.

#### Types

##### `LockManager`

Manages file locks to prevent concurrent operations.

**Methods:**

###### `NewLockManager() *LockManager`

Creates a new lock manager.

###### `Lock(path string) error`

Acquires a lock on a file path (blocking).

###### `Unlock(path string) error`

Releases a lock on a file path.

###### `TryLock(path string) bool`

Attempts to acquire a lock without blocking.

###### `IsLocked(path string) bool`

Checks if a file path is currently locked.

###### `WithLock(path string, fn func() error) error`

Executes a function while holding a lock.

###### `CleanupStale(maxAge time.Duration) int`

Removes locks that haven't been used recently.

###### `Size() int`

Returns the number of locks currently managed.

**Example:**

```go
lm := filelock.NewLockManager()

// Option 1: Manual locking
if err := lm.Lock("/path/to/file"); err != nil {
    return err
}
defer lm.Unlock("/path/to/file")

// Option 2: Automatic with function
err := lm.WithLock("/path/to/file", func() error {
    // Perform file operation
    return nil
})

// Option 3: Non-blocking
if lm.TryLock("/path/to/file") {
    defer lm.Unlock("/path/to/file")
    // Perform operation
}
```

### pkg/errors

Error handling utilities.

#### Functions

##### `WrapError(err error, format string, args ...interface{}) error`

Wraps an error with additional context.

**Parameters:**

- `err` - The error to wrap
- `format` - Format string for context message
- `args` - Format arguments

**Returns:**

- `error` - Wrapped error, or nil if input is nil

**Example:**

```go
if err := operation(); err != nil {
    return errors.WrapError(err, "failed to process file: %s", filename)
}
```

##### `FirstError(errors ...error) error`

Returns the first non-nil error from a list.

##### `CombineErrors(errors []error) error`

Combines multiple errors into a single error.

## Internal Packages

### internal/analyzer

File metadata analysis and directory scanning.

#### Types

##### `FileAnalyzer`

Implements file analysis operations.

**Methods:**

###### `Analyze(ctx context.Context, path string) (*FileMetadata, error)`

Extracts complete metadata for a single file.

###### `AnalyzeDirectory(ctx context.Context, path string, opts *ScanOptions) ([]*FileMetadata, error)`

Scans a directory and returns metadata for all matching files.

**Features:**

- Concurrent scanning with worker pool
- Optional hash calculation
- File filtering and exclusion
- Context-aware cancellation

##### `ScanOptions`

Controls directory scanning behavior.

**Fields:**

- `Recursive bool` - Scan subdirectories
- `IncludeHidden bool` - Include hidden files
- `Filter *FileFilter` - File filtering criteria
- `ExcludeExtensions []string` - Extensions to exclude
- `ExcludePatterns []string` - Filename patterns to exclude
- `ExcludeDirs []string` - Directory names to exclude
- `CalculateHash bool` - Whether to calculate file hashes
- `Workers int` - Number of concurrent workers (default: 4)

### internal/organizer

File organization operations.

#### Types

##### `Organizer`

Handles file organization operations.

**Methods:**

###### `Rename(ctx context.Context, source, newName string, opts *RenameOptions) (*OperationResult, error)`

Renames a file with conflict resolution.

###### `Move(ctx context.Context, source, targetDir string, opts *MoveOptions) (*OperationResult, error)`

Moves a file to a target directory.

###### `Delete(ctx context.Context, source, trashDir string) (*OperationResult, error)`

Moves a file to trash.

###### `Organize(ctx context.Context, files []*FileMetadata, strategy *OrganizeStrategy) (*OrganizePlan, error)`

Generates an execution plan for organizing files.

**Features:**

- Concurrent AI processing with caching
- Rule-based file organization
- Conflict resolution strategies
- Transaction support

### internal/transaction

Transaction management for file operations.

#### Types

##### `Manager`

Handles transaction logging and rollback.

**Methods:**

###### `Begin() *Transaction`

Starts a new transaction.

###### `Commit(tx *Transaction) error`

Commits a transaction and persists it.

###### `Rollback(tx *Transaction) error`

Rolls back a transaction by reversing its operations.

**Features:**

- Error-tolerant rollback (continues on individual failures)
- Persistent transaction log
- Undo support

### internal/ai/cache

AI response caching.

#### Types

##### `Cache`

Provides thread-safe caching for AI responses.

**Methods:**

###### `NewCache(ttl time.Duration) *Cache`

Creates a new AI response cache.

###### `Get(key string) ([]string, bool)`

Retrieves a cached response.

###### `Set(key string, response []string)`

Stores a response in the cache.

###### `GenerateKey(prefix, content string) string`

Creates a cache key from file metadata.

## Best Practices

### Error Handling

Always wrap errors with context:

```go
if err := operation(); err != nil {
    return errors.WrapError(err, "operation failed")
}
```

### File Operations

Use safe utilities for file operations:

```go
// Instead of os.Rename
if err := fileutil.SafeRename(src, dst); err != nil {
    return err
}
```

### Concurrent Operations

Use file locking for concurrent file access:

```go
lm := filelock.NewLockManager()
err := lm.WithLock(filePath, func() error {
    // Perform file operation
    return nil
})
```

### Input Validation

Always validate user input:

```go
if err := validator.ValidateFilename(userInput); err != nil {
    return fmt.Errorf("invalid filename: %w", err)
}
```

## Performance Considerations

### File Scanning

- Use `Workers` parameter to control concurrency
- Set `CalculateHash: false` if hashes aren't needed
- Use exclusion patterns to skip unnecessary files

### AI Processing

- AI responses are automatically cached for 24 hours
- Concurrent processing with configurable workers
- Cache keys based on file content

### Memory Usage

- File metadata is loaded into memory
- For very large directories (>10,000 files), consider batch processing
- Use exclusion patterns to reduce memory footprint

## Thread Safety

All public APIs are thread-safe:

- `filelock.LockManager` - Thread-safe lock management
- `ai.Cache` - Thread-safe caching with RWMutex
- `transaction.Manager` - Thread-safe transaction management

## Error Types

Common error patterns:

- Validation errors: Input validation failures
- File operation errors: I/O failures, permission issues
- Transaction errors: Commit/rollback failures
- Lock errors: Lock acquisition failures

All errors support Go 1.13+ error wrapping with `errors.Is()` and `errors.As()`.

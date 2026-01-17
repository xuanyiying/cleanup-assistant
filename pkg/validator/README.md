# Validator Package

The validator package provides utilities for validating and sanitizing file names and paths to ensure security and cross-platform compatibility.

## Functions

### ValidateFilename

Validates a filename against common restrictions and security concerns.

```go
func ValidateFilename(name string) error
```

**Checks**:

- Non-empty filename
- No illegal characters: `/ \ : * ? " < > | \x00`
- Not a Windows reserved name (CON, PRN, AUX, COM1-9, LPT1-9)
- Not consisting only of dots
- Length ≤ 255 characters

**Example**:

```go
if err := validator.ValidateFilename("my-file.txt"); err != nil {
    log.Fatal(err)
}
```

### ValidatePath

Validates a file path for safety, preventing path traversal attacks.

```go
func ValidatePath(path string) error
```

**Checks**:

- Non-empty path
- No parent directory references (`..`)

**Example**:

```go
if err := validator.ValidatePath("documents/file.txt"); err != nil {
    log.Fatal(err)
}
```

### SanitizeFilename

Sanitizes a filename by replacing invalid characters and ensuring it meets requirements.

```go
func SanitizeFilename(name string) string
```

**Actions**:

- Replaces invalid characters with underscore
- Trims leading/trailing spaces, dots, and underscores
- Returns "unnamed" if empty after sanitization
- Truncates to 255 characters if too long

**Example**:

```go
safe := validator.SanitizeFilename("my/file:name?.txt")
// Result: "my_file_name_.txt"
```

## Usage in Cleanup CLI

The validator package is used throughout the codebase to ensure file operations are safe:

```go
// In organizer.Rename()
if err := validator.ValidateFilename(newName); err != nil {
    return &OperationResult{
        Success: false,
        Error:   fmt.Errorf("invalid filename: %w", err),
    }, nil
}
```

## Cross-Platform Compatibility

The validator handles platform-specific restrictions:

- **Windows**: Reserved names (CON, PRN, etc.)
- **All platforms**: Illegal characters, length limits
- **Security**: Path traversal prevention

## Testing

The package includes comprehensive tests covering:

- Valid and invalid filenames
- Edge cases (empty, too long, special characters)
- Reserved names
- Path traversal attempts
- Sanitization behavior

Run tests:

```bash
go test ./pkg/validator/...
```

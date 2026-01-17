# Contributing to Cleanup CLI

Thank you for your interest in contributing to Cleanup CLI! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Make (optional, for using Makefile commands)

### Getting Started

1. **Fork and Clone**

   ```bash
   git clone https://github.com/yourusername/cleanup-cli.git
   cd cleanup-cli
   ```

2. **Install Dependencies**

   ```bash
   go mod download
   ```

3. **Build the Project**

   ```bash
   go build -o cleanup ./cmd/cleanup
   ```

4. **Run Tests**
   ```bash
   go test ./...
   ```

## Project Structure

```
cleanup-cli/
├── cmd/cleanup/          # Main application entry point
├── internal/             # Internal packages
│   ├── analyzer/         # File analysis
│   ├── organizer/        # File organization
│   ├── transaction/      # Transaction management
│   ├── ai/              # AI integration
│   └── ...
├── pkg/                  # Public packages
│   ├── validator/        # Input validation
│   ├── fileutil/         # File utilities
│   ├── filelock/         # File locking
│   └── errors/           # Error handling
├── docs/                 # Documentation
└── integration_test/     # Integration tests
```

## Coding Standards

### Go Style Guide

Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines and:

- Use `gofmt` to format code
- Use `golint` for linting
- Follow Go naming conventions
- Write clear, self-documenting code

### Code Organization

- Keep functions small and focused
- Use meaningful variable and function names
- Add comments for exported functions (godoc format)
- Group related functionality in packages

### Example

```go
// AnalyzeFile extracts metadata from a file.
//
// It returns FileMetadata containing information about the file including
// name, size, type, and content preview for text files.
//
// Example:
//
//	metadata, err := analyzer.AnalyzeFile(ctx, "/path/to/file.txt")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("File: %s, Size: %d\n", metadata.Name, metadata.Size)
func AnalyzeFile(ctx context.Context, path string) (*FileMetadata, error) {
    // Implementation
}
```

## Testing

### Writing Tests

- Write tests for all new functionality
- Aim for 80%+ code coverage
- Use table-driven tests where appropriate
- Include edge cases and error scenarios

### Test Structure

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "expected", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/analyzer/...

# Run with race detector
go test -race ./...
```

## Pull Request Process

### Before Submitting

1. **Create a Branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Changes**
   - Write code following our standards
   - Add tests for new functionality
   - Update documentation as needed

3. **Test Your Changes**

   ```bash
   go test ./...
   go test -race ./...
   ```

4. **Format Code**

   ```bash
   gofmt -w .
   ```

5. **Commit Changes**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks

**Examples:**

```
feat(analyzer): add concurrent file scanning

Implement worker pool pattern for parallel file analysis.
This improves scanning speed by 3-5x for large directories.

Closes #123
```

### Submitting Pull Request

1. **Push to Your Fork**

   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create Pull Request**
   - Go to GitHub and create a PR
   - Fill out the PR template
   - Link related issues

3. **PR Requirements**
   - All tests must pass
   - Code coverage should not decrease
   - Documentation updated if needed
   - No merge conflicts

4. **Code Review**
   - Address reviewer feedback
   - Make requested changes
   - Push updates to your branch

## Development Workflow

### Adding a New Feature

1. **Plan the Feature**
   - Discuss in an issue first
   - Get feedback on approach
   - Consider backward compatibility

2. **Implement**
   - Write code in small, logical commits
   - Add tests as you go
   - Update documentation

3. **Test Thoroughly**
   - Unit tests
   - Integration tests if needed
   - Manual testing

4. **Document**
   - Add godoc comments
   - Update API documentation
   - Add examples if helpful

### Fixing a Bug

1. **Reproduce the Bug**
   - Write a failing test
   - Verify the issue

2. **Fix the Bug**
   - Make minimal changes
   - Ensure test passes

3. **Prevent Regression**
   - Add test coverage
   - Consider edge cases

## Package Guidelines

### Creating New Packages

- Use `internal/` for internal packages
- Use `pkg/` for reusable public packages
- Keep packages focused and cohesive
- Minimize dependencies

### Package Documentation

Every package should have:

- Package-level documentation
- Examples in godoc
- README if complex

## Performance Considerations

- Profile before optimizing
- Use benchmarks to measure improvements
- Consider memory allocations
- Use concurrent patterns appropriately

### Writing Benchmarks

```go
func BenchmarkFunction(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Function()
    }
}
```

## Documentation

### Types of Documentation

1. **Code Comments** - Godoc format
2. **API Documentation** - `docs/API_DOCUMENTATION.md`
3. **User Guides** - `docs/` directory
4. **Examples** - In code and docs

### Documentation Standards

- Clear and concise
- Include examples
- Explain why, not just what
- Keep up to date

## Getting Help

- **Issues**: Open an issue for bugs or features
- **Discussions**: Use GitHub Discussions for questions
- **Documentation**: Check `docs/` directory

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Follow GitHub's Community Guidelines

## License

By contributing, you agree that your contributions will be licensed under the project's license.

## Recognition

Contributors will be recognized in:

- CONTRIBUTORS.md file
- Release notes
- Project README

Thank you for contributing to Cleanup CLI!

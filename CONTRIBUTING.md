# Contributing to Multi-NIC URL Crawler v10

First off, thank you for considering contributing to the Multi-NIC URL Crawler! It's people like you that make this project better.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Pull Requests](#pull-requests)
- [Development Setup](#development-setup)
- [Coding Standards](#coding-standards)
- [Commit Messages](#commit-messages)
- [Testing](#testing)
- [Documentation](#documentation)

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

**Bug Report Template:**

```markdown
**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Configure '...'
2. Run command '...'
3. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Environment:**
 - OS: [e.g. Linux, Ubuntu 22.04]
 - Go Version: [e.g. 1.24]
 - Crawler Version: [e.g. v10.0]
 - Network Interfaces: [e.g. 2x 10GbE]

**Logs**
Please include relevant logs or error messages.

**Additional context**
Add any other context about the problem here.
```

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

- **Use a clear and descriptive title**
- **Provide a detailed description** of the suggested enhancement
- **Explain why this enhancement would be useful** to most users
- **List any similar features** in other tools if applicable

### Pull Requests

1. **Fork the repository** and create your branch from `develop`
2. **Follow the coding standards** outlined below
3. **Add tests** if you're adding functionality
4. **Update documentation** if needed
5. **Ensure the test suite passes** (`make test`)
6. **Run linters** (`make lint`)
7. **Format your code** (`make fmt`)
8. **Submit your pull request**

**Pull Request Process:**

1. Update the README.md or relevant documentation with details of changes if applicable
2. Update the CHANGELOG.md with notes about your changes
3. The PR will be merged once you have approval from at least one maintainer

## Development Setup

### Prerequisites

- Go 1.24 or higher
- Git
- Make (optional, but recommended)

### Setup Steps

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/url_crawler_twotier.git
cd url_crawler_twotier

# Add upstream remote
git remote add upstream https://github.com/ORIGINAL_OWNER/url_crawler_twotier.git

# Create a new branch
git checkout -b feature/your-feature-name

# Install dependencies
make deps-download

# Verify setup
make check
```

### Building

```bash
# Build the project
make build

# Run the crawler
./bin/url_crawler_twotier
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run benchmarks
make bench
```

## Coding Standards

### Go Code Style

We follow the standard Go code style guidelines:

- **Use `gofmt`** for formatting (automatically done with `make fmt`)
- **Follow [Effective Go](https://golang.org/doc/effective_go.html)** guidelines
- **Use meaningful variable names** (avoid single-letter names except for short-lived variables)
- **Add comments** for exported functions, types, and constants
- **Keep functions focused** - a function should do one thing well
- **Avoid deep nesting** - prefer early returns

### Package Organization

- Each package should have a **single, well-defined responsibility**
- Use **internal packages** for code that shouldn't be imported elsewhere
- Keep **package-level documentation** in `doc.go` files

### Error Handling

```go
// Good - explicit error handling
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to process: %w", err)
}

// Bad - ignoring errors
result, _ := someFunction()
```

### Concurrency

- Use **channels** for communication between goroutines
- Protect shared state with **mutexes** or use **atomic operations**
- Always document **goroutine lifetimes** and shutdown procedures
- Use **context.Context** for cancellation and timeouts

### Comments

```go
// Package crawler implements the two-tier web crawling system.
//
// The crawler uses an intelligent fast/slow path routing mechanism
// to optimize HTML processing based on page complexity.
package crawler

// ProcessPage analyzes an HTML page and routes it to either the fast
// or slow processing path based on link density.
//
// Fast path is used for pages with link density < 0.05 (simple pages).
// Slow path is used for pages with link density >= 0.05 (complex pages).
//
// Returns the extracted URLs and any error encountered.
func ProcessPage(html []byte) ([]string, error) {
    // Implementation
}
```

## Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that don't affect code meaning (formatting, etc.)
- **refactor**: Code change that neither fixes a bug nor adds a feature
- **perf**: Performance improvement
- **test**: Adding or updating tests
- **chore**: Changes to build process or auxiliary tools

### Examples

```
feat(tokenizer): add adaptive threshold for fast/slow path selection

Implement dynamic threshold adjustment based on historical performance
metrics. This allows the tokenizer to adapt to different website
structures automatically.

Closes #123
```

```
fix(downloader): prevent worker pool deadlock on shutdown

Add proper context cancellation handling to ensure workers exit cleanly
when the application is shutting down.

Fixes #456
```

## Testing

### Writing Tests

- Write **table-driven tests** when testing multiple scenarios
- Use **subtests** (`t.Run()`) for better organization
- Test **edge cases** and error conditions
- Use **mocks** for external dependencies

### Test Example

```go
func TestURLNormalization(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid URL",
            input:    "https://example.com/path",
            expected: "https://example.com/path",
            wantErr:  false,
        },
        {
            name:     "URL with fragment",
            input:    "https://example.com/path#section",
            expected: "https://example.com/path",
            wantErr:  false,
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := NormalizeURL(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("NormalizeURL() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.expected {
                t.Errorf("NormalizeURL() = %v, want %v", got, tt.expected)
            }
        })
    }
}
```

### Benchmarks

Write benchmarks for performance-critical code:

```go
func BenchmarkFastPath(b *testing.B) {
    html := []byte("<html><body><a href='/test'>Link</a></body></html>")
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _, _ = FastPath(html)
    }
}
```

## Documentation

### Code Documentation

- All **exported functions, types, and constants** must have comments
- Package-level documentation goes in a `doc.go` file or the main package file
- Use **complete sentences** in comments
- Include **examples** in documentation when helpful

### Project Documentation

When adding features, update:

- **README.md**: If it affects usage or features
- **ARCHITECTURE.md**: If it changes the system architecture
- **Migration guides**: If it breaks backward compatibility
- **API documentation**: If it adds or changes APIs

### Example Documentation

```go
// Package monitor provides real-time performance monitoring and auto-scaling
// capabilities for the URL crawler.
//
// The monitor package tracks crawler performance metrics including:
//   - URLs crawled per second
//   - Download success/failure rates
//   - Worker pool utilization
//   - Memory usage and GC activity
//   - Network interface statistics
//
// It also implements automatic worker scaling based on queue depth and
// system resource utilization.
package monitor
```

## Performance Considerations

When contributing performance-sensitive code:

1. **Profile first**: Use `go test -bench` and `pprof` to identify bottlenecks
2. **Measure changes**: Benchmark before and after your changes
3. **Document trade-offs**: Explain why you chose one approach over another
4. **Consider memory**: Minimize allocations in hot paths

## Questions?

Feel free to:

- **Open an issue** with the "question" label
- **Start a discussion** on GitHub Discussions
- **Reach out** to the maintainers

Thank you for contributing! 🎉

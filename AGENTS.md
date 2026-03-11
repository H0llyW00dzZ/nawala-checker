# AGENTS.md - Nawala Checker Development Guidelines

This document provides comprehensive guidelines for agentic coding assistants working on the nawala-checker codebase. It covers linting, testing, code style, and development practices.

## Lint/Test Commands

### Testing Commands
- `make test` - Run all tests with race detector (mirrors CI)
- `make test-verbose` - Run tests with verbose output and race detector
- `make test-cover` - Run tests with coverage report and race detector, writes to `coverage.txt`
- `make test-short` - Run unit tests only (skip live DNS tests)
- `go test -race -run TestSpecificFunction ./src/nawala/...` - Run a single test function
- `go test -race -run TestSpecificFunction$ ./src/nawala/...` - Run exactly one test function (with $ anchor)
- `go test -race -run "TestCheck.*" ./src/nawala/...` - Run tests matching a pattern

### Linting and Formatting
- `go vet ./...` - Run go vet (static analysis)
- `gofmt -s -w .` - Format code with gofmt (simplify and write)
- `gofmt -d .` - Show formatting differences without modifying files

### Coverage Analysis
- `go tool cover -html=coverage.txt` - View coverage report in browser
- `go tool cover -func=coverage.txt` - Show coverage percentages per function

## Code Style Guidelines

### General Principles
- **GoDoc Comments**: Use GoDoc comments for all exported functions, types, and methods. Follow standard Go documentation conventions.
- **Idiomatic Go**: Follow effective Go patterns and standard library conventions.
- **Functional Options**: All configuration uses the functional options pattern defined in `option.go`.
- **Context-first**: All I/O-performing methods accept `context.Context` as the first parameter.
- **Typed Errors**: Use sentinel errors with `errors.Is()` and `errors.As()` for error checking.

### Imports
```go
import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/miekg/dns"
    "github.com/stretchr/testify/assert"
)
```
- Standard library imports first, grouped by blank lines
- Third-party imports second, alphabetically sorted
- Local imports (if any) last

### Naming Conventions
- **Exported functions/types**: PascalCase (e.g., `NewChecker`, `WithTimeout`)
- **Unexported functions/types**: camelCase (e.g., `queryWithRetries`, `defaultServers`)
- **Constants**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase throughout
- **Test functions**: `TestFunctionName` or `TestFunctionName_Scenario`

### Error Handling
- Use sentinel errors defined in `errors.go`:
  - `ErrNoDNSServers`
  - `ErrAllDNSFailed`
  - `ErrInvalidDomain`
  - `ErrDNSTimeout`
  - `ErrInternalPanic`
  - `ErrNXDOMAIN`
  - `ErrQueryRejected`

- Error checking patterns:
```go
if errors.Is(err, ErrInvalidDomain) {
    // handle invalid domain
}
```

- Wrap errors with context when appropriate:
```go
return fmt.Errorf("failed to query %s: %w", domain, err)
```

### Types and Structs
- Define clear, minimal struct types in `types.go`
- Use meaningful field names with proper JSON tags when needed
- Implement interfaces cleanly (e.g., `Cache` interface)

### Constants
- Group related constants together
- Use meaningful names with units when applicable:
```go
const (
    defaultTimeout     = 5 * time.Second
    defaultRetries     = 2
    defaultCacheTTL    = 5 * time.Minute
    defaultConcurrency = 100
    defaultEDNS0Size   = 1232
)
```

### Functions and Methods
- Keep functions focused and single-purpose
- Use early returns to reduce nesting
- Handle panics gracefully in goroutines with recover

### Testing Patterns
- Use `testify/assert` and `testify/require` for assertions
- Table-driven tests for multiple test cases:
```go
func TestIsValidDomain(t *testing.T) {
    tests := []struct {
        name   string
        domain string
        want   bool
    }{
        {"valid .com", "example.com", true},
        {"invalid empty", "", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.want, IsValidDomain(tt.domain))
        })
    }
}
```

- Test both success and error cases
- Use `t.Parallel()` for independent tests
- Mock external dependencies when possible

### Project Structure

```
nawala-checker/
├── .github/            # CI workflows and Dependabot configuration
├── cmd/
│   └── nawala/         # CLI entry point
├── examples/           # Runnable usage examples (basic, custom, status)
├── internal/
│   └── cli/            # CLI package (commands, config, output)
├── Makefile            # Build and test shortcuts
└── src/
    └── nawala/         # Core SDK package (checker, cache, DNS, options, types)
```

### Security Considerations
- Never log or cache sensitive information
- Use timeout contexts for all network operations
- Validate input domains before processing
- Handle TLS connections securely (configurable skip verification for testing only)

### Performance Guidelines
- Use buffered channels for concurrency control
- Implement proper connection pooling for TCP/TLS
- Cache results with configurable TTL
- Avoid unnecessary allocations in hot paths

### CI/CD Integration
- All changes must pass `go vet` and tests with race detection
- Maintain or improve test coverage
- Update documentation for API changes
- Test across multiple Go versions (1.25.6+)

### Dependencies
- Minimize third-party dependencies
- Use only well-maintained, security-audited packages
- Pin versions in `go.mod` appropriately
- Update dependencies regularly via Dependabot

This document should be updated when development practices change. Always run `make test-cover` before submitting changes to ensure quality and coverage requirements are met.

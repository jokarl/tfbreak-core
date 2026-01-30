# tfbreak Developer Guide

This guide covers the internal architecture of tfbreak and how to contribute to its development.

## Contents

1. [Architecture Overview](architecture.md) - Core components and design decisions
2. [Contributing](#contributing)
3. [Development Setup](#development-setup)
4. [Testing](#testing)
5. [Adding Rules](#adding-rules)

## Development Setup

### Prerequisites

- Go 1.23 or later
- Git

### Clone and Build

```bash
git clone https://github.com/jokarl/tfbreak-core.git
cd tfbreak-core
go build -o tfbreak ./cmd/tfbreak
```

### Run Tests

```bash
go test ./...
```

### Run with Verbose Output

```bash
go run ./cmd/tfbreak check ./testdata/old ./testdata/new --verbose
```

## Project Structure

```
tfbreak-core/
├── cmd/
│   └── tfbreak/           # Main entry point
├── internal/
│   ├── annotation/        # Annotation parsing and matching
│   ├── cli/               # CLI commands (check, explain, init, version)
│   ├── config/            # Configuration loading and validation
│   ├── loader/            # Terraform module loading
│   ├── output/            # Output rendering (text, JSON)
│   ├── pathfilter/        # Glob pattern path filtering
│   ├── rules/             # Rule implementations
│   └── types/             # Core types (Finding, Severity, etc.)
├── plugin/                # Plugin discovery and runner
├── docs/                  # Documentation
├── testdata/              # Test fixtures
└── tfbreak-plugin-sdk/    # Plugin SDK (submodule)
```

## Adding Rules

### Rule Interface

Rules implement the `Rule` interface:

```go
type Rule interface {
    ID() string                    // Unique identifier (e.g., "BC001")
    Name() string                  // Human-readable name (e.g., "required-input-added")
    Description() string           // Short description
    DefaultSeverity() types.Severity
    Evaluate(old, new *types.ModuleSnapshot) []*types.Finding
}
```

### Creating a New Rule

1. Create a new file in `internal/rules/` (e.g., `bc999.go`)

2. Implement the rule:

```go
package rules

import "github.com/jokarl/tfbreak-core/internal/types"

func init() {
    Register(&BC999{})
}

type BC999 struct{}

func (r *BC999) ID() string          { return "BC999" }
func (r *BC999) Name() string        { return "my-new-rule" }
func (r *BC999) Description() string { return "Detects a specific breaking change" }
func (r *BC999) DefaultSeverity() types.Severity { return types.SeverityError }

func (r *BC999) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
    var findings []*types.Finding

    // Detection logic here
    // Compare old and new snapshots

    return findings
}
```

3. Add documentation:

```go
// In internal/rules/doc.go or a separate file
func init() {
    RegisterDocumentation("BC999", &Documentation{
        Summary:     "Brief summary of what this rule detects",
        Description: "Detailed explanation of the rule",
        Remediation: "How to fix issues found by this rule",
    })
}
```

4. Add tests in `bc999_test.go`

5. Update `docs/rules.md` with the rule documentation

### Rule Naming Conventions

- **BC*** - Breaking Change (severity: ERROR)
- **RC*** - Risky Change (severity: WARNING)
- Ranges:
  - 001-099: Variable rules
  - 100-199: Resource/module rules
  - 200-299: Version constraint rules
  - 300-399: Module call rules

## Contributing

### Code Style

- Follow Go conventions
- Run `go fmt` before committing
- Add tests for new functionality

### Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Update documentation
6. Submit a pull request

### Commit Message Format

Follow conventional commits:

```
type: short description

Longer description if needed.

Co-Authored-By: Your Name <email@example.com>
```

Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/rules/...

# Run with coverage
go test -cover ./...
```

### Integration Tests

Integration tests are in `internal/integration_test.go`:

```bash
go test -run Integration ./internal/...
```

### Test Fixtures

Test fixtures are in `testdata/`:

```
testdata/
├── old/          # Old module versions for testing
├── new/          # New module versions for testing
└── ...
```

## Release Process

tfbreak uses [release-please](https://github.com/googleapis/release-please) for automated releases.

1. Merge changes to `main`
2. release-please creates a release PR
3. Merge the release PR to trigger the release
4. GitHub Actions builds and publishes binaries

## Architecture Decisions

Architecture Decision Records (ADRs) are in `docs/adr/`:

- [ADR-0001](../adr/ADR-0001-project-inception-and-technology-stack.md) - Project inception and technology stack
- [ADR-0002](../adr/ADR-0002-plugin-architecture.md) - Plugin architecture

## Related Documentation

- [Architecture](architecture.md) - Detailed architecture overview
- [Rules Reference](../rules.md) - All built-in rules
- [User Guide](../user-guide/README.md) - User-facing documentation

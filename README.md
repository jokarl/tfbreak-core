# tfbreak

A static analysis tool that compares two Terraform configurations and reports breaking changes to module interfaces and state safety.

## What is tfbreak?

tfbreak analyzes Terraform module changes between two versions (e.g., a pull request diff) and detects changes that would:

- **Break callers** - Changes to inputs, outputs, or type constraints that would cause errors for existing module consumers
- **Destroy state** - Resource or module removals without proper `moved` blocks that would cause unintended destruction
- **Change behavior unexpectedly** - Default value changes, validation additions, or sensitivity changes that may surprise consumers

### tfbreak vs tflint

| Aspect | tflint | tfbreak |
|--------|--------|---------|
| Focus | Linting a single Terraform configuration | Comparing two configurations to detect breaking changes |
| Use case | Code quality, best practices | Semantic versioning, safe module releases |
| Analysis | Static analysis of one config | Diff analysis between old and new configs |
| Rules | Provider-specific linting rules | Breaking change detection rules |

tfbreak is complementary to tflint - use tflint for code quality and tfbreak for safe releases.

## Installation

### Using Go

```bash
go install github.com/jokarl/tfbreak-core/cmd/tfbreak@latest
```

### Binary Releases

Download pre-built binaries from the [GitHub Releases](https://github.com/jokarl/tfbreak-core/releases) page.

Available platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

### From Source

```bash
git clone https://github.com/jokarl/tfbreak-core.git
cd tfbreak-core
go build -o tfbreak ./cmd/tfbreak
```

## Quick Start

### Basic Usage

Compare two Terraform module versions:

```bash
# Directory mode - compare two local directories
tfbreak check ./old-version ./new-version

# Git ref mode - compare current directory against a branch/tag
tfbreak check --base main ./

# Compare two git refs directly
tfbreak check --base v1.0.0 --head v2.0.0
```

### Example Output

```
tfbreak - Terraform Breaking Change Detector

Comparing:
  Old: ./old-version
  New: ./new-version

Findings:

ERROR  BC001  required-input-added
  New required variable "api_key" has no default
  Location: variables.tf:15

WARNING  RC006  input-default-changed
  Variable "instance_type" default changed from "t3.micro" to "t3.small"
  Location: variables.tf:8

Summary: 1 error, 1 warning, 0 notices
Result: FAIL
```

### Git Ref Comparison

tfbreak can compare against git refs (branches, tags, commits) without manual checkout.

**Single-module repositories** (Terraform config at repo root):

```bash
# Compare current directory against a branch
tfbreak check --base main ./

# Compare against a tag
tfbreak check --base v1.0.0 ./

# Compare two git refs directly
tfbreak check --base v1.0.0 --head v2.0.0

# Compare refs in a remote repository
tfbreak check --repo https://github.com/org/terraform-aws-vpc --base v1.0.0 --head v2.0.0
```

**Monorepos** (multiple modules in subdirectories):

Use the `ref:path` syntax to specify which subdirectory to compare:

```bash
# Compare modules/vpc at main branch against local modules/vpc
tfbreak check --base main:modules/vpc ./modules/vpc

# Compare a module between two tags
tfbreak check --base v1.0.0:modules/vpc --head v2.0.0:modules/vpc

# Handle module renames between versions
tfbreak check --base v1:modules/old-vpc --head v2:modules/vpc

# Remote monorepo comparison
tfbreak check --repo https://github.com/org/infra --base v1:terraform/prod --head v2:terraform/prod
```

The `ref:path` syntax follows git's convention (like `git show REVISION:path`).

### CI Integration

```bash
# Fail on any errors (default)
tfbreak check ./old ./new --fail-on ERROR

# Fail on warnings too
tfbreak check ./old ./new --fail-on WARNING

# Output as JSON for programmatic processing
tfbreak check ./old ./new --format json

# CI pipeline: compare PR against base branch
tfbreak check --base origin/main ./
```

## Configuration

tfbreak looks for `.tfbreak.hcl` in the current directory or the old directory.

### Minimal Configuration

```hcl
version = 1
```

### Full Configuration Example

```hcl
version = 1

# Path filtering
paths {
  include = ["**/*.tf"]
  exclude = [".terraform/**", "**/examples/**"]
}

# Output settings
output {
  format = "text"  # text, json
  color  = "auto"  # auto, always, never
}

# CI policy
policy {
  fail_on                  = "ERROR"  # ERROR, WARNING, NOTICE
  treat_warnings_as_errors = false
}

# Annotation settings
annotations {
  enabled        = true
  require_reason = false
}

# Per-rule configuration
rules "BC001" {
  enabled  = true
  severity = "ERROR"
}
```

See [Configuration Reference](docs/user-guide/config.md) for all options.

## Suppressing Findings

Use inline annotations to suppress findings (tflint-style):

```hcl
# tfbreak:ignore required-input-added # intentional breaking change for v2.0
variable "new_required_var" {
  type = string
}
```

See [Annotations Guide](docs/user-guide/annotations.md) for more details.

## Built-in Rules

tfbreak includes rules for detecting:

| Category | Rules | Description |
|----------|-------|-------------|
| Variable Changes | BC001-BC005, RC003, RC006-RC008, RC012-RC013 | Input additions, removals, type changes |
| Output Changes | BC009-BC010, RC011 | Output removals and sensitivity changes |
| Resource/Module Changes | BC100-BC103, RC300-RC301 | State safety and moved blocks |
| Version Constraints | BC200-BC201 | Terraform and provider version changes |

See [Rules Reference](docs/rules.md) for detailed documentation of all rules.

## Plugin System

tfbreak supports plugins for provider-specific rules (e.g., Azure ForceNew detection).

```hcl
plugin "azurerm" {
  enabled = true
  version = "0.1.0"
  source  = "github.com/jokarl/tfbreak-ruleset-azurerm"
}
```

See [Plugin Guide](docs/user-guide/plugins.md) for details.

## Documentation

- [User Guide](docs/user-guide/README.md)
  - [Configuration Reference](docs/user-guide/config.md)
  - [Annotations Guide](docs/user-guide/annotations.md)
  - [Plugin Guide](docs/user-guide/plugins.md)
- [Rules Reference](docs/rules.md)
- [Developer Guide](docs/developer-guide/README.md)
  - [Architecture](docs/developer-guide/architecture.md)

## Commands

```bash
# Compare two directories
tfbreak check <old_dir> <new_dir> [flags]

# Compare against a git ref
tfbreak check --base <ref[:path]> [new_dir] [flags]

# Compare two git refs
tfbreak check --base <ref[:path]> --head <ref[:path]> [flags]

# Compare remote repository refs
tfbreak check --repo <url> --base <ref[:path]> --head <ref[:path]> [flags]

# Show rule documentation
tfbreak explain <rule_id>

# Generate default config file
tfbreak init

# Show version
tfbreak version
```

### Check Command Flags

```
Git ref flags:
  --base string         Git ref for old config (branch, tag, SHA), supports ref:path syntax
  --head string         Git ref for new config, supports ref:path syntax
  --repo string         Remote repository URL (requires --base)

Output flags:
  --format string       Output format: text, json
  -o, --output string   Write output to file
  --color string        Color mode: auto, always, never
  -q, --quiet           Suppress non-error output
  -v, --verbose         Verbose output

Policy flags:
  --fail-on string      Fail on severity: ERROR, WARNING, NOTICE
  --enable strings      Enable specific rules
  --disable strings     Disable specific rules
  --severity strings    Override rule severity (RULE=SEV)

Config flags:
  -c, --config string   Path to config file
  --include strings     Include patterns
  --exclude strings     Exclude patterns

Annotation flags:
  --no-annotations      Disable annotation processing
  --require-reason      Require reason in annotations

Enhancement flags:
  --include-remediation Include remediation guidance
```

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

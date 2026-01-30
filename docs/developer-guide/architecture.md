# Architecture Overview

This document describes the internal architecture of tfbreak, including its core components, data flow, and design decisions.

## High-Level Architecture

```
                                    ┌─────────────────┐
                                    │   CLI (cobra)   │
                                    └────────┬────────┘
                                             │
              ┌──────────────────────────────┼──────────────────────────────┐
              │                              │                              │
              ▼                              ▼                              ▼
    ┌─────────────────┐            ┌─────────────────┐            ┌─────────────────┐
    │  Config Loader  │            │  Module Loader  │            │ Output Renderer │
    └────────┬────────┘            └────────┬────────┘            └────────┬────────┘
             │                              │                              │
             ▼                              ▼                              │
    ┌─────────────────┐            ┌─────────────────┐                     │
    │   .tfbreak.hcl  │            │  Old Snapshot   │                     │
    └─────────────────┘            │  New Snapshot   │                     │
                                   └────────┬────────┘                     │
                                            │                              │
                                            ▼                              │
                                   ┌─────────────────┐                     │
                                   │  Rules Engine   │                     │
                                   │  + Registry     │                     │
                                   └────────┬────────┘                     │
                                            │                              │
              ┌─────────────────────────────┼─────────────────────────────┐
              │                             │                             │
              ▼                             ▼                             ▼
    ┌─────────────────┐            ┌─────────────────┐            ┌─────────────────┐
    │  Built-in Rules │            │     Plugins     │            │   Annotations   │
    │  BC001, BC002...│            │  (via gRPC)     │            │   Processor     │
    └─────────────────┘            └─────────────────┘            └─────────────────┘
                                                                          │
                                                                          ▼
                                                                 ┌─────────────────┐
                                                                 │   Findings      │
                                                                 │   (filtered)    │
                                                                 └────────┬────────┘
                                                                          │
                                                                          ▼
                                                                 ┌─────────────────┐
                                                                 │  CheckResult    │
                                                                 └─────────────────┘
```

## Core Components

### CLI (`internal/cli`)

The CLI layer uses [Cobra](https://github.com/spf13/cobra) for command handling:

- **`root.go`** - Root command and global configuration
- **`check.go`** - Main `check` command that compares two configurations
- **`explain.go`** - Displays rule documentation
- **`init.go`** - Generates default configuration file
- **`version.go`** - Version information

The CLI orchestrates the overall flow:
1. Load configuration
2. Load old and new module snapshots
3. Run rules engine
4. Process annotations
5. Render output

### Configuration (`internal/config`)

Configuration management handles:

- **Discovery** - Finding `.tfbreak.hcl` in standard locations
- **Parsing** - Using HCL to parse configuration files
- **Defaults** - Providing sensible defaults when not configured
- **Validation** - Ensuring configuration is valid

Key files:
- `config.go` - Main configuration structs and loading logic
- `defaults.go` - Default configuration values

### Module Loader (`internal/loader`)

The loader extracts Terraform module structure from `.tf` files:

```go
type ModuleSnapshot struct {
    Path              string
    Variables         map[string]*VariableSignature
    Outputs           map[string]*OutputSignature
    Resources         map[string]*ResourceSignature
    Modules           map[string]*ModuleCallSignature
    MovedBlocks       []*MovedBlock
    RequiredVersion   string
    RequiredProviders map[string]*ProviderRequirement
}
```

The loader uses:
- **terraform-config-inspect** - HashiCorp library for parsing Terraform configurations
- **Custom HCL parsing** - For features not supported by terraform-config-inspect (nullable, validations, moved blocks)

### Rules Engine (`internal/rules`)

The rules engine is the core of tfbreak's detection capabilities.

#### Registry

The registry holds all registered rules:

```go
type Registry struct {
    rules map[string]Rule
    order []string  // preserve registration order
}
```

Rules self-register using `init()` functions:

```go
func init() {
    Register(&BC001{})
}
```

#### Rule Interface

```go
type Rule interface {
    ID() string
    Name() string
    Description() string
    DefaultSeverity() types.Severity
    Evaluate(old, new *types.ModuleSnapshot) []*types.Finding
}
```

#### Engine

The engine evaluates rules and applies configuration:

```go
type Engine struct {
    registry *Registry
    config   map[string]*RuleConfig
}

func (e *Engine) Evaluate(old, new *ModuleSnapshot) []*Finding {
    var findings []*Finding
    for _, rule := range e.registry.All() {
        cfg := e.GetConfig(rule.ID())
        if cfg.Enabled {
            findings = append(findings, rule.Evaluate(old, new)...)
        }
    }
    return findings
}
```

### Annotations (`internal/annotation`)

The annotation system handles inline ignores:

- **Parser** - Extracts annotations from HCL comments
- **Matcher** - Associates annotations with findings based on location
- **Governance** - Enforces governance rules (require_reason, allow/deny lists)

### Output (`internal/output`)

Output rendering supports multiple formats:

- **Text** - Human-readable colored output for terminals
- **JSON** - Machine-readable format for CI/CD integration

### Types (`internal/types`)

Core types used throughout the codebase:

```go
// Severity levels (tflint-aligned)
type Severity int
const (
    SeverityNotice Severity = iota
    SeverityWarning
    SeverityError
)

// Finding represents a single rule violation
type Finding struct {
    RuleID      string
    RuleName    string
    Severity    Severity
    Message     string
    Detail      string
    OldLocation *FileRange
    NewLocation *FileRange
    Ignored     bool
    IgnoreReason string
    Remediation string
}

// CheckResult is the final output
type CheckResult struct {
    OldPath  string
    NewPath  string
    Findings []*Finding
    Summary  Summary
    Result   string  // PASS or FAIL
    FailOn   Severity
}
```

## Plugin Architecture

Plugins extend tfbreak with provider-specific rules.

### Plugin Discovery (`plugin/discovery.go`)

Plugins are discovered in priority order:
1. Config `plugin_dir`
2. `TFBREAK_PLUGIN_DIR` environment variable
3. `./.tfbreak.d/plugins/`
4. `~/.tfbreak.d/plugins/`

### Plugin Communication

Plugins communicate via gRPC using HashiCorp's [go-plugin](https://github.com/hashicorp/go-plugin) library:

```
tfbreak (host) <--gRPC--> plugin (subprocess)
```

The Runner interface provides plugins access to both configurations:

```go
type Runner interface {
    // Old configuration
    GetOldModuleContent(schema, opts) (*Content, error)
    GetOldResourceContent(type, schema, opts) (*Content, error)

    // New configuration
    GetNewModuleContent(schema, opts) (*Content, error)
    GetNewResourceContent(type, schema, opts) (*Content, error)

    // Emit findings
    EmitIssue(rule Rule, message string, location hcl.Range) error
}
```

## Data Flow

### Check Command Flow

```
1. CLI parses arguments (old_dir, new_dir)
         │
         ▼
2. Load configuration (.tfbreak.hcl)
         │
         ▼
3. Apply CLI flag overrides
         │
         ▼
4. Create path filter from include/exclude patterns
         │
         ▼
5. Load old module snapshot (with filtering)
         │
         ▼
6. Load new module snapshot (with filtering)
         │
         ▼
7. Create and configure rules engine
         │
         ▼
8. Run built-in rules: engine.Evaluate(old, new)
         │
         ▼
9. Run plugin rules (if any)
         │
         ▼
10. Process annotations (match to findings)
         │
         ▼
11. Compute result (PASS/FAIL based on policy)
         │
         ▼
12. Render output (text/JSON)
         │
         ▼
13. Set exit code (0=PASS, 1=FAIL)
```

### Rule Evaluation Flow

```
For each rule in registry:
    1. Check if rule is enabled
    2. Call rule.Evaluate(old, new)
    3. For each finding:
        a. Apply configured severity override
        b. Add to findings list

After all rules:
    1. Apply rename suppression (if enabled)
    2. Return combined findings
```

### Annotation Matching Flow

```
1. Parse annotations from all .tf files in new_dir
         │
         ▼
2. Find block start lines in each file
         │
         ▼
3. For each finding:
    a. Check for file-level annotation
    b. Check for block-level annotation (line before block)
    c. If matched:
        - Check governance rules
        - If valid, mark finding as ignored
```

## Key Design Decisions

### Why Two Snapshots?

Unlike tflint which analyzes a single configuration, tfbreak needs both old and new configurations to detect changes. This is fundamental to breaking change detection.

### Why Process Isolation for Plugins?

Using gRPC and separate processes:
- Prevents plugin crashes from affecting tfbreak
- Allows plugins in different languages (theoretically)
- Follows established Terraform ecosystem patterns

### Why tflint-Aligned Severity Levels?

Using ERROR/WARNING/NOTICE (instead of BREAKING/RISKY/INFO):
- Consistency with the Terraform ecosystem
- Familiar to tflint users
- Better alignment with CI/CD conventions

### Why HCL for Configuration?

Using HCL (same as Terraform):
- Familiar to Terraform users
- Consistent with tflint
- Supports blocks and complex structures

## Module Snapshot Details

The `ModuleSnapshot` captures the "shape" of a Terraform module:

### Variables

```go
type VariableSignature struct {
    Name            string
    Type            string       // Type constraint expression
    Default         interface{}  // Default value (nil if required)
    Description     string
    Sensitive       bool
    Nullable        *bool        // nil means default (true)
    Required        bool         // Computed: no default
    ValidationCount int
    Validations     []*Validation
    DeclRange       FileRange
}
```

### Outputs

```go
type OutputSignature struct {
    Name        string
    Description string
    Sensitive   bool
    DeclRange   FileRange
}
```

### Resources and Modules

```go
type ResourceSignature struct {
    Type      string    // e.g., "aws_instance"
    Name      string    // e.g., "main"
    Address   string    // e.g., "aws_instance.main"
    DeclRange FileRange
}

type ModuleCallSignature struct {
    Name      string    // e.g., "vpc"
    Source    string    // e.g., "registry.terraform.io/..."
    Version   string    // Version constraint
    Address   string    // e.g., "module.vpc"
    DeclRange FileRange
}
```

### Moved Blocks

```go
type MovedBlock struct {
    From      string    // e.g., "aws_instance.old"
    To        string    // e.g., "aws_instance.new"
    DeclRange FileRange
}
```

## Future Considerations

### Plugin Installation

Future work includes automated plugin installation:

```bash
tfbreak plugin install github.com/jokarl/tfbreak-ruleset-azurerm
```

### Remote Comparison

Comparing against remote module registries:

```bash
tfbreak check registry.terraform.io/org/module/aws@1.0.0 ./
```

### GitHub Integration

Direct PR comparison:

```bash
tfbreak check --pr 123
```

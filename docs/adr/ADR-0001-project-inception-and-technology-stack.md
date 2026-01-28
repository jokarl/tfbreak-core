---
status: proposed
date: 2026-01-28
decision-makers: [project maintainers]
consulted: []
informed: []
---

# tfbreak: Project Inception and Technology Stack Selection

## Context and Problem Statement

Terraform module maintainers need a way to detect breaking changes before they impact consumers. Currently, there is no standardized tooling to compare two versions of a Terraform module and identify changes that would break callers or destroy state. Teams rely on manual review, which is error-prone and time-consuming.

How should we build a static analysis tool that compares Terraform configurations and reports breaking changes to module interfaces and state safety?

## Decision Drivers

* Must provide a Terraform-native experience (HCL configuration, familiar patterns)
* Must be CI-first with clear exit codes and machine-readable output
* Must support incremental adoption with sensible defaults
* Must be maintainable and testable
* Must have no runtime dependencies on cloud providers or Terraform binary
* Must be easy to distribute as a single binary

## Considered Options

* **Option 1: Go with terraform-config-inspect and HCL/v2**
* **Option 2: Python with python-hcl2**
* **Option 3: Rust with hcl-rs**
* **Option 4: TypeScript/Node.js with @cdktf/hcl2json**

## Decision Outcome

Chosen option: "Go with terraform-config-inspect and HCL/v2", because it provides native HCL parsing from HashiCorp themselves, produces a single static binary with no runtime dependencies, and aligns with the Terraform ecosystem's language choice.

### Consequences

* Good, because terraform-config-inspect is maintained by HashiCorp and handles edge cases correctly
* Good, because Go produces statically linked binaries that are easy to distribute
* Good, because the ecosystem (cobra, viper, etc.) is mature for CLI development
* Good, because no CGO dependencies means easy cross-compilation
* Bad, because Go's error handling is verbose
* Bad, because generics support is relatively new, may affect some abstractions

### Confirmation

This decision will be confirmed by:
1. Successfully parsing the test fixtures in the specification
2. Achieving the acceptance criteria outlined in spec/001_IDEA.md section 13
3. CI pipeline successfully building for linux/amd64, darwin/amd64, darwin/arm64

## Pros and Cons of the Options

### Option 1: Go with terraform-config-inspect and HCL/v2

* Good, because terraform-config-inspect is the official HashiCorp library for this purpose
* Good, because HCL/v2 is the reference implementation of HCL parsing
* Good, because single binary distribution with no runtime dependencies
* Good, because strong typing catches errors at compile time
* Good, because excellent concurrency support if needed for performance
* Neutral, because Go's module system requires careful dependency management
* Bad, because verbose error handling patterns

### Option 2: Python with python-hcl2

* Good, because rapid prototyping
* Good, because rich ecosystem for text processing
* Bad, because runtime dependency on Python
* Bad, because python-hcl2 is a community library, not officially maintained
* Bad, because distribution complexity (pip, virtualenv, etc.)

### Option 3: Rust with hcl-rs

* Good, because memory safety and performance
* Good, because single binary distribution
* Bad, because hcl-rs has less community adoption and may have edge cases
* Bad, because steeper learning curve
* Bad, because longer development time

### Option 4: TypeScript/Node.js with @cdktf/hcl2json

* Good, because rapid development
* Good, because familiar to many developers
* Bad, because runtime dependency on Node.js
* Bad, because @cdktf/hcl2json shells out to Go binary anyway
* Bad, because distribution complexity

## More Information

### Referenced Specification

The complete specification for tfbreak is documented in `spec/001_IDEA.md`, which defines:
- 17 rules across 4 categories (variables, outputs, state safety, toolchain)
- Three severity levels (BREAKING, RISKY, INFO)
- Configuration via `.tfbreak.hcl`
- Annotation system for in-code ignores
- CLI interface with check, diff, explain, init, and version commands
- Text and JSON output formats

### Key Dependencies (from spec section 8.1)

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/hashicorp/terraform-config-inspect v0.0.0-latest
    github.com/hashicorp/hcl/v2 v2.19.0
    github.com/bmatcuk/doublestar/v4 v4.6.0
    github.com/fatih/color v1.16.0
    github.com/google/go-cmp v0.6.0 // for testing
)
```

### Implementation Phases (from spec section 10)

1. **Phase 1: MVP** - Core check command with essential rules
2. **Phase 2: Configuration & Annotations** - Full config and ignore system
3. **Phase 3: Extended Rules** - Complete rule coverage
4. **Phase 4: CI Enhancements** - Production-ready CI integration

### Related Documents

- Specification: `spec/001_IDEA.md`

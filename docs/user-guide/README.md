# tfbreak User Guide

This guide covers how to configure and use tfbreak for detecting breaking changes in Terraform modules.

## Contents

1. [Getting Started](#getting-started)
2. [Configuration](config.md) - Configure tfbreak behavior via `.tfbreak.hcl`
3. [Annotations](annotations.md) - Suppress findings with inline comments
4. [Plugins](plugins.md) - Extend tfbreak with provider-specific rules
5. [Rules Reference](../rules.md) - Documentation for all built-in rules

## Getting Started

### Installation

Install tfbreak using Go:

```bash
go install github.com/jokarl/tfbreak-core/cmd/tfbreak@latest
```

Or download a pre-built binary from the [releases page](https://github.com/jokarl/tfbreak-core/releases).

### Basic Usage

tfbreak compares two Terraform configuration directories and reports breaking changes:

```bash
tfbreak check <old_dir> <new_dir>
```

For example, comparing two versions of a module:

```bash
# Compare local directories
tfbreak check ./v1.0.0 ./v2.0.0
```

### Git Ref Comparison

tfbreak can compare directly against git refs without manual checkout.

#### Single-Module Repositories

For repositories where the Terraform configuration is at the root (e.g., `terraform-aws-vpc`):

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

#### Monorepos

For repositories containing multiple Terraform modules in subdirectories, use the `ref:path` syntax to specify which module to compare:

```
repo/
├── modules/
│   ├── vpc/          # terraform-aws-vpc module
│   ├── eks/          # terraform-aws-eks module
│   └── rds/          # terraform-aws-rds module
└── environments/
    └── prod/
```

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

The `ref:path` syntax follows git's convention (like `git show REVISION:path`). Each ref can have its own path, which is useful when modules are renamed between versions.

### Understanding the Output

tfbreak produces findings with three severity levels:

- **ERROR** - Breaking changes that will definitely cause failures for consumers
- **WARNING** - Risky changes that may cause unexpected behavior
- **NOTICE** - Informational changes worth noting

Example output:

```
tfbreak - Terraform Breaking Change Detector

Comparing:
  Old: ./v1.0.0
  New: ./v2.0.0

Findings:

ERROR  BC002  input-removed
  Variable "legacy_option" was removed
  Location: variables.tf:25

WARNING  RC006  input-default-changed
  Variable "instance_type" default changed from "t3.micro" to "t3.small"
  Location: variables.tf:8

Summary: 1 error, 1 warning, 0 notices
Result: FAIL
```

### Exit Codes

- `0` - No findings at or above the fail threshold (PASS)
- `1` - One or more findings at or above the fail threshold (FAIL)

### CI Integration

Use tfbreak in CI pipelines to prevent accidental breaking changes:

```yaml
# GitHub Actions example - using git ref mode (recommended)
- name: Check for breaking changes
  run: tfbreak check --base origin/main ./
```

Or with manual worktree creation for more control:

```yaml
# GitHub Actions example - manual worktree
- name: Check for breaking changes
  run: |
    git fetch origin main
    git worktree add /tmp/main-branch origin/main
    tfbreak check /tmp/main-branch ./
```

### JSON Output

For programmatic processing, use JSON output:

```bash
tfbreak check ./old ./new --format json | jq '.findings[] | select(.severity == "ERROR")'
```

### Remediation Guidance

Include remediation guidance for each finding:

```bash
tfbreak check ./old ./new --include-remediation
```

This adds helpful suggestions for fixing each issue.

## Common Workflows

### Pre-commit Hook

Add tfbreak to your pre-commit hooks to catch breaking changes before they are committed:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: tfbreak
        name: tfbreak
        entry: bash -c 'tfbreak check --base origin/main ./'
        language: system
        pass_filenames: false
```

### Pull Request Checks

Run tfbreak as part of your PR validation:

```yaml
# GitHub Actions
name: Breaking Change Check
on: [pull_request]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - name: Install tfbreak
        run: go install github.com/jokarl/tfbreak-core/cmd/tfbreak@latest
      - name: Check for breaking changes
        run: tfbreak check --base ${{ github.event.pull_request.base.sha }} ./
```

For monorepos with multiple modules:

```yaml
      - name: Check VPC module
        run: tfbreak check --base ${{ github.event.pull_request.base.sha }}:modules/vpc ./modules/vpc
      - name: Check EKS module
        run: tfbreak check --base ${{ github.event.pull_request.base.sha }}:modules/eks ./modules/eks
```

### Semantic Versioning

Use tfbreak findings to determine version bumps:

- **ERROR findings** = Major version bump required
- **WARNING findings** = Minor version bump recommended
- **NOTICE only** = Patch version acceptable

## Next Steps

- [Configure tfbreak](config.md) with a `.tfbreak.hcl` file
- [Suppress findings](annotations.md) with annotations when changes are intentional
- [Add plugins](plugins.md) for provider-specific detection
- [Review all rules](../rules.md) to understand what tfbreak detects

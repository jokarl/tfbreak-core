# tfbreak — Complete Implementation Specification

> A static analysis tool that compares two Terraform configurations and reports breaking changes to module interfaces and state safety.

---

## 1. Overview

### 1.1 Purpose

`tfbreak` compares an "old" and "new" Terraform configuration directory, extracts structural signatures, and reports changes that would break callers or destroy state.

### 1.2 What It Is

- Static analysis of `.tf` files (with `.tf.json` support planned)
- Module interface contract checker (variables, outputs)
- State safety checker (resources, modules, `moved` blocks)

### 1.3 What It Is Not

- Not infrastructure drift detection (no cloud API calls)
- Not a `terraform plan` replacement
- Not a linter for style/best practices

### 1.4 Design Principles

- Terraform-native feel (HCL config, familiar patterns)
- CI-first (clear exit codes, machine-readable output)
- Incremental adoption (sensible defaults, progressive configuration)

---

## 2. Rules

### 2.1 Severity Levels

| Severity | Meaning | Default CI Behavior |
|----------|---------|---------------------|
| BREAKING | Will break callers or destroy state | Fail |
| RISKY | May cause unexpected behavior changes | Warn |
| INFO | Informational, no action needed | Log |

### 2.2 Complete Rule Catalog

#### Variables (Inputs)

| ID | Name | Default Severity | Description |
|----|------|------------------|-------------|
| BC001 | `required-input-added` | BREAKING | New variable without a default value |
| BC002 | `input-removed` | BREAKING | Variable exists in old, missing in new |
| BC003 | `input-renamed` | BREAKING | Variable removed and similar one added (heuristic, opt-in) |
| BC004 | `input-type-changed` | BREAKING | Variable `type` constraint changed |
| BC005 | `input-default-removed` | BREAKING | Variable had default in old, no default in new |
| RC006 | `input-default-changed` | RISKY | Variable default value differs |
| RC007 | `input-nullable-changed` | RISKY | Variable `nullable` attribute changed |
| RC008 | `input-sensitive-changed` | RISKY | Variable `sensitive` attribute changed |

#### Outputs

| ID | Name | Default Severity | Description |
|----|------|------------------|-------------|
| BC009 | `output-removed` | BREAKING | Output exists in old, missing in new |
| BC010 | `output-renamed` | BREAKING | Output removed and similar one added (heuristic, opt-in) |
| RC011 | `output-sensitive-changed` | RISKY | Output `sensitive` attribute changed |

#### State Safety (Resources & Modules)

| ID | Name | Default Severity | Description |
|----|------|------------------|-------------|
| BC100 | `resource-removed-no-moved` | BREAKING | Resource removed without `moved` block coverage |
| BC101 | `module-removed-no-moved` | BREAKING | Module call removed without `moved` block coverage |
| BC102 | `invalid-moved-block` | BREAKING | `moved` block cannot be parsed or has invalid syntax |
| BC103 | `conflicting-moved` | BREAKING | Duplicate `from` addresses, cycles, or ambiguous mappings |

#### Toolchain Constraints

| ID | Name | Default Severity | Description |
|----|------|------------------|-------------|
| BC200 | `terraform-version-constrained` | BREAKING | `required_version` constraint tightened |
| BC201 | `provider-version-constrained` | BREAKING | Provider version constraint tightened or provider removed |

### 2.3 Rule Evaluation Details

#### BC001 — required-input-added

```
Condition: variable V exists in new AND NOT in old AND new.V.default is unset
```

#### BC002 — input-removed

```
Condition: variable V exists in old AND NOT in new
```

Note: If BC003 (rename detection) is enabled and finds a match, BC002 is suppressed for that variable.

#### BC003 — input-renamed (Phase 4, opt-in)

```
Condition: BC002 triggered for V_old AND BC001 triggered for V_new
           AND similarity(V_old.name, V_new.name) >= threshold
           AND types are compatible
```

Default threshold: 0.85 (Levenshtein-based similarity)

#### BC004 — input-type-changed

```
Condition: variable V exists in both AND old.V.type != new.V.type
```

Type comparison uses normalized string representation. `any` is treated as a wildcard (changing from `any` to specific type is not breaking; reverse is breaking).

#### BC005 — input-default-removed

```
Condition: variable V exists in both AND old.V.default is set AND new.V.default is unset
```

#### RC006 — input-default-changed

```
Condition: variable V exists in both AND both have defaults AND old.V.default != new.V.default
```

Default comparison uses canonical JSON serialization for complex values.

#### RC007 — input-nullable-changed

```
Condition: variable V exists in both AND old.V.nullable != new.V.nullable
```

Note: Terraform defaults `nullable = true` when unspecified.

#### RC008 — input-sensitive-changed

```
Condition: variable V exists in both AND old.V.sensitive != new.V.sensitive
```

#### BC009 — output-removed

```
Condition: output O exists in old AND NOT in new
```

#### BC010 — output-renamed (Phase 4, opt-in)

```
Condition: BC009 triggered for O_old AND new output O_new exists
           AND similarity(O_old.name, O_new.name) >= threshold
```

#### RC011 — output-sensitive-changed

```
Condition: output O exists in both AND old.O.sensitive != new.O.sensitive
```

#### BC100 — resource-removed-no-moved

```
Condition: resource address A exists in old AND NOT in new
           AND no moved block has from = A
```

Address format: `<type>.<name>` (e.g., `aws_s3_bucket.main`)

#### BC101 — module-removed-no-moved

```
Condition: module call M exists in old AND NOT in new
           AND no moved block has from = module.M
```

Address format: `module.<name>` (e.g., `module.vpc`)

#### BC102 — invalid-moved-block

```
Condition: moved block exists but from or to cannot be parsed as valid address
```

#### BC103 — conflicting-moved

```
Condition: multiple moved blocks have same from address
           OR moved blocks form a cycle
           OR moved block references non-existent to address
```

#### BC200 — terraform-version-constrained

```
Condition: new.required_version is more restrictive than old.required_version
```

MVP: Any change to `required_version` triggers this rule. Future: semantic constraint comparison.

#### BC201 — provider-version-constrained

```
Condition: provider P in old.required_providers is missing in new
           OR new constraint for P is more restrictive than old
```

MVP: Any constraint change or removal triggers this rule.

---

## 3. Architecture

### 3.1 Pipeline

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Load Old   │    │  Load New   │    │   Parse     │
│  Directory  │───▶│  Directory  │───▶│   Config    │
└─────────────┘    └─────────────┘    └─────────────┘
                                            │
                                            ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Render    │◀───│   Apply     │◀───│    Run      │
│   Output    │    │   Ignores   │    │   Rules     │
└─────────────┘    └─────────────┘    └─────────────┘
```

### 3.2 Data Model

```go
// ModuleSnapshot represents the extracted signature of a Terraform module
type ModuleSnapshot struct {
    Path string // directory path

    Variables map[string]*VariableSignature
    Outputs   map[string]*OutputSignature
    Resources map[string]*ResourceSignature // key: type.name
    Modules   map[string]*ModuleCallSignature // key: name

    RequiredVersion   string // terraform.required_version
    RequiredProviders map[string]*ProviderRequirement

    MovedBlocks []*MovedBlock
}

type VariableSignature struct {
    Name        string
    Type        string // normalized type expression
    Default     *string // JSON-serialized, nil if no default
    Description string
    Sensitive   bool
    Nullable    bool // defaults to true if unset

    DeclRange FileRange
}

type OutputSignature struct {
    Name        string
    Description string
    Sensitive   bool

    DeclRange FileRange
}

type ResourceSignature struct {
    Type    string // e.g., "aws_s3_bucket"
    Name    string // e.g., "main"
    Address string // e.g., "aws_s3_bucket.main"

    DeclRange FileRange
}

type ModuleCallSignature struct {
    Name    string
    Source  string
    Address string // e.g., "module.vpc"

    DeclRange FileRange
}

type MovedBlock struct {
    From string
    To   string

    DeclRange FileRange
}

type ProviderRequirement struct {
    Source  string // e.g., "hashicorp/aws"
    Version string // version constraint
}

type FileRange struct {
    Filename  string
    StartLine int
    EndLine   int
}
```

### 3.3 Rule Engine

```go
type Severity string

const (
    SeverityBreaking Severity = "BREAKING"
    SeverityRisky    Severity = "RISKY"
    SeverityInfo     Severity = "INFO"
)

type Rule interface {
    ID() string
    Name() string
    Description() string
    DefaultSeverity() Severity
    Evaluate(old, new *ModuleSnapshot, cfg *RuleConfig) []*Finding
}

type RuleConfig struct {
    Enabled  bool
    Severity Severity
    Options  map[string]interface{} // rule-specific options
}

type Finding struct {
    RuleID      string
    RuleName    string
    Severity    Severity
    Message     string
    Detail      string

    OldLocation *FileRange // nil if not applicable
    NewLocation *FileRange // nil if not applicable

    Ignored      bool
    IgnoreReason string
}
```

### 3.4 Ignore System

```go
type Ignore struct {
    Scope    IgnoreScope
    RuleIDs  []string // empty = all rules
    Reason   string
    Ticket   string
    Expires  *time.Time

    FileRange FileRange // where the directive was found
    BlockRange *FileRange // the block it applies to (for block-level)
}

type IgnoreScope int

const (
    IgnoreScopeFile  IgnoreScope = iota // applies to entire file
    IgnoreScopeBlock                    // applies to next block
)
```

---

## 4. Configuration

### 4.1 File Location

Search order:

1. `--config` flag value
2. `.tfbreak.hcl` in current directory
3. `.tfbreak.hcl` in old directory
4. Default configuration (all rules enabled with default severities)

### 4.2 Configuration Schema

```hcl
# .tfbreak.hcl

version = 1 # required, currently only version 1

# Path filtering
paths {
  include = ["**/*.tf"]           # glob patterns to include
  exclude = [                      # glob patterns to exclude
    ".terraform/**",
    "**/examples/**",
    "**/test/**"
  ]
}

# Output settings
output {
  format = "text"  # text | json
  color  = "auto"  # auto | always | never
}

# CI policy
policy {
  fail_on                   = "BREAKING" # BREAKING | RISKY | INFO
  treat_warnings_as_errors  = false
}

# Annotation controls
annotations {
  enabled         = true
  require_reason  = false
  allow_rule_ids  = []  # empty = all allowed
  deny_rule_ids   = []  # these rules cannot be ignored via annotations
}

# Rule configuration
rules {
  # Each rule can be configured individually
  BC001 {
    enabled  = true
    severity = "BREAKING"
  }

  BC002 {
    enabled  = true
    severity = "BREAKING"
  }

  BC004 {
    enabled  = true
    severity = "BREAKING"
  }

  BC005 {
    enabled  = true
    severity = "BREAKING"
  }

  RC006 {
    enabled  = true
    severity = "RISKY"
  }

  RC007 {
    enabled  = true
    severity = "RISKY"
  }

  RC008 {
    enabled  = true
    severity = "RISKY"
  }

  BC009 {
    enabled  = true
    severity = "BREAKING"
  }

  RC011 {
    enabled  = true
    severity = "RISKY"
  }

  BC100 {
    enabled  = true
    severity = "BREAKING"
    # Allow specific resources to be removed without moved blocks
    allow_addresses = []
  }

  BC101 {
    enabled  = true
    severity = "BREAKING"
    allow_addresses = []
  }

  BC102 {
    enabled  = true
    severity = "BREAKING"
  }

  BC103 {
    enabled  = true
    severity = "BREAKING"
  }

  BC200 {
    enabled  = true
    severity = "BREAKING"
  }

  BC201 {
    enabled  = true
    severity = "BREAKING"
  }

  # Rename detection (Phase 4)
  rename_detection {
    enabled             = false
    similarity_threshold = 0.85
  }
}
```

### 4.3 Configuration Defaults

When no configuration file is found, use these defaults:

- All rules enabled
- Default severities as specified in rule catalog
- `fail_on = "BREAKING"`
- `format = "text"`
- `color = "auto"`
- `annotations.enabled = true`
- `annotations.require_reason = false`
- `paths.include = ["**/*.tf"]`
- `paths.exclude = [".terraform/**"]`

---

## 5. Annotations (In-Code Ignores)

### 5.1 Syntax

Prefix: `tfbreak:`

#### File-Level Ignore

Must appear at top of file (before any blocks):

```hcl
# tfbreak:ignore-file
```

```hcl
# tfbreak:ignore-file BC001,BC002 reason="generated file"
```

#### Block-Level Ignore

Comment immediately preceding a block:

```hcl
# tfbreak:ignore BC100 reason="intentional removal"
resource "aws_s3_bucket" "legacy" {
  # ...
}
```

```hcl
# tfbreak:ignore BC002,RC006 reason="deprecated input" ticket="INFRA-1234"
variable "old_setting" {
  type    = string
  default = "value"
}
```

### 5.2 Annotation Grammar

```
annotation     = "tfbreak:" directive [rule_list] [metadata]
directive      = "ignore-file" | "ignore"
rule_list      = rule_id ("," rule_id)*
rule_id        = /[A-Z]{2}[0-9]{3}/
metadata       = metadata_item+
metadata_item  = key "=" quoted_string
key            = "reason" | "ticket" | "expires"
quoted_string  = '"' [^"]* '"'
```

### 5.3 Matching Logic

1. Parse all comments in `.tf` files
2. Extract annotations with their file positions
3. For file-level ignores: apply to all findings in that file
4. For block-level ignores: match to the immediately following block's findings
5. Check against `annotations.allow_rule_ids` and `annotations.deny_rule_ids`
6. If `annotations.require_reason = true`, reject annotations without `reason`
7. If `expires` is set and in the past, ignore the annotation

---

## 6. CLI Interface

### 6.1 Commands

```
tfbreak check <old_dir> <new_dir>   Compare directories and evaluate policy
tfbreak diff <old_dir> <new_dir>    Show structural differences (no policy)
tfbreak explain <rule_id>           Show rule documentation
tfbreak init                        Create starter .tfbreak.hcl
tfbreak version                     Print version information
```

### 6.2 `check` Command

**Usage:**

```
tfbreak check [flags] <old_dir> <new_dir>
```

**Flags:**

Input:

```
--config, -c PATH         Path to config file
```

Policy:

```
--fail-on SEVERITY        Override policy.fail_on (BREAKING|RISKY|INFO)
--enable RULES            Enable rules (comma-separated IDs)
--disable RULES           Disable rules (comma-separated IDs)
--severity RULE=SEV       Override rule severity (repeatable)
```

Paths:

```
--include GLOB            Include pattern (repeatable, overrides config)
--exclude GLOB            Exclude pattern (repeatable, overrides config)
```

Annotations:

```
--no-annotations          Disable annotation processing
--require-reason          Require reason in annotations
```

Output:

```
--format FORMAT           Output format: text, json (default: text)
--output, -o FILE         Write output to file instead of stdout
--color MODE              Color mode: auto, always, never (default: auto)
--quiet, -q               Suppress non-error output
--verbose, -v             Verbose output
```

**Exit Codes:**

- `0`: No findings at or above `fail_on` severity
- `1`: Findings at or above `fail_on` severity
- `2`: Configuration or usage error
- `3`: Internal error

### 6.3 `diff` Command

Same flags as `check` except policy flags. Always exits 0 unless error.

### 6.4 `explain` Command

```
tfbreak explain BC001
```

Output:

```
BC001: required-input-added
Severity: BREAKING

A new variable was added without a default value. This breaks existing
callers that do not provide this variable.

Example (old):
  # (no variable)

Example (new):
  variable "new_required" {
    type = string
  }

Remediation:
  - Add a default value to make the variable optional
  - Or ensure all callers are updated to provide the new variable
  - Or use an annotation to suppress if callers are updated in same change:
    # tfbreak:ignore BC001 reason="callers updated"
```

### 6.5 `init` Command

```
tfbreak init [--force]
```

Creates `.tfbreak.hcl` with documented defaults. Fails if file exists unless `--force`.

---

## 7. Output Formats

### 7.1 Text Format

```
tfbreak: comparing old/ -> new/

BREAKING  BC001  required-input-added
  new/variables.tf:15
  New required variable "cluster_name" has no default

BREAKING  BC100  resource-removed-no-moved
  old/main.tf:42
  Resource "aws_s3_bucket.logs" removed without moved block

RISKY  RC006  input-default-changed
  new/variables.tf:8 (from old/variables.tf:8)
  Variable "instance_type" default changed: "t3.micro" -> "t3.small"
  [IGNORED] reason="intentional upgrade"

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Summary: 2 breaking, 0 risky (1 ignored), 0 info
Result: FAIL (breaking changes detected)
```

### 7.2 JSON Format

```json
{
  "version": "1.0",
  "old_path": "old/",
  "new_path": "new/",
  "findings": [
    {
      "rule_id": "BC001",
      "rule_name": "required-input-added",
      "severity": "BREAKING",
      "message": "New required variable \"cluster_name\" has no default",
      "detail": "",
      "old_location": null,
      "new_location": {
        "filename": "new/variables.tf",
        "start_line": 15,
        "end_line": 18
      },
      "ignored": false,
      "ignore_reason": ""
    },
    {
      "rule_id": "BC100",
      "rule_name": "resource-removed-no-moved",
      "severity": "BREAKING",
      "message": "Resource \"aws_s3_bucket.logs\" removed without moved block",
      "detail": "",
      "old_location": {
        "filename": "old/main.tf",
        "start_line": 42,
        "end_line": 50
      },
      "new_location": null,
      "ignored": false,
      "ignore_reason": ""
    },
    {
      "rule_id": "RC006",
      "rule_name": "input-default-changed",
      "severity": "RISKY",
      "message": "Variable \"instance_type\" default changed: \"t3.micro\" -> \"t3.small\"",
      "detail": "",
      "old_location": {
        "filename": "old/variables.tf",
        "start_line": 8,
        "end_line": 12
      },
      "new_location": {
        "filename": "new/variables.tf",
        "start_line": 8,
        "end_line": 12
      },
      "ignored": true,
      "ignore_reason": "intentional upgrade"
    }
  ],
  "summary": {
    "breaking": 2,
    "risky": 0,
    "info": 0,
    "ignored": 1,
    "total": 3
  },
  "result": "FAIL",
  "fail_on": "BREAKING"
}
```

---

## 8. Dependencies

### 8.1 Go Modules

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

### 8.2 Build Requirements

- Go 1.21+
- No CGO dependencies

---

## 9. Project Structure

```
tfbreak/
├── cmd/
│   └── tfbreak/
│       └── main.go              # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go              # Root command setup
│   │   ├── check.go             # check command
│   │   ├── diff.go              # diff command
│   │   ├── explain.go           # explain command
│   │   ├── init.go              # init command
│   │   └── version.go           # version command
│   ├── config/
│   │   ├── config.go            # Config struct and loading
│   │   ├── defaults.go          # Default configuration
│   │   └── validate.go          # Config validation
│   ├── loader/
│   │   ├── loader.go            # Main loader interface
│   │   ├── snapshot.go          # ModuleSnapshot extraction
│   │   ├── variables.go         # Variable parsing
│   │   ├── outputs.go           # Output parsing
│   │   ├── resources.go         # Resource/module parsing
│   │   ├── moved.go             # Moved block parsing
│   │   └── providers.go         # Provider requirements
│   ├── rules/
│   │   ├── engine.go            # Rule engine
│   │   ├── rule.go              # Rule interface
│   │   ├── registry.go          # Rule registry
│   │   ├── bc001.go             # required-input-added
│   │   ├── bc002.go             # input-removed
│   │   ├── bc004.go             # input-type-changed
│   │   ├── bc005.go             # input-default-removed
│   │   ├── rc006.go             # input-default-changed
│   │   ├── rc007.go             # input-nullable-changed
│   │   ├── rc008.go             # input-sensitive-changed
│   │   ├── bc009.go             # output-removed
│   │   ├── rc011.go             # output-sensitive-changed
│   │   ├── bc100.go             # resource-removed-no-moved
│   │   ├── bc101.go             # module-removed-no-moved
│   │   ├── bc102.go             # invalid-moved-block
│   │   ├── bc103.go             # conflicting-moved
│   │   ├── bc200.go             # terraform-version-constrained
│   │   └── bc201.go             # provider-version-constrained
│   ├── annotation/
│   │   ├── parser.go            # Annotation parsing
│   │   ├── matcher.go           # Finding-to-annotation matching
│   │   └── types.go             # Annotation types
│   ├── output/
│   │   ├── renderer.go          # Output interface
│   │   ├── text.go              # Text renderer
│   │   └── json.go              # JSON renderer
│   └── types/
│       ├── snapshot.go          # ModuleSnapshot types
│       ├── finding.go           # Finding types
│       └── severity.go          # Severity enum
├── testdata/
│   ├── scenarios/               # Fixture tests with real .tf files
│   │   ├── bc001_required_input_added/
│   │   │   ├── old/
│   │   │   ├── new/
│   │   │   └── expected.json
│   │   └── ...
│   └── ...
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 10. Implementation Phases

### Phase 1: MVP (Core Functionality)

**Goal:** Working `check` command with core rules

**Deliverables:**

- [ ] Project setup (go.mod, structure, Makefile)
- [ ] Types package (Severity, Finding, ModuleSnapshot)
- [ ] Loader: extract variables, outputs, resources, modules via terraform-config-inspect
- [ ] Loader: parse moved blocks via HCL
- [ ] Rules: BC001, BC002, BC005, RC006, BC009, BC100, BC101, BC102, BC103
- [ ] Rule engine with default configuration
- [ ] CLI: `check` command with basic flags
- [ ] CLI: `version` command
- [ ] Output: text format
- [ ] Output: JSON format
- [ ] Exit codes
- [ ] Unit tests for each rule
- [ ] Integration tests with fixture directories (real .tf files)

### Phase 2: Configuration & Annotations

**Goal:** Full configuration and ignore system

**Deliverables:**

- [ ] Config file loading (.tfbreak.hcl)
- [ ] Config validation
- [ ] All CLI flags for policy/paths/output
- [ ] Annotation parser
- [ ] Annotation matching to findings
- [ ] Annotation governance (allow/deny rules, require reason)
- [ ] CLI: `init` command
- [ ] CLI: `explain` command
- [ ] Documentation for each rule

### Phase 3: Extended Rules

**Goal:** Complete rule coverage

**Deliverables:**

- [ ] Rules: BC004 (type changed)
- [ ] Rules: RC007 (nullable changed)
- [ ] Rules: RC008 (sensitive changed)
- [ ] Rules: RC011 (output sensitive changed)
- [ ] Rules: BC200 (terraform version)
- [ ] Rules: BC201 (provider version)
- [ ] Enhanced tests for edge cases

### Phase 4: CI Enhancements

**Goal:** Production-ready CI integration

**Deliverables:**

- [ ] Git ref mode (--git-base, --git-head) with unit tests only
- [ ] SARIF output format
- [ ] Rename heuristics (BC003, BC010) - opt-in
- [ ] Performance optimization
- [ ] GitHub Actions example workflow
- [ ] GitLab CI example
- [ ] Comprehensive documentation

---

## 11. Testing Strategy

### 11.1 Unit Tests

Each rule has dedicated unit tests using in-memory ModuleSnapshot structs:

```go
func TestBC001_RequiredInputAdded(t *testing.T) {
    tests := []struct {
        name     string
        old      *ModuleSnapshot
        new      *ModuleSnapshot
        expected []*Finding
    }{
        {
            name: "new required variable triggers finding",
            old:  &ModuleSnapshot{Variables: map[string]*VariableSignature{}},
            new: &ModuleSnapshot{Variables: map[string]*VariableSignature{
                "foo": {Name: "foo", Default: nil},
            }},
            expected: []*Finding{{RuleID: "BC001", Severity: SeverityBreaking}},
        },
        {
            name: "new optional variable does not trigger",
            old:  &ModuleSnapshot{Variables: map[string]*VariableSignature{}},
            new: &ModuleSnapshot{Variables: map[string]*VariableSignature{
                "foo": {Name: "foo", Default: ptr(`"default"`)},
            }},
            expected: nil,
        },
    }
    // ...
}
```

### 11.2 Fixture Tests (Real .tf Files)

Fixture directories under `testdata/` with actual Terraform files:

```
testdata/
├── scenarios/
│   ├── basic_variable_changes/
│   │   ├── old/
│   │   │   └── variables.tf     # real Terraform file
│   │   ├── new/
│   │   │   └── variables.tf     # real Terraform file
│   │   ├── expected.json        # expected findings
│   │   └── config.hcl           # optional test config
│   ├── resource_removed_no_moved/
│   │   ├── old/
│   │   │   └── main.tf
│   │   ├── new/
│   │   │   └── main.tf
│   │   └── expected.json
│   ├── resource_moved_correctly/
│   │   ├── old/
│   │   │   └── main.tf
│   │   ├── new/
│   │   │   ├── main.tf
│   │   │   └── moved.tf         # contains moved blocks
│   │   └── expected.json        # should be empty (no findings)
```

Test runner loads old/ and new/ directories, runs the check, compares output against expected.json.

### 11.3 Git Mode Testing

Git integration (Phase 4) is tested via **unit tests only**, mocking the git operations:

- Mock git ref resolution
- Mock checkout/worktree operations
- Test that correct paths are passed to the core check logic

No actual git repositories in test fixtures.

### 11.4 Edge Cases to Cover

- Empty directories
- No changes (clean comparison)
- Only `.tf.json` files (when supported)
- Deeply nested modules
- Unicode in variable names
- Very large modules (performance)
- Malformed HCL (graceful error handling)
- Circular moved blocks
- Duplicate moved blocks
- Mixed severity findings
- All findings ignored

---

## 12. Future Considerations

### 12.1 `.tf.json` Support

- terraform-config-inspect handles JSON natively
- Moved block parsing needs JSON-specific path
- Annotations not supported (JSON has no comments)

### 12.2 Count/For_each Detection

- Detect when resource gains/loses count or for_each
- Address changes: `aws_s3_bucket.foo` -> `aws_s3_bucket.foo[0]`
- Requires deeper HCL analysis of resource blocks

### 12.3 Module Discovery in Monorepos

- `--discover` flag to find all modules
- Compare each module's old vs new
- Parallel processing

### 12.4 Terraform Plan Integration

- Optional `--with-plan` to detect actual state impact
- Would detect if default change causes recreation
- Requires Terraform binary and credentials

---

## 13. Acceptance Criteria

The implementation is complete when:

1. `tfbreak check old/ new/` correctly identifies all breaking changes in test fixtures
2. All 17 rules are implemented and tested
3. Configuration file is fully functional
4. Annotations work with governance controls
5. Text and JSON output formats are stable
6. Exit codes are correct
7. CI example workflows are documented and tested
8. `tfbreak explain <rule>` provides useful guidance for all rules
9. Performance is acceptable for modules with 100+ resources
10. Code coverage > 80%
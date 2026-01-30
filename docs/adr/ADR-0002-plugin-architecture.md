---
status: proposed
date: 2026-01-29
decision-makers: [project maintainers]
consulted: []
informed: []
---

# Plugin Architecture for Provider-Specific and Extensible Rules

## Context and Problem Statement

tfbreak-core provides provider-agnostic detection of breaking changes in Terraform modules. However, provider-specific concerns cannot be detected by generic rules:

1. Azure resources with "ForceNew" attributes that cause resource recreation
2. AWS resources with specific lifecycle behaviors
3. GCP naming constraints
4. Organization-specific policies

Should tfbreak adopt a plugin architecture, and if so, what minimal interface should plugins implement to report findings back to tfbreak?

## Decision Drivers

* Must align with existing Terraform ecosystem patterns (tflint)
* Must provide minimal interface - don't force plugins into specific detection patterns
* Must allow plugins to decide what they detect (schema-driven, rule-based, etc.)
* Must maintain single-binary distribution for core (no mandatory plugins)
* Must support clear versioning and protocol negotiation
* Must be testable in isolation
* Should minimize maintenance burden

## Considered Options

* **Option 1: tflint-aligned minimal interface with go-plugin/gRPC**
* **Option 2: Custom JSON-RPC protocol**
* **Option 3: Shared library plugins**
* **Option 4: No plugin architecture**

## Decision Outcome

Chosen option: "Option 1: tflint-aligned interface with go-plugin/gRPC", because it:
- Aligns with the established Terraform ecosystem (tflint, Terraform providers)
- Provides process isolation and language flexibility
- Has battle-tested versioning and handshake mechanisms
- Uses familiar patterns: RuleSet, Rule, Runner interfaces from tflint-plugin-sdk
- Gives plugins full flexibility in how they detect issues

### Consequences

* Good, because familiar patterns for Terraform ecosystem developers (Rule.Check(runner) pattern)
* Good, because plugins have full flexibility in detection approach
* Good, because process isolation prevents plugin crashes from affecting core
* Good, because go-plugin handles versioning and health checks
* Good, because separate Rule interface allows per-rule Severity, Enabled, Link
* Bad, because adds complexity to build/distribution
* Bad, because requires maintaining protocol compatibility

### Confirmation

This decision will be confirmed by:
1. Successfully implementing a reference plugin (tfbreak-ruleset-azurerm)
2. Demonstrating that plugins can use any detection approach
3. Validating protocol versioning works correctly

## Pros and Cons of the Options

### Option 1: tflint-aligned interface with go-plugin/gRPC

Follow tflint's proven architecture: plugins are separate binaries communicating via gRPC. Plugins implement RuleSet (container) and Rule (individual checks) interfaces, matching tflint-plugin-sdk patterns.

```
tfbreak (host) <--gRPC--> plugin (subprocess)
    │                          │
    ├─ Provides Runner         ├─ Implements RuleSet
    │   - GetOldModuleContent()│   - RuleSetName()
    │   - GetNewModuleContent()│   - RuleSetVersion()
    │   - EmitIssue()          │   - ConfigSchema()
    │   - DecodeRuleConfig()   │   - ApplyConfig()
    │                          │
    │                          ├─ Implements Rule (per rule)
    │                          │   - Name(), Severity()
    │                          │   - Enabled(), Link()
    │                          │   - Check(runner)
    │                          │
    └─ Aggregates findings     └─ Decides what to detect
```

* Good, because aligns with tflint (familiar to Terraform users)
* Good, because separate Rule interface allows per-rule metadata
* Good, because go-plugin is battle-tested
* Good, because plugins can use any detection approach
* Good, because ConfigSchema/ApplyConfig enables plugin configuration
* Neutral, because requires understanding gRPC/protobuf
* Bad, because each plugin requires separate binary

### Option 2: Custom JSON-RPC protocol

* Good, because simpler to implement
* Bad, because no ecosystem support
* Bad, because must build versioning from scratch

### Option 3: Shared library plugins

* Bad, because Go plugin package has severe limitations
* Bad, because no process isolation

### Option 4: No plugin architecture

* Bad, because core becomes bloated with provider-specific code
* Bad, because cannot support organizational custom rules

## More Information

### tflint-Aligned Plugin Interface

Following tflint-plugin-sdk patterns with separate RuleSet, Rule, and Runner interfaces:

```go
// ==================== Plugin Side ====================

// RuleSet is implemented by plugins (tflint-aligned)
type RuleSet interface {
    // Metadata
    RuleSetName() string
    RuleSetVersion() string
    RuleNames() []string
    VersionConstraint() string

    // Configuration (tflint-aligned)
    ConfigSchema() *hclext.BodySchema
    ApplyGlobalConfig(config *Config) error
    ApplyConfig(content *hclext.BodyContent) error

    // Execution
    NewRunner(runner Runner) (Runner, error)

    // Internal
    BuiltinImpl() *BuiltinRuleSet
}

// Rule is implemented by individual rules (tflint-aligned)
type Rule interface {
    Name() string
    Enabled() bool
    Severity() Severity
    Link() string
    Check(runner Runner) error
}

// DefaultRule provides default implementations for embedding
type DefaultRule struct{}

func (r *DefaultRule) Enabled() bool      { return true }
func (r *DefaultRule) Severity() Severity { return WARNING }
func (r *DefaultRule) Link() string       { return "" }

// ==================== Host Side (tfbreak-core) ====================

// Runner is provided by tfbreak to plugins
// DEVIATION: Dual old/new methods (tflint only has single config)
type Runner interface {
    // Working directory (tflint-aligned)
    GetOriginalwd() (string, error)
    GetModulePath() ([]string, error)

    // Access to OLD configuration (tfbreak-specific)
    GetOldModuleContent(schema *hclext.BodySchema, opts *GetModuleContentOption) (*hclext.BodyContent, error)
    GetOldResourceContent(resourceType string, schema *hclext.BodySchema, opts *GetModuleContentOption) (*hclext.BodyContent, error)
    GetOldFiles() (map[string]*hcl.File, error)

    // Access to NEW configuration (tfbreak-specific)
    GetNewModuleContent(schema *hclext.BodySchema, opts *GetModuleContentOption) (*hclext.BodyContent, error)
    GetNewResourceContent(resourceType string, schema *hclext.BodySchema, opts *GetModuleContentOption) (*hclext.BodyContent, error)
    GetNewFiles() (map[string]*hcl.File, error)

    // Report findings back to tfbreak (tflint signature)
    EmitIssue(rule Rule, message string, issueRange hcl.Range) error

    // Rule configuration (tflint-aligned)
    DecodeRuleConfig(ruleName string, ret interface{}) error
}

// Severity levels (tflint-aligned: ERROR, WARNING, NOTICE)
type Severity int
const (
    ERROR   Severity = iota  // Resource will be destroyed/recreated
    WARNING                   // Potential issue, review recommended
    NOTICE                    // Informational
)
```

### Why tflint-Aligned Interface Matters

The interface follows tflint-plugin-sdk patterns for familiarity, while intentionally NOT prescribing:
- How plugins detect issues (schema-driven, hardcoded rules, ML, etc.)
- What plugins look for (ForceNew, naming, security, custom policies)
- How plugins are structured internally

This means:
- **Schema-driven plugin**: Can embed provider schema and query dynamically
- **Rule-based plugin**: Can implement specific rules for known resources
- **Policy plugin**: Can enforce organizational standards
- **Hybrid plugin**: Can combine approaches

### Key Deviation from tflint

tfbreak requires access to BOTH old and new configurations to detect breaking changes. This is a fundamental difference from tflint which only analyzes a single configuration. The Runner interface reflects this with dual methods:

| tflint | tfbreak |
|--------|---------|
| `GetModuleContent()` | `GetOldModuleContent()` / `GetNewModuleContent()` |
| `GetResourceContent()` | `GetOldResourceContent()` / `GetNewResourceContent()` |
| `GetFiles()` | `GetOldFiles()` / `GetNewFiles()` |

This deviation is intentional and necessary for tfbreak's core purpose.

### Plugin Discovery (tflint-aligned)

Plugins are discovered in priority order (first match wins):

1. `plugin_dir` attribute in `.tfbreak.hcl` config block
2. `TFBREAK_PLUGIN_DIR` environment variable
3. `./.tfbreak.d/plugins/` - Project-local plugins
4. `~/.tfbreak.d/plugins/` - User-installed plugins

Binary naming convention: `tfbreak-ruleset-{name}` (e.g., `tfbreak-ruleset-azurerm`)

### Plugin Configuration

Configuration via `.tfbreak.hcl`:

```hcl
plugin "azurerm" {
  enabled = true
  version = "0.1.0"
  source  = "github.com/jokarl/tfbreak-ruleset-azurerm"
}

# Plugin-specific rule configuration
rule "AZURERM_RESOURCE_GROUP_FORCE_NEW" {
  enabled  = true
  severity = "WARNING"  # Override default
}
```

### Handshake and Versioning (tflint-aligned)

```go
// Following tflint pattern: use a secure random MagicCookieValue
var Handshake = plugin.HandshakeConfig{
    ProtocolVersion:  1,
    MagicCookieKey:   "TFBREAK_RULESET_PLUGIN",
    MagicCookieValue: "8Jx2vKmPnQ3wYzL5bN7cR9dF1gH4jT6uA0sE2iO8pU5mW3xC7vB9",
}
```

Plugins can specify version constraints:
- `VersionConstraint() string` returns semver constraint (e.g., ">= 0.8.0")
- tfbreak checks compatibility before loading

### Protocol Buffer Definition

```protobuf
syntax = "proto3";
package tfbreak.plugin.v1;

service RuleSet {
  rpc GetRuleSetName(Empty) returns (GetRuleSetNameResponse);
  rpc GetRuleSetVersion(Empty) returns (GetRuleSetVersionResponse);
  rpc GetRuleNames(Empty) returns (GetRuleNamesResponse);
  rpc GetVersionConstraint(Empty) returns (GetVersionConstraintResponse);
  rpc Check(CheckRequest) returns (CheckResponse);
}

service Runner {
  rpc GetOldModuleContent(GetModuleContentRequest) returns (GetModuleContentResponse);
  rpc GetNewModuleContent(GetModuleContentRequest) returns (GetModuleContentResponse);
  rpc GetOldResourceContent(GetResourceContentRequest) returns (GetModuleContentResponse);
  rpc GetNewResourceContent(GetResourceContentRequest) returns (GetModuleContentResponse);
  rpc GetOldFiles(Empty) returns (GetFilesResponse);
  rpc GetNewFiles(Empty) returns (GetFilesResponse);
  rpc EmitIssue(EmitIssueRequest) returns (EmitIssueResponse);
}

message Issue {
  string rule_name = 1;
  Severity severity = 2;
  string message = 3;
  Range old_range = 4;
  Range new_range = 5;
  string remediation = 6;
}

enum Severity {
  SEVERITY_UNSPECIFIED = 0;
  SEVERITY_ERROR = 1;
  SEVERITY_WARNING = 2;
  SEVERITY_NOTICE = 3;
}

message Range {
  string filename = 1;
  Position start = 2;
  Position end = 3;
}

message Position {
  int32 line = 1;
  int32 column = 2;
  int32 byte = 3;
}
```

### Example Plugin Implementation (tflint-aligned)

```go
// cmd/tfbreak-ruleset-azurerm/main.go
package main

import (
    "github.com/jokarl/tfbreak-core/plugin"
    "github.com/jokarl/tfbreak-ruleset-azurerm/rules"
)

func main() {
    plugin.Serve(&plugin.ServeOpts{
        RuleSet: rules.NewRuleSet(),
    })
}

// internal/rules/ruleset.go
type RuleSet struct {
    tflint.BuiltinRuleSet  // Embed for default implementations
    rules []tflint.Rule
}

func NewRuleSet() *RuleSet {
    return &RuleSet{
        rules: []tflint.Rule{
            &ForceNewLocationRule{},
        },
    }
}

func (r *RuleSet) RuleSetName() string      { return "azurerm" }
func (r *RuleSet) RuleSetVersion() string   { return "0.1.0" }
func (r *RuleSet) RuleNames() []string      { return []string{"azurerm_force_new_location"} }
func (r *RuleSet) VersionConstraint() string { return ">= 0.8.0" }

// internal/rules/force_new_location.go
type ForceNewLocationRule struct {
    tflint.DefaultRule  // Embed for default Enabled(), Severity(), Link()
}

func (r *ForceNewLocationRule) Name() string       { return "azurerm_force_new_location" }
func (r *ForceNewLocationRule) Severity() Severity { return tflint.ERROR }
func (r *ForceNewLocationRule) Link() string       { return "https://docs.tfbreak.io/rules/azurerm_force_new_location" }

func (r *ForceNewLocationRule) Check(runner tflint.Runner) error {
    schema := &hclext.BodySchema{
        Attributes: []hclext.AttributeSchema{{Name: "location"}},
    }

    // Get resources from BOTH configs (tfbreak-specific)
    oldContent, _ := runner.GetOldResourceContent("azurerm_resource_group", schema, nil)
    newContent, _ := runner.GetNewResourceContent("azurerm_resource_group", schema, nil)

    // Plugin's detection logic: compare old vs new
    for _, newRes := range newContent.Blocks {
        oldRes := findMatchingResource(oldContent, newRes)
        if oldRes != nil && locationChanged(oldRes, newRes) {
            runner.EmitIssue(
                r,  // Pass rule object (tflint pattern)
                "Changing 'location' forces resource recreation",
                newRes.Body.Attributes["location"].Expr.Range(),
            )
        }
    }

    return nil
}
```

### Phased Implementation

**Phase 1: Core Plugin Infrastructure**
- Define protobuf schema
- Implement plugin discovery and loading
- Implement Runner (host side)
- Create plugin SDK package

**Phase 2: Reference Plugin**
- Build tfbreak-ruleset-azurerm
- Demonstrate schema-driven approach
- Document plugin development

**Phase 3: Ecosystem**
- Plugin installation command
- Plugin registry integration
- Community guidelines

### References

- [tflint architecture](https://github.com/terraform-linters/tflint/blob/master/docs/developer-guide/architecture.md)
- [tflint-plugin-sdk](https://github.com/terraform-linters/tflint-plugin-sdk)
- [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)

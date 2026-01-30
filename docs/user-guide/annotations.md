# Annotations Guide

tfbreak supports inline annotations to suppress findings. This is useful when breaking changes are intentional and have been properly communicated to consumers.

## Annotation Format

tfbreak uses a tflint-compatible annotation format:

```hcl
# tfbreak:ignore <rule_name_or_id> [# reason]
```

Annotations are placed as comments immediately before the block they apply to.

## Basic Usage

### Suppressing a Single Rule

Use the rule name or rule ID:

```hcl
# Suppress by rule name (recommended)
# tfbreak:ignore required-input-added
variable "new_required_var" {
  type = string
}

# Suppress by rule ID
# tfbreak:ignore BC001
variable "another_required_var" {
  type = string
}
```

### Suppressing Multiple Rules

Separate rule names with commas:

```hcl
# tfbreak:ignore required-input-added,input-type-changed
variable "new_var" {
  type = number
}
```

### Suppressing All Rules

Use the `all` keyword:

```hcl
# tfbreak:ignore all
variable "experimental_var" {
  type = any
}
```

## Adding Reasons

Always document why a finding is being suppressed. Reasons are added after a `#` following the rule names:

```hcl
# tfbreak:ignore input-removed # deprecated in v2.0, removing in v3.0
```

```hcl
# tfbreak:ignore required-input-added # intentional breaking change for JIRA-1234
variable "api_key" {
  type = string
}
```

Reasons are displayed in tfbreak output and stored for audit purposes.

## Annotation Scopes

### Block-Level Annotations

By default, annotations apply to the immediately following block:

```hcl
# tfbreak:ignore input-removed # this variable is deprecated
variable "old_option" {
  type    = string
  default = ""
}

# This variable is NOT affected by the annotation above
variable "another_var" {
  type = string
}
```

### File-Level Annotations

Use `tfbreak:ignore-file` to suppress findings for an entire file:

```hcl
# tfbreak:ignore-file all # this file contains intentional breaking changes

variable "new_required_1" {
  type = string
}

variable "new_required_2" {
  type = string
}

output "new_output" {
  value = "..."
}
```

File-level annotations must appear before any blocks in the file.

## Rule Identifiers

You can use either the rule ID or rule name in annotations:

| Rule ID | Rule Name |
|---------|-----------|
| BC001 | required-input-added |
| BC002 | input-removed |
| BC003 | input-renamed |
| BC004 | input-type-changed |
| BC005 | input-default-removed |
| RC003 | input-renamed-optional |
| RC006 | input-default-changed |
| RC007 | input-nullable-changed |
| RC008 | input-sensitive-changed |
| RC012 | validation-added |
| RC013 | validation-value-removed |
| BC009 | output-removed |
| BC010 | output-renamed |
| RC011 | output-sensitive-changed |
| BC100 | resource-removed-no-moved |
| BC101 | module-removed-no-moved |
| BC102 | invalid-moved-block |
| BC103 | conflicting-moved |
| RC300 | module-source-changed |
| RC301 | module-version-changed |
| BC200 | terraform-version-constrained |
| BC201 | provider-version-constrained |

Using rule names is recommended as they are more descriptive.

## Governance

tfbreak supports governance rules to control how annotations are used.

### Requiring Reasons

Force all annotations to include a reason:

```hcl
# .tfbreak.hcl
annotations {
  require_reason = true
}
```

With this setting, annotations without reasons will not suppress findings:

```hcl
# This WILL NOT suppress the finding (no reason)
# tfbreak:ignore required-input-added

# This WILL suppress the finding (has reason)
# tfbreak:ignore required-input-added # approved in PR-123
```

### Allow Lists

Only allow ignoring specific rules:

```hcl
# .tfbreak.hcl
annotations {
  allow_rule_ids = ["BC001", "BC002", "RC006"]
}
```

With this setting, only the listed rules can be ignored. Attempting to ignore other rules will have no effect.

### Deny Lists

Prevent specific rules from ever being ignored:

```hcl
# .tfbreak.hcl
annotations {
  deny_rule_ids = ["BC100", "BC101"]
}
```

This is useful for critical rules (like state safety rules) that should never be bypassed.

### Combining Allow and Deny Lists

When both are specified, deny takes precedence:

```hcl
# .tfbreak.hcl
annotations {
  allow_rule_ids = ["BC001", "BC002", "BC100"]
  deny_rule_ids  = ["BC100"]
}
```

In this example, BC001 and BC002 can be ignored, but BC100 cannot (even though it appears in the allow list).

## Legacy Metadata Format

For backward compatibility, tfbreak also supports a legacy metadata format:

```hcl
# tfbreak:ignore required-input-added reason="deprecated" ticket="JIRA-123" expires="2025-12-31"
```

However, the tflint-style format with `# reason` is preferred for new annotations.

### Metadata Fields

| Field | Description |
|-------|-------------|
| `reason` | Documentation for why the ignore is needed |
| `ticket` | Issue tracker reference |
| `expires` | ISO date when the ignore should be reviewed (YYYY-MM-DD) |

## Disabling Annotations

### Via Configuration

```hcl
# .tfbreak.hcl
annotations {
  enabled = false
}
```

### Via CLI

```bash
tfbreak check ./old ./new --no-annotations
```

## Best Practices

### Do Document Everything

Always include a reason explaining why the finding is being suppressed:

```hcl
# Good
# tfbreak:ignore input-removed # migrated to new_option in v2.0, see MIGRATION.md

# Bad (no reason)
# tfbreak:ignore input-removed
```

### Do Use Narrow Scopes

Prefer block-level annotations over file-level when possible:

```hcl
# Good - suppresses only this specific variable
# tfbreak:ignore input-removed
variable "deprecated_var" { ... }

# Less good - suppresses everything in the file
# tfbreak:ignore-file all
```

### Do Use Rule Names

Rule names are more descriptive than IDs:

```hcl
# Good - clear what's being suppressed
# tfbreak:ignore required-input-added

# Less clear
# tfbreak:ignore BC001
```

### Do Reference Tickets

Link to issue trackers for audit trails:

```hcl
# tfbreak:ignore input-removed # JIRA-1234: deprecated in v2.0
```

### Do Set Expiration Dates

For temporary suppressions, use the expires metadata:

```hcl
# tfbreak:ignore input-removed reason="grace period" expires="2025-06-01"
```

### Do Configure Governance

Set up governance rules to enforce best practices across your organization:

```hcl
# .tfbreak.hcl
annotations {
  require_reason = true
  deny_rule_ids  = ["BC100", "BC101"]  # Never ignore state safety
}
```

## Examples

### Intentional Breaking Change

```hcl
# tfbreak:ignore required-input-added # v3.0 breaking change, documented in CHANGELOG.md
variable "encryption_key" {
  type        = string
  description = "Required encryption key (breaking change in v3.0)"
}
```

### Deprecation Period

```hcl
# tfbreak:ignore input-removed # grace period until 2025-06-01, see MIGRATION.md
# Note: This variable is deprecated. Use new_option instead.
variable "legacy_option" {
  type    = string
  default = ""
}
```

### Type Migration

```hcl
# tfbreak:ignore input-type-changed # migrating from string to number, backward compatible
variable "port" {
  type        = number
  default     = 443
  description = "Port number (changed from string to number in v2.1)"
}
```

### Multiple Suppressions

```hcl
# tfbreak:ignore all # experimental module, expect frequent breaking changes
variable "experimental_config" {
  type = object({
    feature_a = optional(bool)
    feature_b = optional(string)
  })
}
```

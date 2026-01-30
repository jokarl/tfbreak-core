# Configuration Reference

tfbreak is configured via a `.tfbreak.hcl` file using HCL syntax (the same format as Terraform).

## Config File Discovery

tfbreak searches for configuration in this order:

1. Explicit path via `--config` / `-c` flag
2. `.tfbreak.hcl` in the current working directory
3. `.tfbreak.hcl` in the old directory (first argument to `check`)

If no config file is found, tfbreak uses sensible defaults.

## Minimal Configuration

```hcl
version = 1
```

The `version` attribute is required and must be `1`.

## Full Configuration Reference

```hcl
# Configuration version (required)
version = 1

# Global settings
config {
  # Directory to search for plugins
  plugin_dir = "/path/to/plugins"
}

# Path filtering
paths {
  # Glob patterns for files to include (default: ["**/*.tf"])
  include = ["**/*.tf"]

  # Glob patterns for files to exclude (default: [".terraform/**"])
  exclude = [
    ".terraform/**",
    "**/examples/**",
    "**/test/**",
  ]
}

# Output settings
output {
  # Output format: text, json (default: text)
  format = "text"

  # Color mode: auto, always, never (default: auto)
  color = "auto"
}

# CI policy settings
policy {
  # Minimum severity to fail the check: ERROR, WARNING, NOTICE (default: ERROR)
  fail_on = "ERROR"

  # Treat WARNING findings as errors (default: false)
  treat_warnings_as_errors = false
}

# Annotation settings
annotations {
  # Enable annotation processing (default: true)
  enabled = true

  # Require reason in annotations (default: false)
  require_reason = false

  # Only allow ignoring these rule IDs (empty = all allowed)
  allow_rule_ids = []

  # Never allow ignoring these rule IDs
  deny_rule_ids = []
}

# Rename detection settings (opt-in)
rename_detection {
  # Enable rename detection (default: false)
  enabled = true

  # Minimum similarity threshold for rename detection (default: 0.85)
  # Value between 0.0 and 1.0, higher = stricter matching
  similarity_threshold = 0.85
}

# Per-rule configuration
rules "BC001" {
  enabled  = true
  severity = "ERROR"
}

rules "RC006" {
  enabled  = true
  severity = "WARNING"
}

# Plugin configuration
plugin "azurerm" {
  enabled = true
  version = "0.1.0"
  source  = "github.com/jokarl/tfbreak-ruleset-azurerm"
}
```

## Configuration Blocks

### `config` Block

Global settings for tfbreak.

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `plugin_dir` | string | (none) | Directory to search for plugins (highest priority) |

### `paths` Block

Controls which Terraform files are analyzed.

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `include` | list(string) | `["**/*.tf"]` | Glob patterns for files to include |
| `exclude` | list(string) | `[".terraform/**"]` | Glob patterns for files to exclude |

Glob patterns use [doublestar](https://github.com/bmatcuk/doublestar) syntax:
- `**` matches any number of directories
- `*` matches any sequence of characters within a path segment
- `?` matches any single character

Examples:
```hcl
paths {
  include = ["**/*.tf"]
  exclude = [
    ".terraform/**",      # Terraform cache
    "**/examples/**",     # Example modules
    "**/test/**",         # Test fixtures
    "modules/deprecated/**", # Deprecated modules
  ]
}
```

### `output` Block

Controls how tfbreak displays results.

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `format` | string | `"text"` | Output format: `text` or `json` |
| `color` | string | `"auto"` | Color mode: `auto`, `always`, or `never` |

The `auto` color mode enables colors when stdout is a terminal.

### `policy` Block

Controls CI behavior and exit codes.

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `fail_on` | string | `"ERROR"` | Minimum severity to fail: `ERROR`, `WARNING`, `NOTICE` |
| `treat_warnings_as_errors` | bool | `false` | Treat WARNING findings as errors |

### `annotations` Block

Controls how inline annotations (ignores) are processed.

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable annotation processing |
| `require_reason` | bool | `false` | Require a reason for all ignores |
| `allow_rule_ids` | list(string) | `[]` | Only allow ignoring these rules (empty = all) |
| `deny_rule_ids` | list(string) | `[]` | Never allow ignoring these rules |

See [Annotations](annotations.md) for more details.

### `rename_detection` Block

Controls the opt-in rename detection feature. When enabled, tfbreak can detect when variables or outputs are renamed rather than simply removed and added.

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `false` | Enable rename detection |
| `similarity_threshold` | float | `0.85` | Minimum similarity for name matching (0.0-1.0) |

When enabled, rename detection enables three additional rules:
- **BC003** - Required variable renamed
- **RC003** - Optional variable renamed
- **BC010** - Output renamed

These rules suppress the corresponding removal/addition rules when a rename is detected.

### `rules` Block

Per-rule configuration. Each block is labeled with the rule ID.

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable or disable the rule |
| `severity` | string | (rule default) | Override severity: `ERROR`, `WARNING`, `NOTICE` |

Example:
```hcl
# Disable a specific rule
rules "BC001" {
  enabled = false
}

# Downgrade a breaking change to a warning
rules "BC002" {
  enabled  = true
  severity = "WARNING"
}

# Upgrade a risky change to an error
rules "RC006" {
  enabled  = true
  severity = "ERROR"
}
```

### `plugin` Block

Plugin configuration. Each block is labeled with the plugin name.

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable or disable the plugin |
| `version` | string | (none) | Version constraint for the plugin |
| `source` | string | (none) | Plugin source (for future plugin installation) |

See [Plugins](plugins.md) for more details.

## CLI Flag Overrides

CLI flags take precedence over config file settings:

| Config | CLI Flag |
|--------|----------|
| `output.format` | `--format` |
| `output.color` | `--color` |
| `policy.fail_on` | `--fail-on` |
| `paths.include` | `--include` |
| `paths.exclude` | `--exclude` |
| `annotations.require_reason` | `--require-reason` |
| `annotations.enabled` | `--no-annotations` (inverse) |

Example:
```bash
# Override output format for this run
tfbreak check ./old ./new --format json

# Override fail threshold
tfbreak check ./old ./new --fail-on WARNING
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `TFBREAK_PLUGIN_DIR` | Directory to search for plugins (second priority after config) |

## Example Configurations

### Strict CI Configuration

```hcl
version = 1

policy {
  fail_on                  = "WARNING"
  treat_warnings_as_errors = true
}

annotations {
  enabled        = true
  require_reason = true
  deny_rule_ids  = ["BC100", "BC101"]  # Never allow ignoring state safety rules
}
```

### Lenient Development Configuration

```hcl
version = 1

policy {
  fail_on = "ERROR"
}

annotations {
  enabled        = true
  require_reason = false
}

# Downgrade some breaking changes to warnings during development
rules "BC001" {
  severity = "WARNING"
}

rules "BC002" {
  severity = "WARNING"
}
```

### Module Library Configuration

```hcl
version = 1

paths {
  include = ["modules/**/*.tf"]
  exclude = [
    ".terraform/**",
    "**/examples/**",
    "**/test/**",
  ]
}

rename_detection {
  enabled              = true
  similarity_threshold = 0.80
}

annotations {
  enabled        = true
  require_reason = true
}
```

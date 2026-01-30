# Plugin Guide

tfbreak supports plugins for extending its detection capabilities with provider-specific rules. This allows the core tool to remain lightweight while enabling specialized detection for providers like Azure, AWS, or GCP.

## Overview

Plugins are separate binaries that communicate with tfbreak via gRPC. This architecture:

- Keeps the core tool provider-agnostic
- Allows plugins to be developed and released independently
- Provides process isolation (plugin crashes do not affect tfbreak)
- Uses familiar patterns from the Terraform ecosystem (similar to tflint)

## Plugin Discovery

tfbreak discovers plugins in the following locations, in priority order:

1. **Config `plugin_dir`** - Directory specified in `.tfbreak.hcl`
2. **Environment variable** - `TFBREAK_PLUGIN_DIR`
3. **Local plugins** - `./.tfbreak.d/plugins/` (relative to current directory)
4. **User plugins** - `~/.tfbreak.d/plugins/` (user home directory)

The first matching plugin takes precedence if the same plugin exists in multiple locations.

## Plugin Naming Convention

Plugin binaries must follow the naming convention:

```
tfbreak-ruleset-{name}
```

For example:
- `tfbreak-ruleset-azurerm` - Azure provider rules
- `tfbreak-ruleset-aws` - AWS provider rules (future)
- `tfbreak-ruleset-gcp` - GCP provider rules (future)

On Windows, add the `.exe` extension: `tfbreak-ruleset-azurerm.exe`

## Installing Plugins

### Manual Installation

1. Download the plugin binary for your platform
2. Place it in one of the plugin discovery directories
3. Ensure the file is executable (on Unix: `chmod +x tfbreak-ruleset-*`)

Example:

```bash
# Create plugin directory
mkdir -p ~/.tfbreak.d/plugins

# Download and install plugin
curl -L -o ~/.tfbreak.d/plugins/tfbreak-ruleset-azurerm \
  https://github.com/jokarl/tfbreak-ruleset-azurerm/releases/download/v0.1.0/tfbreak-ruleset-azurerm-linux-amd64

chmod +x ~/.tfbreak.d/plugins/tfbreak-ruleset-azurerm
```

### Project-Local Plugins

For project-specific plugins, use the local plugin directory:

```bash
mkdir -p .tfbreak.d/plugins
cp /path/to/tfbreak-ruleset-custom .tfbreak.d/plugins/
```

## Plugin Configuration

Configure plugins in `.tfbreak.hcl`:

```hcl
version = 1

# Optional: override plugin directory
config {
  plugin_dir = "/custom/plugin/path"
}

# Enable and configure a plugin
plugin "azurerm" {
  enabled = true
  version = "0.1.0"
  source  = "github.com/jokarl/tfbreak-ruleset-azurerm"
}
```

### Plugin Block Attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable or disable the plugin |
| `version` | string | (none) | Version constraint (for future use) |
| `source` | string | (none) | Plugin source (for future use) |

### Disabling a Plugin

```hcl
plugin "azurerm" {
  enabled = false
}
```

## Available Plugins

### tfbreak-ruleset-azurerm

Detects Azure-specific breaking changes, including:

- **ForceNew attributes** - Changes that cause resource recreation in Azure
- **Resource group location changes** - Changing location forces recreation
- **SKU changes** - Some SKU changes cannot be done in-place

Repository: [github.com/jokarl/tfbreak-ruleset-azurerm](https://github.com/jokarl/tfbreak-ruleset-azurerm)

Example findings:

```
ERROR  AZURERM001  azurerm-force-new-location
  Changing 'location' on azurerm_resource_group forces recreation
  Location: main.tf:15
```

### More Plugins

Additional plugins for other providers are planned:

- `tfbreak-ruleset-aws` - AWS provider rules
- `tfbreak-ruleset-gcp` - GCP provider rules

## Developing Plugins

Plugins are built using the [tfbreak-plugin-sdk](https://github.com/jokarl/tfbreak-plugin-sdk).

### Plugin Architecture

```
tfbreak (host) <--gRPC--> plugin (subprocess)
    |                          |
    +- Provides Runner         +- Implements RuleSet
    |   - GetOldModuleContent()|   - RuleSetName()
    |   - GetNewModuleContent()|   - RuleSetVersion()
    |   - EmitIssue()          |   - RuleNames()
    |                          |
    +- Aggregates findings     +- Contains Rules
                                   - Check(runner)
```

### SDK Installation

```bash
go get github.com/jokarl/tfbreak-plugin-sdk
```

### Creating a Plugin

1. Create a new Go module:

```bash
mkdir tfbreak-ruleset-myprovider
cd tfbreak-ruleset-myprovider
go mod init github.com/yourorg/tfbreak-ruleset-myprovider
```

2. Implement your rules:

```go
package rules

import (
    "github.com/hashicorp/hcl/v2"
    sdk "github.com/jokarl/tfbreak-plugin-sdk"
)

type MyRule struct {
    sdk.DefaultRule
}

func (r *MyRule) Name() string {
    return "my-provider-rule"
}

func (r *MyRule) Severity() sdk.Severity {
    return sdk.ERROR
}

func (r *MyRule) Check(runner sdk.Runner) error {
    // Get old and new configurations
    oldContent, _ := runner.GetOldResourceContent("my_resource", schema, nil)
    newContent, _ := runner.GetNewResourceContent("my_resource", schema, nil)

    // Compare and emit issues
    for _, newRes := range newContent.Blocks {
        // ... detection logic ...
        runner.EmitIssue(r, "Breaking change detected", newRes.Range())
    }

    return nil
}
```

3. Create the plugin main:

```go
package main

import (
    "github.com/jokarl/tfbreak-plugin-sdk/plugin"
    "github.com/yourorg/tfbreak-ruleset-myprovider/rules"
)

func main() {
    plugin.Serve(&plugin.ServeOpts{
        RuleSet: rules.NewRuleSet(),
    })
}
```

4. Build the plugin:

```bash
go build -o tfbreak-ruleset-myprovider ./cmd/tfbreak-ruleset-myprovider
```

### Plugin Interface

The key interfaces from the SDK:

```go
// RuleSet is the container for a plugin's rules
type RuleSet interface {
    RuleSetName() string
    RuleSetVersion() string
    RuleNames() []string
}

// Rule is implemented by individual detection rules
type Rule interface {
    Name() string
    Enabled() bool
    Severity() Severity
    Check(runner Runner) error
}

// Runner provides access to configurations and emitting issues
type Runner interface {
    // Old configuration access
    GetOldModuleContent(schema, opts) (*Content, error)
    GetOldResourceContent(type, schema, opts) (*Content, error)

    // New configuration access
    GetNewModuleContent(schema, opts) (*Content, error)
    GetNewResourceContent(type, schema, opts) (*Content, error)

    // Emit findings
    EmitIssue(rule Rule, message string, location hcl.Range) error
}
```

### Key Differences from tflint

Unlike tflint which analyzes a single configuration, tfbreak plugins have access to both old and new configurations via separate methods:

| tflint | tfbreak |
|--------|---------|
| `GetModuleContent()` | `GetOldModuleContent()` / `GetNewModuleContent()` |
| `GetResourceContent()` | `GetOldResourceContent()` / `GetNewResourceContent()` |

This enables diff-based analysis for detecting breaking changes.

## Troubleshooting

### Plugin Not Found

If a plugin is not being discovered:

1. Verify the binary is in one of the search paths
2. Check the naming convention (`tfbreak-ruleset-{name}`)
3. Ensure the file is executable

```bash
# List discovered plugins
ls ~/.tfbreak.d/plugins/
ls ./.tfbreak.d/plugins/

# Check if executable
file ~/.tfbreak.d/plugins/tfbreak-ruleset-azurerm
```

### Plugin Errors

Plugin errors are logged to stderr. Enable verbose mode for more details:

```bash
tfbreak check ./old ./new --verbose
```

### Version Mismatch

If a plugin requires a newer version of tfbreak, you will see an error like:

```
Plugin "azurerm" requires tfbreak >= 0.8.0, but current version is 0.7.0
```

Upgrade tfbreak to the required version.

## Future: Plugin Installation

A future version of tfbreak will include a plugin installation command:

```bash
# Install from GitHub releases (future)
tfbreak plugin install github.com/jokarl/tfbreak-ruleset-azurerm

# Install specific version (future)
tfbreak plugin install github.com/jokarl/tfbreak-ruleset-azurerm@0.1.0

# List installed plugins (future)
tfbreak plugin list
```

Until then, plugins must be installed manually as described above.

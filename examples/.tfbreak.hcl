# tfbreak configuration file
# Copy this to your project root as .tfbreak.hcl
# Documentation: https://github.com/jokarl/tfbreak-core

version = 1

# Global settings
config {
  # Custom plugin directory (overrides default locations)
  # plugin_dir = "/path/to/plugins"
}

# Path filtering - controls which files are analyzed
paths {
  include = ["**/*.tf"]
  exclude = [
    ".terraform/**",
    # "**/examples/**",
    # "**/test/**",
  ]
}

# Output settings
output {
  format = "text"  # text, json, compact, checkstyle, junit, sarif
  color = "auto"   # auto, always, never
}

# CI policy settings
policy {
  fail_on = "ERROR"  # ERROR, WARNING, NOTICE
  treat_warnings_as_errors = false
}

# Annotation settings (in-code ignores: # tfbreak:ignore)
annotations {
  enabled = true
  require_reason = false
  allow_rule_ids = []  # Empty = all allowed
  deny_rule_ids = []   # Never allow ignoring these
}

# Rename detection (opt-in heuristic rules)
# rename_detection {
#   enabled = true
#   similarity_threshold = 0.85  # 0.0-1.0, higher = stricter
# }

# Plugin configuration
# Plugins are discovered from:
# 1. plugin_dir from config block
# 2. TFBREAK_PLUGIN_DIR environment variable
# 3. ./.tfbreak.d/plugins/ (project-local)
# 4. ~/.tfbreak.d/plugins/ (user home)
#
# plugin "azurerm" {
#   enabled = true
#   version = "0.1.0"
#   source  = "github.com/jokarl/tfbreak-ruleset-azurerm"
# }

# Per-rule configuration
# rules "BC001" {
#   enabled  = true
#   severity = "ERROR"
# }

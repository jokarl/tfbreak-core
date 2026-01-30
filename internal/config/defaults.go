package config

// Default returns the default configuration
func Default() *Config {
	enabled := true
	renameDisabled := false
	defaultThreshold := DefaultSimilarityThreshold
	return &Config{
		Version: 1,
		ConfigBlock: &ConfigBlockConfig{
			PluginDir: "",
		},
		Paths: &PathsConfig{
			Include: []string{"**/*.tf"},
			Exclude: []string{".terraform/**"},
		},
		Output: &OutputConfig{
			Format: "text",
			Color:  "auto",
		},
		Policy: &PolicyConfig{
			FailOn:                "ERROR",
			TreatWarningsAsErrors: false,
		},
		Annotations: &AnnotationsConfig{
			Enabled:       &enabled,
			RequireReason: false,
			AllowRuleIDs:  []string{},
			DenyRuleIDs:   []string{},
		},
		RenameDetection: &RenameDetectionConfig{
			Enabled:             &renameDisabled,
			SimilarityThreshold: &defaultThreshold,
		},
		Rules:   []*RuleConfig{},
		Plugins: []*PluginConfig{},
	}
}

// DefaultConfigHCL returns the default configuration as an HCL string
// with documentation comments for the init command
func DefaultConfigHCL() string {
	return `# tfbreak configuration file
# Documentation: https://github.com/jokarl/tfbreak-core

# Configuration version (required)
version = 1

# Path filtering
# Controls which files are analyzed
paths {
  # Glob patterns for files to include (default: ["**/*.tf"])
  include = ["**/*.tf"]

  # Glob patterns for files to exclude (default: [".terraform/**"])
  exclude = [
    ".terraform/**",
    # "**/examples/**",
    # "**/test/**",
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
# Controls how in-code ignores (# tfbreak:ignore) are processed
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
# Enables heuristic rules that detect renamed variables and outputs
# rename_detection {
#   # Enable rename detection (default: false)
#   enabled = true
#
#   # Minimum similarity threshold for rename detection (default: 0.85)
#   # Value between 0.0 and 1.0, higher = stricter matching
#   similarity_threshold = 0.85
# }

# Global settings (tflint-aligned)
# config {
#   # Custom plugin directory (overrides default locations)
#   plugin_dir = "/path/to/plugins"
# }

# Plugin configuration
# Plugins are discovered from:
# 1. plugin_dir from config block above
# 2. TFBREAK_PLUGIN_DIR environment variable
# 3. ./.tfbreak.d/plugins/ (project-local)
# 4. ~/.tfbreak.d/plugins/ (user home)
#
# plugin "azurerm" {
#   enabled = true
#   version = "0.1.0"
#   source  = "github.com/jokarl/tfbreak-ruleset-azurerm"
# }
#
# plugin "aws" {
#   enabled = true
# }

# Per-rule configuration
# Uncomment and modify to customize rule behavior
#
# rules "BC001" {
#   enabled  = true
#   severity = "ERROR"
# }
#
# rules "BC002" {
#   enabled  = true
#   severity = "ERROR"
# }
#
# rules "BC005" {
#   enabled  = true
#   severity = "ERROR"
# }
#
# rules "RC006" {
#   enabled  = true
#   severity = "WARNING"
# }
#
# rules "BC009" {
#   enabled  = true
#   severity = "ERROR"
# }
#
# rules "BC100" {
#   enabled  = true
#   severity = "ERROR"
# }
#
# rules "BC101" {
#   enabled  = true
#   severity = "ERROR"
# }
#
# rules "BC102" {
#   enabled  = true
#   severity = "ERROR"
# }
#
# rules "BC103" {
#   enabled  = true
#   severity = "ERROR"
# }
`
}

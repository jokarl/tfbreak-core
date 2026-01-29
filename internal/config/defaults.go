package config

// Default returns the default configuration
func Default() *Config {
	enabled := true
	renameDisabled := false
	defaultThreshold := DefaultSimilarityThreshold
	return &Config{
		Version: 1,
		Paths: &PathsConfig{
			Include: []string{"**/*.tf"},
			Exclude: []string{".terraform/**"},
		},
		Output: &OutputConfig{
			Format: "text",
			Color:  "auto",
		},
		Policy: &PolicyConfig{
			FailOn:                "BREAKING",
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
		Rules: []*RuleConfig{},
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
  # Minimum severity to fail the check: BREAKING, RISKY, INFO (default: BREAKING)
  fail_on = "BREAKING"

  # Treat RISKY findings as errors (default: false)
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

# Per-rule configuration
# Uncomment and modify to customize rule behavior
#
# rules "BC001" {
#   enabled  = true
#   severity = "BREAKING"
# }
#
# rules "BC002" {
#   enabled  = true
#   severity = "BREAKING"
# }
#
# rules "BC005" {
#   enabled  = true
#   severity = "BREAKING"
# }
#
# rules "RC006" {
#   enabled  = true
#   severity = "RISKY"
# }
#
# rules "BC009" {
#   enabled  = true
#   severity = "BREAKING"
# }
#
# rules "BC100" {
#   enabled  = true
#   severity = "BREAKING"
# }
#
# rules "BC101" {
#   enabled  = true
#   severity = "BREAKING"
# }
#
# rules "BC102" {
#   enabled  = true
#   severity = "BREAKING"
# }
#
# rules "BC103" {
#   enabled  = true
#   severity = "BREAKING"
# }
`
}

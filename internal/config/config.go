// Package config handles loading and validating tfbreak configuration files.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// Config represents the tfbreak configuration
type Config struct {
	Version     int                `hcl:"version,attr"`
	Paths       *PathsConfig       `hcl:"paths,block"`
	Output      *OutputConfig      `hcl:"output,block"`
	Policy      *PolicyConfig      `hcl:"policy,block"`
	Annotations *AnnotationsConfig `hcl:"annotations,block"`
	Rules       []*RuleConfig      `hcl:"rules,block"`

	// Internal: path to the loaded config file (empty if using defaults)
	configPath string
}

// PathsConfig defines path filtering settings
type PathsConfig struct {
	Include []string `hcl:"include,attr"`
	Exclude []string `hcl:"exclude,attr"`
}

// OutputConfig defines output settings
type OutputConfig struct {
	Format string `hcl:"format,attr"`
	Color  string `hcl:"color,attr"`
}

// PolicyConfig defines CI policy settings
type PolicyConfig struct {
	FailOn                 string `hcl:"fail_on,attr"`
	TreatWarningsAsErrors  bool   `hcl:"treat_warnings_as_errors,optional"`
}

// AnnotationsConfig defines annotation/ignore settings
type AnnotationsConfig struct {
	Enabled      *bool    `hcl:"enabled,attr"`
	RequireReason bool    `hcl:"require_reason,optional"`
	AllowRuleIDs []string `hcl:"allow_rule_ids,optional"`
	DenyRuleIDs  []string `hcl:"deny_rule_ids,optional"`
}

// RuleConfig defines per-rule configuration
type RuleConfig struct {
	ID       string  `hcl:"id,label"`
	Enabled  *bool   `hcl:"enabled,attr"`
	Severity *string `hcl:"severity,attr"`
}

// ConfigPath returns the path to the loaded config file, or empty if using defaults
func (c *Config) ConfigPath() string {
	return c.configPath
}

// GetRuleConfig returns the configuration for a specific rule, or nil if not configured
func (c *Config) GetRuleConfig(ruleID string) *RuleConfig {
	for _, rc := range c.Rules {
		if rc.ID == ruleID {
			return rc
		}
	}
	return nil
}

// IsRuleEnabled returns whether a rule is enabled based on config
func (c *Config) IsRuleEnabled(ruleID string) bool {
	rc := c.GetRuleConfig(ruleID)
	if rc == nil || rc.Enabled == nil {
		return true // enabled by default
	}
	return *rc.Enabled
}

// GetRuleSeverity returns the configured severity for a rule, or the default if not configured
func (c *Config) GetRuleSeverity(ruleID string, defaultSeverity types.Severity) types.Severity {
	rc := c.GetRuleConfig(ruleID)
	if rc == nil || rc.Severity == nil {
		return defaultSeverity
	}
	sev, err := types.ParseSeverity(*rc.Severity)
	if err != nil {
		return defaultSeverity
	}
	return sev
}

// IsAnnotationsEnabled returns whether annotations are enabled
func (c *Config) IsAnnotationsEnabled() bool {
	if c.Annotations == nil || c.Annotations.Enabled == nil {
		return true // enabled by default
	}
	return *c.Annotations.Enabled
}

// Load loads configuration from the specified path or searches for it
// Search order: configPath (if provided), .tfbreak.hcl in cwd, .tfbreak.hcl in oldDir
func Load(configPath, oldDir string) (*Config, error) {
	var path string

	if configPath != "" {
		// Explicit path provided
		path = configPath
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
	} else {
		// Search for config file
		path = findConfigFile(oldDir)
	}

	if path == "" {
		// No config found, use defaults
		return Default(), nil
	}

	return loadFromFile(path)
}

// findConfigFile searches for .tfbreak.hcl in standard locations
func findConfigFile(oldDir string) string {
	// Check current directory
	cwd, err := os.Getwd()
	if err == nil {
		cwdPath := filepath.Join(cwd, ".tfbreak.hcl")
		if _, err := os.Stat(cwdPath); err == nil {
			return cwdPath
		}
	}

	// Check old directory
	if oldDir != "" {
		oldDirPath := filepath.Join(oldDir, ".tfbreak.hcl")
		if _, err := os.Stat(oldDirPath); err == nil {
			return oldDirPath
		}
	}

	return ""
}

// loadFromFile loads and parses a configuration file
func loadFromFile(path string) (*Config, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse config file: %s", formatDiagnostics(diags))
	}

	var config Config
	decodeDiags := gohcl.DecodeBody(file.Body, nil, &config)
	if decodeDiags.HasErrors() {
		return nil, fmt.Errorf("failed to decode config: %s", formatDiagnostics(decodeDiags))
	}

	config.configPath = path

	// Apply defaults for missing optional blocks
	applyDefaults(&config)

	// Validate
	if err := Validate(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// formatDiagnostics formats HCL diagnostics into a readable error string
func formatDiagnostics(diags hcl.Diagnostics) string {
	if len(diags) == 0 {
		return ""
	}

	var b strings.Builder
	for i, diag := range diags {
		if i > 0 {
			b.WriteString("; ")
		}
		if diag.Subject != nil {
			fmt.Fprintf(&b, "%s:%d: ", diag.Subject.Filename, diag.Subject.Start.Line)
		}
		b.WriteString(diag.Summary)
		if diag.Detail != "" {
			b.WriteString(": ")
			b.WriteString(diag.Detail)
		}
	}
	return b.String()
}

// applyDefaults fills in default values for missing optional config blocks
func applyDefaults(cfg *Config) {
	defaults := Default()

	if cfg.Paths == nil {
		cfg.Paths = defaults.Paths
	} else {
		if len(cfg.Paths.Include) == 0 {
			cfg.Paths.Include = defaults.Paths.Include
		}
		if len(cfg.Paths.Exclude) == 0 {
			cfg.Paths.Exclude = defaults.Paths.Exclude
		}
	}

	if cfg.Output == nil {
		cfg.Output = defaults.Output
	} else {
		if cfg.Output.Format == "" {
			cfg.Output.Format = defaults.Output.Format
		}
		if cfg.Output.Color == "" {
			cfg.Output.Color = defaults.Output.Color
		}
	}

	if cfg.Policy == nil {
		cfg.Policy = defaults.Policy
	} else {
		if cfg.Policy.FailOn == "" {
			cfg.Policy.FailOn = defaults.Policy.FailOn
		}
	}

	if cfg.Annotations == nil {
		cfg.Annotations = defaults.Annotations
	}
}

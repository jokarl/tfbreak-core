package rules

import "github.com/jokarl/tfbreak-core/internal/types"

// Rule defines the interface for a breaking change detection rule
type Rule interface {
	// ID returns the unique identifier for this rule (e.g., "BC001")
	ID() string

	// Name returns the human-readable name (e.g., "required-input-added")
	Name() string

	// Description returns a description of what this rule detects
	Description() string

	// DefaultSeverity returns the default severity level for this rule
	DefaultSeverity() types.Severity

	// Evaluate checks the old and new snapshots and returns any findings
	Evaluate(old, new *types.ModuleSnapshot) []*types.Finding
}

// RuleConfig holds configuration for a single rule
type RuleConfig struct {
	Enabled  bool
	Severity types.Severity
	Options  map[string]interface{}
}

// DefaultRuleConfig returns the default configuration for a rule
func DefaultRuleConfig(r Rule) *RuleConfig {
	return &RuleConfig{
		Enabled:  true,
		Severity: r.DefaultSeverity(),
		Options:  make(map[string]interface{}),
	}
}

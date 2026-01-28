package config

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// ValidRuleIDs contains all valid rule IDs
var ValidRuleIDs = map[string]bool{
	"BC001": true,
	"BC002": true,
	"BC005": true,
	"RC006": true,
	"BC009": true,
	"BC100": true,
	"BC101": true,
	"BC102": true,
	"BC103": true,
	// Phase 3 rules (not yet implemented but reserve the IDs)
	"BC003": true,
	"BC004": true,
	"RC007": true,
	"RC008": true,
	"BC010": true,
	"RC011": true,
	"BC200": true,
	"BC201": true,
}

// Validate validates the configuration
func Validate(cfg *Config) error {
	// Version check
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version: %d (only version 1 is supported)", cfg.Version)
	}

	// Validate output format
	if cfg.Output != nil && cfg.Output.Format != "" {
		switch cfg.Output.Format {
		case "text", "json":
			// valid
		default:
			return fmt.Errorf("invalid output format: %s (must be 'text' or 'json')", cfg.Output.Format)
		}
	}

	// Validate output color
	if cfg.Output != nil && cfg.Output.Color != "" {
		switch cfg.Output.Color {
		case "auto", "always", "never":
			// valid
		default:
			return fmt.Errorf("invalid color mode: %s (must be 'auto', 'always', or 'never')", cfg.Output.Color)
		}
	}

	// Validate policy fail_on
	if cfg.Policy != nil && cfg.Policy.FailOn != "" {
		if _, err := types.ParseSeverity(cfg.Policy.FailOn); err != nil {
			return fmt.Errorf("invalid fail_on severity: %s (must be 'BREAKING', 'RISKY', or 'INFO')", cfg.Policy.FailOn)
		}
	}

	// Validate rule configurations
	for _, rule := range cfg.Rules {
		if !ValidRuleIDs[rule.ID] {
			return fmt.Errorf("unknown rule ID: %s", rule.ID)
		}

		if rule.Severity != nil {
			if _, err := types.ParseSeverity(*rule.Severity); err != nil {
				return fmt.Errorf("invalid severity for rule %s: %s", rule.ID, *rule.Severity)
			}
		}
	}

	// Validate annotation allow_rule_ids
	if cfg.Annotations != nil {
		for _, ruleID := range cfg.Annotations.AllowRuleIDs {
			if !ValidRuleIDs[ruleID] {
				return fmt.Errorf("unknown rule ID in allow_rule_ids: %s", ruleID)
			}
		}

		for _, ruleID := range cfg.Annotations.DenyRuleIDs {
			if !ValidRuleIDs[ruleID] {
				return fmt.Errorf("unknown rule ID in deny_rule_ids: %s", ruleID)
			}
		}
	}

	return nil
}

// ValidateRuleID checks if a rule ID is valid
func ValidateRuleID(ruleID string) bool {
	return ValidRuleIDs[ruleID]
}

package config

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// RuleValidator validates rule identifiers
type RuleValidator interface {
	// IsValidRuleID returns true if the rule ID is valid
	IsValidRuleID(ruleID string) bool
	// IsValidRuleName returns true if the rule name is valid
	IsValidRuleName(name string) bool
	// ResolveToID resolves a rule name or ID to a canonical rule ID
	// Returns the ID and true if valid, empty string and false if invalid
	ResolveToID(nameOrID string) (string, bool)
}

// defaultValidator is set by the rules package during init
var defaultValidator RuleValidator

// SetRuleValidator sets the global rule validator
func SetRuleValidator(v RuleValidator) {
	defaultValidator = v
}

// fallbackValidator is used when no validator is set (for testing)
type fallbackValidator struct{}

func (f fallbackValidator) IsValidRuleID(ruleID string) bool {
	// Check if any rule name maps to this ID
	for _, id := range ValidRuleNames {
		if id == ruleID {
			return true
		}
	}
	return false
}

func (f fallbackValidator) IsValidRuleName(name string) bool {
	_, ok := ValidRuleNames[name]
	return ok
}

func (f fallbackValidator) ResolveToID(nameOrID string) (string, bool) {
	// Only accept rule names, not legacy IDs
	if id, ok := ValidRuleNames[nameOrID]; ok {
		return id, true
	}
	return "", false
}

// ValidRuleNames maps rule names to IDs (fallback when no validator is set)
// Only rule names are accepted - legacy rule codes (BC001, etc.) are not supported
var ValidRuleNames = map[string]string{
	"required-input-added":          "BC001",
	"input-removed":                 "BC002",
	"input-renamed":                 "BC003",
	"input-type-changed":            "BC004",
	"input-default-removed":         "BC005",
	"output-removed":                "BC009",
	"output-renamed":                "BC010",
	"resource-removed-no-moved":     "BC100",
	"module-removed-no-moved":       "BC101",
	"invalid-moved-block":           "BC102",
	"conflicting-moved":             "BC103",
	"input-renamed-optional":        "RC003",
	"input-default-changed":         "RC006",
	"input-nullable-changed":        "RC007",
	"input-sensitive-changed":       "RC008",
	"output-sensitive-changed":      "RC011",
	"validation-added":              "RC012",
	"validation-value-removed":      "RC013",
	"terraform-version-constrained": "BC200",
	"provider-version-constrained":  "BC201",
	"module-source-changed":         "RC300",
	"module-version-changed":        "RC301",
}

// getValidator returns the current rule validator
func getValidator() RuleValidator {
	if defaultValidator != nil {
		return defaultValidator
	}
	return fallbackValidator{}
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

	// Validate rename detection config
	if cfg.RenameDetection != nil && cfg.RenameDetection.SimilarityThreshold != nil {
		threshold := *cfg.RenameDetection.SimilarityThreshold
		if threshold < 0.0 || threshold > 1.0 {
			return fmt.Errorf("invalid similarity_threshold: %f (must be between 0.0 and 1.0)", threshold)
		}
	}

	validator := getValidator()

	// Validate rule configurations
	for _, rule := range cfg.Rules {
		if _, ok := validator.ResolveToID(rule.ID); !ok {
			return fmt.Errorf("unknown rule: %s", rule.ID)
		}

		if rule.Severity != nil {
			if _, err := types.ParseSeverity(*rule.Severity); err != nil {
				return fmt.Errorf("invalid severity for rule %s: %s", rule.ID, *rule.Severity)
			}
		}
	}

	// Validate annotation allow_rule_ids and deny_rule_ids
	// Only rule names are accepted (e.g., required-input-added)
	if cfg.Annotations != nil {
		for _, ruleSpec := range cfg.Annotations.AllowRuleIDs {
			if _, ok := validator.ResolveToID(ruleSpec); !ok {
				return fmt.Errorf("unknown rule in allow_rule_ids: %s", ruleSpec)
			}
		}

		for _, ruleSpec := range cfg.Annotations.DenyRuleIDs {
			if _, ok := validator.ResolveToID(ruleSpec); !ok {
				return fmt.Errorf("unknown rule in deny_rule_ids: %s", ruleSpec)
			}
		}
	}

	return nil
}

// ValidateRuleID checks if a rule ID or name is valid
func ValidateRuleID(ruleIDOrName string) bool {
	_, ok := getValidator().ResolveToID(ruleIDOrName)
	return ok
}

package rules

import (
	"encoding/json"
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// RC006 detects when a variable's default value changes
type RC006 struct{}

func init() {
	Register(&RC006{})
}

func (r *RC006) ID() string {
	return "RC006"
}

func (r *RC006) Name() string {
	return "input-default-changed"
}

func (r *RC006) Description() string {
	return "A variable's default value changed, which may cause unexpected behavior"
}

func (r *RC006) DefaultSeverity() types.Severity {
	return types.SeverityWarning
}

func (r *RC006) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `variable "instance_type" {
  type    = string
  default = "t3.micro"
}`,
		ExampleNew: `variable "instance_type" {
  type    = string
  default = "t3.small"  # Changed!
}`,
		Remediation: `This is a RISKY change because callers that rely on the default
may experience different behavior. Consider:
1. Documenting the change in your changelog
2. Notifying callers of the change
3. Using an annotation if this is intentional:
   # tfbreak:ignore input-default-changed # intentional upgrade`,
	}
}

func (r *RC006) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldVar := range old.Variables {
		newVar, exists := new.Variables[name]
		if !exists {
			// Variable was removed - handled by BC002
			continue
		}

		// Both must have defaults to compare
		if oldVar.Required || newVar.Required {
			continue
		}

		// Compare default values using JSON serialization
		if !defaultsEqual(oldVar.Default, newVar.Default) {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Variable %q default changed: %v -> %v", name,
					formatDefault(oldVar.Default), formatDefault(newVar.Default)),
			).WithOldLocation(&oldVar.DeclRange).
				WithNewLocation(&newVar.DeclRange)

			findings = append(findings, finding)
		}
	}

	return findings
}

// defaultsEqual compares two default values using JSON serialization
func defaultsEqual(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Serialize to JSON for comparison
	aJSON, errA := json.Marshal(a)
	bJSON, errB := json.Marshal(b)

	if errA != nil || errB != nil {
		// If serialization fails, fall back to direct comparison
		return a == b
	}

	return string(aJSON) == string(bJSON)
}

// formatDefault formats a default value for display
func formatDefault(v interface{}) string {
	if v == nil {
		return "null"
	}

	// Try to format as JSON for complex types
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}

	// For strings, show them quoted
	s := string(data)
	if len(s) > 50 {
		return s[:47] + "..."
	}
	return s
}

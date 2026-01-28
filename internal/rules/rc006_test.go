package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestRC006_DefaultChanged(t *testing.T) {
	rule := &RC006{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: false,
		Default:  "old_value",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: false,
		Default:  "new_value",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "RC006" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "RC006")
	}
	if f.Severity != types.SeverityRisky {
		t.Errorf("Severity = %v, want %v", f.Severity, types.SeverityRisky)
	}
}

func TestRC006_DefaultUnchanged(t *testing.T) {
	rule := &RC006{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: false,
		Default:  "same_value",
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: false,
		Default:  "same_value",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when default unchanged, got %d", len(findings))
	}
}

func TestRC006_ComplexDefaultChanged(t *testing.T) {
	rule := &RC006{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: false,
		Default:  map[string]interface{}{"key": "old"},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: false,
		Default:  map[string]interface{}{"key": "new"},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for complex default change, got %d", len(findings))
	}
}

func TestRC006_ComplexDefaultUnchanged(t *testing.T) {
	rule := &RC006{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: false,
		Default:  map[string]interface{}{"key": "value"},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: false,
		Default:  map[string]interface{}{"key": "value"},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when complex default unchanged, got %d", len(findings))
	}
}

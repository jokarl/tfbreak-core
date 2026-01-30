package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC001_NewRequiredVariable(t *testing.T) {
	rule := &BC001{}

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	new.Variables["new_var"] = &types.VariableSignature{
		Name:     "new_var",
		Required: true,
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
	if f.RuleID != "BC001" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC001")
	}
	if f.Severity != types.SeverityError {
		t.Errorf("Severity = %v, want %v", f.Severity, types.SeverityError)
	}
}

func TestBC001_NewOptionalVariable(t *testing.T) {
	rule := &BC001{}

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	new.Variables["new_var"] = &types.VariableSignature{
		Name:     "new_var",
		Required: false, // has default
		Default:  "default_value",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for optional variable, got %d", len(findings))
	}
}

func TestBC001_ExistingVariable(t *testing.T) {
	rule := &BC001{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["existing_var"] = &types.VariableSignature{
		Name:     "existing_var",
		Required: true,
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["existing_var"] = &types.VariableSignature{
		Name:     "existing_var",
		Required: true,
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for existing variable, got %d", len(findings))
	}
}

package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC002_RemovedVariable(t *testing.T) {
	rule := &BC002{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["removed_var"] = &types.VariableSignature{
		Name: "removed_var",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "BC002" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC002")
	}
	if f.Severity != types.SeverityError {
		t.Errorf("Severity = %v, want %v", f.Severity, types.SeverityError)
	}
}

func TestBC002_VariableStillExists(t *testing.T) {
	rule := &BC002{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["existing_var"] = &types.VariableSignature{
		Name: "existing_var",
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["existing_var"] = &types.VariableSignature{
		Name: "existing_var",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when variable exists, got %d", len(findings))
	}
}

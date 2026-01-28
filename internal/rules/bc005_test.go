package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC005_DefaultRemoved(t *testing.T) {
	rule := &BC005{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: false, // has default
		Default:  "value",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: true, // no default
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
	if f.RuleID != "BC005" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC005")
	}
}

func TestBC005_DefaultStillExists(t *testing.T) {
	rule := &BC005{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: false,
		Default:  "value",
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Required: false,
		Default:  "value",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when default exists, got %d", len(findings))
	}
}

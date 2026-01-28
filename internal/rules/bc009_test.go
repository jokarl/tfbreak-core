package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC009_RemovedOutput(t *testing.T) {
	rule := &BC009{}

	old := types.NewModuleSnapshot("/old")
	old.Outputs["my_output"] = &types.OutputSignature{
		Name: "my_output",
		DeclRange: types.FileRange{
			Filename: "outputs.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "BC009" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC009")
	}
	if f.Severity != types.SeverityBreaking {
		t.Errorf("Severity = %v, want %v", f.Severity, types.SeverityBreaking)
	}
}

func TestBC009_OutputStillExists(t *testing.T) {
	rule := &BC009{}

	old := types.NewModuleSnapshot("/old")
	old.Outputs["my_output"] = &types.OutputSignature{
		Name: "my_output",
	}

	new := types.NewModuleSnapshot("/new")
	new.Outputs["my_output"] = &types.OutputSignature{
		Name: "my_output",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when output exists, got %d", len(findings))
	}
}

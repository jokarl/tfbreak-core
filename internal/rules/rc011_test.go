package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestRC011_SensitiveChanged_FalseToTrue(t *testing.T) {
	rule := &RC011{}

	old := types.NewModuleSnapshot("/old")
	old.Outputs["my_output"] = &types.OutputSignature{
		Name:      "my_output",
		Sensitive: false,
		DeclRange: types.FileRange{
			Filename: "outputs.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Outputs["my_output"] = &types.OutputSignature{
		Name:      "my_output",
		Sensitive: true,
		DeclRange: types.FileRange{
			Filename: "outputs.tf",
			Line:     1,
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "RC011" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "RC011")
	}
	if f.Severity != types.SeverityRisky {
		t.Errorf("Severity = %v, want %v", f.Severity, types.SeverityRisky)
	}
}

func TestRC011_SensitiveChanged_TrueToFalse(t *testing.T) {
	rule := &RC011{}

	old := types.NewModuleSnapshot("/old")
	old.Outputs["my_output"] = &types.OutputSignature{
		Name:      "my_output",
		Sensitive: true,
		DeclRange: types.FileRange{
			Filename: "outputs.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Outputs["my_output"] = &types.OutputSignature{
		Name:      "my_output",
		Sensitive: false,
		DeclRange: types.FileRange{
			Filename: "outputs.tf",
			Line:     1,
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for true->false, got %d", len(findings))
	}
}

func TestRC011_SensitiveUnchanged_BothFalse(t *testing.T) {
	rule := &RC011{}

	old := types.NewModuleSnapshot("/old")
	old.Outputs["my_output"] = &types.OutputSignature{
		Name:      "my_output",
		Sensitive: false,
	}

	new := types.NewModuleSnapshot("/new")
	new.Outputs["my_output"] = &types.OutputSignature{
		Name:      "my_output",
		Sensitive: false,
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when sensitive unchanged (both false), got %d", len(findings))
	}
}

func TestRC011_SensitiveUnchanged_BothTrue(t *testing.T) {
	rule := &RC011{}

	old := types.NewModuleSnapshot("/old")
	old.Outputs["my_output"] = &types.OutputSignature{
		Name:      "my_output",
		Sensitive: true,
	}

	new := types.NewModuleSnapshot("/new")
	new.Outputs["my_output"] = &types.OutputSignature{
		Name:      "my_output",
		Sensitive: true,
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when sensitive unchanged (both true), got %d", len(findings))
	}
}

func TestRC011_OutputRemoved_NoFinding(t *testing.T) {
	rule := &RC011{}

	old := types.NewModuleSnapshot("/old")
	old.Outputs["my_output"] = &types.OutputSignature{
		Name:      "my_output",
		Sensitive: true,
	}

	new := types.NewModuleSnapshot("/new")
	// Output removed - should be handled by BC009, not RC011

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when output removed, got %d", len(findings))
	}
}

func TestRC011_MultipleOutputs(t *testing.T) {
	rule := &RC011{}

	old := types.NewModuleSnapshot("/old")
	old.Outputs["out1"] = &types.OutputSignature{Name: "out1", Sensitive: false, DeclRange: types.FileRange{Filename: "o.tf", Line: 1}}
	old.Outputs["out2"] = &types.OutputSignature{Name: "out2", Sensitive: true, DeclRange: types.FileRange{Filename: "o.tf", Line: 5}}
	old.Outputs["out3"] = &types.OutputSignature{Name: "out3", Sensitive: false, DeclRange: types.FileRange{Filename: "o.tf", Line: 9}}

	new := types.NewModuleSnapshot("/new")
	new.Outputs["out1"] = &types.OutputSignature{Name: "out1", Sensitive: true, DeclRange: types.FileRange{Filename: "o.tf", Line: 1}}  // changed
	new.Outputs["out2"] = &types.OutputSignature{Name: "out2", Sensitive: true, DeclRange: types.FileRange{Filename: "o.tf", Line: 5}}  // unchanged
	new.Outputs["out3"] = &types.OutputSignature{Name: "out3", Sensitive: true, DeclRange: types.FileRange{Filename: "o.tf", Line: 9}} // changed

	findings := rule.Evaluate(old, new)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings for multiple sensitive changes, got %d", len(findings))
	}
}

func TestRC011_NewOutput_NoFinding(t *testing.T) {
	rule := &RC011{}

	old := types.NewModuleSnapshot("/old")
	// No outputs

	new := types.NewModuleSnapshot("/new")
	new.Outputs["new_output"] = &types.OutputSignature{
		Name:      "new_output",
		Sensitive: true,
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for new output, got %d", len(findings))
	}
}

func TestRC011_Documentation(t *testing.T) {
	rule := &RC011{}

	doc := rule.Documentation()

	if doc.ID != "RC011" {
		t.Errorf("Documentation ID = %q, want %q", doc.ID, "RC011")
	}
	if doc.Name != "output-sensitive-changed" {
		t.Errorf("Documentation Name = %q, want %q", doc.Name, "output-sensitive-changed")
	}
	if doc.DefaultSeverity != types.SeverityRisky {
		t.Errorf("Documentation Severity = %v, want %v", doc.DefaultSeverity, types.SeverityRisky)
	}
	if doc.ExampleOld == "" {
		t.Error("Documentation ExampleOld should not be empty")
	}
	if doc.ExampleNew == "" {
		t.Error("Documentation ExampleNew should not be empty")
	}
	if doc.Remediation == "" {
		t.Error("Documentation Remediation should not be empty")
	}
}

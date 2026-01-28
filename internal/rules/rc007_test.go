package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func boolPtr(b bool) *bool {
	return &b
}

func TestRC007_NullableChanged_TrueToFalse(t *testing.T) {
	rule := &RC007{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(true),
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(false),
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
	if f.RuleID != "RC007" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "RC007")
	}
	if f.Severity != types.SeverityRisky {
		t.Errorf("Severity = %v, want %v", f.Severity, types.SeverityRisky)
	}
}

func TestRC007_NullableChanged_FalseToTrue(t *testing.T) {
	rule := &RC007{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(false),
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(true),
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for false->true change, got %d", len(findings))
	}
}

func TestRC007_NullableUnchanged_BothTrue(t *testing.T) {
	rule := &RC007{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(true),
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(true),
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when nullable unchanged, got %d", len(findings))
	}
}

func TestRC007_NullableUnchanged_BothFalse(t *testing.T) {
	rule := &RC007{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(false),
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(false),
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when nullable unchanged, got %d", len(findings))
	}
}

func TestRC007_UnsetToExplicitFalse(t *testing.T) {
	rule := &RC007{}

	// Unset nullable defaults to true
	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: nil, // unset = defaults to true
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(false), // explicit false
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when unset->false, got %d", len(findings))
	}
}

func TestRC007_UnsetToExplicitTrue_NoFinding(t *testing.T) {
	rule := &RC007{}

	// Unset nullable defaults to true, so unset -> explicit true is not a change
	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: nil, // unset = defaults to true
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(true), // explicit true = same as default
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when unset->explicit true (same effective value), got %d", len(findings))
	}
}

func TestRC007_ExplicitTrueToUnset_NoFinding(t *testing.T) {
	rule := &RC007{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(true), // explicit true
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: nil, // unset = defaults to true
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when explicit true->unset (same effective value), got %d", len(findings))
	}
}

func TestRC007_BothUnset_NoFinding(t *testing.T) {
	rule := &RC007{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: nil,
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: nil,
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when both unset, got %d", len(findings))
	}
}

func TestRC007_VariableRemoved_NoFinding(t *testing.T) {
	rule := &RC007{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name:     "my_var",
		Nullable: boolPtr(true),
	}

	new := types.NewModuleSnapshot("/new")
	// Variable removed - should be handled by BC002, not RC007

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when variable removed, got %d", len(findings))
	}
}

func TestRC007_MultipleVariables(t *testing.T) {
	rule := &RC007{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["var1"] = &types.VariableSignature{Name: "var1", Nullable: boolPtr(true), DeclRange: types.FileRange{Filename: "v.tf", Line: 1}}
	old.Variables["var2"] = &types.VariableSignature{Name: "var2", Nullable: boolPtr(false), DeclRange: types.FileRange{Filename: "v.tf", Line: 5}}
	old.Variables["var3"] = &types.VariableSignature{Name: "var3", Nullable: nil, DeclRange: types.FileRange{Filename: "v.tf", Line: 9}}

	new := types.NewModuleSnapshot("/new")
	new.Variables["var1"] = &types.VariableSignature{Name: "var1", Nullable: boolPtr(false), DeclRange: types.FileRange{Filename: "v.tf", Line: 1}} // changed
	new.Variables["var2"] = &types.VariableSignature{Name: "var2", Nullable: boolPtr(false), DeclRange: types.FileRange{Filename: "v.tf", Line: 5}} // unchanged
	new.Variables["var3"] = &types.VariableSignature{Name: "var3", Nullable: boolPtr(false), DeclRange: types.FileRange{Filename: "v.tf", Line: 9}} // changed (nil->false)

	findings := rule.Evaluate(old, new)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings for multiple nullable changes, got %d", len(findings))
	}
}

func TestRC007_Documentation(t *testing.T) {
	rule := &RC007{}

	doc := rule.Documentation()

	if doc.ID != "RC007" {
		t.Errorf("Documentation ID = %q, want %q", doc.ID, "RC007")
	}
	if doc.Name != "input-nullable-changed" {
		t.Errorf("Documentation Name = %q, want %q", doc.Name, "input-nullable-changed")
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

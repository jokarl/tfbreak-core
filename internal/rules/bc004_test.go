package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC004_TypeChanged(t *testing.T) {
	rule := &BC004{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "string",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "number",
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
	if f.RuleID != "BC004" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC004")
	}
	if f.Severity != types.SeverityError {
		t.Errorf("Severity = %v, want %v", f.Severity, types.SeverityError)
	}
}

func TestBC004_TypeUnchanged(t *testing.T) {
	rule := &BC004{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "string",
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "string",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when type unchanged, got %d", len(findings))
	}
}

func TestBC004_AnyToSpecific_NonBreaking(t *testing.T) {
	rule := &BC004{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "any",
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "string",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when narrowing from any to specific type, got %d", len(findings))
	}
}

func TestBC004_SpecificToAny_Breaking(t *testing.T) {
	rule := &BC004{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "string",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "any",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when widening from specific to any, got %d", len(findings))
	}

	if findings[0].RuleID != "BC004" {
		t.Errorf("RuleID = %q, want %q", findings[0].RuleID, "BC004")
	}
}

func TestBC004_EmptyTypeAsAny_NonBreaking(t *testing.T) {
	rule := &BC004{}

	// Empty type (unspecified) should be treated as "any"
	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "", // unspecified = any
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "string",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when narrowing from empty (any) to specific, got %d", len(findings))
	}
}

func TestBC004_EmptyToAny_NoChange(t *testing.T) {
	rule := &BC004{}

	// Empty type and "any" should be equivalent
	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "",
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "any",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when both are any (empty vs explicit), got %d", len(findings))
	}
}

func TestBC004_ComplexTypeChange(t *testing.T) {
	rule := &BC004{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "list(string)",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "list(number)",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for complex type change, got %d", len(findings))
	}
}

func TestBC004_ObjectTypeChange(t *testing.T) {
	rule := &BC004{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["config"] = &types.VariableSignature{
		Name: "config",
		Type: "object({name = string})",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["config"] = &types.VariableSignature{
		Name: "config",
		Type: "object({name = string, age = number})",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for object type change, got %d", len(findings))
	}
}

func TestBC004_VariableRemoved_NoFinding(t *testing.T) {
	rule := &BC004{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["my_var"] = &types.VariableSignature{
		Name: "my_var",
		Type: "string",
	}

	new := types.NewModuleSnapshot("/new")
	// Variable removed - should be handled by BC002, not BC004

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when variable removed (handled by BC002), got %d", len(findings))
	}
}

func TestBC004_MultipleVariables(t *testing.T) {
	rule := &BC004{}

	old := types.NewModuleSnapshot("/old")
	old.Variables["var1"] = &types.VariableSignature{Name: "var1", Type: "string", DeclRange: types.FileRange{Filename: "v.tf", Line: 1}}
	old.Variables["var2"] = &types.VariableSignature{Name: "var2", Type: "number", DeclRange: types.FileRange{Filename: "v.tf", Line: 5}}
	old.Variables["var3"] = &types.VariableSignature{Name: "var3", Type: "bool", DeclRange: types.FileRange{Filename: "v.tf", Line: 9}}

	new := types.NewModuleSnapshot("/new")
	new.Variables["var1"] = &types.VariableSignature{Name: "var1", Type: "number", DeclRange: types.FileRange{Filename: "v.tf", Line: 1}} // changed
	new.Variables["var2"] = &types.VariableSignature{Name: "var2", Type: "number", DeclRange: types.FileRange{Filename: "v.tf", Line: 5}} // unchanged
	new.Variables["var3"] = &types.VariableSignature{Name: "var3", Type: "string", DeclRange: types.FileRange{Filename: "v.tf", Line: 9}} // changed

	findings := rule.Evaluate(old, new)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings for multiple type changes, got %d", len(findings))
	}
}

func TestBC004_Documentation(t *testing.T) {
	rule := &BC004{}

	doc := rule.Documentation()

	if doc.ID != "BC004" {
		t.Errorf("Documentation ID = %q, want %q", doc.ID, "BC004")
	}
	if doc.Name != "input-type-changed" {
		t.Errorf("Documentation Name = %q, want %q", doc.Name, "input-type-changed")
	}
	if doc.DefaultSeverity != types.SeverityError {
		t.Errorf("Documentation Severity = %v, want %v", doc.DefaultSeverity, types.SeverityError)
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

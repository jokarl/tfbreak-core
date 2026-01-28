package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestEngineRunsAllRules(t *testing.T) {
	engine := NewDefaultEngine()

	old := types.NewModuleSnapshot("/old")
	old.Variables["removed_var"] = &types.VariableSignature{
		Name: "removed_var",
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}
	old.Outputs["removed_output"] = &types.OutputSignature{
		Name: "removed_output",
		DeclRange: types.FileRange{
			Filename: "outputs.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["new_required"] = &types.VariableSignature{
		Name:     "new_required",
		Required: true,
		DeclRange: types.FileRange{
			Filename: "variables.tf",
			Line:     1,
		},
	}

	findings := engine.Evaluate(old, new)

	// Should have findings from:
	// - BC001: new required variable
	// - BC002: removed variable
	// - BC009: removed output
	ruleIDs := make(map[string]bool)
	for _, f := range findings {
		ruleIDs[f.RuleID] = true
	}

	expectedRules := []string{"BC001", "BC002", "BC009"}
	for _, expected := range expectedRules {
		if !ruleIDs[expected] {
			t.Errorf("expected finding from rule %s", expected)
		}
	}
}

func TestEngineDisableRule(t *testing.T) {
	engine := NewDefaultEngine()
	engine.DisableRule("BC001")

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	new.Variables["new_required"] = &types.VariableSignature{
		Name:     "new_required",
		Required: true,
	}

	findings := engine.Evaluate(old, new)

	for _, f := range findings {
		if f.RuleID == "BC001" {
			t.Error("BC001 should be disabled")
		}
	}
}

func TestEngineCheck(t *testing.T) {
	engine := NewDefaultEngine()

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	new.Variables["new_required"] = &types.VariableSignature{
		Name:     "new_required",
		Required: true,
	}

	result := engine.Check("/old", "/new", old, new, types.SeverityBreaking)

	if result.Result != "FAIL" {
		t.Errorf("Result = %q, want FAIL", result.Result)
	}
	if result.Summary.Breaking != 1 {
		t.Errorf("Summary.Breaking = %d, want 1", result.Summary.Breaking)
	}
}

func TestEngineCheckPass(t *testing.T) {
	engine := NewDefaultEngine()

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")

	result := engine.Check("/old", "/new", old, new, types.SeverityBreaking)

	if result.Result != "PASS" {
		t.Errorf("Result = %q, want PASS", result.Result)
	}
}

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

	result := engine.Check("/old", "/new", old, new, types.SeverityError)

	if result.Result != "FAIL" {
		t.Errorf("Result = %q, want FAIL", result.Result)
	}
	if result.Summary.Error != 1 {
		t.Errorf("Summary.Error = %d, want 1", result.Summary.Error)
	}
}

func TestEngineCheckPass(t *testing.T) {
	engine := NewDefaultEngine()

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")

	result := engine.Check("/old", "/new", old, new, types.SeverityError)

	if result.Result != "PASS" {
		t.Errorf("Result = %q, want PASS", result.Result)
	}
}

func TestEngine_BC003_SuppressesBC001_BC002(t *testing.T) {
	// Enable rename detection
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.70,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	engine := NewDefaultEngine()

	old := types.NewModuleSnapshot("/old")
	old.Variables["api_key"] = &types.VariableSignature{
		Name:     "api_key",
		Default:  nil,
		Required: true,
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["api_key_v2"] = &types.VariableSignature{
		Name:     "api_key_v2",
		Default:  nil,
		Required: true, // BC001 checks Required field
	}

	findings := engine.Evaluate(old, new)

	// Should only have BC003, not BC001 or BC002
	ruleIDs := make(map[string]int)
	for _, f := range findings {
		ruleIDs[f.RuleID]++
	}

	if ruleIDs["BC003"] != 1 {
		t.Errorf("Expected exactly 1 BC003 finding, got %d", ruleIDs["BC003"])
	}
	if ruleIDs["BC001"] != 0 {
		t.Errorf("Expected BC001 to be suppressed, got %d findings", ruleIDs["BC001"])
	}
	if ruleIDs["BC002"] != 0 {
		t.Errorf("Expected BC002 to be suppressed, got %d findings", ruleIDs["BC002"])
	}
}

func TestEngine_RC003_SuppressesBC002(t *testing.T) {
	// Enable rename detection
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.50,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	engine := NewDefaultEngine()

	defaultVal := `"30s"`
	old := types.NewModuleSnapshot("/old")
	old.Variables["timeout"] = &types.VariableSignature{
		Name:    "timeout",
		Default: &defaultVal, // Optional
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["timeout_ms"] = &types.VariableSignature{
		Name:    "timeout_ms",
		Default: &defaultVal, // Optional
	}

	findings := engine.Evaluate(old, new)

	// Should only have RC003, not BC002
	ruleIDs := make(map[string]int)
	for _, f := range findings {
		ruleIDs[f.RuleID]++
	}

	if ruleIDs["RC003"] != 1 {
		t.Errorf("Expected exactly 1 RC003 finding, got %d", ruleIDs["RC003"])
	}
	if ruleIDs["BC002"] != 0 {
		t.Errorf("Expected BC002 to be suppressed, got %d findings", ruleIDs["BC002"])
	}
}

func TestEngine_BC010_SuppressesBC009(t *testing.T) {
	// Enable rename detection
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.50,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	engine := NewDefaultEngine()

	old := types.NewModuleSnapshot("/old")
	old.Outputs["vpc_id"] = &types.OutputSignature{
		Name: "vpc_id",
	}

	new := types.NewModuleSnapshot("/new")
	new.Outputs["main_vpc_id"] = &types.OutputSignature{
		Name: "main_vpc_id",
	}

	findings := engine.Evaluate(old, new)

	// Should only have BC010, not BC009
	ruleIDs := make(map[string]int)
	for _, f := range findings {
		ruleIDs[f.RuleID]++
	}

	if ruleIDs["BC010"] != 1 {
		t.Errorf("Expected exactly 1 BC010 finding, got %d", ruleIDs["BC010"])
	}
	if ruleIDs["BC009"] != 0 {
		t.Errorf("Expected BC009 to be suppressed, got %d findings", ruleIDs["BC009"])
	}
}

func TestEngine_RenameDetectionDisabled_NoSuppression(t *testing.T) {
	// Ensure rename detection is disabled
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             false,
		SimilarityThreshold: 0.70,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	engine := NewDefaultEngine()

	old := types.NewModuleSnapshot("/old")
	old.Variables["api_key"] = &types.VariableSignature{
		Name:     "api_key",
		Default:  nil,
		Required: true,
	}

	new := types.NewModuleSnapshot("/new")
	new.Variables["api_key_v2"] = &types.VariableSignature{
		Name:     "api_key_v2",
		Default:  nil,
		Required: true, // BC001 checks Required field
	}

	findings := engine.Evaluate(old, new)

	// Should have BC001 and BC002, no BC003
	ruleIDs := make(map[string]int)
	for _, f := range findings {
		ruleIDs[f.RuleID]++
	}

	if ruleIDs["BC003"] != 0 {
		t.Errorf("Expected no BC003 findings when disabled, got %d", ruleIDs["BC003"])
	}
	if ruleIDs["BC001"] != 1 {
		t.Errorf("Expected 1 BC001 finding, got %d", ruleIDs["BC001"])
	}
	if ruleIDs["BC002"] != 1 {
		t.Errorf("Expected 1 BC002 finding, got %d", ruleIDs["BC002"])
	}
}

func TestExtractQuotedName(t *testing.T) {
	tests := []struct {
		message  string
		expected string
	}{
		{`Variable "foo" was removed`, "foo"},
		{`New required variable "bar" has no default`, "bar"},
		{`Output "baz" was removed`, "baz"},
		{`No quotes here`, ""},
		{`One "quote`, ""},
		{`Empty "" quotes`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			result := extractQuotedName(tt.message)
			if result != tt.expected {
				t.Errorf("extractQuotedName(%q) = %q, want %q", tt.message, result, tt.expected)
			}
		})
	}
}

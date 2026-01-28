package internal

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jokarl/tfbreak-core/internal/loader"
	"github.com/jokarl/tfbreak-core/internal/rules"
	"github.com/jokarl/tfbreak-core/internal/types"
)

func getTestdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "testdata", "scenarios")
}

func TestScenario_BC001_RequiredInputAdded(t *testing.T) {
	runScenario(t, "bc001_required_input_added", []string{"BC001"})
}

func TestScenario_BC002_InputRemoved(t *testing.T) {
	runScenario(t, "bc002_input_removed", []string{"BC002"})
}

func TestScenario_BC005_DefaultRemoved(t *testing.T) {
	runScenario(t, "bc005_default_removed", []string{"BC005"})
}

func TestScenario_RC006_DefaultChanged(t *testing.T) {
	runScenario(t, "rc006_default_changed", []string{"RC006"})
}

func TestScenario_BC009_OutputRemoved(t *testing.T) {
	runScenario(t, "bc009_output_removed", []string{"BC009"})
}

func TestScenario_BC100_ResourceRemoved(t *testing.T) {
	runScenario(t, "bc100_resource_removed", []string{"BC100"})
}

func TestScenario_BC100_ResourceMoved(t *testing.T) {
	// This should have NO findings because the moved block is present
	runScenario(t, "bc100_resource_moved", []string{})
}

func TestScenario_BC101_ModuleRemoved(t *testing.T) {
	runScenario(t, "bc101_module_removed", []string{"BC101"})
}

func TestScenario_BC103_NonexistentTarget(t *testing.T) {
	// Should find BC103 (non-existent target) - BC100 won't trigger because moved block exists
	runScenario(t, "bc103_nonexistent_target", []string{"BC103"})
}

func TestScenario_NoChanges(t *testing.T) {
	runScenario(t, "no_changes", []string{})
}

func TestScenario_BC004_TypeChanged(t *testing.T) {
	runScenario(t, "bc004_type_changed", []string{"BC004"})
}

func TestScenario_BC004_AnyToSpecific(t *testing.T) {
	// any -> specific type is non-breaking, should have no findings
	runScenario(t, "bc004_any_to_specific", []string{})
}

func TestScenario_RC007_NullableChanged(t *testing.T) {
	runScenario(t, "rc007_nullable_changed", []string{"RC007"})
}

func TestScenario_RC008_SensitiveChanged(t *testing.T) {
	runScenario(t, "rc008_sensitive_changed", []string{"RC008"})
}

func TestScenario_RC011_OutputSensitiveChanged(t *testing.T) {
	runScenario(t, "rc011_output_sensitive_changed", []string{"RC011"})
}

func TestScenario_BC200_VersionAdded(t *testing.T) {
	runScenario(t, "bc200_version_added", []string{"BC200"})
}

func TestScenario_BC201_ProviderRemoved(t *testing.T) {
	runScenario(t, "bc201_provider_removed", []string{"BC201"})
}

func runScenario(t *testing.T, name string, expectedRuleIDs []string) {
	t.Helper()

	baseDir := getTestdataDir()
	oldDir := filepath.Join(baseDir, name, "old")
	newDir := filepath.Join(baseDir, name, "new")

	// Load configs
	oldSnap, err := loader.Load(oldDir)
	if err != nil {
		t.Fatalf("failed to load old config: %v", err)
	}

	newSnap, err := loader.Load(newDir)
	if err != nil {
		t.Fatalf("failed to load new config: %v", err)
	}

	// Run engine
	engine := rules.NewDefaultEngine()
	result := engine.Check(oldDir, newDir, oldSnap, newSnap, types.SeverityBreaking)

	// Build set of found rule IDs
	foundRuleIDs := make(map[string]bool)
	for _, f := range result.Findings {
		foundRuleIDs[f.RuleID] = true
	}

	// Check expected rules were found
	for _, expected := range expectedRuleIDs {
		if !foundRuleIDs[expected] {
			t.Errorf("expected finding for rule %s, but not found", expected)
		}
	}

	// Check no unexpected rules (allow for expected rules only)
	expectedSet := make(map[string]bool)
	for _, id := range expectedRuleIDs {
		expectedSet[id] = true
	}
	for id := range foundRuleIDs {
		if !expectedSet[id] {
			t.Errorf("unexpected finding for rule %s", id)
		}
	}

	// Verify pass/fail based on expected findings
	if len(expectedRuleIDs) == 0 {
		if result.Result != "PASS" {
			t.Errorf("expected PASS result, got %s", result.Result)
		}
	} else {
		// Check if any breaking rules expected
		hasBreaking := false
		for _, id := range expectedRuleIDs {
			if id[0] == 'B' { // Breaking rules start with BC
				hasBreaking = true
				break
			}
		}
		if hasBreaking && result.Result != "FAIL" {
			t.Errorf("expected FAIL result for breaking changes, got %s", result.Result)
		}
	}
}

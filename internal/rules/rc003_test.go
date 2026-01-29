package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestRC003_Disabled(t *testing.T) {
	// Ensure rename detection is disabled
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             false,
		SimilarityThreshold: 0.85,
	})

	rule := &RC003{}

	defaultVal := `"30s"`
	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"timeout": {Name: "timeout", Default: &defaultVal},
		},
	}

	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"timeout_seconds": {Name: "timeout_seconds", Default: &defaultVal},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when rename detection is disabled, got %d", len(findings))
	}
}

func TestRC003_OptionalRenameDetected(t *testing.T) {
	// Enable rename detection
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.50,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &RC003{}

	defaultVal := `"30s"`
	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"timeout": {Name: "timeout", Default: &defaultVal},
		},
	}

	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"timeout_ms": {Name: "timeout_ms", Default: &defaultVal},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "RC003" {
		t.Errorf("Expected rule ID RC003, got %s", f.RuleID)
	}
	if f.Severity != types.SeverityWarning {
		t.Errorf("Expected RISKY severity, got %s", f.Severity)
	}
	if f.Metadata["old_name"] != "timeout" {
		t.Errorf("Expected old_name 'timeout', got %s", f.Metadata["old_name"])
	}
	if f.Metadata["new_name"] != "timeout_ms" {
		t.Errorf("Expected new_name 'timeout_ms', got %s", f.Metadata["new_name"])
	}
}

func TestRC003_RequiredNewVar_NoMatch(t *testing.T) {
	// Enable rename detection
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.50,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &RC003{}

	defaultVal := `"30s"`
	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"timeout": {Name: "timeout", Default: &defaultVal},
		},
	}

	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			// New variable has no default, so it's required - RC003 shouldn't match
			"timeout_ms": {Name: "timeout_ms", Default: nil},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when new variable is required, got %d", len(findings))
	}
}

func TestRC003_BelowThreshold(t *testing.T) {
	// Enable rename detection with high threshold
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.95,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &RC003{}

	defaultVal := `"value"`
	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"foo": {Name: "foo", Default: &defaultVal},
		},
	}

	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			// Similarity between "foo" and "bar_foo" is below 0.95
			"bar_foo": {Name: "bar_foo", Default: &defaultVal},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when similarity below threshold, got %d", len(findings))
	}
}

func TestRC003_MultipleMatches_BestWins(t *testing.T) {
	// Enable rename detection
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.50,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &RC003{}

	defaultVal := `"value"`
	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"database_timeout": {Name: "database_timeout", Default: &defaultVal},
		},
	}

	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"database_timeout_ms": {Name: "database_timeout_ms", Default: &defaultVal}, // Better match
			"db_timeout":          {Name: "db_timeout", Default: &defaultVal},          // Worse match
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}

	// Should match with the more similar name
	if findings[0].Metadata["new_name"] != "database_timeout_ms" {
		t.Errorf("Expected best match 'database_timeout_ms', got %s", findings[0].Metadata["new_name"])
	}
}

func TestRC003_RuleMetadata(t *testing.T) {
	rule := &RC003{}

	if rule.ID() != "RC003" {
		t.Errorf("Expected ID 'RC003', got %s", rule.ID())
	}
	if rule.Name() != "input-renamed-optional" {
		t.Errorf("Expected name 'input-renamed-optional', got %s", rule.Name())
	}
	if rule.DefaultSeverity() != types.SeverityWarning {
		t.Errorf("Expected severity RISKY, got %s", rule.DefaultSeverity())
	}
	if rule.Documentation() == nil {
		t.Error("Expected non-nil documentation")
	}
}

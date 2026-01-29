package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC003_Disabled(t *testing.T) {
	// Ensure rename detection is disabled
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             false,
		SimilarityThreshold: 0.85,
	})

	rule := &BC003{}

	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"api_key": {Name: "api_key", Default: nil},
		},
	}

	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"api_key_v2": {Name: "api_key_v2", Default: nil},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when rename detection is disabled, got %d", len(findings))
	}
}

func TestBC003_RequiredRenameDetected(t *testing.T) {
	// Enable rename detection
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.70,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &BC003{}

	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"api_key": {Name: "api_key", Default: nil},
		},
	}

	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"api_key_v2": {Name: "api_key_v2", Default: nil},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "BC003" {
		t.Errorf("Expected rule ID BC003, got %s", f.RuleID)
	}
	if f.Severity != types.SeverityError {
		t.Errorf("Expected BREAKING severity, got %s", f.Severity)
	}
	if f.Metadata["old_name"] != "api_key" {
		t.Errorf("Expected old_name 'api_key', got %s", f.Metadata["old_name"])
	}
	if f.Metadata["new_name"] != "api_key_v2" {
		t.Errorf("Expected new_name 'api_key_v2', got %s", f.Metadata["new_name"])
	}
}

func TestBC003_OptionalNewVar_NoMatch(t *testing.T) {
	// Enable rename detection
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.70,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &BC003{}

	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"api_key": {Name: "api_key", Default: nil},
		},
	}

	defaultVal := `"default_value"`
	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			// New variable has a default, so it's optional - BC003 shouldn't match
			"api_key_v2": {Name: "api_key_v2", Default: &defaultVal},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when new variable has default, got %d", len(findings))
	}
}

func TestBC003_BelowThreshold(t *testing.T) {
	// Enable rename detection with high threshold
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.95,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &BC003{}

	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"foo": {Name: "foo", Default: nil},
		},
	}

	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			// Similarity between "foo" and "bar_foo" is below 0.95
			"bar_foo": {Name: "bar_foo", Default: nil},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when similarity below threshold, got %d", len(findings))
	}
}

func TestBC003_MultipleMatches_BestWins(t *testing.T) {
	// Enable rename detection
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.50,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &BC003{}

	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"database_host": {Name: "database_host", Default: nil},
		},
	}

	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"database_hostname": {Name: "database_hostname", Default: nil}, // Better match
			"db_host":           {Name: "db_host", Default: nil},           // Worse match
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}

	// Should match with the more similar name
	if findings[0].Metadata["new_name"] != "database_hostname" {
		t.Errorf("Expected best match 'database_hostname', got %s", findings[0].Metadata["new_name"])
	}
}

func TestBC003_NoRemovedVariables(t *testing.T) {
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.70,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &BC003{}

	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"foo": {Name: "foo", Default: nil},
		},
	}

	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"foo": {Name: "foo", Default: nil}, // Same variable, not removed
			"bar": {Name: "bar", Default: nil}, // New variable
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when no variables removed, got %d", len(findings))
	}
}

func TestBC003_NoAddedRequiredVariables(t *testing.T) {
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.70,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &BC003{}

	old := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			"foo": {Name: "foo", Default: nil},
		},
	}

	new := &types.ModuleSnapshot{
		Variables: map[string]*types.VariableSignature{
			// No new required variables
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when no new required variables, got %d", len(findings))
	}
}

func TestBC003_RuleMetadata(t *testing.T) {
	rule := &BC003{}

	if rule.ID() != "BC003" {
		t.Errorf("Expected ID 'BC003', got %s", rule.ID())
	}
	if rule.Name() != "input-renamed" {
		t.Errorf("Expected name 'input-renamed', got %s", rule.Name())
	}
	if rule.DefaultSeverity() != types.SeverityError {
		t.Errorf("Expected severity BREAKING, got %s", rule.DefaultSeverity())
	}
	if rule.Documentation() == nil {
		t.Error("Expected non-nil documentation")
	}
}

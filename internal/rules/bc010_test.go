package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC010_Disabled(t *testing.T) {
	// Ensure rename detection is disabled
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             false,
		SimilarityThreshold: 0.85,
	})

	rule := &BC010{}

	old := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			"vpc_id": {Name: "vpc_id"},
		},
	}

	new := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			"main_vpc_id": {Name: "main_vpc_id"},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when rename detection is disabled, got %d", len(findings))
	}
}

func TestBC010_OutputRenameDetected(t *testing.T) {
	// Enable rename detection
	// Similarity of "vpc_id" to "main_vpc_id" is ~0.55, so use threshold of 0.50
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.50,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &BC010{}

	old := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			"vpc_id": {Name: "vpc_id"},
		},
	}

	new := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			"main_vpc_id": {Name: "main_vpc_id"},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "BC010" {
		t.Errorf("Expected rule ID BC010, got %s", f.RuleID)
	}
	if f.Severity != types.SeverityBreaking {
		t.Errorf("Expected BREAKING severity, got %s", f.Severity)
	}
	if f.Metadata["old_name"] != "vpc_id" {
		t.Errorf("Expected old_name 'vpc_id', got %s", f.Metadata["old_name"])
	}
	if f.Metadata["new_name"] != "main_vpc_id" {
		t.Errorf("Expected new_name 'main_vpc_id', got %s", f.Metadata["new_name"])
	}
}

func TestBC010_BelowThreshold(t *testing.T) {
	// Enable rename detection with high threshold
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.95,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &BC010{}

	old := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			"foo": {Name: "foo"},
		},
	}

	new := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			// Similarity between "foo" and "bar_baz" is well below 0.95
			"bar_baz": {Name: "bar_baz"},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when similarity below threshold, got %d", len(findings))
	}
}

func TestBC010_MultipleMatches_BestWins(t *testing.T) {
	// Enable rename detection
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.50,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &BC010{}

	old := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			"cluster_endpoint": {Name: "cluster_endpoint"},
		},
	}

	new := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			"cluster_endpoint_url": {Name: "cluster_endpoint_url"}, // Better match
			"endpoint":             {Name: "endpoint"},             // Worse match
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(findings))
	}

	// Should match with the more similar name
	if findings[0].Metadata["new_name"] != "cluster_endpoint_url" {
		t.Errorf("Expected best match 'cluster_endpoint_url', got %s", findings[0].Metadata["new_name"])
	}
}

func TestBC010_NoRemovedOutputs(t *testing.T) {
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.70,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &BC010{}

	old := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			"foo": {Name: "foo"},
		},
	}

	new := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			"foo": {Name: "foo"}, // Same output, not removed
			"bar": {Name: "bar"}, // New output
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when no outputs removed, got %d", len(findings))
	}
}

func TestBC010_NoAddedOutputs(t *testing.T) {
	SetRenameDetectionSettings(&RenameDetectionSettings{
		Enabled:             true,
		SimilarityThreshold: 0.70,
	})
	defer SetRenameDetectionSettings(DefaultRenameDetectionSettings())

	rule := &BC010{}

	old := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			"foo": {Name: "foo"},
		},
	}

	new := &types.ModuleSnapshot{
		Outputs: map[string]*types.OutputSignature{
			// No outputs
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("Expected no findings when no new outputs, got %d", len(findings))
	}
}

func TestBC010_RuleMetadata(t *testing.T) {
	rule := &BC010{}

	if rule.ID() != "BC010" {
		t.Errorf("Expected ID 'BC010', got %s", rule.ID())
	}
	if rule.Name() != "output-renamed" {
		t.Errorf("Expected name 'output-renamed', got %s", rule.Name())
	}
	if rule.DefaultSeverity() != types.SeverityBreaking {
		t.Errorf("Expected severity BREAKING, got %s", rule.DefaultSeverity())
	}
	if rule.Documentation() == nil {
		t.Error("Expected non-nil documentation")
	}
}

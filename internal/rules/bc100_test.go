package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC100_ResourceRemovedNoMoved(t *testing.T) {
	rule := &BC100{}

	old := types.NewModuleSnapshot("/old")
	old.Resources["aws_s3_bucket.main"] = &types.ResourceSignature{
		Type:    "aws_s3_bucket",
		Name:    "main",
		Address: "aws_s3_bucket.main",
		DeclRange: types.FileRange{
			Filename: "main.tf",
			Line:     1,
		},
	}

	new := types.NewModuleSnapshot("/new")

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "BC100" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC100")
	}
}

func TestBC100_ResourceMovedCorrectly(t *testing.T) {
	rule := &BC100{}

	old := types.NewModuleSnapshot("/old")
	old.Resources["aws_s3_bucket.old_name"] = &types.ResourceSignature{
		Type:    "aws_s3_bucket",
		Name:    "old_name",
		Address: "aws_s3_bucket.old_name",
	}

	new := types.NewModuleSnapshot("/new")
	new.Resources["aws_s3_bucket.new_name"] = &types.ResourceSignature{
		Type:    "aws_s3_bucket",
		Name:    "new_name",
		Address: "aws_s3_bucket.new_name",
	}
	new.MovedBlocks = []*types.MovedBlock{
		{
			From: "aws_s3_bucket.old_name",
			To:   "aws_s3_bucket.new_name",
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings with moved block, got %d", len(findings))
	}
}

func TestBC100_ResourceStillExists(t *testing.T) {
	rule := &BC100{}

	old := types.NewModuleSnapshot("/old")
	old.Resources["aws_s3_bucket.main"] = &types.ResourceSignature{
		Type:    "aws_s3_bucket",
		Name:    "main",
		Address: "aws_s3_bucket.main",
	}

	new := types.NewModuleSnapshot("/new")
	new.Resources["aws_s3_bucket.main"] = &types.ResourceSignature{
		Type:    "aws_s3_bucket",
		Name:    "main",
		Address: "aws_s3_bucket.main",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when resource exists, got %d", len(findings))
	}
}

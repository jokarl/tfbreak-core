package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC103_DuplicateFrom(t *testing.T) {
	rule := &BC103{}

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	new.Resources["aws_s3_bucket.target1"] = &types.ResourceSignature{
		Address: "aws_s3_bucket.target1",
	}
	new.Resources["aws_s3_bucket.target2"] = &types.ResourceSignature{
		Address: "aws_s3_bucket.target2",
	}
	new.MovedBlocks = []*types.MovedBlock{
		{
			From: "aws_s3_bucket.source",
			To:   "aws_s3_bucket.target1",
			DeclRange: types.FileRange{
				Filename: "moved.tf",
				Line:     1,
			},
		},
		{
			From: "aws_s3_bucket.source", // duplicate
			To:   "aws_s3_bucket.target2",
			DeclRange: types.FileRange{
				Filename: "moved.tf",
				Line:     5,
			},
		},
	}

	findings := rule.Evaluate(old, new)

	// Should find duplicate from address
	found := false
	for _, f := range findings {
		if f.RuleID == "BC103" && f.Message == "Duplicate moved block 'from' address: \"aws_s3_bucket.source\"" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected finding for duplicate 'from' address")
	}
}

func TestBC103_CyclicMoved(t *testing.T) {
	rule := &BC103{}

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	// Create resources so the "to" targets exist (avoiding BC103 non-existent target)
	new.Resources["aws_s3_bucket.a"] = &types.ResourceSignature{Address: "aws_s3_bucket.a"}
	new.Resources["aws_s3_bucket.b"] = &types.ResourceSignature{Address: "aws_s3_bucket.b"}
	new.MovedBlocks = []*types.MovedBlock{
		{
			From: "aws_s3_bucket.a",
			To:   "aws_s3_bucket.b",
			DeclRange: types.FileRange{
				Filename: "moved.tf",
				Line:     1,
			},
		},
		{
			From: "aws_s3_bucket.b",
			To:   "aws_s3_bucket.a",
			DeclRange: types.FileRange{
				Filename: "moved.tf",
				Line:     5,
			},
		},
	}

	findings := rule.Evaluate(old, new)

	// Should find a cycle
	found := false
	for _, f := range findings {
		if f.RuleID == "BC103" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected finding for cyclic moved blocks")
	}
}

func TestBC103_NonExistentTarget(t *testing.T) {
	rule := &BC103{}

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	// Note: aws_s3_bucket.new_name does NOT exist in new
	new.MovedBlocks = []*types.MovedBlock{
		{
			From: "aws_s3_bucket.old_name",
			To:   "aws_s3_bucket.new_name",
			DeclRange: types.FileRange{
				Filename: "moved.tf",
				Line:     1,
			},
		},
	}

	findings := rule.Evaluate(old, new)

	found := false
	for _, f := range findings {
		if f.RuleID == "BC103" && f.Message == "Moved block 'to' target does not exist: \"aws_s3_bucket.new_name\"" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected finding for non-existent target, got findings: %v", findings)
	}
}

func TestBC103_ValidMoves(t *testing.T) {
	rule := &BC103{}

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	new.Resources["aws_s3_bucket.new1"] = &types.ResourceSignature{Address: "aws_s3_bucket.new1"}
	new.Resources["aws_s3_bucket.new2"] = &types.ResourceSignature{Address: "aws_s3_bucket.new2"}
	new.MovedBlocks = []*types.MovedBlock{
		{
			From: "aws_s3_bucket.old1",
			To:   "aws_s3_bucket.new1",
		},
		{
			From: "aws_s3_bucket.old2",
			To:   "aws_s3_bucket.new2",
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for valid moves, got %d: %v", len(findings), findings)
	}
}

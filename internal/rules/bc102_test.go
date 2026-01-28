package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC102_TypeMismatchResourceToModule(t *testing.T) {
	rule := &BC102{}

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	new.MovedBlocks = []*types.MovedBlock{
		{
			From: "aws_s3_bucket.main",
			To:   "module.bucket",
			DeclRange: types.FileRange{
				Filename: "moved.tf",
				Line:     1,
			},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "BC102" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC102")
	}
}

func TestBC102_TypeMismatchModuleToResource(t *testing.T) {
	rule := &BC102{}

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	new.MovedBlocks = []*types.MovedBlock{
		{
			From: "module.bucket",
			To:   "aws_s3_bucket.main",
			DeclRange: types.FileRange{
				Filename: "moved.tf",
				Line:     1,
			},
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestBC102_ValidResourceMove(t *testing.T) {
	rule := &BC102{}

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	new.MovedBlocks = []*types.MovedBlock{
		{
			From: "aws_s3_bucket.old",
			To:   "aws_s3_bucket.new",
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for valid resource move, got %d", len(findings))
	}
}

func TestBC102_ValidModuleMove(t *testing.T) {
	rule := &BC102{}

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")
	new.MovedBlocks = []*types.MovedBlock{
		{
			From: "module.old",
			To:   "module.new",
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for valid module move, got %d", len(findings))
	}
}

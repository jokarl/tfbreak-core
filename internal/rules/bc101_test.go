package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC101_ModuleRemovedNoMoved(t *testing.T) {
	rule := &BC101{}

	old := types.NewModuleSnapshot("/old")
	old.Modules["vpc"] = &types.ModuleCallSignature{
		Name:    "vpc",
		Source:  "./modules/vpc",
		Address: "module.vpc",
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
	if f.RuleID != "BC101" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC101")
	}
}

func TestBC101_ModuleMovedCorrectly(t *testing.T) {
	rule := &BC101{}

	old := types.NewModuleSnapshot("/old")
	old.Modules["old_vpc"] = &types.ModuleCallSignature{
		Name:    "old_vpc",
		Source:  "./modules/vpc",
		Address: "module.old_vpc",
	}

	new := types.NewModuleSnapshot("/new")
	new.Modules["new_vpc"] = &types.ModuleCallSignature{
		Name:    "new_vpc",
		Source:  "./modules/vpc",
		Address: "module.new_vpc",
	}
	new.MovedBlocks = []*types.MovedBlock{
		{
			From: "module.old_vpc",
			To:   "module.new_vpc",
		},
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings with moved block, got %d", len(findings))
	}
}

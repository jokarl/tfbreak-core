package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC200_VersionAdded(t *testing.T) {
	rule := &BC200{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredVersion = ""

	new := types.NewModuleSnapshot("/new")
	new.RequiredVersion = ">= 1.5.0"

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when version added, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "BC200" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC200")
	}
	if f.Severity != types.SeverityBreaking {
		t.Errorf("Severity = %v, want %v", f.Severity, types.SeverityBreaking)
	}
}

func TestBC200_VersionChanged(t *testing.T) {
	rule := &BC200{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredVersion = ">= 1.0.0"

	new := types.NewModuleSnapshot("/new")
	new.RequiredVersion = ">= 1.5.0"

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when version changed, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "BC200" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC200")
	}
}

func TestBC200_VersionRemoved_NoFinding(t *testing.T) {
	rule := &BC200{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredVersion = ">= 1.0.0"

	new := types.NewModuleSnapshot("/new")
	new.RequiredVersion = ""

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when version removed (loosening), got %d", len(findings))
	}
}

func TestBC200_VersionUnchanged(t *testing.T) {
	rule := &BC200{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredVersion = ">= 1.0.0"

	new := types.NewModuleSnapshot("/new")
	new.RequiredVersion = ">= 1.0.0"

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when version unchanged, got %d", len(findings))
	}
}

func TestBC200_BothEmpty_NoFinding(t *testing.T) {
	rule := &BC200{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredVersion = ""

	new := types.NewModuleSnapshot("/new")
	new.RequiredVersion = ""

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when both empty, got %d", len(findings))
	}
}

func TestBC200_ComplexConstraint(t *testing.T) {
	rule := &BC200{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredVersion = ">= 1.0.0, < 2.0.0"

	new := types.NewModuleSnapshot("/new")
	new.RequiredVersion = ">= 1.5.0, < 2.0.0"

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when complex constraint changed, got %d", len(findings))
	}
}

func TestBC200_PessimisticConstraint(t *testing.T) {
	rule := &BC200{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredVersion = "~> 1.0"

	new := types.NewModuleSnapshot("/new")
	new.RequiredVersion = "~> 1.5"

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when pessimistic constraint changed, got %d", len(findings))
	}
}

func TestBC200_Documentation(t *testing.T) {
	rule := &BC200{}

	doc := rule.Documentation()

	if doc.ID != "BC200" {
		t.Errorf("Documentation ID = %q, want %q", doc.ID, "BC200")
	}
	if doc.Name != "terraform-version-constrained" {
		t.Errorf("Documentation Name = %q, want %q", doc.Name, "terraform-version-constrained")
	}
	if doc.DefaultSeverity != types.SeverityBreaking {
		t.Errorf("Documentation Severity = %v, want %v", doc.DefaultSeverity, types.SeverityBreaking)
	}
	if doc.ExampleOld == "" {
		t.Error("Documentation ExampleOld should not be empty")
	}
	if doc.ExampleNew == "" {
		t.Error("Documentation ExampleNew should not be empty")
	}
	if doc.Remediation == "" {
		t.Error("Documentation Remediation should not be empty")
	}
}

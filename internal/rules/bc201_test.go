package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestBC201_ProviderRemoved(t *testing.T) {
	rule := &BC201{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredProviders["aws"] = &types.ProviderRequirement{
		Source:  "hashicorp/aws",
		Version: ">= 4.0",
	}

	new := types.NewModuleSnapshot("/new")
	// aws provider removed

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when provider removed, got %d", len(findings))
	}

	f := findings[0]
	if f.RuleID != "BC201" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC201")
	}
	if f.Severity != types.SeverityBreaking {
		t.Errorf("Severity = %v, want %v", f.Severity, types.SeverityBreaking)
	}
}

func TestBC201_ProviderVersionChanged(t *testing.T) {
	rule := &BC201{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredProviders["aws"] = &types.ProviderRequirement{
		Source:  "hashicorp/aws",
		Version: ">= 4.0",
	}

	new := types.NewModuleSnapshot("/new")
	new.RequiredProviders["aws"] = &types.ProviderRequirement{
		Source:  "hashicorp/aws",
		Version: ">= 5.0",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when version changed, got %d", len(findings))
	}
}

func TestBC201_ProviderSourceChanged(t *testing.T) {
	rule := &BC201{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredProviders["custom"] = &types.ProviderRequirement{
		Source:  "hashicorp/custom",
		Version: ">= 1.0",
	}

	new := types.NewModuleSnapshot("/new")
	new.RequiredProviders["custom"] = &types.ProviderRequirement{
		Source:  "other/custom",
		Version: ">= 1.0",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when source changed, got %d", len(findings))
	}
}

func TestBC201_ProviderAdded_NoFinding(t *testing.T) {
	rule := &BC201{}

	old := types.NewModuleSnapshot("/old")
	// No providers

	new := types.NewModuleSnapshot("/new")
	new.RequiredProviders["aws"] = &types.ProviderRequirement{
		Source:  "hashicorp/aws",
		Version: ">= 4.0",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when provider added, got %d", len(findings))
	}
}

func TestBC201_ProviderUnchanged(t *testing.T) {
	rule := &BC201{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredProviders["aws"] = &types.ProviderRequirement{
		Source:  "hashicorp/aws",
		Version: ">= 4.0",
	}

	new := types.NewModuleSnapshot("/new")
	new.RequiredProviders["aws"] = &types.ProviderRequirement{
		Source:  "hashicorp/aws",
		Version: ">= 4.0",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when provider unchanged, got %d", len(findings))
	}
}

func TestBC201_VersionConstraintAdded(t *testing.T) {
	rule := &BC201{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredProviders["aws"] = &types.ProviderRequirement{
		Source:  "hashicorp/aws",
		Version: "", // No version constraint
	}

	new := types.NewModuleSnapshot("/new")
	new.RequiredProviders["aws"] = &types.ProviderRequirement{
		Source:  "hashicorp/aws",
		Version: ">= 5.0",
	}

	findings := rule.Evaluate(old, new)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when version constraint added, got %d", len(findings))
	}
}

func TestBC201_VersionConstraintRemoved(t *testing.T) {
	rule := &BC201{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredProviders["aws"] = &types.ProviderRequirement{
		Source:  "hashicorp/aws",
		Version: ">= 4.0",
	}

	new := types.NewModuleSnapshot("/new")
	new.RequiredProviders["aws"] = &types.ProviderRequirement{
		Source:  "hashicorp/aws",
		Version: "", // Version constraint removed
	}

	findings := rule.Evaluate(old, new)

	// Version constraint removal is still flagged as a significant change
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when version constraint removed, got %d", len(findings))
	}
}

func TestBC201_MultipleProviders(t *testing.T) {
	rule := &BC201{}

	old := types.NewModuleSnapshot("/old")
	old.RequiredProviders["aws"] = &types.ProviderRequirement{Source: "hashicorp/aws", Version: ">= 4.0"}
	old.RequiredProviders["google"] = &types.ProviderRequirement{Source: "hashicorp/google", Version: ">= 4.0"}
	old.RequiredProviders["azurerm"] = &types.ProviderRequirement{Source: "hashicorp/azurerm", Version: ">= 3.0"}

	new := types.NewModuleSnapshot("/new")
	new.RequiredProviders["aws"] = &types.ProviderRequirement{Source: "hashicorp/aws", Version: ">= 5.0"} // changed
	new.RequiredProviders["google"] = &types.ProviderRequirement{Source: "hashicorp/google", Version: ">= 4.0"} // unchanged
	// azurerm removed

	findings := rule.Evaluate(old, new)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings (aws changed, azurerm removed), got %d", len(findings))
	}
}

func TestBC201_BothEmpty_NoFinding(t *testing.T) {
	rule := &BC201{}

	old := types.NewModuleSnapshot("/old")
	new := types.NewModuleSnapshot("/new")

	findings := rule.Evaluate(old, new)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings when both have no providers, got %d", len(findings))
	}
}

func TestBC201_Documentation(t *testing.T) {
	rule := &BC201{}

	doc := rule.Documentation()

	if doc.ID != "BC201" {
		t.Errorf("Documentation ID = %q, want %q", doc.ID, "BC201")
	}
	if doc.Name != "provider-version-constrained" {
		t.Errorf("Documentation Name = %q, want %q", doc.Name, "provider-version-constrained")
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

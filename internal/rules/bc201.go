package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC201 detects when provider requirements are removed or changed
type BC201 struct{}

func init() {
	Register(&BC201{})
}

func (r *BC201) ID() string {
	return "BC201"
}

func (r *BC201) Name() string {
	return "provider-version-constrained"
}

func (r *BC201) Description() string {
	return "Provider requirement was removed or changed, which may break consumers using different provider versions"
}

func (r *BC201) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC201) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.0"
    }
  }
}`,
		ExampleNew: `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"  # Tightened constraint
    }
  }
}`,
		Remediation: `This is a BREAKING change because consumers using older provider versions will fail.

Common scenarios:
- Provider removed: consumers depending on that provider will fail
- Version constraint tightened: consumers with older versions must upgrade
- Source changed: effectively a different provider

Before making this change:
1. Verify all consumers can use the new provider version
2. Test with the minimum required version
3. Document the provider version change in your changelog

Use an annotation if this is intentional:
   # tfbreak:ignore BC201 reason="provider upgrade for new features"`,
	}
}

func (r *BC201) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	// Check for providers that were removed or changed
	for name, oldProvider := range old.RequiredProviders {
		newProvider, exists := new.RequiredProviders[name]

		if !exists {
			// Provider was removed
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Provider %q was removed from required_providers", name),
			)
			findings = append(findings, finding)
			continue
		}

		// Check if source changed
		if oldProvider.Source != newProvider.Source {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Provider %q source changed: %q -> %q", name, oldProvider.Source, newProvider.Source),
			)
			findings = append(findings, finding)
			continue
		}

		// Check if version constraint changed
		if oldProvider.Version != newProvider.Version {
			var message string
			if oldProvider.Version == "" {
				message = fmt.Sprintf("Provider %q version constraint added: %q", name, newProvider.Version)
			} else if newProvider.Version == "" {
				// Version constraint removed - still flag as it's a significant change
				message = fmt.Sprintf("Provider %q version constraint removed (was %q)", name, oldProvider.Version)
			} else {
				message = fmt.Sprintf("Provider %q version constraint changed: %q -> %q", name, oldProvider.Version, newProvider.Version)
			}

			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				message,
			)
			findings = append(findings, finding)
		}
	}

	// Note: New providers added are not flagged (adding a dependency is not breaking)

	return findings
}

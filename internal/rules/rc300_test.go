package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestRC300_Metadata(t *testing.T) {
	r := &RC300{}

	if r.ID() != "RC300" {
		t.Errorf("expected ID 'RC300', got %q", r.ID())
	}
	if r.Name() != "module-source-changed" {
		t.Errorf("expected Name 'module-source-changed', got %q", r.Name())
	}
	if r.DefaultSeverity() != types.SeverityRisky {
		t.Errorf("expected severity RISKY, got %v", r.DefaultSeverity())
	}
	if r.Documentation() == nil {
		t.Error("expected Documentation to be non-nil")
	}
}

func TestRC300_Evaluate(t *testing.T) {
	tests := []struct {
		name          string
		oldModules    map[string]*types.ModuleCallSignature
		newModules    map[string]*types.ModuleCallSignature
		wantFindings  int
		wantMessage   string
	}{
		{
			name: "source changed",
			oldModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "git::https://github.com/org/vpc.git",
					Version: "~> 3.0",
					Address: "module.vpc",
				},
			},
			newModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "registry.terraform.io/org/vpc/aws",
					Version: "~> 3.0",
					Address: "module.vpc",
				},
			},
			wantFindings: 1,
			wantMessage:  `Module "vpc" source changed: "git::https://github.com/org/vpc.git" -> "registry.terraform.io/org/vpc/aws"`,
		},
		{
			name: "source unchanged",
			oldModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "./modules/vpc",
					Version: "~> 3.0",
					Address: "module.vpc",
				},
			},
			newModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "./modules/vpc",
					Version: "~> 4.0", // version changed but not source
					Address: "module.vpc",
				},
			},
			wantFindings: 0,
		},
		{
			name: "module added - no finding",
			oldModules: map[string]*types.ModuleCallSignature{},
			newModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "./modules/vpc",
					Address: "module.vpc",
				},
			},
			wantFindings: 0,
		},
		{
			name: "module removed - no finding (handled by BC101)",
			oldModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "./modules/vpc",
					Address: "module.vpc",
				},
			},
			newModules:   map[string]*types.ModuleCallSignature{},
			wantFindings: 0,
		},
		{
			name: "multiple modules - only changed ones reported",
			oldModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "./modules/vpc",
					Address: "module.vpc",
				},
				"rds": {
					Name:    "rds",
					Source:  "git::https://github.com/org/rds.git",
					Address: "module.rds",
				},
				"s3": {
					Name:    "s3",
					Source:  "./modules/s3",
					Address: "module.s3",
				},
			},
			newModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "./modules/networking/vpc", // changed
					Address: "module.vpc",
				},
				"rds": {
					Name:    "rds",
					Source:  "registry.terraform.io/org/rds/aws", // changed
					Address: "module.rds",
				},
				"s3": {
					Name:    "s3",
					Source:  "./modules/s3", // unchanged
					Address: "module.s3",
				},
			},
			wantFindings: 2,
		},
		{
			name: "empty source to non-empty",
			oldModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "",
					Address: "module.vpc",
				},
			},
			newModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "./modules/vpc",
					Address: "module.vpc",
				},
			},
			wantFindings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := &types.ModuleSnapshot{
				Modules: tt.oldModules,
			}
			new := &types.ModuleSnapshot{
				Modules: tt.newModules,
			}

			r := &RC300{}
			findings := r.Evaluate(old, new)

			if len(findings) != tt.wantFindings {
				t.Errorf("expected %d findings, got %d", tt.wantFindings, len(findings))
				for _, f := range findings {
					t.Logf("  finding: %s", f.Message)
				}
			}

			if tt.wantMessage != "" && len(findings) > 0 {
				if findings[0].Message != tt.wantMessage {
					t.Errorf("expected message %q, got %q", tt.wantMessage, findings[0].Message)
				}
			}

			// Verify all findings have correct metadata
			for _, f := range findings {
				if f.RuleID != "RC300" {
					t.Errorf("expected RuleID 'RC300', got %q", f.RuleID)
				}
				if f.RuleName != "module-source-changed" {
					t.Errorf("expected RuleName 'module-source-changed', got %q", f.RuleName)
				}
				if f.Severity != types.SeverityRisky {
					t.Errorf("expected severity RISKY, got %v", f.Severity)
				}
			}
		})
	}
}

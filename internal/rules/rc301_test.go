package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestRC301_Metadata(t *testing.T) {
	r := &RC301{}

	if r.ID() != "RC301" {
		t.Errorf("expected ID 'RC301', got %q", r.ID())
	}
	if r.Name() != "module-version-changed" {
		t.Errorf("expected Name 'module-version-changed', got %q", r.Name())
	}
	if r.DefaultSeverity() != types.SeverityRisky {
		t.Errorf("expected severity RISKY, got %v", r.DefaultSeverity())
	}
	if r.Documentation() == nil {
		t.Error("expected Documentation to be non-nil")
	}
}

func TestRC301_Evaluate(t *testing.T) {
	tests := []struct {
		name         string
		oldModules   map[string]*types.ModuleCallSignature
		newModules   map[string]*types.ModuleCallSignature
		wantFindings int
		wantMessage  string
	}{
		{
			name: "version constraint changed",
			oldModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "registry.terraform.io/org/vpc/aws",
					Version: "~> 3.0",
					Address: "module.vpc",
				},
			},
			newModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "registry.terraform.io/org/vpc/aws",
					Version: "~> 4.0",
					Address: "module.vpc",
				},
			},
			wantFindings: 1,
			wantMessage:  `Module "vpc" version constraint changed: "~> 3.0" -> "~> 4.0"`,
		},
		{
			name: "version constraint added",
			oldModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "registry.terraform.io/org/vpc/aws",
					Version: "",
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
			wantMessage:  `Module "vpc" version constraint added: "~> 3.0"`,
		},
		{
			name: "version constraint removed",
			oldModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "registry.terraform.io/org/vpc/aws",
					Version: "~> 3.0",
					Address: "module.vpc",
				},
			},
			newModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "registry.terraform.io/org/vpc/aws",
					Version: "",
					Address: "module.vpc",
				},
			},
			wantFindings: 1,
			wantMessage:  `Module "vpc" version constraint removed (was "~> 3.0")`,
		},
		{
			name: "version unchanged",
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
					Source:  "./modules/networking/vpc", // source changed but not version
					Version: "~> 3.0",
					Address: "module.vpc",
				},
			},
			wantFindings: 0,
		},
		{
			name: "both empty - no finding",
			oldModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "./modules/vpc",
					Version: "",
					Address: "module.vpc",
				},
			},
			newModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "./modules/vpc",
					Version: "",
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
					Version: "~> 3.0",
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
					Version: "~> 3.0",
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
					Version: "~> 3.0",
					Address: "module.vpc",
				},
				"rds": {
					Name:    "rds",
					Source:  "./modules/rds",
					Version: "~> 2.0",
					Address: "module.rds",
				},
				"s3": {
					Name:    "s3",
					Source:  "./modules/s3",
					Version: "~> 1.0",
					Address: "module.s3",
				},
			},
			newModules: map[string]*types.ModuleCallSignature{
				"vpc": {
					Name:    "vpc",
					Source:  "./modules/vpc",
					Version: "~> 4.0", // changed
					Address: "module.vpc",
				},
				"rds": {
					Name:    "rds",
					Source:  "./modules/rds",
					Version: "", // removed
					Address: "module.rds",
				},
				"s3": {
					Name:    "s3",
					Source:  "./modules/s3",
					Version: "~> 1.0", // unchanged
					Address: "module.s3",
				},
			},
			wantFindings: 2,
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

			r := &RC301{}
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
				if f.RuleID != "RC301" {
					t.Errorf("expected RuleID 'RC301', got %q", f.RuleID)
				}
				if f.RuleName != "module-version-changed" {
					t.Errorf("expected RuleName 'module-version-changed', got %q", f.RuleName)
				}
				if f.Severity != types.SeverityRisky {
					t.Errorf("expected severity RISKY, got %v", f.Severity)
				}
			}
		})
	}
}

func TestFormatVersionChangeMessage(t *testing.T) {
	tests := []struct {
		name       string
		moduleName string
		oldVersion string
		newVersion string
		want       string
	}{
		{
			name:       "version changed",
			moduleName: "vpc",
			oldVersion: "~> 3.0",
			newVersion: "~> 4.0",
			want:       `Module "vpc" version constraint changed: "~> 3.0" -> "~> 4.0"`,
		},
		{
			name:       "version added",
			moduleName: "rds",
			oldVersion: "",
			newVersion: "~> 2.0",
			want:       `Module "rds" version constraint added: "~> 2.0"`,
		},
		{
			name:       "version removed",
			moduleName: "s3",
			oldVersion: "~> 1.0",
			newVersion: "",
			want:       `Module "s3" version constraint removed (was "~> 1.0")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVersionChangeMessage(tt.moduleName, tt.oldVersion, tt.newVersion)
			if got != tt.want {
				t.Errorf("formatVersionChangeMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

package rules

import (
	"strings"
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestRC013_Metadata(t *testing.T) {
	r := &RC013{}

	if r.ID() != "RC013" {
		t.Errorf("expected ID 'RC013', got %q", r.ID())
	}
	if r.Name() != "validation-value-removed" {
		t.Errorf("expected Name 'validation-value-removed', got %q", r.Name())
	}
	if r.DefaultSeverity() != types.SeverityRisky {
		t.Errorf("expected severity RISKY, got %v", r.DefaultSeverity())
	}
	if r.Documentation() == nil {
		t.Error("expected Documentation to be non-nil")
	}
}

func TestRC013_Evaluate(t *testing.T) {
	tests := []struct {
		name              string
		oldVars           map[string]*types.VariableSignature
		newVars           map[string]*types.VariableSignature
		wantFindings      int
		wantMessageSubstr string
	}{
		{
			name: "value removed from contains",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["dev", "staging", "prod"], var.environment)`},
					},
				},
			},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["dev", "staging"], var.environment)`},
					},
				},
			},
			wantFindings:      1,
			wantMessageSubstr: `"prod"`,
		},
		{
			name: "multiple values removed",
			oldVars: map[string]*types.VariableSignature{
				"tier": {
					Name:            "tier",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["free", "basic", "pro", "enterprise"], var.tier)`},
					},
				},
			},
			newVars: map[string]*types.VariableSignature{
				"tier": {
					Name:            "tier",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["basic", "pro"], var.tier)`},
					},
				},
			},
			wantFindings:      1,
			wantMessageSubstr: `"free"`,
		},
		{
			name: "value added - no finding",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["dev", "staging"], var.environment)`},
					},
				},
			},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["dev", "staging", "prod"], var.environment)`},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "same values - no finding",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["dev", "prod"], var.environment)`},
					},
				},
			},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["dev", "prod"], var.environment)`},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "no contains pattern - no finding",
			oldVars: map[string]*types.VariableSignature{
				"name": {
					Name:            "name",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `length(var.name) > 0`},
					},
				},
			},
			newVars: map[string]*types.VariableSignature{
				"name": {
					Name:            "name",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `length(var.name) > 3`},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "dynamic list - no finding",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(var.allowed_envs, var.environment)`},
					},
				},
			},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(var.other_envs, var.environment)`},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "contains validation removed entirely - no finding",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["dev", "prod"], var.environment)`},
					},
				},
			},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `var.environment != ""`}, // different validation
					},
				},
			},
			wantFindings: 0,
		},
		{
			name:    "variable added - no finding",
			oldVars: map[string]*types.VariableSignature{},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["dev"], var.environment)`},
					},
				},
			},
			wantFindings: 0,
		},
		{
			name: "variable removed - no finding",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["dev", "prod"], var.environment)`},
					},
				},
			},
			newVars:      map[string]*types.VariableSignature{},
			wantFindings: 0,
		},
		{
			name: "complete replacement of values",
			oldVars: map[string]*types.VariableSignature{
				"region": {
					Name:            "region",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["us-east-1", "us-west-2"], var.region)`},
					},
				},
			},
			newVars: map[string]*types.VariableSignature{
				"region": {
					Name:            "region",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["eu-west-1", "eu-central-1"], var.region)`},
					},
				},
			},
			wantFindings:      1,
			wantMessageSubstr: `"us-east-1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := &types.ModuleSnapshot{
				Variables: tt.oldVars,
			}
			new := &types.ModuleSnapshot{
				Variables: tt.newVars,
			}

			r := &RC013{}
			findings := r.Evaluate(old, new)

			if len(findings) != tt.wantFindings {
				t.Errorf("expected %d findings, got %d", tt.wantFindings, len(findings))
				for _, f := range findings {
					t.Logf("  finding: %s", f.Message)
				}
			}

			if tt.wantMessageSubstr != "" && len(findings) > 0 {
				if !strings.Contains(findings[0].Message, tt.wantMessageSubstr) {
					t.Errorf("expected message to contain %q, got %q", tt.wantMessageSubstr, findings[0].Message)
				}
			}

			// Verify all findings have correct metadata
			for _, f := range findings {
				if f.RuleID != "RC013" {
					t.Errorf("expected RuleID 'RC013', got %q", f.RuleID)
				}
				if f.RuleName != "validation-value-removed" {
					t.Errorf("expected RuleName 'validation-value-removed', got %q", f.RuleName)
				}
				if f.Severity != types.SeverityRisky {
					t.Errorf("expected severity RISKY, got %v", f.Severity)
				}
			}
		})
	}
}

package rules

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestRC012_Metadata(t *testing.T) {
	r := &RC012{}

	if r.ID() != "RC012" {
		t.Errorf("expected ID 'RC012', got %q", r.ID())
	}
	if r.Name() != "validation-added" {
		t.Errorf("expected Name 'validation-added', got %q", r.Name())
	}
	if r.DefaultSeverity() != types.SeverityWarning {
		t.Errorf("expected severity RISKY, got %v", r.DefaultSeverity())
	}
	if r.Documentation() == nil {
		t.Error("expected Documentation to be non-nil")
	}
}

func TestRC012_Evaluate(t *testing.T) {
	tests := []struct {
		name         string
		oldVars      map[string]*types.VariableSignature
		newVars      map[string]*types.VariableSignature
		wantFindings int
		wantMessage  string
	}{
		{
			name: "validation added (0 -> 1)",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					Type:            "string",
					ValidationCount: 0,
				},
			},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					Type:            "string",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `contains(["dev", "staging", "prod"], var.environment)`},
					},
				},
			},
			wantFindings: 1,
			wantMessage:  `Variable "environment": validation block added (now has 1)`,
		},
		{
			name: "multiple validations added (1 -> 3)",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					Type:            "string",
					ValidationCount: 1,
					Validations: []types.ValidationBlock{
						{Condition: `var.environment != ""`},
					},
				},
			},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					Type:            "string",
					ValidationCount: 3,
					Validations: []types.ValidationBlock{
						{Condition: `var.environment != ""`},
						{Condition: `length(var.environment) <= 20`},
						{Condition: `contains(["dev", "staging", "prod"], var.environment)`},
					},
				},
			},
			wantFindings: 1,
			wantMessage:  `Variable "environment": validation blocks increased from 1 to 3`,
		},
		{
			name: "validation removed (2 -> 1) - no finding",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					Type:            "string",
					ValidationCount: 2,
				},
			},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					Type:            "string",
					ValidationCount: 1,
				},
			},
			wantFindings: 0,
		},
		{
			name: "validation count unchanged - no finding",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					Type:            "string",
					ValidationCount: 1,
				},
			},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					Type:            "string",
					ValidationCount: 1,
				},
			},
			wantFindings: 0,
		},
		{
			name: "both zero - no finding",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					Type:            "string",
					ValidationCount: 0,
				},
			},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					Type:            "string",
					ValidationCount: 0,
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
					Type:            "string",
					ValidationCount: 1,
				},
			},
			wantFindings: 0,
		},
		{
			name: "variable removed - no finding (handled by BC002)",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					Type:            "string",
					ValidationCount: 1,
				},
			},
			newVars:      map[string]*types.VariableSignature{},
			wantFindings: 0,
		},
		{
			name: "multiple variables - only changed ones reported",
			oldVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 0,
				},
				"region": {
					Name:            "region",
					ValidationCount: 1,
				},
				"tier": {
					Name:            "tier",
					ValidationCount: 0,
				},
			},
			newVars: map[string]*types.VariableSignature{
				"environment": {
					Name:            "environment",
					ValidationCount: 1, // added
				},
				"region": {
					Name:            "region",
					ValidationCount: 1, // unchanged
				},
				"tier": {
					Name:            "tier",
					ValidationCount: 2, // added
				},
			},
			wantFindings: 2,
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

			r := &RC012{}
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
				if f.RuleID != "RC012" {
					t.Errorf("expected RuleID 'RC012', got %q", f.RuleID)
				}
				if f.RuleName != "validation-added" {
					t.Errorf("expected RuleName 'validation-added', got %q", f.RuleName)
				}
				if f.Severity != types.SeverityWarning {
					t.Errorf("expected severity RISKY, got %v", f.Severity)
				}
			}
		})
	}
}

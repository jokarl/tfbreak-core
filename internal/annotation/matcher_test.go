package annotation

import (
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestMatcherMatch(t *testing.T) {
	tests := []struct {
		name        string
		annotations []*Annotation
		blockStarts map[string]map[int]string
		finding     *types.Finding
		expectMatch bool
	}{
		{
			name:        "no annotations",
			annotations: nil,
			finding: &types.Finding{
				RuleID:      "BC001",
				NewLocation: &types.FileRange{Filename: "test.tf", Line: 5},
			},
			expectMatch: false,
		},
		{
			name: "file level ignore matches",
			annotations: []*Annotation{
				{Scope: ScopeFile, RuleIDs: nil, Filename: "test.tf", Line: 1},
			},
			finding: &types.Finding{
				RuleID:      "BC001",
				NewLocation: &types.FileRange{Filename: "test.tf", Line: 10},
			},
			expectMatch: true,
		},
		{
			name: "file level ignore with specific rules matches",
			annotations: []*Annotation{
				{Scope: ScopeFile, RuleIDs: []string{"BC001"}, Filename: "test.tf", Line: 1},
			},
			finding: &types.Finding{
				RuleID:      "BC001",
				NewLocation: &types.FileRange{Filename: "test.tf", Line: 10},
			},
			expectMatch: true,
		},
		{
			name: "file level ignore with different rule doesn't match",
			annotations: []*Annotation{
				{Scope: ScopeFile, RuleIDs: []string{"BC002"}, Filename: "test.tf", Line: 1},
			},
			finding: &types.Finding{
				RuleID:      "BC001",
				NewLocation: &types.FileRange{Filename: "test.tf", Line: 10},
			},
			expectMatch: false,
		},
		{
			name: "block level ignore on previous line matches",
			annotations: []*Annotation{
				{Scope: ScopeBlock, RuleIDs: []string{"BC001"}, Filename: "test.tf", Line: 4},
			},
			finding: &types.Finding{
				RuleID:      "BC001",
				NewLocation: &types.FileRange{Filename: "test.tf", Line: 5},
			},
			expectMatch: true,
		},
		{
			name: "block level ignore not on previous line doesn't match",
			annotations: []*Annotation{
				{Scope: ScopeBlock, RuleIDs: []string{"BC001"}, Filename: "test.tf", Line: 2},
			},
			finding: &types.Finding{
				RuleID:      "BC001",
				NewLocation: &types.FileRange{Filename: "test.tf", Line: 5},
			},
			expectMatch: false,
		},
		{
			name: "different file doesn't match",
			annotations: []*Annotation{
				{Scope: ScopeFile, RuleIDs: nil, Filename: "other.tf", Line: 1},
			},
			finding: &types.Finding{
				RuleID:      "BC001",
				NewLocation: &types.FileRange{Filename: "test.tf", Line: 5},
			},
			expectMatch: false,
		},
		{
			name: "uses old location if new location is nil",
			annotations: []*Annotation{
				{Scope: ScopeFile, RuleIDs: nil, Filename: "old.tf", Line: 1},
			},
			finding: &types.Finding{
				RuleID:      "BC002",
				OldLocation: &types.FileRange{Filename: "old.tf", Line: 5},
				NewLocation: nil,
			},
			expectMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMatcher(tt.annotations, tt.blockStarts)
			result := m.Match(tt.finding)
			if result.Matched != tt.expectMatch {
				t.Errorf("expected match=%v, got match=%v", tt.expectMatch, result.Matched)
			}
		})
	}
}

func TestCheckGovernance(t *testing.T) {
	tests := []struct {
		name           string
		annotation     *Annotation
		config         GovernanceConfig
		expectViolation bool
		violationMsg   string
	}{
		{
			name: "disabled governance allows all",
			annotation: &Annotation{
				RuleIDs: []string{"BC001"},
				Reason:  "",
			},
			config: GovernanceConfig{
				Enabled:       false,
				RequireReason: true,
			},
			expectViolation: false,
		},
		{
			name: "require_reason without reason",
			annotation: &Annotation{
				RuleIDs: []string{"BC001"},
				Reason:  "",
			},
			config: GovernanceConfig{
				Enabled:       true,
				RequireReason: true,
			},
			expectViolation: true,
			violationMsg:    "annotation requires a reason",
		},
		{
			name: "require_reason with reason passes",
			annotation: &Annotation{
				RuleIDs: []string{"BC001"},
				Reason:  "intentional change",
			},
			config: GovernanceConfig{
				Enabled:       true,
				RequireReason: true,
			},
			expectViolation: false,
		},
		{
			name: "deny_rule_ids blocks specific rule",
			annotation: &Annotation{
				RuleIDs: []string{"BC100"},
			},
			config: GovernanceConfig{
				Enabled:      true,
				DenyRuleIDs:  []string{"BC100"},
			},
			expectViolation: true,
			violationMsg:    "rule BC100 cannot be ignored (in deny_rule_ids)",
		},
		{
			name: "deny_rule_ids allows other rules",
			annotation: &Annotation{
				RuleIDs: []string{"BC001"},
			},
			config: GovernanceConfig{
				Enabled:      true,
				DenyRuleIDs:  []string{"BC100"},
			},
			expectViolation: false,
		},
		{
			name: "deny_rule_ids blocks wildcard annotation",
			annotation: &Annotation{
				RuleIDs: nil, // all rules
			},
			config: GovernanceConfig{
				Enabled:      true,
				DenyRuleIDs:  []string{"BC100"},
			},
			expectViolation: true,
			violationMsg:    "cannot ignore all rules when deny_rule_ids is set",
		},
		{
			name: "allow_rule_ids only allows listed rules",
			annotation: &Annotation{
				RuleIDs: []string{"BC002"},
			},
			config: GovernanceConfig{
				Enabled:      true,
				AllowRuleIDs: []string{"BC001"},
			},
			expectViolation: true,
			violationMsg:    "rule BC002 is not in allow_rule_ids",
		},
		{
			name: "allow_rule_ids allows listed rule",
			annotation: &Annotation{
				RuleIDs: []string{"BC001"},
			},
			config: GovernanceConfig{
				Enabled:      true,
				AllowRuleIDs: []string{"BC001", "BC002"},
			},
			expectViolation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violation := CheckGovernance(tt.annotation, tt.config)
			if tt.expectViolation {
				if violation == nil {
					t.Error("expected violation, got nil")
				} else if tt.violationMsg != "" && violation.Message != tt.violationMsg {
					t.Errorf("expected message %q, got %q", tt.violationMsg, violation.Message)
				}
			} else {
				if violation != nil {
					t.Errorf("expected no violation, got: %s", violation.Message)
				}
			}
		})
	}
}

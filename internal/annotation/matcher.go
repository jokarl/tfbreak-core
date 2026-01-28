package annotation

import (
	"slices"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// Matcher matches annotations to findings
type Matcher struct {
	annotations map[string][]*Annotation // filename -> annotations
	blockStarts map[string]map[int]string // filename -> line -> block type
}

// NewMatcher creates a new Matcher with the given annotations and block information
func NewMatcher(annotations []*Annotation, blockStarts map[string]map[int]string) *Matcher {
	annByFile := make(map[string][]*Annotation)
	for _, ann := range annotations {
		annByFile[ann.Filename] = append(annByFile[ann.Filename], ann)
	}

	return &Matcher{
		annotations: annByFile,
		blockStarts: blockStarts,
	}
}

// MatchResult contains the result of matching an annotation to a finding
type MatchResult struct {
	Matched    bool
	Annotation *Annotation
}

// Match finds an annotation that applies to the given finding
func (m *Matcher) Match(finding *types.Finding) MatchResult {
	// Determine which file location to use
	var filename string
	var line int

	if finding.NewLocation != nil {
		filename = finding.NewLocation.Filename
		line = finding.NewLocation.Line
	} else if finding.OldLocation != nil {
		filename = finding.OldLocation.Filename
		line = finding.OldLocation.Line
	} else {
		return MatchResult{Matched: false}
	}

	anns := m.annotations[filename]
	if len(anns) == 0 {
		return MatchResult{Matched: false}
	}

	// Check file-level annotations first
	for _, ann := range anns {
		if ann.Scope == ScopeFile && ann.MatchesRule(finding.RuleID) {
			return MatchResult{Matched: true, Annotation: ann}
		}
	}

	// Check block-level annotations
	// Find the annotation on the line immediately before the finding's block
	for _, ann := range anns {
		if ann.Scope != ScopeBlock {
			continue
		}

		if !ann.MatchesRule(finding.RuleID) {
			continue
		}

		// Check if the annotation is on the line immediately before the finding
		// or immediately before the block containing the finding
		if ann.Line == line-1 {
			return MatchResult{Matched: true, Annotation: ann}
		}

		// Check if annotation is immediately before a block that contains this line
		if blocks, ok := m.blockStarts[filename]; ok {
			for blockLine := range blocks {
				if ann.Line == blockLine-1 && line >= blockLine {
					// The annotation is right before a block, and the finding is at or after that block
					return MatchResult{Matched: true, Annotation: ann}
				}
			}
		}
	}

	return MatchResult{Matched: false}
}

// GovernanceConfig contains settings for annotation governance
type GovernanceConfig struct {
	Enabled       bool
	RequireReason bool
	AllowRuleIDs  []string
	DenyRuleIDs   []string
}

// CheckGovernance checks if an annotation violates governance rules
func CheckGovernance(ann *Annotation, cfg GovernanceConfig) *GovernanceViolation {
	if !cfg.Enabled {
		return nil
	}

	// Check expiration
	if ann.IsExpired() {
		return &GovernanceViolation{
			Annotation: ann,
			Message:    "annotation has expired",
		}
	}

	// Check require_reason
	if cfg.RequireReason && ann.Reason == "" {
		return &GovernanceViolation{
			Annotation: ann,
			Message:    "annotation requires a reason",
		}
	}

	// Check allow_rule_ids (if non-empty, only these rules can be ignored)
	if len(cfg.AllowRuleIDs) > 0 {
		for _, ruleID := range ann.RuleIDs {
			if !slices.Contains(cfg.AllowRuleIDs, ruleID) {
				return &GovernanceViolation{
					Annotation: ann,
					Message:    "rule " + ruleID + " is not in allow_rule_ids",
				}
			}
		}
	}

	// Check deny_rule_ids
	if len(cfg.DenyRuleIDs) > 0 {
		for _, ruleID := range ann.RuleIDs {
			if slices.Contains(cfg.DenyRuleIDs, ruleID) {
				return &GovernanceViolation{
					Annotation: ann,
					Message:    "rule " + ruleID + " cannot be ignored (in deny_rule_ids)",
				}
			}
		}
		// Also check if annotation has empty RuleIDs (means all rules)
		// and any deny rules exist
		if len(ann.RuleIDs) == 0 && len(cfg.DenyRuleIDs) > 0 {
			return &GovernanceViolation{
				Annotation: ann,
				Message:    "cannot ignore all rules when deny_rule_ids is set",
			}
		}
	}

	return nil
}

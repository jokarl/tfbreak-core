// Package annotation handles parsing and matching of tfbreak ignore annotations.
package annotation

import (
	"slices"
	"time"
)

// Scope defines where an annotation applies
type Scope int

const (
	// ScopeBlock applies to the immediately following block
	ScopeBlock Scope = iota
	// ScopeFile applies to the entire file
	ScopeFile
)

// Annotation represents a parsed tfbreak ignore annotation
type Annotation struct {
	// Scope determines whether this applies to a block or entire file
	Scope Scope

	// RuleIDs is the list of rule IDs to ignore (empty = all rules)
	RuleIDs []string

	// Reason is the documented reason for ignoring
	Reason string

	// Ticket is an optional ticket/issue reference
	Ticket string

	// Expires is an optional expiration date
	Expires *time.Time

	// Location is where the annotation was found
	Filename string
	Line     int

	// BlockLine is the line of the block this annotation applies to (for ScopeBlock)
	// This is set during matching, not parsing
	BlockLine int
}

// IsExpired returns true if the annotation has an expiration date that has passed
func (a *Annotation) IsExpired() bool {
	if a.Expires == nil {
		return false
	}
	return time.Now().After(*a.Expires)
}

// MatchesRule returns true if this annotation applies to the given rule ID
func (a *Annotation) MatchesRule(ruleID string) bool {
	// Empty list means all rules
	if len(a.RuleIDs) == 0 {
		return true
	}

	return slices.Contains(a.RuleIDs, ruleID)
}

// GovernanceViolation represents a violation of annotation governance rules
type GovernanceViolation struct {
	Annotation *Annotation
	Message    string
}

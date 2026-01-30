package plugin

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/jokarl/tfbreak-plugin-sdk/tflint"
)

// Issue represents a finding emitted by a plugin rule.
type Issue struct {
	// Rule is the rule that emitted the issue.
	Rule tflint.Rule
	// Message is the issue message.
	Message string
	// Range is the source location of the issue.
	Range hcl.Range
}

// Issues is a slice of Issue for convenience.
type Issues []Issue

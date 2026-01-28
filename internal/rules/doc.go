package rules

import "github.com/jokarl/tfbreak-core/internal/types"

// RuleDoc contains documentation for a rule
type RuleDoc struct {
	ID              string
	Name            string
	DefaultSeverity types.Severity
	Description     string
	ExampleOld      string
	ExampleNew      string
	Remediation     string
}

// Documentable is implemented by rules that provide documentation
type Documentable interface {
	Documentation() *RuleDoc
}

// GetDocumentation returns the documentation for a rule if available
func GetDocumentation(ruleID string) *RuleDoc {
	r, ok := DefaultRegistry.Get(ruleID)
	if !ok {
		return nil
	}

	if doc, ok := r.(Documentable); ok {
		return doc.Documentation()
	}

	// Fallback to basic info from the rule interface
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
	}
}

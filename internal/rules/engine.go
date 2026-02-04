package rules

import "github.com/jokarl/tfbreak-core/internal/types"

// Engine evaluates rules against module snapshots
type Engine struct {
	registry *Registry
	config   map[string]*RuleConfig
}

// NewEngine creates a new Engine with the given registry
func NewEngine(registry *Registry) *Engine {
	return &Engine{
		registry: registry,
		config:   make(map[string]*RuleConfig),
	}
}

// NewDefaultEngine creates an Engine with the default registry and default configs
func NewDefaultEngine() *Engine {
	e := NewEngine(DefaultRegistry)
	// Initialize default configs for all registered rules
	for _, rule := range DefaultRegistry.All() {
		e.config[rule.ID()] = DefaultRuleConfig(rule)
	}
	return e
}

// SetConfig sets the configuration for a specific rule
func (e *Engine) SetConfig(ruleID string, config *RuleConfig) {
	e.config[ruleID] = config
}

// GetConfig returns the configuration for a specific rule
func (e *Engine) GetConfig(ruleID string) *RuleConfig {
	if cfg, ok := e.config[ruleID]; ok {
		return cfg
	}
	// Return default config if not explicitly set
	if rule, ok := e.registry.Get(ruleID); ok {
		return DefaultRuleConfig(rule)
	}
	return nil
}

// EnableRule enables a rule
func (e *Engine) EnableRule(ruleID string) {
	if cfg := e.GetConfig(ruleID); cfg != nil {
		cfg.Enabled = true
		e.config[ruleID] = cfg
	}
}

// DisableRule disables a rule
func (e *Engine) DisableRule(ruleID string) {
	if cfg := e.GetConfig(ruleID); cfg != nil {
		cfg.Enabled = false
		e.config[ruleID] = cfg
	}
}

// DisableAllRules disables all rules in the engine
func (e *Engine) DisableAllRules() {
	for _, rule := range e.registry.All() {
		cfg := e.GetConfig(rule.ID())
		if cfg != nil {
			cfg.Enabled = false
			e.config[rule.ID()] = cfg
		}
	}
}

// Evaluate runs all enabled rules against the old and new snapshots
func (e *Engine) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for _, rule := range e.registry.All() {
		cfg := e.GetConfig(rule.ID())
		if cfg == nil || !cfg.Enabled {
			continue
		}

		ruleFindings := rule.Evaluate(old, new)
		for _, f := range ruleFindings {
			// Apply configured severity if different from default
			if cfg.Severity != rule.DefaultSeverity() {
				f.Severity = cfg.Severity
			}
			findings = append(findings, f)
		}
	}

	// Apply rename detection suppression if enabled
	if IsRenameDetectionEnabled() {
		findings = applyRenameSuppression(findings)
	}

	return findings
}

// applyRenameSuppression removes findings that should be suppressed by rename detection
// - BC003 (input-renamed) suppresses BC001 and BC002 for the matched variable pair
// - RC003 (input-renamed-optional) suppresses BC002 for the matched variable
// - BC010 (output-renamed) suppresses BC009 for the matched output
func applyRenameSuppression(findings []*types.Finding) []*types.Finding {
	// Collect suppression data from rename findings
	suppressedVarRemovals := make(map[string]bool)    // old var names to suppress BC002
	suppressedVarAdditions := make(map[string]bool)   // new var names to suppress BC001
	suppressedOutputRemovals := make(map[string]bool) // old output names to suppress BC009

	for _, f := range findings {
		if f.Metadata == nil {
			continue
		}

		switch f.RuleID {
		case "BC003":
			// BC003 suppresses both BC002 (removal) and BC001 (addition)
			if oldName, ok := f.Metadata["old_name"]; ok {
				suppressedVarRemovals[oldName] = true
			}
			if newName, ok := f.Metadata["new_name"]; ok {
				suppressedVarAdditions[newName] = true
			}
		case "RC003":
			// RC003 only suppresses BC002 (removal)
			if oldName, ok := f.Metadata["old_name"]; ok {
				suppressedVarRemovals[oldName] = true
			}
		case "BC010":
			// BC010 suppresses BC009 (output removal)
			if oldName, ok := f.Metadata["old_name"]; ok {
				suppressedOutputRemovals[oldName] = true
			}
		}
	}

	// Filter out suppressed findings
	var filtered []*types.Finding
	for _, f := range findings {
		suppressed := false

		switch f.RuleID {
		case "BC001":
			// Check if this is a suppressed variable addition
			// BC001 message format: "New required variable %q has no default"
			// We need to extract the variable name from the message or metadata
			if name := extractVariableNameFromBC001(f); name != "" {
				suppressed = suppressedVarAdditions[name]
			}
		case "BC002":
			// Check if this is a suppressed variable removal
			// BC002 message format: "Variable %q was removed"
			if name := extractVariableNameFromBC002(f); name != "" {
				suppressed = suppressedVarRemovals[name]
			}
		case "BC009":
			// Check if this is a suppressed output removal
			// BC009 message format: "Output %q was removed"
			if name := extractOutputNameFromBC009(f); name != "" {
				suppressed = suppressedOutputRemovals[name]
			}
		}

		if !suppressed {
			filtered = append(filtered, f)
		}
	}

	return filtered
}

// extractVariableNameFromBC001 extracts the variable name from a BC001 finding
func extractVariableNameFromBC001(f *types.Finding) string {
	// Message format: "New required variable %q has no default"
	// Try to extract the quoted variable name
	return extractQuotedName(f.Message)
}

// extractVariableNameFromBC002 extracts the variable name from a BC002 finding
func extractVariableNameFromBC002(f *types.Finding) string {
	// Message format: "Variable %q was removed"
	return extractQuotedName(f.Message)
}

// extractOutputNameFromBC009 extracts the output name from a BC009 finding
func extractOutputNameFromBC009(f *types.Finding) string {
	// Message format: "Output %q was removed"
	return extractQuotedName(f.Message)
}

// extractQuotedName extracts the first quoted string from a message
func extractQuotedName(message string) string {
	// Find first occurrence of quoted string
	start := -1
	for i, c := range message {
		if c == '"' {
			if start == -1 {
				start = i + 1
			} else {
				return message[start:i]
			}
		}
	}
	return ""
}

// CheckOptions configures the behavior of the Check method
type CheckOptions struct {
	// IncludeRemediation populates remediation text for each finding
	IncludeRemediation bool
}

// Check runs the engine and returns a complete CheckResult
func (e *Engine) Check(oldPath, newPath string, old, new *types.ModuleSnapshot, failOn types.Severity) *types.CheckResult {
	return e.CheckWithOptions(oldPath, newPath, old, new, failOn, CheckOptions{})
}

// CheckWithOptions runs the engine with additional options
func (e *Engine) CheckWithOptions(oldPath, newPath string, old, new *types.ModuleSnapshot, failOn types.Severity, opts CheckOptions) *types.CheckResult {
	result := types.NewCheckResult(oldPath, newPath, failOn)

	findings := e.Evaluate(old, new)
	for _, f := range findings {
		// Populate remediation if requested
		if opts.IncludeRemediation {
			e.populateRemediation(f)
		}
		result.AddFinding(f)
	}

	result.Compute()
	return result
}

// populateRemediation adds remediation text to a finding from its rule's documentation
func (e *Engine) populateRemediation(f *types.Finding) {
	doc := GetDocumentation(f.RuleID)
	if doc != nil && doc.Remediation != "" {
		f.Remediation = doc.Remediation
	}
}

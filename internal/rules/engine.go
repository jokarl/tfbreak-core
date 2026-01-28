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

	return findings
}

// Check runs the engine and returns a complete CheckResult
func (e *Engine) Check(oldPath, newPath string, old, new *types.ModuleSnapshot, failOn types.Severity) *types.CheckResult {
	result := types.NewCheckResult(oldPath, newPath, failOn)

	findings := e.Evaluate(old, new)
	for _, f := range findings {
		result.AddFinding(f)
	}

	result.Compute()
	return result
}

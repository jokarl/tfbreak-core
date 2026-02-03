// Package plugin provides plugin discovery, loading, and execution for tfbreak.
//
// This file implements the plugin manager, which coordinates plugin lifecycle
// (discovery, loading, execution, cleanup) and aggregates results from plugins.
package plugin

import (
	"fmt"
	"os"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"

	"github.com/jokarl/tfbreak-core/internal/config"
	"github.com/jokarl/tfbreak-core/internal/types"
	"github.com/jokarl/tfbreak-plugin-sdk/tflint"
)

// RuleSetWithCheck extends tflint.RuleSet with a Check method.
// The SDK's GRPCRuleSetClient implements this interface.
type RuleSetWithCheck interface {
	tflint.RuleSet
	// Check executes all enabled rules using the provided runner.
	Check(runner tflint.Runner) error
}

// Manager manages plugin lifecycle and execution.
type Manager struct {
	loader  *Loader
	plugins []*LoadedPlugin
	config  *config.Config
	logger  hclog.Logger
	mu      sync.RWMutex
}

// NewManager creates a new plugin manager.
func NewManager(cfg *config.Config) *Manager {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "tfbreak-plugin-manager",
		Level:  hclog.Warn,
		Output: os.Stderr,
	})

	return &Manager{
		loader: NewLoaderWithLogger(logger),
		config: cfg,
		logger: logger,
	}
}

// DiscoverAndLoad discovers plugins and loads enabled ones.
// Returns the number of plugins loaded and any errors encountered.
func (m *Manager) DiscoverAndLoad() (int, []error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Discover plugins
	discovered, err := Discover(m.config)
	if err != nil {
		return 0, []error{fmt.Errorf("plugin discovery failed: %w", err)}
	}

	// Filter to enabled plugins
	enabled := GetEnabledPlugins(discovered)

	if len(enabled) == 0 {
		return 0, nil
	}

	// Load enabled plugins
	loaded, errs := m.loader.LoadAll(enabled)
	m.plugins = loaded

	// Apply configuration to loaded plugins
	for _, p := range m.plugins {
		if err := m.configurePlugin(p); err != nil {
			errs = append(errs, fmt.Errorf("failed to configure plugin %s: %w", p.Info.Name, err))
		}
	}

	return len(m.plugins), errs
}

// configurePlugin applies configuration to a loaded plugin.
func (m *Manager) configurePlugin(p *LoadedPlugin) error {
	// Convert internal config to SDK config format
	sdkConfig := m.toSDKConfig()

	// Apply global configuration
	if err := p.RuleSet.ApplyGlobalConfig(sdkConfig); err != nil {
		return fmt.Errorf("ApplyGlobalConfig failed: %w", err)
	}

	// Note: Plugin-specific configuration (ApplyConfig) would be parsed
	// from the config file and passed here. For now, we pass nil.
	// Future: parse plugin-specific config blocks and pass BodyContent.

	return nil
}

// toSDKConfig converts internal config to SDK tflint.Config format.
func (m *Manager) toSDKConfig() *tflint.Config {
	if m.config == nil {
		return &tflint.Config{}
	}

	rules := make(map[string]*tflint.RuleConfig)
	for _, rc := range m.config.Rules {
		enabled := true
		if rc.Enabled != nil {
			enabled = *rc.Enabled
		}
		rules[rc.ID] = &tflint.RuleConfig{
			Name:    rc.ID,
			Enabled: enabled,
		}
	}

	return &tflint.Config{
		Rules:     rules,
		PluginDir: m.config.GetPluginDir(),
	}
}

// ExecuteRules executes all loaded plugin rules against the provided configurations.
// Returns a slice of findings from all plugins.
func (m *Manager) ExecuteRules(oldFiles, newFiles map[string]*hcl.File) ([]*types.Finding, []error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allFindings []*types.Finding
	var allErrors []error

	for _, p := range m.plugins {
		findings, err := m.executePluginRules(p, oldFiles, newFiles)
		if err != nil {
			allErrors = append(allErrors, fmt.Errorf("plugin %s: %w", p.Info.Name, err))
			continue
		}
		allFindings = append(allFindings, findings...)
	}

	return allFindings, allErrors
}

// executePluginRules executes rules for a single plugin.
func (m *Manager) executePluginRules(p *LoadedPlugin, oldFiles, newFiles map[string]*hcl.File) ([]*types.Finding, error) {
	// Create a runner that provides old/new configurations to the plugin
	runner := NewRunner(oldFiles, newFiles)

	// The RuleSet from the loader should implement RuleSetWithCheck
	// (the SDK's GRPCRuleSetClient has a Check method)
	checker, ok := p.RuleSet.(RuleSetWithCheck)
	if !ok {
		return nil, fmt.Errorf("plugin does not support Check method")
	}

	// Execute the plugin's Check method, which runs all enabled rules
	// The plugin will call back to our runner to get configurations
	// and emit issues
	if err := checker.Check(runner); err != nil {
		return nil, fmt.Errorf("check failed: %w", err)
	}

	// Convert plugin issues to internal findings
	findings := make([]*types.Finding, 0, len(runner.Issues))
	for _, issue := range runner.Issues {
		finding := m.issueToFinding(issue, p.Info.Name)
		findings = append(findings, finding)
	}

	return findings, nil
}

// issueToFinding converts a plugin Issue to an internal Finding.
func (m *Manager) issueToFinding(issue Issue, pluginName string) *types.Finding {
	// Convert SDK severity to internal severity
	severity := m.convertSeverity(issue.Rule.Severity())

	// Create the finding
	finding := &types.Finding{
		RuleID:   fmt.Sprintf("%s/%s", pluginName, issue.Rule.Name()),
		RuleName: issue.Rule.Name(),
		Severity: severity,
		Message:  issue.Message,
	}

	// Set location from range
	if issue.Range.Filename != "" {
		finding.NewLocation = &types.FileRange{
			Filename: issue.Range.Filename,
			Line:     issue.Range.Start.Line,
		}
	}

	return finding
}

// convertSeverity converts SDK severity to internal severity.
func (m *Manager) convertSeverity(s tflint.Severity) types.Severity {
	switch s {
	case tflint.ERROR:
		return types.SeverityError
	case tflint.WARNING:
		return types.SeverityWarning
	case tflint.NOTICE:
		return types.SeverityNotice
	default:
		return types.SeverityError
	}
}

// GetLoadedPlugins returns information about loaded plugins.
func (m *Manager) GetLoadedPlugins() []PluginSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summaries := make([]PluginSummary, 0, len(m.plugins))
	for _, p := range m.plugins {
		summaries = append(summaries, PluginSummary{
			Name:      p.RuleSet.RuleSetName(),
			Version:   p.RuleSet.RuleSetVersion(),
			RuleCount: len(p.RuleSet.RuleNames()),
			Rules:     p.RuleSet.RuleNames(),
		})
	}
	return summaries
}

// PluginSummary contains summary information about a loaded plugin.
type PluginSummary struct {
	Name      string
	Version   string
	RuleCount int
	Rules     []string
}

// Close terminates all loaded plugins.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	CloseAll(m.plugins)
	m.plugins = nil
}

// HasPlugins returns true if any plugins are loaded.
func (m *Manager) HasPlugins() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.plugins) > 0
}

// PluginCount returns the number of loaded plugins.
func (m *Manager) PluginCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.plugins)
}

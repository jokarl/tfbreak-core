package plugin

import (
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/jokarl/tfbreak-core/internal/config"
	"github.com/jokarl/tfbreak-core/internal/types"
	"github.com/jokarl/tfbreak-plugin-sdk/tflint"
)

func TestNewManager(t *testing.T) {
	cfg := config.Default()
	mgr := NewManager(cfg)
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
	if mgr.loader == nil {
		t.Error("manager.loader is nil")
	}
	if mgr.config != cfg {
		t.Error("manager.config does not match provided config")
	}
}

func TestManager_Close_NoPlugins(t *testing.T) {
	cfg := config.Default()
	mgr := NewManager(cfg)

	// Should not panic with no plugins loaded
	mgr.Close()
}

func TestManager_HasPlugins_Empty(t *testing.T) {
	cfg := config.Default()
	mgr := NewManager(cfg)

	if mgr.HasPlugins() {
		t.Error("HasPlugins() should return false when no plugins loaded")
	}
}

func TestManager_PluginCount_Empty(t *testing.T) {
	cfg := config.Default()
	mgr := NewManager(cfg)

	if mgr.PluginCount() != 0 {
		t.Errorf("PluginCount() = %d, want 0", mgr.PluginCount())
	}
}

func TestManager_GetLoadedPlugins_Empty(t *testing.T) {
	cfg := config.Default()
	mgr := NewManager(cfg)

	summaries := mgr.GetLoadedPlugins()
	if len(summaries) != 0 {
		t.Errorf("GetLoadedPlugins() returned %d summaries, want 0", len(summaries))
	}
}

func TestManager_DiscoverAndLoad_NoPlugins(t *testing.T) {
	cleanup := isolatePluginDiscovery(t)
	defer cleanup()

	// Create config with non-existent plugin directory
	cfg := config.Default()
	cfg.ConfigBlock = &config.ConfigBlockConfig{
		PluginDir: t.TempDir(), // Empty temp directory
	}

	mgr := NewManager(cfg)
	defer mgr.Close()

	count, errors := mgr.DiscoverAndLoad()

	if count != 0 {
		t.Errorf("DiscoverAndLoad() count = %d, want 0", count)
	}

	if len(errors) != 0 {
		t.Errorf("DiscoverAndLoad() errors = %v, want none", errors)
	}
}

func TestManager_ExecuteRules_NoPlugins(t *testing.T) {
	cfg := config.Default()
	mgr := NewManager(cfg)
	defer mgr.Close()

	oldFiles := make(map[string]*hcl.File)
	newFiles := make(map[string]*hcl.File)

	findings, errors := mgr.ExecuteRules(oldFiles, newFiles)

	if len(findings) != 0 {
		t.Errorf("ExecuteRules() returned %d findings, want 0", len(findings))
	}

	if len(errors) != 0 {
		t.Errorf("ExecuteRules() returned %d errors, want 0", len(errors))
	}
}

func TestManager_toSDKConfig_NilConfig(t *testing.T) {
	mgr := &Manager{config: nil}
	sdkConfig := mgr.toSDKConfig()

	if sdkConfig == nil {
		t.Fatal("toSDKConfig() returned nil")
	}

	if sdkConfig.Rules == nil && sdkConfig.PluginDir != "" {
		// Either rules is nil or pluginDir is empty is fine
	}
}

func TestManager_toSDKConfig_WithRules(t *testing.T) {
	enabled := true
	disabled := false

	cfg := config.Default()
	cfg.Rules = []*config.RuleConfig{
		{ID: "rule1", Enabled: &enabled},
		{ID: "rule2", Enabled: &disabled},
	}

	mgr := NewManager(cfg)
	sdkConfig := mgr.toSDKConfig()

	if len(sdkConfig.Rules) != 2 {
		t.Errorf("got %d rules, want 2", len(sdkConfig.Rules))
	}

	if rule1, ok := sdkConfig.Rules["rule1"]; ok {
		if !rule1.Enabled {
			t.Error("rule1 should be enabled")
		}
	} else {
		t.Error("rule1 not found in SDK config")
	}

	if rule2, ok := sdkConfig.Rules["rule2"]; ok {
		if rule2.Enabled {
			t.Error("rule2 should be disabled")
		}
	} else {
		t.Error("rule2 not found in SDK config")
	}
}

func TestManager_convertSeverity(t *testing.T) {
	mgr := &Manager{}

	tests := []struct {
		name  string
		input tflint.Severity
		want  types.Severity
	}{
		{"ERROR maps to SeverityError", tflint.ERROR, types.SeverityError},
		{"WARNING maps to SeverityWarning", tflint.WARNING, types.SeverityWarning},
		{"NOTICE maps to SeverityNotice", tflint.NOTICE, types.SeverityNotice},
		{"Unknown severity (0) defaults to SeverityError", tflint.Severity(0), types.SeverityError},
		{"Unknown severity (99) defaults to SeverityError", tflint.Severity(99), types.SeverityError},
		{"Unknown severity (-1) defaults to SeverityError", tflint.Severity(-1), types.SeverityError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgr.convertSeverity(tt.input)
			if got != tt.want {
				t.Errorf("convertSeverity(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestSeverityMappingConsistency verifies that severity mappings between
// tflint SDK and internal types produce consistent string outputs.
// This test guards against unintentional changes to severity semantics.
func TestSeverityMappingConsistency(t *testing.T) {
	mgr := &Manager{}

	// Verify that mapped severities produce the same string representation
	tests := []struct {
		sdkSeverity  tflint.Severity
		expectedStr  string
	}{
		{tflint.ERROR, "ERROR"},
		{tflint.WARNING, "WARNING"},
		{tflint.NOTICE, "NOTICE"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedStr, func(t *testing.T) {
			// SDK severity string
			sdkStr := tt.sdkSeverity.String()
			if sdkStr != tt.expectedStr {
				t.Errorf("SDK severity String() = %q, want %q", sdkStr, tt.expectedStr)
			}

			// Converted internal severity string
			internal := mgr.convertSeverity(tt.sdkSeverity)
			internalStr := internal.String()
			if internalStr != tt.expectedStr {
				t.Errorf("Internal severity String() = %q, want %q", internalStr, tt.expectedStr)
			}
		})
	}
}

func TestPluginSummary_Fields(t *testing.T) {
	summary := PluginSummary{
		Name:      "test-plugin",
		Version:   "1.0.0",
		RuleCount: 5,
		Rules:     []string{"rule1", "rule2", "rule3", "rule4", "rule5"},
	}

	if summary.Name != "test-plugin" {
		t.Errorf("Name = %q, want %q", summary.Name, "test-plugin")
	}

	if summary.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", summary.Version, "1.0.0")
	}

	if summary.RuleCount != 5 {
		t.Errorf("RuleCount = %d, want 5", summary.RuleCount)
	}

	if len(summary.Rules) != 5 {
		t.Errorf("len(Rules) = %d, want 5", len(summary.Rules))
	}
}

func TestManager_toSDKConfig_RulesWithNilEnabled(t *testing.T) {
	cfg := config.Default()
	cfg.Rules = []*config.RuleConfig{
		{ID: "rule1", Enabled: nil}, // nil Enabled should default to true
	}

	mgr := NewManager(cfg)
	sdkConfig := mgr.toSDKConfig()

	if rule1, ok := sdkConfig.Rules["rule1"]; ok {
		if !rule1.Enabled {
			t.Error("rule1 should default to enabled when Enabled is nil")
		}
	} else {
		t.Error("rule1 not found in SDK config")
	}
}

func TestManager_HasPlugins_AfterClose(t *testing.T) {
	cfg := config.Default()
	mgr := NewManager(cfg)

	// Close should clear plugins
	mgr.Close()

	if mgr.HasPlugins() {
		t.Error("HasPlugins() should return false after Close()")
	}

	if mgr.PluginCount() != 0 {
		t.Errorf("PluginCount() = %d, want 0 after Close()", mgr.PluginCount())
	}
}

func TestManager_ExecuteRules_WithNilFiles(t *testing.T) {
	cfg := config.Default()
	mgr := NewManager(cfg)
	defer mgr.Close()

	// Should not panic with nil file maps
	findings, errors := mgr.ExecuteRules(nil, nil)

	if len(findings) != 0 {
		t.Errorf("ExecuteRules(nil, nil) returned %d findings, want 0", len(findings))
	}

	if len(errors) != 0 {
		t.Errorf("ExecuteRules(nil, nil) returned %d errors, want 0", len(errors))
	}
}

func TestManager_DiscoverAndLoad_MultipleCallsAreSafe(t *testing.T) {
	cleanup := isolatePluginDiscovery(t)
	defer cleanup()

	cfg := config.Default()
	cfg.ConfigBlock = &config.ConfigBlockConfig{
		PluginDir: t.TempDir(),
	}

	mgr := NewManager(cfg)
	defer mgr.Close()

	// First call
	count1, err1 := mgr.DiscoverAndLoad()
	if len(err1) != 0 {
		t.Errorf("First DiscoverAndLoad() errors = %v", err1)
	}

	// Second call should be safe
	count2, err2 := mgr.DiscoverAndLoad()
	if len(err2) != 0 {
		t.Errorf("Second DiscoverAndLoad() errors = %v", err2)
	}

	// Both should return same count (0 in this case since no plugins)
	if count1 != count2 {
		t.Errorf("DiscoverAndLoad() counts differ: %d vs %d", count1, count2)
	}
}

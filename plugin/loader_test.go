package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
	if loader.logger == nil {
		t.Error("loader.logger is nil")
	}
}

func TestLoader_Load_NonexistentPlugin(t *testing.T) {
	loader := NewLoader()

	info := PluginInfo{
		Name: "nonexistent",
		Path: "/path/to/nonexistent/plugin",
	}

	_, err := loader.Load(info)
	if err == nil {
		t.Error("expected error for nonexistent plugin")
	}
}

func TestLoader_Load_InvalidPlugin(t *testing.T) {
	// Create a fake plugin binary that's not a valid go-plugin
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "tfbreak-ruleset-fake")
	if runtime.GOOS == "windows" {
		pluginPath += ".exe"
	}

	// Write a non-executable or invalid binary
	if err := os.WriteFile(pluginPath, []byte("not a valid plugin"), 0755); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	info := PluginInfo{
		Name:    "fake",
		Path:    pluginPath,
		Enabled: true,
	}

	// This should fail because it's not a valid go-plugin binary
	_, err := loader.Load(info)
	if err == nil {
		t.Error("expected error for invalid plugin binary")
	}
}

func TestLoader_LoadAll_SkipsDisabled(t *testing.T) {
	loader := NewLoader()

	// Create a list of plugins, some disabled
	plugins := []PluginInfo{
		{Name: "disabled1", Path: "/fake/path1", Enabled: false},
		{Name: "disabled2", Path: "/fake/path2", Enabled: false},
	}

	loaded, errors := loader.LoadAll(plugins)

	// Should not load any plugins (all disabled)
	if len(loaded) != 0 {
		t.Errorf("got %d loaded plugins, want 0", len(loaded))
	}

	// Should not have errors (disabled plugins are skipped, not failed)
	if len(errors) != 0 {
		t.Errorf("got %d errors, want 0", len(errors))
	}
}

func TestCloseAll(t *testing.T) {
	// Test with nil slice - should not panic
	CloseAll(nil)

	// Test with empty slice - should not panic
	CloseAll([]*LoadedPlugin{})

	// Test with plugin that has nil client - should not panic
	plugins := []*LoadedPlugin{
		{Info: PluginInfo{Name: "test"}, Client: nil},
	}
	CloseAll(plugins)
}

func TestLoadedPlugin_Close_NilClient(t *testing.T) {
	p := &LoadedPlugin{
		Info:   PluginInfo{Name: "test"},
		Client: nil,
	}

	// Should not panic
	p.Close()
}

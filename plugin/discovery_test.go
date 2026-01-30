package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jokarl/tfbreak-core/internal/config"
)

func TestDiscover_ConfigPath(t *testing.T) {
	// Create temp directory with a plugin
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "tfbreak-ruleset-test")
	if runtime.GOOS == "windows" {
		pluginPath += ".exe"
	}

	if err := os.WriteFile(pluginPath, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.ConfigBlock = &config.ConfigBlockConfig{
		PluginDir: tmpDir,
	}

	plugins, err := Discover(cfg)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	found := false
	for _, p := range plugins {
		if p.Name == "test" {
			found = true
			if p.Path != pluginPath {
				t.Errorf("Path = %q, want %q", p.Path, pluginPath)
			}
			break
		}
	}

	if !found {
		t.Error("plugin 'test' not found")
	}
}

func TestDiscover_EnvVar(t *testing.T) {
	// Create temp directory with a plugin
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "tfbreak-ruleset-envtest")
	if runtime.GOOS == "windows" {
		pluginPath += ".exe"
	}

	if err := os.WriteFile(pluginPath, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}

	// Set env var
	oldEnv := os.Getenv(PluginDirEnv)
	os.Setenv(PluginDirEnv, tmpDir)
	defer os.Setenv(PluginDirEnv, oldEnv)

	cfg := config.Default()

	plugins, err := Discover(cfg)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	found := false
	for _, p := range plugins {
		if p.Name == "envtest" {
			found = true
			break
		}
	}

	if !found {
		t.Error("plugin 'envtest' not found from env var path")
	}
}

func TestDiscover_NamingConvention(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with different naming conventions
	files := []string{
		"tfbreak-ruleset-valid",     // Valid
		"tfbreak-ruleset-",          // Invalid - empty name
		"tflint-ruleset-wrong",      // Invalid - wrong prefix
		"tfbreak-valid",             // Invalid - missing 'ruleset-'
		"random-file",               // Invalid - no prefix
	}

	for _, f := range files {
		name := f
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("fake"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.Default()
	cfg.ConfigBlock = &config.ConfigBlockConfig{
		PluginDir: tmpDir,
	}

	plugins, err := Discover(cfg)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should only find "valid"
	if len(plugins) != 1 {
		t.Errorf("got %d plugins, want 1", len(plugins))
	}

	if len(plugins) > 0 && plugins[0].Name != "valid" {
		t.Errorf("plugin name = %q, want %q", plugins[0].Name, "valid")
	}
}

func TestDiscover_Priority(t *testing.T) {
	// Create two directories with plugins of the same name
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Plugin in dir1 (config path - higher priority)
	plugin1 := filepath.Join(dir1, "tfbreak-ruleset-priority")
	// Plugin in dir2 (env var - lower priority)
	plugin2 := filepath.Join(dir2, "tfbreak-ruleset-priority")

	if runtime.GOOS == "windows" {
		plugin1 += ".exe"
		plugin2 += ".exe"
	}

	if err := os.WriteFile(plugin1, []byte("dir1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plugin2, []byte("dir2"), 0755); err != nil {
		t.Fatal(err)
	}

	// Set env var to dir2
	oldEnv := os.Getenv(PluginDirEnv)
	os.Setenv(PluginDirEnv, dir2)
	defer os.Setenv(PluginDirEnv, oldEnv)

	// Set config to dir1
	cfg := config.Default()
	cfg.ConfigBlock = &config.ConfigBlockConfig{
		PluginDir: dir1,
	}

	plugins, err := Discover(cfg)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find only one plugin, from dir1 (config takes precedence)
	var found *PluginInfo
	for i, p := range plugins {
		if p.Name == "priority" {
			found = &plugins[i]
			break
		}
	}

	if found == nil {
		t.Fatal("plugin 'priority' not found")
	}

	if found.Path != plugin1 {
		t.Errorf("Path = %q, want %q (config path should take precedence)", found.Path, plugin1)
	}
}

func TestDiscover_EnabledStatus(t *testing.T) {
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "tfbreak-ruleset-status")
	if runtime.GOOS == "windows" {
		pluginPath += ".exe"
	}

	if err := os.WriteFile(pluginPath, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with plugin disabled
	enabled := false
	cfg := config.Default()
	cfg.ConfigBlock = &config.ConfigBlockConfig{
		PluginDir: tmpDir,
	}
	cfg.Plugins = []*config.PluginConfig{
		{Name: "status", Enabled: &enabled},
	}

	plugins, err := Discover(cfg)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	var found *PluginInfo
	for i, p := range plugins {
		if p.Name == "status" {
			found = &plugins[i]
			break
		}
	}

	if found == nil {
		t.Fatal("plugin 'status' not found")
	}

	if found.Enabled {
		t.Error("plugin should be disabled based on config")
	}
}

func TestDiscover_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.Default()
	cfg.ConfigBlock = &config.ConfigBlockConfig{
		PluginDir: tmpDir,
	}

	plugins, err := Discover(cfg)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(plugins) != 0 {
		t.Errorf("got %d plugins, want 0", len(plugins))
	}
}

func TestDiscover_NonexistentDirectory(t *testing.T) {
	cfg := config.Default()
	cfg.ConfigBlock = &config.ConfigBlockConfig{
		PluginDir: "/nonexistent/path/that/does/not/exist",
	}

	// Should not error, just return empty
	plugins, err := Discover(cfg)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// May find plugins in other paths, but not from the nonexistent one
	_ = plugins
}

func TestGetEnabledPlugins(t *testing.T) {
	plugins := []PluginInfo{
		{Name: "enabled1", Enabled: true},
		{Name: "disabled1", Enabled: false},
		{Name: "enabled2", Enabled: true},
		{Name: "disabled2", Enabled: false},
	}

	enabled := GetEnabledPlugins(plugins)

	if len(enabled) != 2 {
		t.Errorf("got %d enabled plugins, want 2", len(enabled))
	}

	for _, p := range enabled {
		if !p.Enabled {
			t.Errorf("plugin %q should be enabled", p.Name)
		}
	}
}

func TestStripExecutableExtension(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"plugin", "plugin"},
		{"plugin.exe", "plugin"}, // Only stripped on Windows
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripExecutableExtension(tt.input)
			if runtime.GOOS == "windows" {
				if got != tt.want {
					t.Errorf("stripExecutableExtension(%q) = %q, want %q", tt.input, got, tt.want)
				}
			} else {
				// On non-Windows, .exe is not stripped
				if got != tt.input {
					t.Errorf("stripExecutableExtension(%q) = %q, want %q (unchanged on non-Windows)", tt.input, got, tt.input)
				}
			}
		})
	}
}

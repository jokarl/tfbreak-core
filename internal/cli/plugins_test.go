package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPluginsCmd_Exists(t *testing.T) {
	// Verify the plugins command is registered
	if pluginsCmd == nil {
		t.Fatal("pluginsCmd is nil")
	}

	if pluginsCmd.Use != "plugins" {
		t.Errorf("pluginsCmd.Use = %q, want %q", pluginsCmd.Use, "plugins")
	}
}

func TestPluginsListCmd_Exists(t *testing.T) {
	// Verify the plugins list command is registered
	if pluginsListCmd == nil {
		t.Fatal("pluginsListCmd is nil")
	}

	if pluginsListCmd.Use != "list" {
		t.Errorf("pluginsListCmd.Use = %q, want %q", pluginsListCmd.Use, "list")
	}
}

func TestPluginsListCmd_HasConfigFlag(t *testing.T) {
	// Verify the -c/--config flag is registered
	flag := pluginsListCmd.Flags().Lookup("config")
	if flag == nil {
		t.Fatal("plugins list command missing --config flag")
	}

	if flag.Shorthand != "c" {
		t.Errorf("--config flag shorthand = %q, want %q", flag.Shorthand, "c")
	}
}

func TestRunPluginsList_NoPlugins(t *testing.T) {
	// Create a temporary directory with no plugins
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	// Create empty plugin directory
	pluginDir := filepath.Join(tmpDir, ".tfbreak.d", "plugins")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin directory: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Clear the config path flag
	pluginsListConfigPath = ""

	// Run should not error even with no plugins
	err = runPluginsList(nil, nil)
	if err != nil {
		t.Errorf("runPluginsList returned error with no plugins: %v", err)
	}
}

func TestRunPluginsList_WithConfigPath(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a valid config file (version is required)
	configContent := `# tfbreak configuration
version = 1
`
	configPath := filepath.Join(tmpDir, ".tfbreak.hcl")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create plugin directory
	pluginDir := filepath.Join(tmpDir, ".tfbreak.d", "plugins")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin directory: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Set config path flag
	pluginsListConfigPath = configPath
	defer func() { pluginsListConfigPath = "" }()

	// Should not error
	err := runPluginsList(nil, nil)
	if err != nil {
		t.Errorf("runPluginsList with config path returned error: %v", err)
	}
}

func TestRunPluginsList_InvalidConfigPath(t *testing.T) {
	// Set to a non-existent config path
	pluginsListConfigPath = "/nonexistent/path/.tfbreak.hcl"
	defer func() { pluginsListConfigPath = "" }()

	// Should error because config file doesn't exist
	err := runPluginsList(nil, nil)
	if err == nil {
		t.Error("expected error for non-existent config path")
	}
}

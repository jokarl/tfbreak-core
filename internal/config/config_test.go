package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}

	if cfg.Paths == nil {
		t.Fatal("expected paths to be set")
	}
	if len(cfg.Paths.Include) != 1 || cfg.Paths.Include[0] != "**/*.tf" {
		t.Errorf("unexpected include patterns: %v", cfg.Paths.Include)
	}
	if len(cfg.Paths.Exclude) != 1 || cfg.Paths.Exclude[0] != ".terraform/**" {
		t.Errorf("unexpected exclude patterns: %v", cfg.Paths.Exclude)
	}

	if cfg.Output == nil {
		t.Fatal("expected output to be set")
	}
	if cfg.Output.Format != "text" {
		t.Errorf("expected format 'text', got %s", cfg.Output.Format)
	}
	if cfg.Output.Color != "auto" {
		t.Errorf("expected color 'auto', got %s", cfg.Output.Color)
	}

	if cfg.Policy == nil {
		t.Fatal("expected policy to be set")
	}
	if cfg.Policy.FailOn != "ERROR" {
		t.Errorf("expected fail_on 'ERROR', got %s", cfg.Policy.FailOn)
	}

	if cfg.Annotations == nil {
		t.Fatal("expected annotations to be set")
	}
	if cfg.Annotations.Enabled == nil || !*cfg.Annotations.Enabled {
		t.Error("expected annotations to be enabled by default")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tfbreak.hcl")

	configContent := `
version = 1

paths {
  include = ["**/*.tf", "**/*.tf.json"]
  exclude = [".terraform/**", "**/examples/**"]
}

output {
  format = "json"
  color  = "never"
}

policy {
  fail_on = "WARNING"
}

annotations {
  enabled        = true
  require_reason = true
}

rules "required-input-added" {
  enabled  = false
  severity = "NOTICE"
}
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath, "")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}

	if len(cfg.Paths.Include) != 2 {
		t.Errorf("expected 2 include patterns, got %d", len(cfg.Paths.Include))
	}
	if len(cfg.Paths.Exclude) != 2 {
		t.Errorf("expected 2 exclude patterns, got %d", len(cfg.Paths.Exclude))
	}

	if cfg.Output.Format != "json" {
		t.Errorf("expected format 'json', got %s", cfg.Output.Format)
	}
	if cfg.Output.Color != "never" {
		t.Errorf("expected color 'never', got %s", cfg.Output.Color)
	}

	if cfg.Policy.FailOn != "WARNING" {
		t.Errorf("expected fail_on 'WARNING', got %s", cfg.Policy.FailOn)
	}

	if !cfg.Annotations.RequireReason {
		t.Error("expected require_reason to be true")
	}

	// Check rule config
	if !cfg.IsRuleEnabled("input-removed") {
		t.Error("expected input-removed to be enabled by default")
	}
	if cfg.IsRuleEnabled("required-input-added") {
		t.Error("expected required-input-added to be disabled")
	}

	sev := cfg.GetRuleSeverity("required-input-added", types.SeverityError)
	if sev != types.SeverityNotice {
		t.Errorf("expected required-input-added severity INFO, got %s", sev)
	}
}

func TestLoadNotFound(t *testing.T) {
	// Load with explicit path that doesn't exist
	_, err := Load("/nonexistent/path/.tfbreak.hcl", "")
	if err == nil {
		t.Error("expected error for nonexistent config")
	}
}

func TestLoadDefaultsWhenNoConfig(t *testing.T) {
	// Load from empty directory (no config file)
	tmpDir := t.TempDir()

	cfg, err := Load("", tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Should return defaults
	if cfg.Version != 1 {
		t.Errorf("expected default version 1, got %d", cfg.Version)
	}
	if cfg.Output.Format != "text" {
		t.Errorf("expected default format 'text', got %s", cfg.Output.Format)
	}
}

func TestLoadInvalidHCL(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tfbreak.hcl")

	// Invalid HCL syntax
	invalidContent := `
version = 1
this is not valid HCL {
`
	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath, "")
	if err == nil {
		t.Error("expected error for invalid HCL")
	}
}

func TestLoadInvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tfbreak.hcl")

	configContent := `version = 2`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath, "")
	if err == nil {
		t.Error("expected error for unsupported version")
	}
}

func TestLoadInvalidSeverity(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tfbreak.hcl")

	configContent := `
version = 1
policy {
  fail_on = "INVALID"
}
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath, "")
	if err == nil {
		t.Error("expected error for invalid severity")
	}
}

func TestLoadInvalidRuleID(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tfbreak.hcl")

	configContent := `
version = 1
rules "INVALID_RULE" {
  enabled = true
}
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath, "")
	if err == nil {
		t.Error("expected error for invalid rule ID")
	}
}

func TestConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tfbreak.hcl")

	configContent := `version = 1`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath, "")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.ConfigPath() != configPath {
		t.Errorf("expected config path %s, got %s", configPath, cfg.ConfigPath())
	}

	// Default config should have empty path
	defaultCfg := Default()
	if defaultCfg.ConfigPath() != "" {
		t.Errorf("expected empty config path for defaults, got %s", defaultCfg.ConfigPath())
	}
}

func TestIsAnnotationsEnabled(t *testing.T) {
	// Default
	cfg := Default()
	if !cfg.IsAnnotationsEnabled() {
		t.Error("expected annotations enabled by default")
	}

	// Explicitly disabled
	disabled := false
	cfg.Annotations.Enabled = &disabled
	if cfg.IsAnnotationsEnabled() {
		t.Error("expected annotations disabled")
	}

	// Nil annotations block
	cfg.Annotations = nil
	if !cfg.IsAnnotationsEnabled() {
		t.Error("expected annotations enabled when block is nil")
	}
}

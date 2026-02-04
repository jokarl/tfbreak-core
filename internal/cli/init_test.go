package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunInit_CreatesConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Reset the flag
	forceFlag = false

	// Run init
	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit returned error: %v", err)
	}

	// Check that the file was created
	configPath := filepath.Join(tmpDir, ".tfbreak.hcl")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("expected config file to be created at %s", configPath)
	}

	// Check that the file has content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	if len(content) == 0 {
		t.Errorf("expected config file to have content")
	}
}

func TestRunInit_ExistingFile_NoForce(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create an existing config file
	configPath := filepath.Join(tmpDir, ".tfbreak.hcl")
	if err := os.WriteFile(configPath, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to create existing config: %v", err)
	}

	// Reset the flag
	forceFlag = false

	// Run init - should fail
	err = runInit(nil, nil)
	if err == nil {
		t.Errorf("expected error when config file exists without --force")
	}

	// Verify existing content was not modified
	content, _ := os.ReadFile(configPath)
	if string(content) != "existing content" {
		t.Errorf("existing config file was modified")
	}
}

func TestRunInit_ExistingFile_WithForce(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create an existing config file
	configPath := filepath.Join(tmpDir, ".tfbreak.hcl")
	if err := os.WriteFile(configPath, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to create existing config: %v", err)
	}

	// Set force flag
	forceFlag = true
	defer func() { forceFlag = false }()

	// Run init - should succeed
	if err := runInit(nil, nil); err != nil {
		t.Errorf("runInit with --force returned error: %v", err)
	}

	// Verify content was overwritten
	content, _ := os.ReadFile(configPath)
	if string(content) == "existing content" {
		t.Errorf("config file was not overwritten with --force")
	}
}

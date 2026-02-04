package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHCLFiles_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	files, err := LoadHCLFiles(tmpDir)
	if err != nil {
		t.Fatalf("LoadHCLFiles() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("got %d files, want 0", len(files))
	}
}

func TestLoadHCLFiles_NonexistentDirectory(t *testing.T) {
	_, err := LoadHCLFiles("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestLoadHCLFiles_NotADirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadHCLFiles(filePath)
	if err == nil {
		t.Error("expected error for file path (not directory)")
	}
}

func TestLoadHCLFiles_ValidTerraformFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid Terraform files
	mainTf := `
variable "name" {
  type    = string
  default = "test"
}

output "result" {
  value = var.name
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(mainTf), 0644); err != nil {
		t.Fatal(err)
	}

	variablesTf := `
variable "environment" {
  type = string
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "variables.tf"), []byte(variablesTf), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := LoadHCLFiles(tmpDir)
	if err != nil {
		t.Fatalf("LoadHCLFiles() error = %v", err)
	}

	if len(files) != 2 {
		t.Errorf("got %d files, want 2", len(files))
	}

	// Verify file paths are absolute
	for path := range files {
		if !filepath.IsAbs(path) {
			t.Errorf("file path %q is not absolute", path)
		}
	}
}

func TestLoadHCLFiles_IgnoresNonTfFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create various files
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(`variable "x" {}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# Readme"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".terraform.lock.hcl"), []byte("# Lock file"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := LoadHCLFiles(tmpDir)
	if err != nil {
		t.Fatalf("LoadHCLFiles() error = %v", err)
	}

	// Should only load the .tf file
	if len(files) != 1 {
		t.Errorf("got %d files, want 1", len(files))
	}
}

func TestLoadHCLFiles_IgnoresSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file in root
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(`variable "x" {}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory with a file
	subDir := filepath.Join(tmpDir, "submodule")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "sub.tf"), []byte(`variable "y" {}`), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := LoadHCLFiles(tmpDir)
	if err != nil {
		t.Fatalf("LoadHCLFiles() error = %v", err)
	}

	// Should only load root level file, not subdirectory
	if len(files) != 1 {
		t.Errorf("got %d files, want 1", len(files))
	}
}

func TestLoadHCLFilesWithFilter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple files
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(`variable "main" {}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "variables.tf"), []byte(`variable "vars" {}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "outputs.tf"), []byte(`output "out" {}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Filter to only include variables.tf
	filter := func(filename string) bool {
		return filename == "variables.tf"
	}

	files, err := LoadHCLFilesWithFilter(tmpDir, filter)
	if err != nil {
		t.Fatalf("LoadHCLFilesWithFilter() error = %v", err)
	}

	if len(files) != 1 {
		t.Errorf("got %d files, want 1", len(files))
	}

	// Verify the correct file was loaded
	found := false
	for path := range files {
		if filepath.Base(path) == "variables.tf" {
			found = true
			break
		}
	}
	if !found {
		t.Error("variables.tf not found in loaded files")
	}
}

func TestLoadHCLFilesWithFilter_NilFilter(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(`variable "x" {}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Nil filter should load all files
	files, err := LoadHCLFilesWithFilter(tmpDir, nil)
	if err != nil {
		t.Fatalf("LoadHCLFilesWithFilter() error = %v", err)
	}

	if len(files) != 1 {
		t.Errorf("got %d files, want 1", len(files))
	}
}

func TestLoadHCLFiles_SkipsInvalidHCL(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid file
	if err := os.WriteFile(filepath.Join(tmpDir, "valid.tf"), []byte(`variable "x" {}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an invalid HCL file
	if err := os.WriteFile(filepath.Join(tmpDir, "invalid.tf"), []byte(`this is not { valid HCL }`), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := LoadHCLFiles(tmpDir)
	if err != nil {
		t.Fatalf("LoadHCLFiles() error = %v", err)
	}

	// Should only load the valid file (invalid file is skipped)
	if len(files) != 1 {
		t.Errorf("got %d files, want 1 (invalid file should be skipped)", len(files))
	}
}

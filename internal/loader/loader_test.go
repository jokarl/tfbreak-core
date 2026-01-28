package loader

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jokarl/tfbreak-core/internal/pathfilter"
)

func getTestdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", "loader")
}

func TestLoadEmptyDirectory(t *testing.T) {
	dir := filepath.Join(getTestdataDir(), "empty")
	snap, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(snap.Variables) != 0 {
		t.Errorf("expected 0 variables, got %d", len(snap.Variables))
	}
	if len(snap.Outputs) != 0 {
		t.Errorf("expected 0 outputs, got %d", len(snap.Outputs))
	}
	if len(snap.Resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(snap.Resources))
	}
	if len(snap.Modules) != 0 {
		t.Errorf("expected 0 modules, got %d", len(snap.Modules))
	}
}

func TestLoadVariables(t *testing.T) {
	dir := filepath.Join(getTestdataDir(), "basic")
	snap, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(snap.Variables) != 3 {
		t.Fatalf("expected 3 variables, got %d", len(snap.Variables))
	}

	// Check required variable
	reqVar, ok := snap.Variables["required_var"]
	if !ok {
		t.Fatal("required_var not found")
	}
	if reqVar.Name != "required_var" {
		t.Errorf("Name = %q, want %q", reqVar.Name, "required_var")
	}
	if reqVar.Type != "string" {
		t.Errorf("Type = %q, want %q", reqVar.Type, "string")
	}
	if !reqVar.Required {
		t.Error("Required = false, want true")
	}
	if reqVar.Description != "A required variable" {
		t.Errorf("Description = %q, want %q", reqVar.Description, "A required variable")
	}

	// Check optional variable
	optVar, ok := snap.Variables["optional_var"]
	if !ok {
		t.Fatal("optional_var not found")
	}
	if optVar.Required {
		t.Error("Required = true, want false")
	}
	if optVar.Default != "default_value" {
		t.Errorf("Default = %v, want %q", optVar.Default, "default_value")
	}

	// Check sensitive variable
	sensVar, ok := snap.Variables["sensitive_var"]
	if !ok {
		t.Fatal("sensitive_var not found")
	}
	if !sensVar.Sensitive {
		t.Error("Sensitive = false, want true")
	}
}

func TestLoadOutputs(t *testing.T) {
	dir := filepath.Join(getTestdataDir(), "basic")
	snap, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(snap.Outputs) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(snap.Outputs))
	}

	// Check simple output
	simpleOut, ok := snap.Outputs["simple_output"]
	if !ok {
		t.Fatal("simple_output not found")
	}
	if simpleOut.Name != "simple_output" {
		t.Errorf("Name = %q, want %q", simpleOut.Name, "simple_output")
	}
	if simpleOut.Description != "A simple output" {
		t.Errorf("Description = %q, want %q", simpleOut.Description, "A simple output")
	}
	if simpleOut.Sensitive {
		t.Error("Sensitive = true, want false")
	}

	// Check sensitive output
	sensOut, ok := snap.Outputs["sensitive_output"]
	if !ok {
		t.Fatal("sensitive_output not found")
	}
	if !sensOut.Sensitive {
		t.Error("Sensitive = false, want true")
	}
}

func TestLoadResources(t *testing.T) {
	dir := filepath.Join(getTestdataDir(), "basic")
	snap, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(snap.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(snap.Resources))
	}

	// Check null_resource
	nullRes, ok := snap.Resources["null_resource.example"]
	if !ok {
		t.Fatal("null_resource.example not found")
	}
	if nullRes.Type != "null_resource" {
		t.Errorf("Type = %q, want %q", nullRes.Type, "null_resource")
	}
	if nullRes.Name != "example" {
		t.Errorf("Name = %q, want %q", nullRes.Name, "example")
	}
	if nullRes.Address != "null_resource.example" {
		t.Errorf("Address = %q, want %q", nullRes.Address, "null_resource.example")
	}

	// Check local_file
	localFile, ok := snap.Resources["local_file.config"]
	if !ok {
		t.Fatal("local_file.config not found")
	}
	if localFile.Type != "local_file" {
		t.Errorf("Type = %q, want %q", localFile.Type, "local_file")
	}
}

func TestLoadModules(t *testing.T) {
	dir := filepath.Join(getTestdataDir(), "basic")
	snap, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(snap.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(snap.Modules))
	}

	mod, ok := snap.Modules["submodule"]
	if !ok {
		t.Fatal("submodule not found")
	}
	if mod.Name != "submodule" {
		t.Errorf("Name = %q, want %q", mod.Name, "submodule")
	}
	if mod.Source != "./submodule" {
		t.Errorf("Source = %q, want %q", mod.Source, "./submodule")
	}
	if mod.Address != "module.submodule" {
		t.Errorf("Address = %q, want %q", mod.Address, "module.submodule")
	}
}

func TestLoadNonExistentDirectory(t *testing.T) {
	_, err := Load("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestLoadFile(t *testing.T) {
	// Load should fail if given a file instead of a directory
	dir := filepath.Join(getTestdataDir(), "basic", "main.tf")
	_, err := Load(dir)
	if err == nil {
		t.Error("expected error when loading a file")
	}
}

func TestLoadWithFilter(t *testing.T) {
	dir := filepath.Join(getTestdataDir(), "basic")

	// Test with default filter (should include all .tf files)
	filter := pathfilter.DefaultFilter()
	snap, err := LoadWithFilter(dir, filter)
	if err != nil {
		t.Fatalf("LoadWithFilter() error = %v", err)
	}

	// Should have all the same content as Load()
	if len(snap.Variables) != 3 {
		t.Errorf("expected 3 variables, got %d", len(snap.Variables))
	}
	if len(snap.Outputs) != 2 {
		t.Errorf("expected 2 outputs, got %d", len(snap.Outputs))
	}
}

func TestLoadWithFilterNil(t *testing.T) {
	dir := filepath.Join(getTestdataDir(), "basic")

	// Test with nil filter (should work like regular Load)
	snap, err := LoadWithFilter(dir, nil)
	if err != nil {
		t.Fatalf("LoadWithFilter() error = %v", err)
	}

	if len(snap.Variables) != 3 {
		t.Errorf("expected 3 variables, got %d", len(snap.Variables))
	}
}

func TestLoadWithFilterExclude(t *testing.T) {
	dir := filepath.Join(getTestdataDir(), "basic")

	// Test excluding outputs.tf
	filter := pathfilter.New([]string{"**/*.tf"}, []string{"outputs.tf"})
	snap, err := LoadWithFilter(dir, filter)
	if err != nil {
		t.Fatalf("LoadWithFilter() error = %v", err)
	}

	// Should have variables but no outputs (outputs are in outputs.tf)
	if len(snap.Variables) != 3 {
		t.Errorf("expected 3 variables, got %d", len(snap.Variables))
	}
	if len(snap.Outputs) != 0 {
		t.Errorf("expected 0 outputs (excluded), got %d", len(snap.Outputs))
	}
}

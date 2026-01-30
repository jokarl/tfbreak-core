package plugin

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/jokarl/tfbreak-plugin-sdk/hclext"
	"github.com/jokarl/tfbreak-plugin-sdk/tflint"
)

// testRule is a minimal rule for testing.
type testRule struct {
	tflint.DefaultRule
	name string
}

func (r *testRule) Name() string              { return r.name }
func (r *testRule) Link() string              { return "" }
func (r *testRule) Check(_ tflint.Runner) error { return nil }

func TestRunner_ImplementsInterface(t *testing.T) {
	runner, err := NewRunnerFromContent(
		map[string]string{},
		map[string]string{},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Verify Runner satisfies tflint.Runner
	var _ tflint.Runner = runner
}

func TestRunner_GetOldResourceContent(t *testing.T) {
	runner, err := NewRunnerFromContent(
		map[string]string{
			"main.tf": `
resource "azurerm_resource_group" "example" {
  name     = "my-rg"
  location = "westeurope"
}`,
		},
		map[string]string{
			"main.tf": `
resource "azurerm_resource_group" "example" {
  name     = "my-rg"
  location = "eastus"
}`,
		},
	)
	if err != nil {
		t.Fatalf("NewRunnerFromContent() error = %v", err)
	}

	schema := &hclext.BodySchema{
		Attributes: []hclext.AttributeSchema{
			{Name: "name"},
			{Name: "location"},
		},
	}

	content, err := runner.GetOldResourceContent("azurerm_resource_group", schema, nil)
	if err != nil {
		t.Fatalf("GetOldResourceContent() error = %v", err)
	}

	if len(content.Blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(content.Blocks))
	}

	block := content.Blocks[0]
	if block.Type != "resource" {
		t.Errorf("block type = %q, want %q", block.Type, "resource")
	}
	if len(block.Labels) < 2 || block.Labels[1] != "example" {
		t.Errorf("block labels = %v, want [azurerm_resource_group example]", block.Labels)
	}
}

func TestRunner_GetNewResourceContent(t *testing.T) {
	runner, err := NewRunnerFromContent(
		map[string]string{
			"main.tf": `
resource "azurerm_resource_group" "example" {
  location = "westeurope"
}`,
		},
		map[string]string{
			"main.tf": `
resource "azurerm_resource_group" "example" {
  location = "eastus"
}`,
		},
	)
	if err != nil {
		t.Fatalf("NewRunnerFromContent() error = %v", err)
	}

	schema := &hclext.BodySchema{
		Attributes: []hclext.AttributeSchema{
			{Name: "location"},
		},
	}

	content, err := runner.GetNewResourceContent("azurerm_resource_group", schema, nil)
	if err != nil {
		t.Fatalf("GetNewResourceContent() error = %v", err)
	}

	if len(content.Blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(content.Blocks))
	}
}

func TestRunner_GetResourceContent_FiltersByType(t *testing.T) {
	runner, err := NewRunnerFromContent(
		map[string]string{
			"main.tf": `
resource "azurerm_resource_group" "rg" {
  name = "my-rg"
}
resource "azurerm_virtual_network" "vnet" {
  name = "my-vnet"
}
resource "azurerm_resource_group" "rg2" {
  name = "my-rg-2"
}`,
		},
		map[string]string{},
	)
	if err != nil {
		t.Fatalf("NewRunnerFromContent() error = %v", err)
	}

	schema := &hclext.BodySchema{
		Attributes: []hclext.AttributeSchema{
			{Name: "name"},
		},
	}

	content, err := runner.GetOldResourceContent("azurerm_resource_group", schema, nil)
	if err != nil {
		t.Fatalf("GetOldResourceContent() error = %v", err)
	}

	if len(content.Blocks) != 2 {
		t.Errorf("got %d resource_group blocks, want 2", len(content.Blocks))
	}

	for _, block := range content.Blocks {
		if block.Labels[0] != "azurerm_resource_group" {
			t.Errorf("unexpected resource type: %s", block.Labels[0])
		}
	}
}

func TestRunner_GetOldModuleContent(t *testing.T) {
	runner, err := NewRunnerFromContent(
		map[string]string{
			"main.tf": `
variable "location" {
  default = "westeurope"
}`,
		},
		map[string]string{},
	)
	if err != nil {
		t.Fatalf("NewRunnerFromContent() error = %v", err)
	}

	schema := &hclext.BodySchema{
		Blocks: []hclext.BlockSchema{
			{
				Type:       "variable",
				LabelNames: []string{"name"},
			},
		},
	}

	content, err := runner.GetOldModuleContent(schema, nil)
	if err != nil {
		t.Fatalf("GetOldModuleContent() error = %v", err)
	}

	if len(content.Blocks) != 1 {
		t.Fatalf("got %d variable blocks, want 1", len(content.Blocks))
	}

	if content.Blocks[0].Labels[0] != "location" {
		t.Errorf("variable name = %q, want %q", content.Blocks[0].Labels[0], "location")
	}
}

func TestRunner_EmitIssue(t *testing.T) {
	runner, err := NewRunnerFromContent(map[string]string{}, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	rule := &testRule{name: "test_rule"}
	issueRange := hcl.Range{
		Filename: "main.tf",
		Start:    hcl.Pos{Line: 3, Column: 3},
		End:      hcl.Pos{Line: 3, Column: 20},
	}

	err = runner.EmitIssue(rule, "test message", issueRange)
	if err != nil {
		t.Fatalf("EmitIssue() error = %v", err)
	}

	if len(runner.Issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(runner.Issues))
	}

	issue := runner.Issues[0]
	if issue.Rule.Name() != "test_rule" {
		t.Errorf("issue rule name = %q, want %q", issue.Rule.Name(), "test_rule")
	}
	if issue.Message != "test message" {
		t.Errorf("issue message = %q, want %q", issue.Message, "test message")
	}
	if issue.Range.Filename != "main.tf" {
		t.Errorf("issue range filename = %q, want %q", issue.Range.Filename, "main.tf")
	}
}

func TestRunner_EmitIssue_Multiple(t *testing.T) {
	runner, err := NewRunnerFromContent(map[string]string{}, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	rule := &testRule{name: "test_rule"}

	_ = runner.EmitIssue(rule, "issue 1", hcl.Range{})
	_ = runner.EmitIssue(rule, "issue 2", hcl.Range{})
	_ = runner.EmitIssue(rule, "issue 3", hcl.Range{})

	if len(runner.Issues) != 3 {
		t.Errorf("got %d issues, want 3", len(runner.Issues))
	}
}

func TestRunner_DecodeRuleConfig(t *testing.T) {
	runner, err := NewRunnerFromContent(map[string]string{}, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	var target struct{ Value string }
	err = runner.DecodeRuleConfig("test_rule", &target)
	if err != nil {
		t.Errorf("DecodeRuleConfig() error = %v", err)
	}
}

func TestNewRunner(t *testing.T) {
	oldFiles := make(map[string]*hcl.File)
	newFiles := make(map[string]*hcl.File)

	runner := NewRunner(oldFiles, newFiles)

	if runner == nil {
		t.Fatal("NewRunner() returned nil")
	}
	if runner.Issues == nil {
		t.Error("runner.Issues should be initialized")
	}
}

func TestNewRunnerFromContent_InvalidHCL(t *testing.T) {
	_, err := NewRunnerFromContent(
		map[string]string{
			"main.tf": `this is not valid HCL {{{`,
		},
		map[string]string{},
	)

	if err == nil {
		t.Error("expected error for invalid HCL, got nil")
	}
}

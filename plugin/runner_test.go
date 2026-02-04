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

func TestRunner_GetResourceContent_WithNestedAttributes(t *testing.T) {
	runner, err := NewRunnerFromContent(
		map[string]string{
			"main.tf": `
resource "azurerm_storage_account" "example" {
  name     = "storageacct"
  location = "westeurope"

  network_rules {
    default_action = "Deny"
    ip_rules       = ["10.0.0.0/8"]
  }
}`,
		},
		map[string]string{},
	)
	if err != nil {
		t.Fatalf("NewRunnerFromContent() error = %v", err)
	}

	// Schema that requests nested block content
	schema := &hclext.BodySchema{
		Attributes: []hclext.AttributeSchema{
			{Name: "name"},
			{Name: "location"},
		},
		Blocks: []hclext.BlockSchema{
			{
				Type: "network_rules",
				Body: &hclext.BodySchema{
					Attributes: []hclext.AttributeSchema{
						{Name: "default_action"},
						{Name: "ip_rules"},
					},
				},
			},
		},
	}

	content, err := runner.GetOldResourceContent("azurerm_storage_account", schema, nil)
	if err != nil {
		t.Fatalf("GetOldResourceContent() error = %v", err)
	}

	if len(content.Blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(content.Blocks))
	}

	block := content.Blocks[0]
	if block.Body == nil {
		t.Fatal("block.Body is nil")
	}

	// Check that we have the nested network_rules block
	networkRulesFound := false
	for _, nestedBlock := range block.Body.Blocks {
		if nestedBlock.Type == "network_rules" {
			networkRulesFound = true
			if nestedBlock.Body == nil {
				t.Error("network_rules body is nil")
			}
		}
	}

	if !networkRulesFound {
		t.Error("network_rules block not found in nested content")
	}
}

func TestRunner_GetResourceContent_DeeplyNested(t *testing.T) {
	// Test three levels of nesting: resource > blob_properties > cors_rule
	runner, err := NewRunnerFromContent(
		map[string]string{
			"main.tf": `
resource "azurerm_storage_account" "example" {
  name = "storageacct"

  blob_properties {
    versioning_enabled = true

    cors_rule {
      allowed_methods = ["GET", "POST"]
      allowed_origins = ["*"]
    }
  }
}`,
		},
		map[string]string{},
	)
	if err != nil {
		t.Fatalf("NewRunnerFromContent() error = %v", err)
	}

	// Three-level nested schema
	schema := &hclext.BodySchema{
		Attributes: []hclext.AttributeSchema{
			{Name: "name"},
		},
		Blocks: []hclext.BlockSchema{
			{
				Type: "blob_properties",
				Body: &hclext.BodySchema{
					Attributes: []hclext.AttributeSchema{
						{Name: "versioning_enabled"},
					},
					Blocks: []hclext.BlockSchema{
						{
							Type: "cors_rule",
							Body: &hclext.BodySchema{
								Attributes: []hclext.AttributeSchema{
									{Name: "allowed_methods"},
									{Name: "allowed_origins"},
								},
							},
						},
					},
				},
			},
		},
	}

	content, err := runner.GetOldResourceContent("azurerm_storage_account", schema, nil)
	if err != nil {
		t.Fatalf("GetOldResourceContent() error = %v", err)
	}

	if len(content.Blocks) != 1 {
		t.Fatalf("got %d resource blocks, want 1", len(content.Blocks))
	}

	resourceBlock := content.Blocks[0]
	if resourceBlock.Body == nil {
		t.Fatal("resource block body is nil")
	}

	// Find blob_properties
	var blobProps *hclext.Block
	for _, b := range resourceBlock.Body.Blocks {
		if b.Type == "blob_properties" {
			blobProps = b
			break
		}
	}
	if blobProps == nil {
		t.Fatal("blob_properties block not found")
	}
	if blobProps.Body == nil {
		t.Fatal("blob_properties body is nil")
	}

	// Verify versioning_enabled attribute
	if blobProps.Body.Attributes["versioning_enabled"] == nil {
		t.Error("versioning_enabled attribute not found in blob_properties")
	}

	// Find cors_rule (third level)
	var corsRule *hclext.Block
	for _, b := range blobProps.Body.Blocks {
		if b.Type == "cors_rule" {
			corsRule = b
			break
		}
	}
	if corsRule == nil {
		t.Fatal("cors_rule block not found (third level)")
	}
	if corsRule.Body == nil {
		t.Fatal("cors_rule body is nil")
	}

	// Verify deeply nested attributes
	if corsRule.Body.Attributes["allowed_methods"] == nil {
		t.Error("allowed_methods attribute not found in cors_rule")
	}
	if corsRule.Body.Attributes["allowed_origins"] == nil {
		t.Error("allowed_origins attribute not found in cors_rule")
	}
}

func TestRunner_GetModuleContent_WithMultipleFiles(t *testing.T) {
	runner, err := NewRunnerFromContent(
		map[string]string{
			"main.tf": `
variable "env" {
  default = "prod"
}`,
			"variables.tf": `
variable "region" {
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

	if len(content.Blocks) != 2 {
		t.Errorf("got %d variable blocks, want 2 (from multiple files)", len(content.Blocks))
	}

	// Verify both variables are found
	varNames := make(map[string]bool)
	for _, block := range content.Blocks {
		if len(block.Labels) > 0 {
			varNames[block.Labels[0]] = true
		}
	}

	if !varNames["env"] {
		t.Error("variable 'env' not found")
	}
	if !varNames["region"] {
		t.Error("variable 'region' not found")
	}
}

func TestRunner_GetResourceContent_NoMatchingResources(t *testing.T) {
	runner, err := NewRunnerFromContent(
		map[string]string{
			"main.tf": `
resource "azurerm_resource_group" "example" {
  name = "my-rg"
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

	// Request a resource type that doesn't exist
	content, err := runner.GetOldResourceContent("azurerm_virtual_network", schema, nil)
	if err != nil {
		t.Fatalf("GetOldResourceContent() error = %v", err)
	}

	if len(content.Blocks) != 0 {
		t.Errorf("got %d blocks, want 0 for non-existent resource type", len(content.Blocks))
	}
}

func TestRunner_EmptyFiles(t *testing.T) {
	runner := NewRunner(nil, nil)

	schema := &hclext.BodySchema{
		Attributes: []hclext.AttributeSchema{
			{Name: "test"},
		},
	}

	// Should not panic with nil files
	content, err := runner.GetOldModuleContent(schema, nil)
	if err != nil {
		t.Fatalf("GetOldModuleContent() error = %v", err)
	}

	if content == nil {
		t.Fatal("content should not be nil")
	}
}

func TestLabelsMatch(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{"both empty", []string{}, []string{}, true},
		{"both nil", nil, nil, true},
		{"same single", []string{"foo"}, []string{"foo"}, true},
		{"same multiple", []string{"foo", "bar"}, []string{"foo", "bar"}, true},
		{"different length", []string{"foo"}, []string{"foo", "bar"}, false},
		{"different values", []string{"foo"}, []string{"bar"}, false},
		{"one nil one empty", nil, []string{}, true},
		{"different order", []string{"foo", "bar"}, []string{"bar", "foo"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := labelsMatch(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("labelsMatch(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

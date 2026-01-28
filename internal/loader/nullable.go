package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// variableBlockSchema defines the schema for extracting variable blocks
var variableBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "variable",
			LabelNames: []string{"name"},
		},
	},
}

// variableNullableSchema defines the schema for extracting nullable from a variable block
var variableNullableSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "nullable", Required: false},
	},
}

// NullableMap maps variable names to their nullable attribute values.
// A nil value means the attribute was not specified (defaults to true in Terraform 1.1+).
type NullableMap map[string]*bool

// parseNullableAttributes parses the nullable attribute from all variable blocks
// in the given directory. Returns a map of variable name to nullable value.
func parseNullableAttributes(dir string) (NullableMap, error) {
	result := make(NullableMap)

	// Find all .tf files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	parser := hclparse.NewParser()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		fileNullables, err := parseNullableFromFile(parser, filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
		}

		// Merge results
		for name, nullable := range fileNullables {
			result[name] = nullable
		}
	}

	return result, nil
}

// parseNullableFromFile parses nullable attributes from variable blocks in a single file
func parseNullableFromFile(parser *hclparse.Parser, filePath string) (NullableMap, error) {
	file, diags := parser.ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL parse error: %s", diags.Error())
	}

	content, _, diags := file.Body.PartialContent(variableBlockSchema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to extract variable blocks: %s", diags.Error())
	}

	result := make(NullableMap)

	for _, block := range content.Blocks {
		if block.Type != "variable" {
			continue
		}

		if len(block.Labels) < 1 {
			continue
		}

		varName := block.Labels[0]
		nullable, err := extractNullableAttribute(block)
		if err != nil {
			return nil, fmt.Errorf("failed to extract nullable from variable %q: %w", varName, err)
		}

		result[varName] = nullable
	}

	return result, nil
}

// extractNullableAttribute extracts the nullable attribute from a variable block.
// Returns nil if the attribute is not specified, or a pointer to the boolean value.
func extractNullableAttribute(block *hcl.Block) (*bool, error) {
	content, _, diags := block.Body.PartialContent(variableNullableSchema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse variable block: %s", diags.Error())
	}

	nullableAttr, exists := content.Attributes["nullable"]
	if !exists {
		// nullable not specified, return nil (defaults to true in Terraform)
		return nil, nil
	}

	// Evaluate the expression to get the value
	val, diags := nullableAttr.Expr.Value(nil)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to evaluate nullable: %s", diags.Error())
	}

	if val.Type() != cty.Bool {
		return nil, fmt.Errorf("nullable must be a boolean, got %s", val.Type().FriendlyName())
	}

	boolVal := val.True()
	return &boolVal, nil
}

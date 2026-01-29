package loader

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/jokarl/tfbreak-core/internal/types"
	"github.com/zclconf/go-cty/cty"
)

// variableWithValidationSchema defines the schema for extracting variable blocks with validations
var variableWithValidationSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "variable",
			LabelNames: []string{"name"},
		},
	},
}

// ValidationMap maps variable names to their validation blocks
type ValidationMap map[string][]types.ValidationBlock

// parseValidationBlocks parses all validation blocks from variable definitions
// in the given directory. Returns a map of variable name to validation blocks.
func parseValidationBlocks(dir string) (ValidationMap, error) {
	result := make(ValidationMap)

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
		fileValidations, err := parseValidationsFromFile(parser, filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
		}

		// Merge results
		for name, validations := range fileValidations {
			result[name] = validations
		}
	}

	return result, nil
}

// parseValidationsFromFile parses validation blocks from variable blocks in a single file
func parseValidationsFromFile(parser *hclparse.Parser, filePath string) (ValidationMap, error) {
	file, diags := parser.ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL parse error: %s", diags.Error())
	}

	content, _, diags := file.Body.PartialContent(variableWithValidationSchema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to extract variable blocks: %s", diags.Error())
	}

	// Read file content for extracting raw expressions
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	result := make(ValidationMap)

	for _, block := range content.Blocks {
		if block.Type != "variable" {
			continue
		}

		if len(block.Labels) < 1 {
			continue
		}

		varName := block.Labels[0]
		validations, err := extractValidationBlocks(block, fileContent)
		if err != nil {
			return nil, fmt.Errorf("failed to extract validations from variable %q: %w", varName, err)
		}

		result[varName] = validations
	}

	return result, nil
}

// extractValidationBlocks extracts all validation blocks from a variable block
func extractValidationBlocks(block *hcl.Block, fileContent []byte) ([]types.ValidationBlock, error) {
	// Use hclsyntax to get the body with blocks
	syntaxBody, ok := block.Body.(*hclsyntax.Body)
	if !ok {
		// Not a syntax body, can't extract validation blocks
		return nil, nil
	}

	var validations []types.ValidationBlock

	for _, innerBlock := range syntaxBody.Blocks {
		if innerBlock.Type != "validation" {
			continue
		}

		validation, err := parseValidationBlock(innerBlock, fileContent)
		if err != nil {
			return nil, err
		}

		validations = append(validations, validation)
	}

	return validations, nil
}

// parseValidationBlock parses a single validation block
func parseValidationBlock(block *hclsyntax.Block, fileContent []byte) (types.ValidationBlock, error) {
	var validation types.ValidationBlock

	for _, attr := range block.Body.Attributes {
		switch attr.Name {
		case "condition":
			// Extract the raw condition expression from source
			validation.Condition = extractExpressionSource(attr.Expr, fileContent)
		case "error_message":
			// Try to evaluate error_message as a string literal
			val, diags := attr.Expr.Value(nil)
			if !diags.HasErrors() && val.Type() == cty.String {
				validation.ErrorMessage = val.AsString()
			} else {
				// Fall back to raw source
				validation.ErrorMessage = extractExpressionSource(attr.Expr, fileContent)
			}
		}
	}

	return validation, nil
}

// extractExpressionSource extracts the raw source code of an expression
func extractExpressionSource(expr hcl.Expression, fileContent []byte) string {
	rng := expr.Range()
	start := rng.Start.Byte
	end := rng.End.Byte

	if start >= len(fileContent) || end > len(fileContent) || start >= end {
		return ""
	}

	return string(bytes.TrimSpace(fileContent[start:end]))
}

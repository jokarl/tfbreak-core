package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/jokarl/tfbreak-core/internal/types"
)

// movedBlockSchema defines the schema for a moved block
var movedBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "moved",
		},
	},
}

// movedContentSchema defines the schema for the content of a moved block
var movedContentSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "from", Required: true},
		{Name: "to", Required: true},
	},
}

// parseMovedBlocks parses all moved blocks from .tf files in the given directory
func parseMovedBlocks(dir string) ([]*types.MovedBlock, error) {
	var movedBlocks []*types.MovedBlock

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
		blocks, err := parseMovedBlocksFromFile(parser, filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
		}
		movedBlocks = append(movedBlocks, blocks...)
	}

	return movedBlocks, nil
}

// parseMovedBlocksFromFile parses moved blocks from a single .tf file
func parseMovedBlocksFromFile(parser *hclparse.Parser, filePath string) ([]*types.MovedBlock, error) {
	file, diags := parser.ParseHCLFile(filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL parse error: %s", diags.Error())
	}

	content, _, diags := file.Body.PartialContent(movedBlockSchema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to extract moved blocks: %s", diags.Error())
	}

	var movedBlocks []*types.MovedBlock

	for _, block := range content.Blocks {
		if block.Type != "moved" {
			continue
		}

		movedBlock, err := parseMovedBlock(block)
		if err != nil {
			return nil, fmt.Errorf("invalid moved block at %s:%d: %w",
				block.DefRange.Filename, block.DefRange.Start.Line, err)
		}
		movedBlocks = append(movedBlocks, movedBlock)
	}

	return movedBlocks, nil
}

// parseMovedBlock parses a single moved block
func parseMovedBlock(block *hcl.Block) (*types.MovedBlock, error) {
	content, diags := block.Body.Content(movedContentSchema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("invalid moved block content: %s", diags.Error())
	}

	fromAttr, exists := content.Attributes["from"]
	if !exists {
		return nil, fmt.Errorf("missing 'from' attribute")
	}

	toAttr, exists := content.Attributes["to"]
	if !exists {
		return nil, fmt.Errorf("missing 'to' attribute")
	}

	// Extract the address from the expression
	// Moved blocks use traversal expressions like: aws_s3_bucket.old or module.foo
	fromAddr, err := extractTraversalAddress(fromAttr.Expr)
	if err != nil {
		return nil, fmt.Errorf("invalid 'from' address: %w", err)
	}

	toAddr, err := extractTraversalAddress(toAttr.Expr)
	if err != nil {
		return nil, fmt.Errorf("invalid 'to' address: %w", err)
	}

	return &types.MovedBlock{
		From: fromAddr,
		To:   toAddr,
		DeclRange: types.FileRange{
			Filename: block.DefRange.Filename,
			Line:     block.DefRange.Start.Line,
		},
	}, nil
}

// extractTraversalAddress extracts a resource or module address from an HCL expression
func extractTraversalAddress(expr hcl.Expression) (string, error) {
	traversal, diags := hcl.AbsTraversalForExpr(expr)
	if diags.HasErrors() {
		return "", fmt.Errorf("expression is not a valid address: %s", diags.Error())
	}

	if len(traversal) < 2 {
		return "", fmt.Errorf("address must have at least two parts (e.g., type.name or module.name)")
	}

	// Build the address string from the traversal
	var parts []string
	for _, step := range traversal {
		switch s := step.(type) {
		case hcl.TraverseRoot:
			parts = append(parts, s.Name)
		case hcl.TraverseAttr:
			parts = append(parts, s.Name)
		default:
			return "", fmt.Errorf("unsupported traversal type in address")
		}
	}

	return strings.Join(parts, "."), nil
}

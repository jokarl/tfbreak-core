package plugin

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/jokarl/tfbreak-plugin-sdk/hclext"
	"github.com/jokarl/tfbreak-plugin-sdk/tflint"
)

// Runner implements tflint.Runner for providing config access to plugins.
// This is the host-side implementation that provides old/new configurations
// to plugin rules during execution.
type Runner struct {
	oldFiles map[string]*hcl.File
	newFiles map[string]*hcl.File
	// Issues contains all issues emitted during rule execution.
	Issues Issues
}

// Ensure Runner implements tflint.Runner at compile time.
var _ tflint.Runner = (*Runner)(nil)

// NewRunner creates a new Runner with the provided old and new HCL files.
func NewRunner(oldFiles, newFiles map[string]*hcl.File) *Runner {
	return &Runner{
		oldFiles: oldFiles,
		newFiles: newFiles,
		Issues:   make(Issues, 0),
	}
}

// NewRunnerFromContent creates a Runner by parsing HCL content from string maps.
// This is useful for testing or when files are provided as strings.
func NewRunnerFromContent(oldContent, newContent map[string]string) (*Runner, error) {
	parser := hclparse.NewParser()

	oldFiles := make(map[string]*hcl.File)
	for name, content := range oldContent {
		file, diags := parser.ParseHCL([]byte(content), name)
		if diags.HasErrors() {
			return nil, diags
		}
		oldFiles[name] = file
	}

	newFiles := make(map[string]*hcl.File)
	for name, content := range newContent {
		file, diags := parser.ParseHCL([]byte(content), name)
		if diags.HasErrors() {
			return nil, diags
		}
		newFiles[name] = file
	}

	return NewRunner(oldFiles, newFiles), nil
}

// GetOldModuleContent retrieves module content from old files.
func (r *Runner) GetOldModuleContent(schema *hclext.BodySchema, _ *tflint.GetModuleContentOption) (*hclext.BodyContent, error) {
	return r.getModuleContent(r.oldFiles, schema)
}

// GetNewModuleContent retrieves module content from new files.
func (r *Runner) GetNewModuleContent(schema *hclext.BodySchema, _ *tflint.GetModuleContentOption) (*hclext.BodyContent, error) {
	return r.getModuleContent(r.newFiles, schema)
}

// GetOldResourceContent retrieves resources of a specific type from old files.
func (r *Runner) GetOldResourceContent(resourceType string, schema *hclext.BodySchema, _ *tflint.GetModuleContentOption) (*hclext.BodyContent, error) {
	return r.getResourceContent(r.oldFiles, resourceType, schema)
}

// GetNewResourceContent retrieves resources of a specific type from new files.
func (r *Runner) GetNewResourceContent(resourceType string, schema *hclext.BodySchema, _ *tflint.GetModuleContentOption) (*hclext.BodyContent, error) {
	return r.getResourceContent(r.newFiles, resourceType, schema)
}

// EmitIssue records an issue from a plugin rule.
func (r *Runner) EmitIssue(rule tflint.Rule, message string, issueRange hcl.Range) error {
	r.Issues = append(r.Issues, Issue{
		Rule:    rule,
		Message: message,
		Range:   issueRange,
	})
	return nil
}

// DecodeRuleConfig decodes rule-specific configuration.
// This is a stub implementation that returns nil (no config).
// Full implementation will be added when gRPC communication is implemented.
func (r *Runner) DecodeRuleConfig(_ string, _ any) error {
	return nil
}

// getModuleContent extracts content from files using the schema.
func (r *Runner) getModuleContent(files map[string]*hcl.File, schema *hclext.BodySchema) (*hclext.BodyContent, error) {
	content := &hclext.BodyContent{
		Attributes: make(map[string]*hclext.Attribute),
		Blocks:     make([]*hclext.Block, 0),
	}

	hclSchema := hclext.ToHCLBodySchema(schema)

	for _, file := range files {
		bodyContent, _, diags := file.Body.PartialContent(hclSchema)
		if diags.HasErrors() {
			return nil, diags
		}

		// Merge attributes
		for name, attr := range bodyContent.Attributes {
			content.Attributes[name] = hclext.FromHCLAttribute(attr)
		}

		// Append blocks
		for _, block := range bodyContent.Blocks {
			b := hclext.FromHCLBlock(block)
			// Process nested body if schema specifies it
			if schema != nil {
				for _, bs := range schema.Blocks {
					if bs.Type == block.Type && bs.Body != nil {
						nestedContent, _ := r.extractBlockContent(block.Body, bs.Body)
						b.Body = nestedContent
					}
				}
			}
			content.Blocks = append(content.Blocks, b)
		}
	}

	return content, nil
}

// getResourceContent extracts resources of a specific type.
func (r *Runner) getResourceContent(files map[string]*hcl.File, resourceType string, bodySchema *hclext.BodySchema) (*hclext.BodyContent, error) {
	// Create a schema that looks for resource blocks
	resourceSchema := &hclext.BodySchema{
		Blocks: []hclext.BlockSchema{
			{
				Type:       "resource",
				LabelNames: []string{"type", "name"},
				Body:       bodySchema,
			},
		},
	}

	allContent, err := r.getModuleContent(files, resourceSchema)
	if err != nil {
		return nil, err
	}

	// Filter to only the requested resource type
	result := &hclext.BodyContent{
		Attributes: make(map[string]*hclext.Attribute),
		Blocks:     make([]*hclext.Block, 0),
	}

	for _, block := range allContent.Blocks {
		if block.Type == "resource" && len(block.Labels) >= 1 && block.Labels[0] == resourceType {
			result.Blocks = append(result.Blocks, block)
		}
	}

	return result, nil
}

// extractBlockContent extracts nested block content recursively.
func (r *Runner) extractBlockContent(body hcl.Body, schema *hclext.BodySchema) (*hclext.BodyContent, error) {
	if body == nil || schema == nil {
		return nil, nil
	}

	hclSchema := hclext.ToHCLBodySchema(schema)
	bodyContent, _, diags := body.PartialContent(hclSchema)
	if diags.HasErrors() {
		return nil, diags
	}

	content := hclext.FromHCLBodyContent(bodyContent)

	// Recursively process nested blocks
	for i, block := range content.Blocks {
		for _, bs := range schema.Blocks {
			if bs.Type == block.Type && bs.Body != nil {
				// Find the original HCL block to get its body
				for _, hclBlock := range bodyContent.Blocks {
					if hclBlock.Type == block.Type && labelsMatch(hclBlock.Labels, block.Labels) {
						nestedContent, err := r.extractBlockContent(hclBlock.Body, bs.Body)
						if err != nil {
							return nil, err
						}
						content.Blocks[i].Body = nestedContent
						break
					}
				}
			}
		}
	}

	return content, nil
}

// labelsMatch checks if two label slices are equal.
func labelsMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

package annotation

import (
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// annotationPrefix is the prefix for tfbreak annotations
const annotationPrefix = "tfbreak:"

// Regex patterns for parsing annotations
var (
	// Matches: tfbreak:ignore or tfbreak:ignore-file
	directiveRe = regexp.MustCompile(`tfbreak:(ignore-file|ignore)(?:\s+(.*))?$`)

	// Matches metadata: key="value" (legacy format)
	metadataRe = regexp.MustCompile(`(\w+)="([^"]*)"`)
)

// Parser parses annotations from HCL files
type Parser struct {
	resolver RuleResolver
}

// NewParser creates a new Parser with the given resolver
func NewParser(resolver RuleResolver) *Parser {
	if resolver == nil {
		resolver = DefaultResolver{}
	}
	return &Parser{resolver: resolver}
}

// defaultParser is used for backward compatibility
var defaultParser = NewParser(nil)

// ParseFile parses all annotations from an HCL file using the default parser
func ParseFile(filename string, src []byte) ([]*Annotation, error) {
	return defaultParser.ParseFile(filename, src)
}

// ParseFile parses all annotations from an HCL file
func (p *Parser) ParseFile(filename string, src []byte) ([]*Annotation, error) {
	tokens, diags := hclsyntax.LexConfig(src, filename, hcl.InitialPos)
	if diags.HasErrors() {
		// Still try to parse what we can
		_ = diags
	}

	var annotations []*Annotation

	for _, token := range tokens {
		if token.Type != hclsyntax.TokenComment {
			continue
		}

		// Get comment text (strip # or // prefix)
		text := string(token.Bytes)
		text = strings.TrimPrefix(text, "#")
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimSpace(text)

		// Check if it's a tfbreak annotation
		if !strings.HasPrefix(text, annotationPrefix) {
			continue
		}

		ann, err := p.parseAnnotation(text, filename, token.Range.Start.Line)
		if err != nil {
			// Skip invalid annotations (could log a warning)
			continue
		}

		annotations = append(annotations, ann)
	}

	return annotations, nil
}

// parseAnnotation parses a single annotation from comment text
func (p *Parser) parseAnnotation(text, filename string, line int) (*Annotation, error) {
	matches := directiveRe.FindStringSubmatch(text)
	if matches == nil {
		return nil, nil
	}

	directive := matches[1]
	rest := ""
	if len(matches) > 2 {
		rest = strings.TrimSpace(matches[2])
	}

	ann := &Annotation{
		Filename: filename,
		Line:     line,
	}

	// Set scope based on directive
	if directive == "ignore-file" {
		ann.Scope = ScopeFile
	} else {
		ann.Scope = ScopeBlock
	}

	// Parse rules and reason if present
	if rest != "" {
		// Check for trailing comment as reason (tflint style: # tfbreak:ignore rule # reason)
		// Split on # but only if there's content after the rules
		var rulesPart, reasonPart string
		if before, after, found := strings.Cut(rest, "#"); found {
			rulesPart = strings.TrimSpace(before)
			reasonPart = strings.TrimSpace(after)
		} else {
			rulesPart = rest
		}

		// Parse rules (can be 'all', rule IDs like BC001, or rule names like required-input-added)
		if rulesPart != "" {
			ann.RuleIDs = p.parseRuleSpecs(rulesPart)
		}

		// Set reason from trailing comment if present
		if reasonPart != "" && ann.Reason == "" {
			ann.Reason = reasonPart
		}

		// Also check for legacy metadata format in rulesPart (for backward compatibility)
		metaMatches := metadataRe.FindAllStringSubmatch(rulesPart, -1)
		for _, m := range metaMatches {
			key := m[1]
			value := m[2]

			switch key {
			case "reason":
				ann.Reason = value
			case "ticket":
				ann.Ticket = value
			case "expires":
				t, err := time.Parse("2006-01-02", value)
				if err == nil {
					ann.Expires = &t
				}
			}
		}
	}

	return ann, nil
}

// parseRuleSpecs parses a comma-separated list of rule specs (names or 'all')
// Only known rule names are resolved; unknown names are ignored
func (p *Parser) parseRuleSpecs(input string) []string {
	// First strip any metadata (key="value" pairs)
	cleaned := metadataRe.ReplaceAllString(input, "")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" {
		return nil
	}

	// Handle 'all' keyword - returns empty slice which means match all rules
	if cleaned == "all" {
		return nil
	}

	var ruleIDs []string
	for _, spec := range strings.Split(cleaned, ",") {
		spec = strings.TrimSpace(spec)
		if spec == "" || spec == "all" {
			continue
		}

		// Resolve the spec to a rule ID
		// Only known rule names are accepted; unknown names are skipped
		if id, ok := p.resolver.ResolveRuleID(spec); ok {
			ruleIDs = append(ruleIDs, id)
		}
		// Unknown rule names are silently ignored
	}

	return ruleIDs
}

// FindBlockStarts finds the starting lines of all blocks in an HCL file
// Returns a map of line number to block type (e.g., "variable", "resource")
func FindBlockStarts(filename string, src []byte) (map[int]string, error) {
	file, diags := hclsyntax.ParseConfig(src, filename, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	blockStarts := make(map[int]string)

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return blockStarts, nil
	}

	for _, block := range body.Blocks {
		blockStarts[block.Range().Start.Line] = block.Type
	}

	return blockStarts, nil
}

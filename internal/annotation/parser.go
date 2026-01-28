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

	// Matches rule IDs: BC001,BC002 or BC001, BC002
	ruleIDRe = regexp.MustCompile(`^([A-Z]{2}[0-9]{3}(?:\s*,\s*[A-Z]{2}[0-9]{3})*)`)

	// Matches metadata: key="value"
	metadataRe = regexp.MustCompile(`(\w+)="([^"]*)"`)
)

// ParseFile parses all annotations from an HCL file
func ParseFile(filename string, src []byte) ([]*Annotation, error) {
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

		ann, err := parseAnnotation(text, filename, token.Range.Start.Line)
		if err != nil {
			// Skip invalid annotations (could log a warning)
			continue
		}

		annotations = append(annotations, ann)
	}

	return annotations, nil
}

// parseAnnotation parses a single annotation from comment text
func parseAnnotation(text, filename string, line int) (*Annotation, error) {
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

	// Parse rule IDs if present
	if rest != "" {
		ruleMatches := ruleIDRe.FindStringSubmatch(rest)
		if ruleMatches != nil {
			ruleStr := ruleMatches[1]
			// Split by comma and clean up
			for _, id := range strings.Split(ruleStr, ",") {
				id = strings.TrimSpace(id)
				if id != "" {
					ann.RuleIDs = append(ann.RuleIDs, id)
				}
			}
			// Remove rule IDs from rest
			rest = strings.TrimSpace(rest[len(ruleMatches[0]):])
		}

		// Parse metadata
		metaMatches := metadataRe.FindAllStringSubmatch(rest, -1)
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

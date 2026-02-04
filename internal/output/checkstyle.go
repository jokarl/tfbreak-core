package output

import (
	"encoding/xml"
	"io"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// CheckstyleRenderer renders output in Checkstyle XML format
// This format is compatible with many CI/CD tools and code quality platforms
type CheckstyleRenderer struct{}

// checkstyleOutput is the root element for Checkstyle XML
type checkstyleOutput struct {
	XMLName xml.Name         `xml:"checkstyle"`
	Version string           `xml:"version,attr"`
	Files   []checkstyleFile `xml:"file"`
}

// checkstyleFile represents a file element in Checkstyle XML
type checkstyleFile struct {
	Name   string            `xml:"name,attr"`
	Errors []checkstyleError `xml:"error"`
}

// checkstyleError represents an error element in Checkstyle XML
type checkstyleError struct {
	Line     int    `xml:"line,attr"`
	Column   int    `xml:"column,attr"`
	Severity string `xml:"severity,attr"`
	Message  string `xml:"message,attr"`
	Source   string `xml:"source,attr"`
}

// Render writes the check result in Checkstyle XML format
func (r *CheckstyleRenderer) Render(w io.Writer, result *types.CheckResult) error {
	// Group findings by file
	fileMap := make(map[string][]checkstyleError)

	for _, f := range result.Findings {
		if f.Ignored {
			continue
		}

		// Determine location
		filename := "<unknown>"
		line := 0
		col := 0

		if f.NewLocation != nil {
			filename = f.NewLocation.Filename
			line = f.NewLocation.Line
			col = f.NewLocation.Column
		} else if f.OldLocation != nil {
			filename = f.OldLocation.Filename
			line = f.OldLocation.Line
			col = f.OldLocation.Column
		}

		// Map severity to Checkstyle severity
		severity := mapToCheckstyleSeverity(f.Severity)

		err := checkstyleError{
			Line:     line,
			Column:   col,
			Severity: severity,
			Message:  f.Message,
			Source:   "tfbreak." + f.RuleID,
		}

		fileMap[filename] = append(fileMap[filename], err)
	}

	// Build the output structure
	output := checkstyleOutput{
		Version: "1.0",
		Files:   make([]checkstyleFile, 0, len(fileMap)),
	}

	for filename, errors := range fileMap {
		output.Files = append(output.Files, checkstyleFile{
			Name:   filename,
			Errors: errors,
		})
	}

	// Write XML header
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return err
	}

	// Encode XML
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	return encoder.Encode(output)
}

// mapToCheckstyleSeverity maps tfbreak severity to Checkstyle severity
func mapToCheckstyleSeverity(s types.Severity) string {
	switch s {
	case types.SeverityError:
		return "error"
	case types.SeverityWarning:
		return "warning"
	case types.SeverityNotice:
		return "info"
	default:
		return "info"
	}
}

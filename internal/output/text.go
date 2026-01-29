package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/jokarl/tfbreak-core/internal/types"
)

// TextRenderer renders output in human-readable text format
type TextRenderer struct {
	ColorEnabled bool
}

// Render writes the check result in text format
func (r *TextRenderer) Render(w io.Writer, result *types.CheckResult) error {
	// Configure color
	if !r.ColorEnabled {
		color.NoColor = true
	}

	// Header
	fmt.Fprintf(w, "tfbreak: comparing %s -> %s\n\n", result.OldPath, result.NewPath)

	// Findings
	for _, f := range result.Findings {
		r.renderFinding(w, f)
	}

	// Separator
	fmt.Fprintln(w, strings.Repeat("-", 60))

	// Summary
	r.renderSummary(w, result)

	// Result
	r.renderResult(w, result)

	return nil
}

func (r *TextRenderer) renderFinding(w io.Writer, f *types.Finding) {
	// Severity with color
	severityStr := r.colorSeverity(f.Severity)
	fmt.Fprintf(w, "%s  %s  %s\n", severityStr, f.RuleID, f.RuleName)

	// Location
	if f.NewLocation != nil {
		fmt.Fprintf(w, "  %s:%d\n", f.NewLocation.Filename, f.NewLocation.Line)
	} else if f.OldLocation != nil {
		fmt.Fprintf(w, "  %s:%d\n", f.OldLocation.Filename, f.OldLocation.Line)
	}

	// Message
	fmt.Fprintf(w, "  %s\n", f.Message)

	// Ignored status
	if f.Ignored {
		if f.IgnoreReason != "" {
			fmt.Fprintf(w, "  [IGNORED] reason=%q\n", f.IgnoreReason)
		} else {
			fmt.Fprintln(w, "  [IGNORED]")
		}
	}

	// Remediation (if included)
	if f.Remediation != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  Remediation:")
		r.renderIndented(w, f.Remediation, "    ")
	}

	fmt.Fprintln(w)
}

// renderIndented writes text with the given prefix on each line
func (r *TextRenderer) renderIndented(w io.Writer, text, prefix string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		fmt.Fprintf(w, "%s%s\n", prefix, line)
	}
}

func (r *TextRenderer) renderSummary(w io.Writer, result *types.CheckResult) {
	parts := []string{}

	if result.Summary.Error > 0 {
		parts = append(parts, fmt.Sprintf("%d error", result.Summary.Error))
	}
	if result.Summary.Warning > 0 {
		if result.Summary.Ignored > 0 {
			parts = append(parts, fmt.Sprintf("%d warning (%d ignored)", result.Summary.Warning, result.Summary.Ignored))
		} else {
			parts = append(parts, fmt.Sprintf("%d warning", result.Summary.Warning))
		}
	}
	if result.Summary.Notice > 0 {
		parts = append(parts, fmt.Sprintf("%d notice", result.Summary.Notice))
	}

	if len(parts) == 0 {
		parts = append(parts, "no issues found")
	}

	fmt.Fprintf(w, "Summary: %s\n", strings.Join(parts, ", "))
}

func (r *TextRenderer) renderResult(w io.Writer, result *types.CheckResult) {
	if result.Result == "PASS" {
		if r.ColorEnabled {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Fprintf(w, "Result: %s\n", green("PASS"))
		} else {
			fmt.Fprintln(w, "Result: PASS")
		}
	} else {
		if r.ColorEnabled {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Fprintf(w, "Result: %s (errors detected)\n", red("FAIL"))
		} else {
			fmt.Fprintln(w, "Result: FAIL (errors detected)")
		}
	}
}

func (r *TextRenderer) colorSeverity(s types.Severity) string {
	str := s.String()
	if !r.ColorEnabled {
		return str
	}

	switch s {
	case types.SeverityError:
		return color.New(color.FgRed, color.Bold).Sprint(str)
	case types.SeverityWarning:
		return color.New(color.FgYellow).Sprint(str)
	case types.SeverityNotice:
		return color.New(color.FgCyan).Sprint(str)
	default:
		return str
	}
}

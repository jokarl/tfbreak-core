package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestTextRenderer(t *testing.T) {
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC001",
				RuleName: "required-input-added",
				Severity: types.SeverityBreaking,
				Message:  "New required variable \"foo\" has no default",
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     10,
				},
			},
		},
		Summary: types.Summary{
			Breaking: 1,
			Total:    1,
		},
		Result: "FAIL",
		FailOn: types.SeverityBreaking,
	}

	renderer := &TextRenderer{ColorEnabled: false}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()

	// Check header
	if !strings.Contains(output, "comparing /old -> /new") {
		t.Error("output should contain paths")
	}

	// Check severity
	if !strings.Contains(output, "BREAKING") {
		t.Error("output should contain severity")
	}

	// Check rule ID
	if !strings.Contains(output, "BC001") {
		t.Error("output should contain rule ID")
	}

	// Check location
	if !strings.Contains(output, "variables.tf:10") {
		t.Error("output should contain file location")
	}

	// Check message
	if !strings.Contains(output, "New required variable") {
		t.Error("output should contain message")
	}

	// Check result
	if !strings.Contains(output, "FAIL") {
		t.Error("output should contain result")
	}
}

func TestTextRendererPass(t *testing.T) {
	result := &types.CheckResult{
		OldPath:  "/old",
		NewPath:  "/new",
		Findings: []*types.Finding{},
		Summary:  types.Summary{},
		Result:   "PASS",
		FailOn:   types.SeverityBreaking,
	}

	renderer := &TextRenderer{ColorEnabled: false}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "PASS") {
		t.Error("output should contain PASS")
	}
	if !strings.Contains(output, "no issues found") {
		t.Error("output should indicate no issues")
	}
}

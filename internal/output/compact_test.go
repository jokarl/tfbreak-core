package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestCompactRenderer(t *testing.T) {
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC001",
				RuleName: "required-input-added",
				Severity: types.SeverityError,
				Message:  "New required variable \"foo\" has no default",
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     10,
					Column:   5,
				},
			},
			{
				RuleID:   "RC006",
				RuleName: "input-default-changed",
				Severity: types.SeverityWarning,
				Message:  "Default value changed for \"bar\"",
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     20,
					Column:   3,
				},
			},
		},
		Summary: types.Summary{
			Error:   1,
			Warning: 1,
			Total:   2,
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &CompactRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %s", len(lines), output)
	}

	// Check first line
	expected1 := "variables.tf:10:5: ERROR: [BC001] New required variable \"foo\" has no default"
	if lines[0] != expected1 {
		t.Errorf("line 1 = %q, want %q", lines[0], expected1)
	}

	// Check second line
	expected2 := "variables.tf:20:3: WARNING: [RC006] Default value changed for \"bar\""
	if lines[1] != expected2 {
		t.Errorf("line 2 = %q, want %q", lines[1], expected2)
	}
}

func TestCompactRenderer_IgnoredFindings(t *testing.T) {
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC001",
				RuleName: "required-input-added",
				Severity: types.SeverityError,
				Message:  "New required variable \"foo\" has no default",
				Ignored:  true, // Should be skipped
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     10,
				},
			},
			{
				RuleID:   "BC002",
				RuleName: "input-removed",
				Severity: types.SeverityError,
				Message:  "Variable \"bar\" removed",
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     20,
				},
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &CompactRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Only non-ignored finding should appear
	if len(lines) != 1 {
		t.Fatalf("expected 1 line (ignored should be skipped), got %d: %s", len(lines), output)
	}

	if !strings.Contains(lines[0], "BC002") {
		t.Errorf("expected BC002, got: %s", lines[0])
	}
}

func TestCompactRenderer_OldLocation(t *testing.T) {
	// Test that old location is used when new location is not available
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC002",
				RuleName: "input-removed",
				Severity: types.SeverityError,
				Message:  "Variable \"foo\" removed",
				OldLocation: &types.FileRange{
					Filename: "old/variables.tf",
					Line:     15,
					Column:   1,
				},
				// No NewLocation
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &CompactRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "old/variables.tf:15:1") {
		t.Errorf("expected old location, got: %s", output)
	}
}

func TestCompactRenderer_NoLocation(t *testing.T) {
	// Test finding with neither new nor old location
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC100",
				RuleName: "resource-removed-no-moved",
				Severity: types.SeverityError,
				Message:  "Resource removed without moved block",
				// No location at all
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &CompactRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()
	// Should use <unknown>:0:0 when no location is available
	if !strings.Contains(output, "<unknown>:0:0") {
		t.Errorf("expected <unknown>:0:0, got: %s", output)
	}
	if !strings.Contains(output, "BC100") {
		t.Errorf("expected BC100 rule ID, got: %s", output)
	}
}

func TestCompactRenderer_Empty(t *testing.T) {
	result := &types.CheckResult{
		OldPath:  "/old",
		NewPath:  "/new",
		Findings: []*types.Finding{},
		Result:   "PASS",
		FailOn:   types.SeverityError,
	}

	renderer := &CompactRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("expected empty output, got: %s", buf.String())
	}
}

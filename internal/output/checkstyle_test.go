package output

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestCheckstyleRenderer(t *testing.T) {
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

	renderer := &CheckstyleRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()

	// Verify XML header
	if !strings.HasPrefix(output, xml.Header) {
		t.Error("expected XML header")
	}

	// Verify valid XML
	var checkstyle checkstyleOutput
	if err := xml.Unmarshal(buf.Bytes(), &checkstyle); err != nil {
		t.Fatalf("Invalid XML: %v\nOutput:\n%s", err, output)
	}

	// Check version
	if checkstyle.Version != "1.0" {
		t.Errorf("version = %s, want 1.0", checkstyle.Version)
	}

	// Check files
	if len(checkstyle.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(checkstyle.Files))
	}

	file := checkstyle.Files[0]
	if file.Name != "variables.tf" {
		t.Errorf("file name = %s, want variables.tf", file.Name)
	}

	// Check errors
	if len(file.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(file.Errors))
	}

	// Check first error
	err1 := file.Errors[0]
	if err1.Severity != "error" {
		t.Errorf("error 1 severity = %s, want error", err1.Severity)
	}
	if err1.Source != "tfbreak.BC001" {
		t.Errorf("error 1 source = %s, want tfbreak.BC001", err1.Source)
	}
}

func TestCheckstyleRenderer_IgnoredFindings(t *testing.T) {
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC001",
				RuleName: "required-input-added",
				Severity: types.SeverityError,
				Message:  "New required variable \"foo\" has no default",
				Ignored:  true,
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     10,
				},
			},
		},
		Result: "PASS",
		FailOn: types.SeverityError,
	}

	renderer := &CheckstyleRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var checkstyle checkstyleOutput
	if err := xml.Unmarshal(buf.Bytes(), &checkstyle); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	// Ignored findings should not appear
	if len(checkstyle.Files) != 0 {
		t.Errorf("expected 0 files (ignored findings skipped), got %d", len(checkstyle.Files))
	}
}

func TestCheckstyleRenderer_MultipleFiles(t *testing.T) {
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC001",
				RuleName: "required-input-added",
				Severity: types.SeverityError,
				Message:  "Error in file1",
				NewLocation: &types.FileRange{
					Filename: "file1.tf",
					Line:     10,
				},
			},
			{
				RuleID:   "BC002",
				RuleName: "input-removed",
				Severity: types.SeverityError,
				Message:  "Error in file2",
				NewLocation: &types.FileRange{
					Filename: "file2.tf",
					Line:     5,
				},
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &CheckstyleRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var checkstyle checkstyleOutput
	if err := xml.Unmarshal(buf.Bytes(), &checkstyle); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	if len(checkstyle.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(checkstyle.Files))
	}
}

func TestCheckstyleRenderer_SeverityMapping(t *testing.T) {
	tests := []struct {
		severity types.Severity
		expected string
	}{
		{types.SeverityError, "error"},
		{types.SeverityWarning, "warning"},
		{types.SeverityNotice, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := &types.CheckResult{
				OldPath: "/old",
				NewPath: "/new",
				Findings: []*types.Finding{
					{
						RuleID:   "TEST",
						RuleName: "test",
						Severity: tt.severity,
						Message:  "test",
						NewLocation: &types.FileRange{
							Filename: "test.tf",
							Line:     1,
						},
					},
				},
				Result: "FAIL",
				FailOn: types.SeverityNotice,
			}

			renderer := &CheckstyleRenderer{}
			var buf bytes.Buffer
			err := renderer.Render(&buf, result)
			if err != nil {
				t.Fatalf("Render error: %v", err)
			}

			var checkstyle checkstyleOutput
			if err := xml.Unmarshal(buf.Bytes(), &checkstyle); err != nil {
				t.Fatalf("Invalid XML: %v", err)
			}

			if len(checkstyle.Files) != 1 || len(checkstyle.Files[0].Errors) != 1 {
				t.Fatal("expected 1 file with 1 error")
			}

			if checkstyle.Files[0].Errors[0].Severity != tt.expected {
				t.Errorf("severity = %s, want %s", checkstyle.Files[0].Errors[0].Severity, tt.expected)
			}
		})
	}
}

func TestCheckstyleRenderer_Empty(t *testing.T) {
	result := &types.CheckResult{
		OldPath:  "/old",
		NewPath:  "/new",
		Findings: []*types.Finding{},
		Result:   "PASS",
		FailOn:   types.SeverityError,
	}

	renderer := &CheckstyleRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var checkstyle checkstyleOutput
	if err := xml.Unmarshal(buf.Bytes(), &checkstyle); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	if len(checkstyle.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(checkstyle.Files))
	}
}

func TestMapToCheckstyleSeverity_AllCases(t *testing.T) {
	tests := []struct {
		severity types.Severity
		expected string
	}{
		{types.SeverityError, "error"},
		{types.SeverityWarning, "warning"},
		{types.SeverityNotice, "info"},
		{types.Severity(-1), "info"},  // Unknown defaults to info
		{types.Severity(99), "info"},  // Unknown defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := mapToCheckstyleSeverity(tt.severity)
			if got != tt.expected {
				t.Errorf("mapToCheckstyleSeverity(%v) = %q, want %q", tt.severity, got, tt.expected)
			}
		})
	}
}

func TestCheckstyleRenderer_FallbackToOldLocation(t *testing.T) {
	// Test that OldLocation is used when NewLocation is nil
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC002",
				RuleName: "input-removed",
				Severity: types.SeverityError,
				Message:  "Variable removed",
				OldLocation: &types.FileRange{
					Filename: "old_file.tf",
					Line:     15,
					Column:   3,
				},
				// No NewLocation
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &CheckstyleRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var checkstyle checkstyleOutput
	if err := xml.Unmarshal(buf.Bytes(), &checkstyle); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	if len(checkstyle.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(checkstyle.Files))
	}

	if checkstyle.Files[0].Name != "old_file.tf" {
		t.Errorf("file name = %s, want old_file.tf (should fallback to OldLocation)", checkstyle.Files[0].Name)
	}

	if checkstyle.Files[0].Errors[0].Line != 15 {
		t.Errorf("line = %d, want 15", checkstyle.Files[0].Errors[0].Line)
	}
}

func TestCheckstyleRenderer_NoLocation(t *testing.T) {
	// Test finding with no location at all
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC100",
				RuleName: "resource-removed",
				Severity: types.SeverityError,
				Message:  "Resource removed without moved block",
				// No location
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &CheckstyleRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var checkstyle checkstyleOutput
	if err := xml.Unmarshal(buf.Bytes(), &checkstyle); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	if len(checkstyle.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(checkstyle.Files))
	}

	// Should use <unknown> as filename
	if checkstyle.Files[0].Name != "<unknown>" {
		t.Errorf("file name = %s, want <unknown>", checkstyle.Files[0].Name)
	}
}

func TestCheckstyleRenderer_DeterministicFileOrder(t *testing.T) {
	// Create findings across multiple files in non-alphabetical order
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC001",
				RuleName: "test",
				Severity: types.SeverityError,
				Message:  "Error in zebra.tf",
				NewLocation: &types.FileRange{
					Filename: "zebra.tf",
					Line:     1,
				},
			},
			{
				RuleID:   "BC001",
				RuleName: "test",
				Severity: types.SeverityError,
				Message:  "Error in alpha.tf",
				NewLocation: &types.FileRange{
					Filename: "alpha.tf",
					Line:     1,
				},
			},
			{
				RuleID:   "BC001",
				RuleName: "test",
				Severity: types.SeverityError,
				Message:  "Error in middle.tf",
				NewLocation: &types.FileRange{
					Filename: "middle.tf",
					Line:     1,
				},
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &CheckstyleRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var checkstyle checkstyleOutput
	if err := xml.Unmarshal(buf.Bytes(), &checkstyle); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	// Files should be sorted alphabetically
	if len(checkstyle.Files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(checkstyle.Files))
	}

	expectedOrder := []string{"alpha.tf", "middle.tf", "zebra.tf"}
	for i, expected := range expectedOrder {
		if checkstyle.Files[i].Name != expected {
			t.Errorf("file[%d].Name = %s, want %s", i, checkstyle.Files[i].Name, expected)
		}
	}
}

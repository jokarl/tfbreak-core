package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestSARIFRenderer(t *testing.T) {
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
					Filename:  "variables.tf",
					Line:      10,
					Column:    5,
					EndLine:   10,
					EndColumn: 20,
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

	renderer := &SARIFRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	// Verify it's valid JSON
	var sarif sarifLog
	if err := json.Unmarshal(buf.Bytes(), &sarif); err != nil {
		t.Fatalf("Invalid JSON: %v\nOutput:\n%s", err, buf.String())
	}

	// Check schema and version
	if sarif.Version != "2.1.0" {
		t.Errorf("version = %s, want 2.1.0", sarif.Version)
	}
	if sarif.Schema == "" {
		t.Error("expected $schema field")
	}

	// Check runs
	if len(sarif.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(sarif.Runs))
	}

	run := sarif.Runs[0]

	// Check tool
	if run.Tool.Driver.Name != "tfbreak" {
		t.Errorf("tool name = %s, want tfbreak", run.Tool.Driver.Name)
	}

	// Check rules
	if len(run.Tool.Driver.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(run.Tool.Driver.Rules))
	}

	// Check results
	if len(run.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(run.Results))
	}

	// Check first result
	r1 := run.Results[0]
	if r1.RuleID != "BC001" {
		t.Errorf("result 1 ruleId = %s, want BC001", r1.RuleID)
	}
	if r1.Level != "error" {
		t.Errorf("result 1 level = %s, want error", r1.Level)
	}

	// Check location
	if len(r1.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(r1.Locations))
	}
	loc := r1.Locations[0]
	if loc.PhysicalLocation.ArtifactLocation.URI != "variables.tf" {
		t.Errorf("location uri = %s, want variables.tf", loc.PhysicalLocation.ArtifactLocation.URI)
	}
	if loc.PhysicalLocation.Region.StartLine != 10 {
		t.Errorf("startLine = %d, want 10", loc.PhysicalLocation.Region.StartLine)
	}
}

func TestSARIFRenderer_IgnoredFindings(t *testing.T) {
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
			{
				RuleID:   "BC002",
				RuleName: "input-removed",
				Severity: types.SeverityError,
				Message:  "Variable removed",
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     20,
				},
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &SARIFRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var sarif sarifLog
	if err := json.Unmarshal(buf.Bytes(), &sarif); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Only non-ignored findings should appear in results
	if len(sarif.Runs[0].Results) != 1 {
		t.Errorf("expected 1 result (ignored skipped), got %d", len(sarif.Runs[0].Results))
	}

	if sarif.Runs[0].Results[0].RuleID != "BC002" {
		t.Errorf("expected BC002, got %s", sarif.Runs[0].Results[0].RuleID)
	}
}

func TestSARIFRenderer_SeverityMapping(t *testing.T) {
	tests := []struct {
		severity types.Severity
		expected string
	}{
		{types.SeverityError, "error"},
		{types.SeverityWarning, "warning"},
		{types.SeverityNotice, "note"},
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

			renderer := &SARIFRenderer{}
			var buf bytes.Buffer
			err := renderer.Render(&buf, result)
			if err != nil {
				t.Fatalf("Render error: %v", err)
			}

			var sarif sarifLog
			if err := json.Unmarshal(buf.Bytes(), &sarif); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			if len(sarif.Runs[0].Results) != 1 {
				t.Fatal("expected 1 result")
			}

			if sarif.Runs[0].Results[0].Level != tt.expected {
				t.Errorf("level = %s, want %s", sarif.Runs[0].Results[0].Level, tt.expected)
			}
		})
	}
}

func TestSARIFRenderer_OldLocation(t *testing.T) {
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

	renderer := &SARIFRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var sarif sarifLog
	if err := json.Unmarshal(buf.Bytes(), &sarif); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	result1 := sarif.Runs[0].Results[0]
	if len(result1.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(result1.Locations))
	}

	loc := result1.Locations[0]
	if loc.PhysicalLocation.ArtifactLocation.URI != "old/variables.tf" {
		t.Errorf("expected old location URI, got %s", loc.PhysicalLocation.ArtifactLocation.URI)
	}
}

func TestSARIFRenderer_Empty(t *testing.T) {
	result := &types.CheckResult{
		OldPath:  "/old",
		NewPath:  "/new",
		Findings: []*types.Finding{},
		Result:   "PASS",
		FailOn:   types.SeverityError,
	}

	renderer := &SARIFRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var sarif sarifLog
	if err := json.Unmarshal(buf.Bytes(), &sarif); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(sarif.Runs) != 1 {
		t.Errorf("expected 1 run, got %d", len(sarif.Runs))
	}

	if len(sarif.Runs[0].Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(sarif.Runs[0].Results))
	}

	if len(sarif.Runs[0].Tool.Driver.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(sarif.Runs[0].Tool.Driver.Rules))
	}
}

func TestSARIFRenderer_NoLocation(t *testing.T) {
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC100",
				RuleName: "resource-removed-no-moved",
				Severity: types.SeverityError,
				Message:  "Resource removed",
				// No location
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &SARIFRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var sarif sarifLog
	if err := json.Unmarshal(buf.Bytes(), &sarif); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	result1 := sarif.Runs[0].Results[0]
	// Should have no locations when neither old nor new location is available
	if len(result1.Locations) != 0 {
		t.Errorf("expected 0 locations, got %d", len(result1.Locations))
	}
}

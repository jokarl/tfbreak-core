package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestJSONRenderer(t *testing.T) {
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
				},
			},
		},
		Summary: types.Summary{
			Error: 1,
			Total:    1,
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	// Verify it's valid JSON
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Check required fields
	if output["version"] != "1.0" {
		t.Errorf("version = %v, want 1.0", output["version"])
	}
	if output["old_path"] != "/old" {
		t.Errorf("old_path = %v, want /old", output["old_path"])
	}
	if output["new_path"] != "/new" {
		t.Errorf("new_path = %v, want /new", output["new_path"])
	}
	if output["result"] != "FAIL" {
		t.Errorf("result = %v, want FAIL", output["result"])
	}

	// Check findings array
	findings, ok := output["findings"].([]interface{})
	if !ok {
		t.Fatal("findings should be an array")
	}
	if len(findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(findings))
	}

	// Check summary
	summary, ok := output["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("summary should be an object")
	}
	if summary["error"].(float64) != 1 {
		t.Errorf("summary.error = %v, want 1", summary["error"])
	}
}

func TestJSONRenderer_WithRemediation(t *testing.T) {
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:      "BC001",
				RuleName:    "required-input-added",
				Severity:    types.SeverityError,
				Message:     "New required variable \"foo\" has no default",
				Remediation: "To fix this issue:\n1. Add a default value\n2. Update callers",
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     10,
				},
			},
		},
		Summary: types.Summary{
			Error: 1,
			Total:    1,
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	// Verify it's valid JSON
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Check findings have remediation
	findings, ok := output["findings"].([]interface{})
	if !ok {
		t.Fatal("findings should be an array")
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	finding := findings[0].(map[string]interface{})
	remediation, ok := finding["remediation"]
	if !ok {
		t.Error("finding should have remediation field")
	}
	if remediation != "To fix this issue:\n1. Add a default value\n2. Update callers" {
		t.Errorf("unexpected remediation: %v", remediation)
	}
}

func TestJSONRenderer_NoRemediation(t *testing.T) {
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC001",
				RuleName: "required-input-added",
				Severity: types.SeverityError,
				Message:  "New required variable \"foo\" has no default",
				// No Remediation field set
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     10,
				},
			},
		},
		Summary: types.Summary{
			Error: 1,
			Total:    1,
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	// Verify it's valid JSON
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Check findings do NOT have remediation
	findings, ok := output["findings"].([]interface{})
	if !ok {
		t.Fatal("findings should be an array")
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	finding := findings[0].(map[string]interface{})
	if _, ok := finding["remediation"]; ok {
		t.Error("finding should NOT have remediation field when not set")
	}
}

func TestJSONRendererEmpty(t *testing.T) {
	result := &types.CheckResult{
		OldPath:  "/old",
		NewPath:  "/new",
		Findings: []*types.Finding{},
		Summary:  types.Summary{},
		Result:   "PASS",
		FailOn:   types.SeverityError,
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	// Verify it's valid JSON
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if output["result"] != "PASS" {
		t.Errorf("result = %v, want PASS", output["result"])
	}
}

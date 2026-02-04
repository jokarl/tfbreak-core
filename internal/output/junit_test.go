package output

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestJUnitRenderer(t *testing.T) {
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
				RuleID:   "BC001",
				RuleName: "required-input-added",
				Severity: types.SeverityError,
				Message:  "New required variable \"bar\" has no default",
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     20,
				},
			},
		},
		Summary: types.Summary{
			Error: 2,
			Total: 2,
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &JUnitRenderer{}
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
	var testSuites junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &testSuites); err != nil {
		t.Fatalf("Invalid XML: %v\nOutput:\n%s", err, output)
	}

	// Check root element
	if testSuites.Name != "tfbreak" {
		t.Errorf("testsuites name = %s, want tfbreak", testSuites.Name)
	}

	if testSuites.Tests != 2 {
		t.Errorf("tests = %d, want 2", testSuites.Tests)
	}

	// Check test suites
	if len(testSuites.TestSuites) != 1 {
		t.Fatalf("expected 1 test suite, got %d", len(testSuites.TestSuites))
	}

	suite := testSuites.TestSuites[0]
	if suite.Name != "tfbreak.BC001" {
		t.Errorf("suite name = %s, want tfbreak.BC001", suite.Name)
	}

	if suite.Tests != 2 {
		t.Errorf("suite tests = %d, want 2", suite.Tests)
	}

	if suite.Failures != 2 {
		t.Errorf("suite failures = %d, want 2", suite.Failures)
	}

	// Check test cases
	if len(suite.TestCases) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(suite.TestCases))
	}

	tc := suite.TestCases[0]
	if tc.Failure == nil {
		t.Error("expected failure element")
	}
	if tc.Failure.Type != "ERROR" {
		t.Errorf("failure type = %s, want ERROR", tc.Failure.Type)
	}
}

func TestJUnitRenderer_IgnoredAsSkipped(t *testing.T) {
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:       "BC001",
				RuleName:     "required-input-added",
				Severity:     types.SeverityError,
				Message:      "New required variable \"foo\" has no default",
				Ignored:      true,
				IgnoreReason: "Known issue",
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     10,
				},
			},
		},
		Result: "PASS",
		FailOn: types.SeverityError,
	}

	renderer := &JUnitRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var testSuites junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &testSuites); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	if len(testSuites.TestSuites) != 1 {
		t.Fatalf("expected 1 test suite, got %d", len(testSuites.TestSuites))
	}

	suite := testSuites.TestSuites[0]
	if suite.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", suite.Skipped)
	}

	if len(suite.TestCases) != 1 {
		t.Fatalf("expected 1 test case, got %d", len(suite.TestCases))
	}

	tc := suite.TestCases[0]
	if tc.Skipped == nil {
		t.Error("expected skipped element")
	}
	if tc.Skipped.Message != "Known issue" {
		t.Errorf("skipped message = %s, want 'Known issue'", tc.Skipped.Message)
	}
	if tc.Failure != nil {
		t.Error("should not have failure when skipped")
	}
}

func TestJUnitRenderer_MultipleRules(t *testing.T) {
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "BC001",
				RuleName: "required-input-added",
				Severity: types.SeverityError,
				Message:  "BC001 finding",
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     10,
				},
			},
			{
				RuleID:   "BC002",
				RuleName: "input-removed",
				Severity: types.SeverityError,
				Message:  "BC002 finding",
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     20,
				},
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &JUnitRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var testSuites junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &testSuites); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	// Each rule should have its own test suite
	if len(testSuites.TestSuites) != 2 {
		t.Errorf("expected 2 test suites, got %d", len(testSuites.TestSuites))
	}

	if testSuites.Tests != 2 {
		t.Errorf("total tests = %d, want 2", testSuites.Tests)
	}
}

func TestJUnitRenderer_Empty(t *testing.T) {
	result := &types.CheckResult{
		OldPath:  "/old",
		NewPath:  "/new",
		Findings: []*types.Finding{},
		Result:   "PASS",
		FailOn:   types.SeverityError,
	}

	renderer := &JUnitRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var testSuites junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &testSuites); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	// Should have a passing test
	if testSuites.Tests != 1 {
		t.Errorf("expected 1 test (passing), got %d", testSuites.Tests)
	}

	if testSuites.Failures != 0 {
		t.Errorf("expected 0 failures, got %d", testSuites.Failures)
	}

	if len(testSuites.TestSuites) != 1 {
		t.Fatalf("expected 1 test suite, got %d", len(testSuites.TestSuites))
	}

	suite := testSuites.TestSuites[0]
	if suite.Name != "tfbreak" {
		t.Errorf("suite name = %s, want tfbreak", suite.Name)
	}
}

func TestJUnitRenderer_WithRemediation(t *testing.T) {
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:      "BC001",
				RuleName:    "required-input-added",
				Severity:    types.SeverityError,
				Message:     "Error message",
				Detail:      "Some detail",
				Remediation: "Fix by adding default",
				NewLocation: &types.FileRange{
					Filename: "variables.tf",
					Line:     10,
				},
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &JUnitRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var testSuites junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &testSuites); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	tc := testSuites.TestSuites[0].TestCases[0]
	if tc.Failure == nil {
		t.Fatal("expected failure element")
	}

	// Failure content should include detail and remediation
	if !strings.Contains(tc.Failure.Content, "Some detail") {
		t.Error("expected detail in failure content")
	}
	if !strings.Contains(tc.Failure.Content, "Fix by adding default") {
		t.Error("expected remediation in failure content")
	}
}

func TestJUnitRenderer_DeterministicSuiteOrder(t *testing.T) {
	// Create findings with multiple rules in non-alphabetical order
	result := &types.CheckResult{
		OldPath: "/old",
		NewPath: "/new",
		Findings: []*types.Finding{
			{
				RuleID:   "RC099",
				RuleName: "rule-z",
				Severity: types.SeverityWarning,
				Message:  "Warning",
				NewLocation: &types.FileRange{
					Filename: "test.tf",
					Line:     1,
				},
			},
			{
				RuleID:   "BC001",
				RuleName: "rule-a",
				Severity: types.SeverityError,
				Message:  "Error",
				NewLocation: &types.FileRange{
					Filename: "test.tf",
					Line:     2,
				},
			},
			{
				RuleID:   "MC050",
				RuleName: "rule-m",
				Severity: types.SeverityError,
				Message:  "Error",
				NewLocation: &types.FileRange{
					Filename: "test.tf",
					Line:     3,
				},
			},
		},
		Result: "FAIL",
		FailOn: types.SeverityError,
	}

	renderer := &JUnitRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, result)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var testSuites junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &testSuites); err != nil {
		t.Fatalf("Invalid XML: %v", err)
	}

	// Test suites should be sorted by rule ID (BC001, MC050, RC099)
	if len(testSuites.TestSuites) != 3 {
		t.Fatalf("expected 3 test suites, got %d", len(testSuites.TestSuites))
	}

	expectedOrder := []string{"tfbreak.BC001", "tfbreak.MC050", "tfbreak.RC099"}
	for i, expected := range expectedOrder {
		if testSuites.TestSuites[i].Name != expected {
			t.Errorf("suite[%d].Name = %s, want %s", i, testSuites.TestSuites[i].Name, expected)
		}
	}
}

package types

import "testing"

func TestNewFinding(t *testing.T) {
	f := NewFinding("BC001", "required-input-added", SeverityError, "test message")

	if f.RuleID != "BC001" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "BC001")
	}
	if f.RuleName != "required-input-added" {
		t.Errorf("RuleName = %q, want %q", f.RuleName, "required-input-added")
	}
	if f.Severity != SeverityError {
		t.Errorf("Severity = %v, want %v", f.Severity, SeverityError)
	}
	if f.Message != "test message" {
		t.Errorf("Message = %q, want %q", f.Message, "test message")
	}
}

func TestFindingChainedSetters(t *testing.T) {
	oldLoc := &FileRange{Filename: "old.tf", Line: 1}
	newLoc := &FileRange{Filename: "new.tf", Line: 2}

	f := NewFinding("BC001", "test-rule", SeverityWarning, "msg").
		WithDetail("detailed info").
		WithOldLocation(oldLoc).
		WithNewLocation(newLoc).
		WithMetadata("key1", "value1").
		WithMetadata("key2", "value2").
		WithRemediation("fix it")

	if f.Detail != "detailed info" {
		t.Errorf("Detail = %q, want %q", f.Detail, "detailed info")
	}
	if f.OldLocation != oldLoc {
		t.Errorf("OldLocation = %v, want %v", f.OldLocation, oldLoc)
	}
	if f.NewLocation != newLoc {
		t.Errorf("NewLocation = %v, want %v", f.NewLocation, newLoc)
	}
	if f.Metadata["key1"] != "value1" {
		t.Errorf("Metadata[key1] = %q, want %q", f.Metadata["key1"], "value1")
	}
	if f.Metadata["key2"] != "value2" {
		t.Errorf("Metadata[key2] = %q, want %q", f.Metadata["key2"], "value2")
	}
	if f.Remediation != "fix it" {
		t.Errorf("Remediation = %q, want %q", f.Remediation, "fix it")
	}
}

func TestNewCheckResult(t *testing.T) {
	r := NewCheckResult("/old/path", "/new/path", SeverityWarning)

	if r.OldPath != "/old/path" {
		t.Errorf("OldPath = %q, want %q", r.OldPath, "/old/path")
	}
	if r.NewPath != "/new/path" {
		t.Errorf("NewPath = %q, want %q", r.NewPath, "/new/path")
	}
	if r.FailOn != SeverityWarning {
		t.Errorf("FailOn = %v, want %v", r.FailOn, SeverityWarning)
	}
	if r.Findings == nil {
		t.Error("Findings slice is nil")
	}
}

func TestCheckResultAddFinding(t *testing.T) {
	r := NewCheckResult("/old", "/new", SeverityError)
	f := NewFinding("BC001", "test", SeverityError, "msg")

	r.AddFinding(f)

	if len(r.Findings) != 1 {
		t.Errorf("len(Findings) = %d, want 1", len(r.Findings))
	}
	if r.Findings[0] != f {
		t.Error("Finding not added correctly")
	}
}

func TestCheckResultCompute(t *testing.T) {
	tests := []struct {
		name       string
		findings   []*Finding
		failOn     Severity
		wantResult string
		wantSummary Summary
	}{
		{
			name:       "no findings passes",
			findings:   nil,
			failOn:     SeverityError,
			wantResult: "PASS",
			wantSummary: Summary{Total: 0},
		},
		{
			name: "error with error threshold fails",
			findings: []*Finding{
				{RuleID: "BC001", Severity: SeverityError},
			},
			failOn:     SeverityError,
			wantResult: "FAIL",
			wantSummary: Summary{Error: 1, Total: 1},
		},
		{
			name: "warning with error threshold passes",
			findings: []*Finding{
				{RuleID: "BC001", Severity: SeverityWarning},
			},
			failOn:     SeverityError,
			wantResult: "PASS",
			wantSummary: Summary{Warning: 1, Total: 1},
		},
		{
			name: "warning with warning threshold fails",
			findings: []*Finding{
				{RuleID: "BC001", Severity: SeverityWarning},
			},
			failOn:     SeverityWarning,
			wantResult: "FAIL",
			wantSummary: Summary{Warning: 1, Total: 1},
		},
		{
			name: "notice with warning threshold passes",
			findings: []*Finding{
				{RuleID: "BC001", Severity: SeverityNotice},
			},
			failOn:     SeverityWarning,
			wantResult: "PASS",
			wantSummary: Summary{Notice: 1, Total: 1},
		},
		{
			name: "notice with notice threshold fails",
			findings: []*Finding{
				{RuleID: "BC001", Severity: SeverityNotice},
			},
			failOn:     SeverityNotice,
			wantResult: "FAIL",
			wantSummary: Summary{Notice: 1, Total: 1},
		},
		{
			name: "ignored findings not counted in severity",
			findings: []*Finding{
				{RuleID: "BC001", Severity: SeverityError, Ignored: true},
			},
			failOn:     SeverityError,
			wantResult: "PASS",
			wantSummary: Summary{Ignored: 1, Total: 1},
		},
		{
			name: "mixed findings with error threshold",
			findings: []*Finding{
				{RuleID: "BC001", Severity: SeverityError},
				{RuleID: "BC002", Severity: SeverityWarning},
				{RuleID: "BC003", Severity: SeverityNotice},
				{RuleID: "BC004", Severity: SeverityError, Ignored: true},
			},
			failOn:     SeverityError,
			wantResult: "FAIL",
			wantSummary: Summary{Error: 1, Warning: 1, Notice: 1, Ignored: 1, Total: 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewCheckResult("/old", "/new", tt.failOn)
			for _, f := range tt.findings {
				r.AddFinding(f)
			}
			r.Compute()

			if r.Result != tt.wantResult {
				t.Errorf("Result = %q, want %q", r.Result, tt.wantResult)
			}
			if r.Summary != tt.wantSummary {
				t.Errorf("Summary = %+v, want %+v", r.Summary, tt.wantSummary)
			}
		})
	}
}

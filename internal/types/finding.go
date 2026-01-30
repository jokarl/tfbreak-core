package types

// Finding represents a single rule violation or observation
type Finding struct {
	// RuleID is the unique identifier for the rule (e.g., "BC001")
	RuleID string `json:"rule_id"`

	// RuleName is the human-readable rule name (e.g., "required-input-added")
	RuleName string `json:"rule_name"`

	// Severity is the severity level of this finding
	Severity Severity `json:"severity"`

	// Message is a short description of the finding
	Message string `json:"message"`

	// Detail provides additional context about the finding
	Detail string `json:"detail,omitempty"`

	// OldLocation is the source location in the old config (nil if not applicable)
	OldLocation *FileRange `json:"old_location,omitempty"`

	// NewLocation is the source location in the new config (nil if not applicable)
	NewLocation *FileRange `json:"new_location,omitempty"`

	// Ignored indicates if this finding was suppressed by an annotation
	Ignored bool `json:"ignored"`

	// IgnoreReason is the reason provided in the ignore annotation
	IgnoreReason string `json:"ignore_reason,omitempty"`

	// Metadata contains rule-specific metadata for advanced processing
	// Used by rename detection rules to store old/new names for suppression logic
	Metadata map[string]string `json:"metadata,omitempty"`

	// Remediation provides guidance on how to fix this issue
	// Only populated when --include-remediation flag is set
	Remediation string `json:"remediation,omitempty"`
}

// NewFinding creates a new Finding with the given parameters
func NewFinding(ruleID, ruleName string, severity Severity, message string) *Finding {
	return &Finding{
		RuleID:   ruleID,
		RuleName: ruleName,
		Severity: severity,
		Message:  message,
	}
}

// WithDetail sets the detail field and returns the finding for chaining
func (f *Finding) WithDetail(detail string) *Finding {
	f.Detail = detail
	return f
}

// WithOldLocation sets the old location and returns the finding for chaining
func (f *Finding) WithOldLocation(loc *FileRange) *Finding {
	f.OldLocation = loc
	return f
}

// WithNewLocation sets the new location and returns the finding for chaining
func (f *Finding) WithNewLocation(loc *FileRange) *Finding {
	f.NewLocation = loc
	return f
}

// WithMetadata sets metadata and returns the finding for chaining
func (f *Finding) WithMetadata(key, value string) *Finding {
	if f.Metadata == nil {
		f.Metadata = make(map[string]string)
	}
	f.Metadata[key] = value
	return f
}

// WithRemediation sets the remediation text and returns the finding for chaining
func (f *Finding) WithRemediation(remediation string) *Finding {
	f.Remediation = remediation
	return f
}

// CheckResult represents the result of running a check
type CheckResult struct {
	// OldPath is the path to the old configuration
	OldPath string `json:"old_path"`

	// NewPath is the path to the new configuration
	NewPath string `json:"new_path"`

	// Findings is the list of all findings
	Findings []*Finding `json:"findings"`

	// Summary contains counts by severity
	Summary Summary `json:"summary"`

	// Result is PASS or FAIL based on the policy
	Result string `json:"result"`

	// FailOn is the severity threshold used for the result
	FailOn Severity `json:"fail_on"`
}

// Summary contains counts of findings by severity
type Summary struct {
	Error   int `json:"error"`
	Warning int `json:"warning"`
	Notice  int `json:"notice"`
	Ignored int `json:"ignored"`
	Total   int `json:"total"`
}

// NewCheckResult creates a new CheckResult
func NewCheckResult(oldPath, newPath string, failOn Severity) *CheckResult {
	return &CheckResult{
		OldPath:  oldPath,
		NewPath:  newPath,
		Findings: make([]*Finding, 0),
		FailOn:   failOn,
	}
}

// AddFinding adds a finding to the result
func (r *CheckResult) AddFinding(f *Finding) {
	r.Findings = append(r.Findings, f)
}

// Compute calculates the summary and result
func (r *CheckResult) Compute() {
	r.Summary = Summary{}
	for _, f := range r.Findings {
		if f.Ignored {
			r.Summary.Ignored++
			continue
		}
		switch f.Severity {
		case SeverityError:
			r.Summary.Error++
		case SeverityWarning:
			r.Summary.Warning++
		case SeverityNotice:
			r.Summary.Notice++
		}
	}
	r.Summary.Total = len(r.Findings)

	// Determine pass/fail based on policy
	failed := false
	switch r.FailOn {
	case SeverityError:
		failed = r.Summary.Error > 0
	case SeverityWarning:
		failed = r.Summary.Error > 0 || r.Summary.Warning > 0
	case SeverityNotice:
		failed = r.Summary.Error > 0 || r.Summary.Warning > 0 || r.Summary.Notice > 0
	}

	if failed {
		r.Result = "FAIL"
	} else {
		r.Result = "PASS"
	}
}

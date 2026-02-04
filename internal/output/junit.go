package output

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// JUnitRenderer renders output in JUnit XML format
// This format is compatible with CI/CD tools like Jenkins, GitLab CI, GitHub Actions
type JUnitRenderer struct{}

// junitTestSuites is the root element for JUnit XML
type junitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Name       string           `xml:"name,attr"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Errors     int              `xml:"errors,attr"`
	Time       float64          `xml:"time,attr"`
	TestSuites []junitTestSuite `xml:"testsuite"`
}

// junitTestSuite represents a testsuite element in JUnit XML
type junitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      float64         `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

// junitTestCase represents a testcase element in JUnit XML
type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
}

// junitFailure represents a failure element in JUnit XML
type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// junitSkipped represents a skipped element in JUnit XML
type junitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// Render writes the check result in JUnit XML format
func (r *JUnitRenderer) Render(w io.Writer, result *types.CheckResult) error {
	// Group findings by rule ID to create test suites per rule
	ruleFindings := make(map[string][]*types.Finding)
	for _, f := range result.Findings {
		ruleFindings[f.RuleID] = append(ruleFindings[f.RuleID], f)
	}

	// Build test suites
	var testSuites []junitTestSuite
	totalTests := 0
	totalFailures := 0
	totalErrors := 0

	timestamp := time.Now().Format(time.RFC3339)

	// Sort rule IDs for deterministic output
	ruleIDs := make([]string, 0, len(ruleFindings))
	for ruleID := range ruleFindings {
		ruleIDs = append(ruleIDs, ruleID)
	}
	sort.Strings(ruleIDs)

	for _, ruleID := range ruleIDs {
		findings := ruleFindings[ruleID]
		suite := junitTestSuite{
			Name:      fmt.Sprintf("tfbreak.%s", ruleID),
			Timestamp: timestamp,
			Time:      0,
		}

		for _, f := range findings {
			testCase := junitTestCase{
				Name:      r.buildTestCaseName(f),
				Classname: fmt.Sprintf("tfbreak.%s", f.RuleID),
				Time:      0,
			}

			if f.Ignored {
				// Ignored findings are skipped tests
				testCase.Skipped = &junitSkipped{
					Message: f.IgnoreReason,
				}
				suite.Skipped++
			} else {
				// Non-ignored findings are failures
				testCase.Failure = &junitFailure{
					Message: f.Message,
					Type:    f.Severity.String(),
					Content: r.buildFailureContent(f),
				}
				suite.Failures++
				if f.Severity == types.SeverityError {
					totalErrors++
				} else {
					totalFailures++
				}
			}

			suite.TestCases = append(suite.TestCases, testCase)
			suite.Tests++
			totalTests++
		}

		testSuites = append(testSuites, suite)
	}

	// If no findings, create a passing test
	if len(testSuites) == 0 {
		testSuites = append(testSuites, junitTestSuite{
			Name:      "tfbreak",
			Tests:     1,
			Timestamp: timestamp,
			TestCases: []junitTestCase{
				{
					Name:      "Breaking change detection",
					Classname: "tfbreak",
					Time:      0,
				},
			},
		})
		totalTests = 1
	}

	output := junitTestSuites{
		Name:       "tfbreak",
		Tests:      totalTests,
		Failures:   totalFailures,
		Errors:     totalErrors,
		Time:       0,
		TestSuites: testSuites,
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

// buildTestCaseName creates a descriptive name for the test case
func (r *JUnitRenderer) buildTestCaseName(f *types.Finding) string {
	location := ""
	if f.NewLocation != nil {
		location = fmt.Sprintf("%s:%d", f.NewLocation.Filename, f.NewLocation.Line)
	} else if f.OldLocation != nil {
		location = fmt.Sprintf("%s:%d", f.OldLocation.Filename, f.OldLocation.Line)
	}

	if location != "" {
		return fmt.Sprintf("%s at %s", f.RuleName, location)
	}
	return f.RuleName
}

// buildFailureContent creates the content for the failure element
func (r *JUnitRenderer) buildFailureContent(f *types.Finding) string {
	content := f.Message
	if f.Detail != "" {
		content += "\n\n" + f.Detail
	}
	if f.Remediation != "" {
		content += "\n\nRemediation:\n" + f.Remediation
	}
	return content
}

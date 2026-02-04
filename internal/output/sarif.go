package output

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// SARIFRenderer renders output in SARIF (Static Analysis Results Interchange Format) JSON
// SARIF is a standardized format for static analysis tools, supported by GitHub, Azure DevOps, etc.
type SARIFRenderer struct{}

// sarifLog is the root SARIF structure (version 2.1.0)
type sarifLog struct {
	Schema  string      `json:"$schema"`
	Version string      `json:"version"`
	Runs    []sarifRun  `json:"runs"`
}

// sarifRun represents a single analysis run
type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

// sarifTool describes the analysis tool
type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

// sarifDriver describes the tool driver
type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Version        string      `json:"version"`
	Rules          []sarifRule `json:"rules"`
}

// sarifRule describes a rule
type sarifRule struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	ShortDescription sarifMessage        `json:"shortDescription"`
	DefaultConfig    sarifDefaultConfig  `json:"defaultConfiguration"`
	HelpURI          string              `json:"helpUri,omitempty"`
}

// sarifDefaultConfig describes the default configuration for a rule
type sarifDefaultConfig struct {
	Level string `json:"level"`
}

// sarifResult represents a single finding
type sarifResult struct {
	RuleID    string           `json:"ruleId"`
	Level     string           `json:"level"`
	Message   sarifMessage     `json:"message"`
	Locations []sarifLocation  `json:"locations,omitempty"`
}

// sarifMessage is a message with text
type sarifMessage struct {
	Text string `json:"text"`
}

// sarifLocation describes where a result was found
type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

// sarifPhysicalLocation describes the physical file location
type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           *sarifRegion          `json:"region,omitempty"`
}

// sarifArtifactLocation describes the artifact (file)
type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

// sarifRegion describes a region within a file
type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
	EndLine     int `json:"endLine,omitempty"`
	EndColumn   int `json:"endColumn,omitempty"`
}

// Render writes the check result in SARIF format
func (r *SARIFRenderer) Render(w io.Writer, result *types.CheckResult) error {
	// Collect unique rules from findings
	ruleMap := make(map[string]*types.Finding)
	for _, f := range result.Findings {
		if _, exists := ruleMap[f.RuleID]; !exists {
			ruleMap[f.RuleID] = f
		}
	}

	// Build rules array - sort by rule ID for deterministic output
	ruleIDs := make([]string, 0, len(ruleMap))
	for ruleID := range ruleMap {
		ruleIDs = append(ruleIDs, ruleID)
	}
	sort.Strings(ruleIDs)

	var rules []sarifRule
	for _, ruleID := range ruleIDs {
		f := ruleMap[ruleID]
		rules = append(rules, sarifRule{
			ID:   ruleID,
			Name: f.RuleName,
			ShortDescription: sarifMessage{
				Text: f.RuleName,
			},
			DefaultConfig: sarifDefaultConfig{
				Level: mapToSARIFLevel(f.Severity),
			},
			HelpURI: "https://github.com/jokarl/tfbreak-core/blob/main/docs/rules.md#" + f.RuleID,
		})
	}

	// Build results array
	var results []sarifResult
	for _, f := range result.Findings {
		if f.Ignored {
			continue
		}

		sarifResult := sarifResult{
			RuleID: f.RuleID,
			Level:  mapToSARIFLevel(f.Severity),
			Message: sarifMessage{
				Text: f.Message,
			},
		}

		// Add location if available
		if f.NewLocation != nil {
			sarifResult.Locations = []sarifLocation{
				{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{
							URI: f.NewLocation.Filename,
						},
						Region: &sarifRegion{
							StartLine:   f.NewLocation.Line,
							StartColumn: f.NewLocation.Column,
							EndLine:     f.NewLocation.EndLine,
							EndColumn:   f.NewLocation.EndColumn,
						},
					},
				},
			}
		} else if f.OldLocation != nil {
			sarifResult.Locations = []sarifLocation{
				{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{
							URI: f.OldLocation.Filename,
						},
						Region: &sarifRegion{
							StartLine:   f.OldLocation.Line,
							StartColumn: f.OldLocation.Column,
							EndLine:     f.OldLocation.EndLine,
							EndColumn:   f.OldLocation.EndColumn,
						},
					},
				},
			}
		}

		results = append(results, sarifResult)
	}

	// Build the SARIF log
	log := sarifLog{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:           "tfbreak",
						InformationURI: "https://github.com/jokarl/tfbreak-core",
						Version:        "1.0.0",
						Rules:          rules,
					},
				},
				Results: results,
			},
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(log)
}

// mapToSARIFLevel maps tfbreak severity to SARIF level
func mapToSARIFLevel(s types.Severity) string {
	switch s {
	case types.SeverityError:
		return "error"
	case types.SeverityWarning:
		return "warning"
	case types.SeverityNotice:
		return "note"
	default:
		return "none"
	}
}

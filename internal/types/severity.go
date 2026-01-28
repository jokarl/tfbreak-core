package types

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Severity represents the severity level of a finding
type Severity int

const (
	// SeverityInfo is informational, no action needed
	SeverityInfo Severity = iota
	// SeverityRisky may cause unexpected behavior changes
	SeverityRisky
	// SeverityBreaking will break callers or destroy state
	SeverityBreaking
)

// String returns the string representation of the severity
func (s Severity) String() string {
	switch s {
	case SeverityBreaking:
		return "BREAKING"
	case SeverityRisky:
		return "RISKY"
	case SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// MarshalJSON implements json.Marshaler
func (s Severity) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON implements json.Unmarshaler
func (s *Severity) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	parsed, err := ParseSeverity(str)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// ParseSeverity parses a string into a Severity
func ParseSeverity(s string) (Severity, error) {
	switch strings.ToUpper(s) {
	case "BREAKING":
		return SeverityBreaking, nil
	case "RISKY":
		return SeverityRisky, nil
	case "INFO":
		return SeverityInfo, nil
	default:
		return SeverityInfo, fmt.Errorf("unknown severity: %s", s)
	}
}

// AtLeast returns true if this severity is at least as severe as other
func (s Severity) AtLeast(other Severity) bool {
	return s >= other
}

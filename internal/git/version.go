package git

import (
	"fmt"
	"regexp"
	"strconv"
)

// Version represents a parsed git version.
type Version struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

// String returns the version as "major.minor.patch".
func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// AtLeast returns true if this version is at least major.minor.
func (v *Version) AtLeast(major, minor int) bool {
	if v.Major > major {
		return true
	}
	if v.Major == major && v.Minor >= minor {
		return true
	}
	return false
}

// versionRegex matches git version strings like:
// - "git version 2.39.0"
// - "git version 2.39.0 (Apple Git-143)"
// - "git version 2.39.0.windows.1"
var versionRegex = regexp.MustCompile(`git version (\d+)\.(\d+)(?:\.(\d+))?`)

// GetVersion returns the installed git version.
func GetVersion() (*Version, error) {
	out, err := Run([]string{"--version"}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get git version: %w", err)
	}
	return ParseVersion(out)
}

// ParseVersion parses a git version string.
func ParseVersion(s string) (*Version, error) {
	matches := versionRegex.FindStringSubmatch(s)
	if matches == nil {
		return nil, fmt.Errorf("failed to parse git version: %q", s)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch := 0
	if len(matches) > 3 && matches[3] != "" {
		patch, _ = strconv.Atoi(matches[3])
	}

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
		Raw:   s,
	}, nil
}

// CheckVersion verifies git is installed and meets the minimum version requirement.
// Returns an error if git is not installed or version is below minMajor.minMinor.
func CheckVersion(minMajor, minMinor int) error {
	v, err := GetVersion()
	if err != nil {
		return err
	}

	if !v.AtLeast(minMajor, minMinor) {
		return &ErrVersionTooOld{
			Current:  v.String(),
			Required: fmt.Sprintf("%d.%d", minMajor, minMinor),
		}
	}

	return nil
}

// MinVersion is the minimum git version required for tfbreak.
const (
	MinVersionMajor = 2
	MinVersionMinor = 5
)

// CheckMinVersion verifies git meets the minimum version for tfbreak.
func CheckMinVersion() error {
	return CheckVersion(MinVersionMajor, MinVersionMinor)
}

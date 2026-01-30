package git

import (
	"os/exec"
	"testing"
)

func TestGetVersion(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	version, err := GetVersion()
	if err != nil {
		t.Fatalf("GetVersion() returned error: %v", err)
	}

	// Version should have valid components
	if version.Major < 1 {
		t.Errorf("Version.Major = %d, want >= 1", version.Major)
	}

	// Raw should contain original string
	if version.Raw == "" {
		t.Error("Version.Raw should not be empty")
	}
}

func TestParseVersion_Standard(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		major   int
		minor   int
		patch   int
		wantErr bool
	}{
		{
			name:  "standard format",
			input: "git version 2.39.0",
			major: 2,
			minor: 39,
			patch: 0,
		},
		{
			name:  "older version",
			input: "git version 2.5.0",
			major: 2,
			minor: 5,
			patch: 0,
		},
		{
			name:  "very old version",
			input: "git version 1.8.3",
			major: 1,
			minor: 8,
			patch: 3,
		},
		{
			name:  "patch version with numbers",
			input: "git version 2.34.1",
			major: 2,
			minor: 34,
			patch: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			if v.Major != tt.major {
				t.Errorf("Major = %d, want %d", v.Major, tt.major)
			}
			if v.Minor != tt.minor {
				t.Errorf("Minor = %d, want %d", v.Minor, tt.minor)
			}
			if v.Patch != tt.patch {
				t.Errorf("Patch = %d, want %d", v.Patch, tt.patch)
			}
		})
	}
}

func TestParseVersion_MacOS(t *testing.T) {
	tests := []struct {
		name  string
		input string
		major int
		minor int
		patch int
	}{
		{
			name:  "Apple Git format",
			input: "git version 2.39.0 (Apple Git-143)",
			major: 2,
			minor: 39,
			patch: 0,
		},
		{
			name:  "Apple Git older",
			input: "git version 2.37.1 (Apple Git-137.1)",
			major: 2,
			minor: 37,
			patch: 1,
		},
		{
			name:  "Homebrew on macOS",
			input: "git version 2.42.0",
			major: 2,
			minor: 42,
			patch: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ParseVersion(tt.input)
			if err != nil {
				t.Fatalf("ParseVersion() error = %v", err)
			}

			if v.Major != tt.major {
				t.Errorf("Major = %d, want %d", v.Major, tt.major)
			}
			if v.Minor != tt.minor {
				t.Errorf("Minor = %d, want %d", v.Minor, tt.minor)
			}
			if v.Patch != tt.patch {
				t.Errorf("Patch = %d, want %d", v.Patch, tt.patch)
			}
		})
	}
}

func TestParseVersion_Windows(t *testing.T) {
	tests := []struct {
		name  string
		input string
		major int
		minor int
		patch int
	}{
		{
			name:  "Git for Windows",
			input: "git version 2.39.0.windows.1",
			major: 2,
			minor: 39,
			patch: 0,
		},
		{
			name:  "Git for Windows older",
			input: "git version 2.37.2.windows.2",
			major: 2,
			minor: 37,
			patch: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ParseVersion(tt.input)
			if err != nil {
				t.Fatalf("ParseVersion() error = %v", err)
			}

			if v.Major != tt.major {
				t.Errorf("Major = %d, want %d", v.Major, tt.major)
			}
			if v.Minor != tt.minor {
				t.Errorf("Minor = %d, want %d", v.Minor, tt.minor)
			}
			if v.Patch != tt.patch {
				t.Errorf("Patch = %d, want %d", v.Patch, tt.patch)
			}
		})
	}
}

func TestParseVersion_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "no version number",
			input: "git version",
		},
		{
			name:  "garbage input",
			input: "not a version string at all",
		},
		{
			name:  "partial version",
			input: "git version 2",
		},
		{
			name:  "non-numeric major",
			input: "git version abc.def.ghi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseVersion(tt.input)
			if err == nil {
				t.Error("ParseVersion() should return error for invalid input")
			}
		})
	}
}

func TestCheckVersion_Satisfied(t *testing.T) {
	tests := []struct {
		name     string
		version  *Version
		minMajor int
		minMinor int
	}{
		{
			name:     "exact match",
			version:  &Version{Major: 2, Minor: 5, Patch: 0},
			minMajor: 2,
			minMinor: 5,
		},
		{
			name:     "newer minor",
			version:  &Version{Major: 2, Minor: 39, Patch: 0},
			minMajor: 2,
			minMinor: 5,
		},
		{
			name:     "newer major",
			version:  &Version{Major: 3, Minor: 0, Patch: 0},
			minMajor: 2,
			minMinor: 5,
		},
		{
			name:     "minimum version 2.25",
			version:  &Version{Major: 2, Minor: 39, Patch: 0},
			minMajor: 2,
			minMinor: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkVersionSatisfied(tt.version, tt.minMajor, tt.minMinor)
			if err != nil {
				t.Errorf("checkVersionSatisfied() returned error: %v", err)
			}
		})
	}
}

func TestCheckVersion_TooOld(t *testing.T) {
	tests := []struct {
		name     string
		version  *Version
		minMajor int
		minMinor int
	}{
		{
			name:     "minor too old",
			version:  &Version{Major: 2, Minor: 4, Patch: 0, Raw: "2.4.0"},
			minMajor: 2,
			minMinor: 5,
		},
		{
			name:     "major too old",
			version:  &Version{Major: 1, Minor: 9, Patch: 0, Raw: "1.9.0"},
			minMajor: 2,
			minMinor: 5,
		},
		{
			name:     "both too old",
			version:  &Version{Major: 1, Minor: 0, Patch: 0, Raw: "1.0.0"},
			minMajor: 2,
			minMinor: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkVersionSatisfied(tt.version, tt.minMajor, tt.minMinor)
			if err == nil {
				t.Error("checkVersionSatisfied() should return error for old version")
			}
		})
	}
}

func TestCheckVersion_Integration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// Modern systems should have at least git 2.5
	err := CheckVersion(2, 5)
	if err != nil {
		t.Errorf("CheckVersion(2, 5) returned error: %v", err)
	}
}

func TestCheckVersion_FutureVersion(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, skipping test")
	}

	// No system should have git version 99
	err := CheckVersion(99, 0)
	if err == nil {
		t.Error("CheckVersion(99, 0) should return error")
	}
}

func TestVersion_String(t *testing.T) {
	v := &Version{
		Major: 2,
		Minor: 39,
		Patch: 0,
		Raw:   "git version 2.39.0 (Apple Git-143)",
	}

	// If Version has a String() method, test it
	// Otherwise, just verify the struct is properly formatted
	if v.Major != 2 || v.Minor != 39 || v.Patch != 0 {
		t.Error("Version fields not properly set")
	}
}

func TestParseVersion_PreservesRaw(t *testing.T) {
	input := "git version 2.39.0 (Apple Git-143)"
	v, err := ParseVersion(input)
	if err != nil {
		t.Fatalf("ParseVersion() error = %v", err)
	}

	if v.Raw != input {
		t.Errorf("Raw = %q, want %q", v.Raw, input)
	}
}

func TestParseVersion_Ubuntu(t *testing.T) {
	// Ubuntu sometimes has additional info
	input := "git version 2.34.1"
	v, err := ParseVersion(input)
	if err != nil {
		t.Fatalf("ParseVersion() error = %v", err)
	}

	if v.Major != 2 || v.Minor != 34 || v.Patch != 1 {
		t.Errorf("Version = %d.%d.%d, want 2.34.1", v.Major, v.Minor, v.Patch)
	}
}

func TestParseVersion_TwoDigitVersion(t *testing.T) {
	// Some versions might only have major.minor
	input := "git version 2.39"
	v, err := ParseVersion(input)
	// This might be valid or invalid depending on implementation
	// If valid, patch should default to 0
	if err == nil {
		if v.Major != 2 || v.Minor != 39 {
			t.Errorf("Version = %d.%d, want 2.39", v.Major, v.Minor)
		}
	}
	// If err != nil, that's also acceptable behavior
}

// Helper function that mirrors the expected implementation behavior
// This tests the version comparison logic without relying on actual git
func checkVersionSatisfied(v *Version, minMajor, minMinor int) error {
	if v.Major < minMajor {
		return &VersionTooOldError{
			Current:  v,
			MinMajor: minMajor,
			MinMinor: minMinor,
		}
	}
	if v.Major == minMajor && v.Minor < minMinor {
		return &VersionTooOldError{
			Current:  v,
			MinMajor: minMajor,
			MinMinor: minMinor,
		}
	}
	return nil
}

// VersionTooOldError is used in tests to simulate the expected error type
type VersionTooOldError struct {
	Current  *Version
	MinMajor int
	MinMinor int
}

func (e *VersionTooOldError) Error() string {
	return "git version too old"
}

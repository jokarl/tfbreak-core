package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRefSpec(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantRef  string
		wantPath string
	}{
		{
			name:     "simple ref",
			input:    "main",
			wantRef:  "main",
			wantPath: "",
		},
		{
			name:     "ref with path",
			input:    "main:modules/vpc",
			wantRef:  "main",
			wantPath: "modules/vpc",
		},
		{
			name:     "tag with path",
			input:    "v1.0.0:src/module",
			wantRef:  "v1.0.0",
			wantPath: "src/module",
		},
		{
			name:     "commit SHA with path",
			input:    "abc1234:path/to/module",
			wantRef:  "abc1234",
			wantPath: "path/to/module",
		},
		{
			name:     "ref with nested path",
			input:    "feature/branch:deeply/nested/path",
			wantRef:  "feature/branch",
			wantPath: "deeply/nested/path",
		},
		{
			name:     "HTTPS URL (no path extraction)",
			input:    "https://github.com/org/repo",
			wantRef:  "https://github.com/org/repo",
			wantPath: "",
		},
		{
			name:     "empty string",
			input:    "",
			wantRef:  "",
			wantPath: "",
		},
		{
			name:     "ref with empty path after colon",
			input:    "main:",
			wantRef:  "main",
			wantPath: "",
		},
		{
			name:     "HEAD~5 with path",
			input:    "HEAD~5:modules/vpc",
			wantRef:  "HEAD~5",
			wantPath: "modules/vpc",
		},
		{
			name:     "origin/main with path",
			input:    "origin/main:terraform",
			wantRef:  "origin/main",
			wantPath: "terraform",
		},
		{
			name:     "Windows drive letter C:",
			input:    "C:\\path\\to\\module",
			wantRef:  "C:\\path\\to\\module",
			wantPath: "",
		},
		{
			name:     "Windows drive letter lowercase",
			input:    "d:\\users\\terraform",
			wantRef:  "d:\\users\\terraform",
			wantPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRefSpec(tt.input)
			if got.Ref != tt.wantRef {
				t.Errorf("parseRefSpec(%q).Ref = %q, want %q", tt.input, got.Ref, tt.wantRef)
			}
			if got.Path != tt.wantPath {
				t.Errorf("parseRefSpec(%q).Path = %q, want %q", tt.input, got.Path, tt.wantPath)
			}
		})
	}
}

func TestValidateSubdirPath(t *testing.T) {
	// Create a temp directory with subdirectories for testing
	rootDir := t.TempDir()
	subDir := filepath.Join(rootDir, "modules", "vpc")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create a file (not a directory)
	filePath := filepath.Join(rootDir, "main.tf")
	if err := os.WriteFile(filePath, []byte("# test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name      string
		rootDir   string
		subPath   string
		ref       string
		wantError bool
		errSubstr string
	}{
		{
			name:      "valid subdirectory",
			rootDir:   rootDir,
			subPath:   "modules/vpc",
			ref:       "main",
			wantError: false,
		},
		{
			name:      "valid nested path with parent",
			rootDir:   rootDir,
			subPath:   "modules",
			ref:       "main",
			wantError: false,
		},
		{
			name:      "nonexistent path",
			rootDir:   rootDir,
			subPath:   "nonexistent/path",
			ref:       "v1.0.0",
			wantError: true,
			errSubstr: "does not exist",
		},
		{
			name:      "path is a file not directory",
			rootDir:   rootDir,
			subPath:   "main.tf",
			ref:       "main",
			wantError: true,
			errSubstr: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSubdirPath(tt.rootDir, tt.subPath, tt.ref)
			if tt.wantError {
				if err == nil {
					t.Errorf("validateSubdirPath() = nil, want error containing %q", tt.errSubstr)
				} else if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateSubdirPath() error = %q, want error containing %q", err.Error(), tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("validateSubdirPath() error = %v, want nil", err)
				}
			}
		})
	}
}

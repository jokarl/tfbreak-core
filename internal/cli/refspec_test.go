package cli

import "testing"

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

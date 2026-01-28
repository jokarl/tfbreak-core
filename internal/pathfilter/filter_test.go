package pathfilter

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestFilterFiles(t *testing.T) {
	// Create test directory structure
	tmpDir := t.TempDir()

	// Create files
	files := []string{
		"main.tf",
		"variables.tf",
		"outputs.tf",
		"README.md",
		"modules/vpc/main.tf",
		"modules/vpc/variables.tf",
		".terraform/providers/test.tf",
		"examples/basic/main.tf",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("# test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	tests := []struct {
		name     string
		include  []string
		exclude  []string
		expected []string
	}{
		{
			name:    "default patterns",
			include: []string{"**/*.tf"},
			exclude: []string{".terraform/**"},
			expected: []string{
				"main.tf",
				"variables.tf",
				"outputs.tf",
				"modules/vpc/main.tf",
				"modules/vpc/variables.tf",
				"examples/basic/main.tf",
			},
		},
		{
			name:    "exclude examples",
			include: []string{"**/*.tf"},
			exclude: []string{".terraform/**", "examples/**"},
			expected: []string{
				"main.tf",
				"variables.tf",
				"outputs.tf",
				"modules/vpc/main.tf",
				"modules/vpc/variables.tf",
			},
		},
		{
			name:    "only root tf files",
			include: []string{"*.tf"},
			exclude: []string{},
			expected: []string{
				"main.tf",
				"variables.tf",
				"outputs.tf",
			},
		},
		{
			name:    "include multiple patterns",
			include: []string{"**/*.tf", "**/*.md"},
			exclude: []string{".terraform/**"},
			expected: []string{
				"main.tf",
				"variables.tf",
				"outputs.tf",
				"README.md",
				"modules/vpc/main.tf",
				"modules/vpc/variables.tf",
				"examples/basic/main.tf",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(tt.include, tt.exclude)
			result, err := f.FilterFiles(tmpDir)
			if err != nil {
				t.Fatalf("FilterFiles failed: %v", err)
			}

			// Sort for comparison
			sort.Strings(result)
			sort.Strings(tt.expected)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d files, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("file %d: expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestFilterFilesAbs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(testFile, []byte("# test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	f := New([]string{"**/*.tf"}, []string{})
	result, err := f.FilterFilesAbs(tmpDir)
	if err != nil {
		t.Fatalf("FilterFilesAbs failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result))
	}

	absDir, _ := filepath.Abs(tmpDir)
	expected := filepath.Join(absDir, "main.tf")
	if result[0] != expected {
		t.Errorf("expected %s, got %s", expected, result[0])
	}
}

func TestMatchFile(t *testing.T) {
	tests := []struct {
		name     string
		include  []string
		exclude  []string
		path     string
		expected bool
	}{
		{
			name:     "matches include",
			include:  []string{"**/*.tf"},
			exclude:  []string{},
			path:     "main.tf",
			expected: true,
		},
		{
			name:     "matches nested include",
			include:  []string{"**/*.tf"},
			exclude:  []string{},
			path:     "modules/vpc/main.tf",
			expected: true,
		},
		{
			name:     "excluded by pattern",
			include:  []string{"**/*.tf"},
			exclude:  []string{".terraform/**"},
			path:     ".terraform/providers/test.tf",
			expected: false,
		},
		{
			name:     "no match",
			include:  []string{"**/*.tf"},
			exclude:  []string{},
			path:     "README.md",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(tt.include, tt.exclude)
			result, err := f.MatchFile(tt.path)
			if err != nil {
				t.Fatalf("MatchFile failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDefaultFilter(t *testing.T) {
	f := DefaultFilter()

	if len(f.include) != 1 || f.include[0] != "**/*.tf" {
		t.Errorf("unexpected include patterns: %v", f.include)
	}

	if len(f.exclude) != 1 || f.exclude[0] != ".terraform/**" {
		t.Errorf("unexpected exclude patterns: %v", f.exclude)
	}
}

func TestEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	f := DefaultFilter()
	result, err := f.FilterFiles(tmpDir)
	if err != nil {
		t.Fatalf("FilterFiles failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 files, got %d", len(result))
	}
}

func TestWalkDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files
	files := []string{
		"main.tf",
		"modules/vpc/main.tf",
		".terraform/providers/test.tf",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("# test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	f := New([]string{"**/*.tf"}, []string{".terraform/**"})

	var walked []string
	err := f.WalkDir(tmpDir, func(path string, d os.DirEntry) error {
		rel, _ := filepath.Rel(tmpDir, path)
		walked = append(walked, rel)
		return nil
	})

	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}

	sort.Strings(walked)
	expected := []string{"main.tf", filepath.Join("modules", "vpc", "main.tf")}
	sort.Strings(expected)

	if len(walked) != len(expected) {
		t.Errorf("expected %d files, got %d: %v", len(expected), len(walked), walked)
	}
}

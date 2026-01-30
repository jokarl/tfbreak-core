package plugin

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSource_GetRelease(t *testing.T) {
	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "tfbreak-file-source-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create version directory
	versionDir := filepath.Join(tmpDir, "0.1.0")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("failed to create version dir: %v", err)
	}

	// Create some files
	files := map[string]string{
		"checksums.txt":                         "abc123  plugin_darwin_arm64.zip\n",
		"tfbreak-ruleset-test_darwin_arm64.zip": "fake zip content",
		"tfbreak-ruleset-test_linux_amd64.zip":  "fake zip content",
	}

	for name, content := range files {
		path := filepath.Join(versionDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", name, err)
		}
	}

	// Create file source
	source, err := NewFileSource("file://" + tmpDir)
	if err != nil {
		t.Fatalf("failed to create file source: %v", err)
	}

	// Get release
	release, err := source.GetRelease("0.1.0")
	if err != nil {
		t.Fatalf("failed to get release: %v", err)
	}

	if release.Version != "0.1.0" {
		t.Errorf("unexpected version: %s", release.Version)
	}

	if len(release.Assets) != 3 {
		t.Errorf("expected 3 assets, got %d", len(release.Assets))
	}

	// Check checksums.txt is present
	checksumAsset := release.FindAsset("checksums.txt")
	if checksumAsset == nil {
		t.Error("expected to find checksums.txt asset")
	}
}

func TestFileSource_GetRelease_NotFound(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tfbreak-file-source-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	source, err := NewFileSource("file://" + tmpDir)
	if err != nil {
		t.Fatalf("failed to create file source: %v", err)
	}

	// Try to get non-existent release
	_, err = source.GetRelease("0.1.0")
	if err == nil {
		t.Error("expected error for non-existent release")
	}
}

func TestFileSource_DownloadAsset(t *testing.T) {
	// Create temp file
	tmpDir, err := os.MkdirTemp("", "tfbreak-file-source-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := "test file content for download"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	source, err := NewFileSource("file://" + tmpDir)
	if err != nil {
		t.Fatalf("failed to create file source: %v", err)
	}

	asset := &SourceAsset{
		Name: "test.txt",
		URL:  testFile,
	}

	var buf bytes.Buffer
	if err := source.DownloadAsset(asset, &buf); err != nil {
		t.Fatalf("failed to download asset: %v", err)
	}

	if buf.String() != testContent {
		t.Errorf("unexpected content: %s", buf.String())
	}
}

func TestFileSource_String(t *testing.T) {
	source, err := NewFileSource("file:///var/plugins/test")
	if err != nil {
		t.Fatalf("failed to create file source: %v", err)
	}

	if source.String() != "file:///var/plugins/test" {
		t.Errorf("unexpected String(): %s", source.String())
	}
}

func TestNewFileSource(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantPath string
	}{
		{
			name:     "unix path",
			url:      "file:///var/plugins/test",
			wantPath: "/var/plugins/test",
		},
		{
			name:     "relative path",
			url:      "file://./plugins",
			wantPath: "plugins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := NewFileSource(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// basePath should be cleaned
			if source.basePath != tt.wantPath {
				t.Errorf("basePath = %q, want %q", source.basePath, tt.wantPath)
			}
		})
	}
}

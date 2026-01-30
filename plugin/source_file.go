package plugin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileSource implements Source for local filesystem.
// This is useful for air-gapped environments where plugins are
// pre-downloaded and stored locally.
type FileSource struct {
	basePath string
	source   string // original source URL for String()
}

// NewFileSource creates a new filesystem source from a file:// URL.
// Format: file:///path/to/plugins/tfbreak-ruleset-foo
func NewFileSource(sourceURL string) (*FileSource, error) {
	// Remove file:// prefix
	path := strings.TrimPrefix(sourceURL, "file://")

	// Clean the path
	path = filepath.Clean(path)

	return &FileSource{
		basePath: path,
		source:   sourceURL,
	}, nil
}

// GetRelease returns release information for a specific version.
// It expects the following directory structure:
//   {basePath}/{version}/
//     ├── tfbreak-ruleset-foo_darwin_arm64.zip
//     ├── tfbreak-ruleset-foo_linux_amd64.zip
//     └── checksums.txt
func (s *FileSource) GetRelease(version string) (*SourceRelease, error) {
	versionDir := filepath.Join(s.basePath, version)

	// Check if version directory exists
	info, err := os.Stat(versionDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("release %s not found at %s", version, s.basePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to access release directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", versionDir)
	}

	// List files in the version directory
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read release directory: %w", err)
	}

	sr := &SourceRelease{
		Version: version,
		Assets:  make([]SourceAsset, 0, len(entries)),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(versionDir, entry.Name())
		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}

		sr.Assets = append(sr.Assets, SourceAsset{
			Name: entry.Name(),
			URL:  filePath, // For file source, URL is the file path
			Size: fileInfo.Size(),
		})
	}

	return sr, nil
}

// DownloadAsset copies a local file to the writer.
// For FileSource, the asset URL is actually a file path.
func (s *FileSource) DownloadAsset(asset *SourceAsset, dest io.Writer) error {
	if asset == nil {
		return fmt.Errorf("asset is nil")
	}

	file, err := os.Open(asset.URL)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(dest, file)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// String returns the source identifier for logging.
func (s *FileSource) String() string {
	return s.source
}

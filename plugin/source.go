package plugin

import (
	"fmt"
	"io"
	"strings"
)

// Source represents a plugin distribution source (GitHub, GitLab, HTTP, etc.)
type Source interface {
	// GetRelease returns metadata for a specific version
	GetRelease(version string) (*SourceRelease, error)

	// DownloadAsset downloads a release asset to the writer
	DownloadAsset(asset *SourceAsset, dest io.Writer) error

	// String returns the source identifier for logging
	String() string
}

// SourceRelease represents a release from any source.
type SourceRelease struct {
	Version string
	Assets  []SourceAsset
}

// SourceAsset represents a downloadable asset.
type SourceAsset struct {
	Name string
	URL  string
	Size int64
}

// FindAsset finds an asset by name in a release.
func (r *SourceRelease) FindAsset(name string) *SourceAsset {
	for i := range r.Assets {
		if r.Assets[i].Name == name {
			return &r.Assets[i]
		}
	}
	return nil
}

// NewSource creates a Source from a source URL string.
// It auto-detects the source type from the URL pattern.
func NewSource(sourceURL string) (Source, error) {
	switch {
	case strings.HasPrefix(sourceURL, "github.com/"):
		return NewGitHubSource(sourceURL)
	case isGitLabSource(sourceURL):
		return NewGitLabSource(sourceURL)
	case strings.HasPrefix(sourceURL, "https://") || strings.HasPrefix(sourceURL, "http://"):
		return NewHTTPSource(sourceURL)
	case strings.HasPrefix(sourceURL, "file://"):
		return NewFileSource(sourceURL)
	default:
		return nil, fmt.Errorf("unknown source type: %s (supported: github.com/*, gitlab.com/*, https://, file://)", sourceURL)
	}
}

// isGitLabSource checks if the source is a GitLab URL.
func isGitLabSource(source string) bool {
	// gitlab.com or self-hosted GitLab instances
	if strings.HasPrefix(source, "gitlab.com/") {
		return true
	}
	// Self-hosted GitLab detection: contains "gitlab" in domain
	// e.g., gitlab.company.com/org/repo
	parts := strings.SplitN(source, "/", 2)
	if len(parts) >= 1 {
		domain := parts[0]
		if strings.Contains(domain, "gitlab") {
			return true
		}
	}
	return false
}

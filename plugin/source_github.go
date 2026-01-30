package plugin

import (
	"fmt"
	"io"
)

// GitHubSource implements Source for GitHub repositories.
type GitHubSource struct {
	owner  string
	repo   string
	client *GitHubClient
	source string // original source URL for String()
}

// NewGitHubSource creates a new GitHub source from a source URL.
// Format: github.com/owner/repo
func NewGitHubSource(sourceURL string) (*GitHubSource, error) {
	owner, repo, err := ParseGitHubSource(sourceURL)
	if err != nil {
		return nil, err
	}

	return &GitHubSource{
		owner:  owner,
		repo:   repo,
		client: NewGitHubClient(),
		source: sourceURL,
	}, nil
}

// GetRelease fetches release information for a specific version.
func (s *GitHubSource) GetRelease(version string) (*SourceRelease, error) {
	tag := "v" + version
	release, err := s.client.GetRelease(s.owner, s.repo, tag)
	if err != nil {
		return nil, err
	}

	// Convert to generic SourceRelease
	sr := &SourceRelease{
		Version: version,
		Assets:  make([]SourceAsset, len(release.Assets)),
	}

	for i, asset := range release.Assets {
		sr.Assets[i] = SourceAsset{
			Name: asset.Name,
			URL:  asset.BrowserDownloadURL,
			Size: asset.Size,
		}
	}

	return sr, nil
}

// DownloadAsset downloads a release asset to the writer.
func (s *GitHubSource) DownloadAsset(asset *SourceAsset, dest io.Writer) error {
	if asset == nil {
		return fmt.Errorf("asset is nil")
	}
	return s.client.DownloadAsset(asset.URL, dest)
}

// String returns the source identifier for logging.
func (s *GitHubSource) String() string {
	return s.source
}

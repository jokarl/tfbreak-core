package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// GitLabSource implements Source for GitLab repositories.
type GitLabSource struct {
	host       string
	projectID  string // URL-encoded project path (owner/repo)
	client     *http.Client
	token      string
	source     string // original source URL for String()
}

// NewGitLabSource creates a new GitLab source from a source URL.
// Formats:
//   - gitlab.com/owner/repo
//   - gitlab.company.com/owner/repo (self-hosted)
func NewGitLabSource(sourceURL string) (*GitLabSource, error) {
	parts := strings.SplitN(sourceURL, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid GitLab source format: %s (expected host/owner/repo)", sourceURL)
	}

	host := parts[0]
	projectPath := parts[1]

	// URL-encode the project path for API calls
	projectID := url.PathEscape(projectPath)

	// Get token from environment
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		token = os.Getenv("GL_TOKEN")
	}

	return &GitLabSource{
		host:      host,
		projectID: projectID,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		token:  token,
		source: sourceURL,
	}, nil
}

// gitlabRelease represents a GitLab release response.
type gitlabRelease struct {
	TagName string `json:"tag_name"`
	Assets  struct {
		Links []gitlabAssetLink `json:"links"`
	} `json:"assets"`
}

// gitlabAssetLink represents a GitLab release asset link.
type gitlabAssetLink struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	DirectURL string `json:"direct_asset_url"`
}

// GetRelease fetches release information for a specific version.
func (s *GitLabSource) GetRelease(version string) (*SourceRelease, error) {
	tag := "v" + version
	apiURL := fmt.Sprintf("https://%s/api/v4/projects/%s/releases/%s", s.host, s.projectID, url.PathEscape(tag))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("release %s not found", tag)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized access to GitLab project. Set GITLAB_TOKEN to authenticate")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d fetching release", resp.StatusCode)
	}

	var release gitlabRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %w", err)
	}

	// Convert to generic SourceRelease
	sr := &SourceRelease{
		Version: version,
		Assets:  make([]SourceAsset, len(release.Assets.Links)),
	}

	for i, link := range release.Assets.Links {
		assetURL := link.DirectURL
		if assetURL == "" {
			assetURL = link.URL
		}
		sr.Assets[i] = SourceAsset{
			Name: link.Name,
			URL:  assetURL,
		}
	}

	return sr, nil
}

// DownloadAsset downloads a release asset to the writer.
func (s *GitLabSource) DownloadAsset(asset *SourceAsset, dest io.Writer) error {
	if asset == nil {
		return fmt.Errorf("asset is nil")
	}

	req, err := http.NewRequest("GET", asset.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d downloading asset", resp.StatusCode)
	}

	_, err = io.Copy(dest, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write asset: %w", err)
	}

	return nil
}

// String returns the source identifier for logging.
func (s *GitLabSource) String() string {
	return s.source
}

// setHeaders sets common headers for GitLab API requests.
func (s *GitLabSource) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "tfbreak")
	if s.token != "" {
		req.Header.Set("PRIVATE-TOKEN", s.token)
	}
}

// Package plugin provides plugin discovery, installation, and runner implementation for tfbreak.
package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GitHubClient handles interactions with the GitHub API for plugin downloads.
type GitHubClient struct {
	httpClient *http.Client
	token      string
	baseURL    string // For GitHub Enterprise support
}

// Release represents a GitHub release.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// NewGitHubClient creates a new GitHub client.
// It reads the token from GITHUB_TOKEN or GH_TOKEN environment variables.
func NewGitHubClient() *GitHubClient {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}

	return &GitHubClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token:   token,
		baseURL: "https://api.github.com",
	}
}

// NewGitHubEnterpriseClient creates a client for GitHub Enterprise.
func NewGitHubEnterpriseClient(host string) *GitHubClient {
	client := NewGitHubClient()
	client.baseURL = fmt.Sprintf("https://%s/api/v3", host)
	return client
}

// GetRelease fetches release information for a specific tag.
func (c *GitHubClient) GetRelease(owner, repo, tag string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", c.baseURL, owner, repo, tag)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("release %s not found in %s/%s", tag, owner, repo)
	}

	if resp.StatusCode == http.StatusForbidden {
		// Check for rate limiting
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return nil, fmt.Errorf("GitHub API rate limit exceeded. Set GITHUB_TOKEN to increase limit")
		}
		return nil, fmt.Errorf("access forbidden to %s/%s (status %d)", owner, repo, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d fetching release", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %w", err)
	}

	return &release, nil
}

// DownloadAsset downloads a release asset to the provided writer.
// It follows redirects as GitHub redirects to the actual download URL.
func (c *GitHubClient) DownloadAsset(url string, dest io.Writer) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// For asset downloads, we need to accept the binary content
	req.Header.Set("Accept", "application/octet-stream")
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}

	resp, err := c.httpClient.Do(req)
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

// FindAsset finds an asset by name in a release.
func (r *Release) FindAsset(name string) *Asset {
	for i := range r.Assets {
		if r.Assets[i].Name == name {
			return &r.Assets[i]
		}
	}
	return nil
}

// setHeaders sets common headers for GitHub API requests.
func (c *GitHubClient) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "tfbreak")
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
}

// ParseGitHubSource parses a source URL like "github.com/owner/repo" into owner and repo.
func ParseGitHubSource(source string) (owner, repo string, err error) {
	// Remove github.com/ prefix if present
	source = strings.TrimPrefix(source, "github.com/")

	parts := strings.SplitN(source, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid GitHub source format: %s (expected owner/repo)", source)
	}

	return parts[0], parts[1], nil
}

// IsGitHubSource checks if the source is a GitHub URL.
func IsGitHubSource(source string) bool {
	return strings.HasPrefix(source, "github.com/")
}

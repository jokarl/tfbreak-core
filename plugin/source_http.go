package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPSource implements Source for custom HTTP endpoints.
// This allows plugins to be distributed from any HTTP server that implements
// a simple metadata protocol.
type HTTPSource struct {
	baseURL string
	client  *http.Client
}

// NewHTTPSource creates a new HTTP source from a base URL.
// Format: https://plugins.example.com/tfbreak-ruleset-foo
func NewHTTPSource(baseURL string) (*HTTPSource, error) {
	return &HTTPSource{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// httpReleaseMetadata represents the metadata.json format for HTTP sources.
type httpReleaseMetadata struct {
	Version string            `json:"version"`
	Assets  []httpAssetInfo   `json:"assets"`
}

// httpAssetInfo represents an asset in the metadata.json.
type httpAssetInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int64  `json:"size,omitempty"`
}

// GetRelease fetches release information for a specific version.
// It expects the server to provide:
//   GET {baseURL}/releases/{version}/metadata.json
// Returns JSON with version and assets array.
func (s *HTTPSource) GetRelease(version string) (*SourceRelease, error) {
	metadataURL := fmt.Sprintf("%s/releases/%s/metadata.json", s.baseURL, version)

	req, err := http.NewRequest("GET", metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "tfbreak")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("release %s not found at %s", version, s.baseURL)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d fetching release metadata", resp.StatusCode)
	}

	var metadata httpReleaseMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode release metadata: %w", err)
	}

	// Convert to generic SourceRelease
	sr := &SourceRelease{
		Version: metadata.Version,
		Assets:  make([]SourceAsset, len(metadata.Assets)),
	}

	for i, asset := range metadata.Assets {
		sr.Assets[i] = SourceAsset{
			Name: asset.Name,
			URL:  asset.URL,
			Size: asset.Size,
		}
	}

	return sr, nil
}

// DownloadAsset downloads a release asset to the writer.
func (s *HTTPSource) DownloadAsset(asset *SourceAsset, dest io.Writer) error {
	if asset == nil {
		return fmt.Errorf("asset is nil")
	}

	req, err := http.NewRequest("GET", asset.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "tfbreak")

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
func (s *HTTPSource) String() string {
	return s.baseURL
}

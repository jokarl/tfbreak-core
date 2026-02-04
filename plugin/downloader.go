package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Downloader handles downloading plugins from GitHub releases.
type Downloader struct {
	httpClient *http.Client
	pluginDir  string
}

// NewDownloader creates a new plugin downloader.
// If pluginDir is empty, the default plugin directory is used.
func NewDownloader(pluginDir string) *Downloader {
	if pluginDir == "" {
		pluginDir = GetDefaultPluginDir()
	}
	return &Downloader{
		httpClient: &http.Client{},
		pluginDir:  pluginDir,
	}
}

// Download downloads a plugin from the given source.
// source format: "github.com/{owner}/{repo}" (e.g., "github.com/jokarl/tfbreak-ruleset-azurerm")
// version: semantic version (e.g., "0.2.0") or "latest"
func (d *Downloader) Download(source, version string) error {
	owner, repo, err := parseSource(source)
	if err != nil {
		return fmt.Errorf("invalid source: %w", err)
	}

	// Resolve "latest" to actual version
	if version == "" || version == "latest" {
		v, err := d.getLatestVersion(owner, repo)
		if err != nil {
			return fmt.Errorf("failed to get latest version: %w", err)
		}
		version = v
	}

	// Build asset name and download URL
	assetName := buildAssetName(repo)
	url := buildDownloadURL(owner, repo, version, assetName)

	// Ensure plugin directory exists
	if err := os.MkdirAll(d.pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Download to plugin directory
	destPath := filepath.Join(d.pluginDir, assetName)
	if err := d.download(url, destPath); err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}

	return nil
}

// parseSource parses a source string in the format "github.com/{owner}/{repo}".
// Returns owner and repo components.
func parseSource(source string) (owner, repo string, err error) {
	// Remove protocol if present
	source = strings.TrimPrefix(source, "https://")
	source = strings.TrimPrefix(source, "http://")

	// Must start with github.com
	if !strings.HasPrefix(source, "github.com/") {
		return "", "", fmt.Errorf("source must be in format 'github.com/{owner}/{repo}'")
	}

	// Extract path after github.com/
	path := strings.TrimPrefix(source, "github.com/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("source must be in format 'github.com/{owner}/{repo}'")
	}

	return parts[0], parts[1], nil
}

// getLatestVersion fetches the latest release version from GitHub API.
func (d *Downloader) getLatestVersion(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode release: %w", err)
	}

	// Strip leading 'v' if present
	version := strings.TrimPrefix(release.TagName, "v")
	return version, nil
}

// buildAssetName builds the asset filename for the current platform.
// Format: {repo}-{os}-{arch}[.exe]
func buildAssetName(repo string) string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	name := fmt.Sprintf("%s-%s-%s", repo, os, arch)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// buildDownloadURL constructs the GitHub release download URL.
// Format: https://github.com/{owner}/{repo}/releases/download/v{version}/{assetName}
func buildDownloadURL(owner, repo, version, assetName string) string {
	// Ensure version has 'v' prefix for the tag
	tag := version
	if !strings.HasPrefix(version, "v") {
		tag = "v" + version
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", owner, repo, tag, assetName)
}

// download downloads a file from a URL to the destination path.
// Makes the file executable on Unix systems.
func (d *Downloader) download(url, destPath string) error {
	resp, err := d.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy content
	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(destPath)
		return err
	}

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(destPath, 0755); err != nil {
			os.Remove(destPath)
			return fmt.Errorf("failed to make executable: %w", err)
		}
	}

	return nil
}

// GetDefaultPluginDir returns the default plugin directory (~/.tfbreak.d/plugins).
func GetDefaultPluginDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, HomePluginDir)
}

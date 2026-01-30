package plugin

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// InstallConfig contains configuration for installing a plugin.
type InstallConfig struct {
	Name       string
	Source     string
	Version    string
	SigningKey string // PGP public key for signature verification (optional)
}

// InstallResult contains the result of a plugin installation.
type InstallResult struct {
	Name      string
	Version   string
	Path      string
	Installed bool // true if newly installed, false if already present
}

// Install downloads and installs a plugin from any supported source.
func (c *InstallConfig) Install(pluginDir string) (*InstallResult, error) {
	// Check if already installed
	installPath := c.InstallPath(pluginDir)
	if c.IsInstalled(pluginDir) {
		return &InstallResult{
			Name:      c.Name,
			Version:   c.Version,
			Path:      installPath,
			Installed: false,
		}, nil
	}

	// Create source from URL (auto-detects source type)
	source, err := NewSource(c.Source)
	if err != nil {
		return nil, err
	}

	// Get release
	release, err := source.GetRelease(c.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get release v%s: %w", c.Version, err)
	}

	// Download checksums.txt
	checksumAsset := release.FindAsset("checksums.txt")
	if checksumAsset == nil {
		return nil, fmt.Errorf("checksums.txt not found in release v%s", c.Version)
	}

	var checksumBuf bytes.Buffer
	if err := source.DownloadAsset(checksumAsset, &checksumBuf); err != nil {
		return nil, fmt.Errorf("failed to download checksums.txt: %w", err)
	}

	// Verify signature if signing key is configured or built-in key exists
	signingKey := GetSigningKey(c.SigningKey, c.Source)
	if signingKey != "" {
		// Download signature file
		sigAsset := release.FindAsset("checksums.txt.sig")
		if sigAsset == nil {
			return nil, fmt.Errorf("checksums.txt.sig not found in release v%s (required when signing_key is configured)", c.Version)
		}

		var sigBuf bytes.Buffer
		if err := source.DownloadAsset(sigAsset, &sigBuf); err != nil {
			return nil, fmt.Errorf("failed to download checksums.txt.sig: %w", err)
		}

		// Verify signature
		verifier, err := NewSignatureVerifier(signingKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse signing key: %w", err)
		}

		if err := verifier.VerifyBytes(checksumBuf.Bytes(), sigBuf.Bytes()); err != nil {
			return nil, fmt.Errorf("signature verification failed for plugin %s: %w", c.Name, err)
		}
	}

	checksummer, err := ParseChecksums(bytes.NewReader(checksumBuf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse checksums.txt: %w", err)
	}

	// Find the correct asset for this OS/arch
	assetName := c.AssetName()
	pluginAsset := release.FindAsset(assetName)
	if pluginAsset == nil {
		return nil, fmt.Errorf("asset %s not found in release v%s", assetName, c.Version)
	}

	// Download plugin zip to temp file
	tmpFile, err := os.CreateTemp("", "tfbreak-plugin-*.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := source.DownloadAsset(pluginAsset, tmpFile); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to download plugin: %w", err)
	}
	tmpFile.Close()

	// Verify checksum
	tmpFile, err = os.Open(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp file for verification: %w", err)
	}
	if err := checksummer.Verify(assetName, tmpFile); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("checksum verification failed: %w", err)
	}
	tmpFile.Close()

	// Create install directory
	installDir := filepath.Dir(installPath)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Extract binary from zip
	if err := extractPluginFromZip(tmpPath, c.BinaryName(), installPath); err != nil {
		return nil, fmt.Errorf("failed to extract plugin: %w", err)
	}

	// Set executable permissions on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(installPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to set executable permissions: %w", err)
		}
	}

	return &InstallResult{
		Name:      c.Name,
		Version:   c.Version,
		Path:      installPath,
		Installed: true,
	}, nil
}

// IsInstalled checks if the plugin is already installed at the correct version.
func (c *InstallConfig) IsInstalled(pluginDir string) bool {
	installPath := c.InstallPath(pluginDir)
	info, err := os.Stat(installPath)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// InstallPath returns the full path where the plugin will be installed.
// Format: [pluginDir]/[source]/[version]/tfbreak-ruleset-[name]
func (c *InstallConfig) InstallPath(pluginDir string) string {
	binaryName := c.BinaryName()
	return filepath.Join(pluginDir, c.Source, c.Version, binaryName)
}

// AssetName returns the expected asset name for the current OS/arch.
func (c *InstallConfig) AssetName() string {
	ext := ".zip"
	return fmt.Sprintf("%s%s_%s_%s%s", PluginPrefix, c.Name, runtime.GOOS, runtime.GOARCH, ext)
}

// BinaryName returns the expected binary name inside the zip.
func (c *InstallConfig) BinaryName() string {
	name := PluginPrefix + c.Name
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// extractPluginFromZip extracts a specific file from a zip archive.
func extractPluginFromZip(zipPath, fileName, destPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// Handle both exact match and path inside zip
		baseName := filepath.Base(f.Name)
		if baseName != fileName && f.Name != fileName {
			continue
		}

		if f.FileInfo().IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}
		defer rc.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create destination file: %w", err)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, rc); err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}

		return nil
	}

	return fmt.Errorf("file %s not found in zip", fileName)
}

// GetDefaultPluginDir returns the default plugin directory.
// Priority: TFBREAK_PLUGIN_DIR > ~/.tfbreak.d/plugins
func GetDefaultPluginDir() string {
	if dir := os.Getenv(PluginDirEnv); dir != "" {
		return dir
	}

	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, HomePluginDir)
	}

	// Fallback to current directory
	return LocalPluginDir
}

// InstallPlugins installs multiple plugins and returns the results.
func InstallPlugins(configs []InstallConfig, pluginDir string) ([]InstallResult, error) {
	var results []InstallResult
	var errors []string

	for _, cfg := range configs {
		result, err := cfg.Install(pluginDir)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", cfg.Name, err))
			continue
		}
		results = append(results, *result)
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("failed to install some plugins:\n  %s", strings.Join(errors, "\n  "))
	}

	return results, nil
}

// Package plugin provides plugin discovery and runner implementation for tfbreak.
// Types and interfaces are imported from github.com/jokarl/tfbreak-plugin-sdk.
package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jokarl/tfbreak-core/internal/config"
)

const (
	// PluginPrefix is the naming convention for tfbreak plugins.
	PluginPrefix = "tfbreak-ruleset-"

	// PluginDirEnv is the environment variable for plugin directory.
	PluginDirEnv = "TFBREAK_PLUGIN_DIR"

	// LocalPluginDir is the local plugin directory relative to cwd.
	LocalPluginDir = ".tfbreak.d/plugins"

	// HomePluginDir is the home directory plugin path.
	HomePluginDir = ".tfbreak.d/plugins"
)

// PluginInfo contains metadata about a discovered plugin.
type PluginInfo struct {
	// Name is the plugin name without prefix (e.g., "azurerm").
	Name string
	// Path is the full path to the plugin binary.
	Path string
	// Enabled indicates whether the plugin is enabled in config.
	Enabled bool
}

// Discover finds plugins from configured paths.
// Priority: config.PluginDir > TFBREAK_PLUGIN_DIR > ./.tfbreak.d/plugins > ~/.tfbreak.d/plugins
func Discover(cfg *config.Config) ([]PluginInfo, error) {
	var plugins []PluginInfo
	seen := make(map[string]bool)

	paths := getSearchPaths(cfg)

	for _, dir := range paths {
		found, err := discoverInDir(dir)
		if err != nil {
			// Skip directories that don't exist or can't be read
			continue
		}

		for _, p := range found {
			if seen[p.Name] {
				continue // First found takes precedence
			}
			seen[p.Name] = true

			// Check if enabled in config
			p.Enabled = cfg.IsPluginEnabled(p.Name)
			plugins = append(plugins, p)
		}
	}

	return plugins, nil
}

// getSearchPaths returns the list of paths to search for plugins in priority order.
func getSearchPaths(cfg *config.Config) []string {
	var paths []string

	// 1. Config plugin_dir (highest priority)
	if dir := cfg.GetPluginDir(); dir != "" {
		paths = append(paths, dir)
	}

	// 2. Environment variable
	if dir := os.Getenv(PluginDirEnv); dir != "" {
		paths = append(paths, dir)
	}

	// 3. Local plugin directory
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, LocalPluginDir))
	}

	// 4. Home directory
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, HomePluginDir))
	}

	return paths
}

// discoverInDir finds plugins in a specific directory.
func discoverInDir(dir string) ([]PluginInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var plugins []PluginInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check naming convention
		if !strings.HasPrefix(name, PluginPrefix) {
			continue
		}

		// Strip prefix and extension to get plugin name
		pluginName := strings.TrimPrefix(name, PluginPrefix)
		pluginName = stripExecutableExtension(pluginName)

		if pluginName == "" {
			continue
		}

		fullPath := filepath.Join(dir, name)

		// Verify it's executable
		if !isExecutable(fullPath) {
			continue
		}

		plugins = append(plugins, PluginInfo{
			Name: pluginName,
			Path: fullPath,
		})
	}

	return plugins, nil
}

// stripExecutableExtension removes OS-specific executable extensions.
func stripExecutableExtension(name string) string {
	if runtime.GOOS == "windows" {
		name = strings.TrimSuffix(name, ".exe")
	}
	return name
}

// isExecutable checks if a file is executable.
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		// On Windows, check for .exe extension
		return strings.HasSuffix(strings.ToLower(path), ".exe")
	}

	// On Unix, check execute permission
	mode := info.Mode()
	return mode.IsRegular() && (mode.Perm()&0111) != 0
}

// GetEnabledPlugins returns only enabled plugins from the discovered list.
func GetEnabledPlugins(plugins []PluginInfo) []PluginInfo {
	var enabled []PluginInfo
	for _, p := range plugins {
		if p.Enabled {
			enabled = append(enabled, p)
		}
	}
	return enabled
}

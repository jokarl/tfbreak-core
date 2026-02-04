package cli

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/jokarl/tfbreak-core/internal/config"
	"github.com/jokarl/tfbreak-core/plugin"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Manage tfbreak plugins",
	Long:  `Commands for managing tfbreak plugins.`,
}

var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered plugins",
	Long: `List all discovered plugins and their status.

Plugins are discovered from the following locations (in priority order):
1. plugin_dir setting in .tfbreak.hcl
2. TFBREAK_PLUGIN_DIR environment variable
3. ./.tfbreak.d/plugins/ (local)
4. ~/.tfbreak.d/plugins/ (home)

Plugins must be named 'tfbreak-ruleset-{name}' to be discovered.`,
	RunE: runPluginsList,
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install <source>[@version]",
	Short: "Install a plugin from GitHub releases",
	Long: `Install a plugin from GitHub releases.

The source must be in the format 'github.com/{owner}/{repo}'.
An optional version can be specified with @version (e.g., @0.2.0).
If no version is specified, the latest release is downloaded.

Examples:
  tfbreak plugins install github.com/jokarl/tfbreak-ruleset-azurerm
  tfbreak plugins install github.com/jokarl/tfbreak-ruleset-azurerm@0.2.0`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInstall,
}

var (
	pluginsListConfigPath    string
	pluginInstallConfigPath  string
)

func init() {
	rootCmd.AddCommand(pluginsCmd)
	pluginsCmd.AddCommand(pluginsListCmd)
	pluginsCmd.AddCommand(pluginInstallCmd)

	pluginsListCmd.Flags().StringVarP(&pluginsListConfigPath, "config", "c", "", "Path to config file")
	pluginInstallCmd.Flags().StringVarP(&pluginInstallConfigPath, "config", "c", "", "Path to config file")
}

func runPluginsList(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load(pluginsListConfigPath, "")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Discover plugins
	plugins, err := plugin.Discover(cfg)
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	if len(plugins) == 0 {
		fmt.Println("No plugins discovered.")
		fmt.Println()
		fmt.Println("Plugins are searched in these locations (in priority order):")
		fmt.Println("  1. plugin_dir setting in .tfbreak.hcl")
		fmt.Println("  2. TFBREAK_PLUGIN_DIR environment variable")
		fmt.Println("  3. ./.tfbreak.d/plugins/")
		fmt.Println("  4. ~/.tfbreak.d/plugins/")
		fmt.Println()
		fmt.Println("Plugin binaries must be named 'tfbreak-ruleset-{name}'.")
		return nil
	}

	fmt.Printf("Discovered %d plugin(s):\n\n", len(plugins))

	enabledColor := color.New(color.FgGreen)
	disabledColor := color.New(color.FgYellow)

	for _, p := range plugins {
		status := "enabled"
		statusColor := enabledColor
		if !p.Enabled {
			status = "disabled"
			statusColor = disabledColor
		}

		fmt.Printf("  %s\n", p.Name)
		fmt.Printf("    Path:   %s\n", p.Path)
		statusColor.Printf("    Status: %s\n", status)
		fmt.Println()
	}

	// Summary
	enabled := plugin.GetEnabledPlugins(plugins)
	fmt.Printf("Total: %d discovered, %d enabled\n", len(plugins), len(enabled))

	return nil
}

func runPluginInstall(cmd *cobra.Command, args []string) error {
	// Parse source and optional version from argument
	arg := args[0]
	source, version := parseSourceArg(arg)

	// Load config to get plugin directory
	cfg, err := config.Load(pluginInstallConfigPath, "")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine plugin directory
	pluginDir := cfg.GetPluginDir()
	if pluginDir == "" {
		pluginDir = plugin.GetDefaultPluginDir()
	}

	// Create downloader
	downloader := plugin.NewDownloader(pluginDir)

	// Display what we're doing
	if version == "" || version == "latest" {
		fmt.Printf("Installing %s (latest)...\n", source)
	} else {
		fmt.Printf("Installing %s@%s...\n", source, version)
	}

	// Download the plugin
	if err := downloader.Download(source, version); err != nil {
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	fmt.Printf("Plugin installed successfully to %s\n", pluginDir)
	return nil
}

// parseSourceArg parses a source[@version] argument.
// Returns the source and version (empty string if not specified).
func parseSourceArg(arg string) (source, version string) {
	// Check for @version suffix
	if idx := strings.LastIndex(arg, "@"); idx != -1 {
		// Make sure @ is not part of the source (e.g., not in the middle of the path)
		potentialVersion := arg[idx+1:]
		// Version should not contain / (which would indicate it's part of the path)
		if !strings.Contains(potentialVersion, "/") && potentialVersion != "" {
			return arg[:idx], potentialVersion
		}
	}
	return arg, ""
}

package cli

import (
	"fmt"

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

var (
	pluginsListConfigPath string
)

func init() {
	rootCmd.AddCommand(pluginsCmd)
	pluginsCmd.AddCommand(pluginsListCmd)

	pluginsListCmd.Flags().StringVarP(&pluginsListConfigPath, "config", "c", "", "Path to config file")
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

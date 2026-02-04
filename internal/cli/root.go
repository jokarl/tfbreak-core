package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jokarl/tfbreak-core/internal/config"
	"github.com/jokarl/tfbreak-core/plugin"
)

var (
	versionStr string
	commitStr  string
	dateStr    string
)

var (
	versionFlag bool
	initFlag    bool
)

func SetVersionInfo(version, commit, date string) {
	versionStr = version
	commitStr = commit
	dateStr = date
}

var rootCmd = &cobra.Command{
	Use:   "tfbreak",
	Short: "Terraform breaking change detector",
	Long: `tfbreak compares two Terraform configurations and reports breaking changes
to module interfaces and state safety.

It performs static analysis of .tf files to detect changes that would break
callers or destroy state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle --version flag
		if versionFlag {
			printVersion()
			return nil
		}
		// Handle --init flag
		if initFlag {
			return runInit()
		}
		// No flags specified, show help
		return cmd.Help()
	},
}

func init() {
	// Disable default help command (keep -h/--help flags on subcommands)
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version information")
	rootCmd.Flags().BoolVar(&initFlag, "init", false, "Install configured plugins")
}

func Execute() error {
	return rootCmd.Execute()
}

func printVersion() {
	fmt.Printf("tfbreak version %s\n", versionStr)
	if commitStr != "none" && commitStr != "" {
		fmt.Printf("  commit: %s\n", commitStr)
	}
	if dateStr != "unknown" && dateStr != "" {
		fmt.Printf("  built:  %s\n", dateStr)
	}
}

// runInit loads config and installs plugins
func runInit() error {
	// Load config (from .tfbreak.hcl or defaults)
	cfg, err := config.Load("", "")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	return initializePlugins(cfg)
}

// initializePlugins downloads plugins that have a source configured.
func initializePlugins(cfg *config.Config) error {
	pluginDir := cfg.GetPluginDir()
	if pluginDir == "" {
		pluginDir = plugin.GetDefaultPluginDir()
	}

	discovered, _ := plugin.Discover(cfg)
	discoveredNames := make(map[string]bool)
	for _, p := range discovered {
		discoveredNames[p.Name] = true
	}

	var installedCount int
	for _, pc := range cfg.Plugins {
		if pc.Enabled != nil && !*pc.Enabled {
			continue
		}
		if pc.Source == "" {
			continue
		}
		if discoveredNames[pc.Name] {
			fmt.Printf("Plugin %s already installed\n", pc.Name)
			continue
		}

		version := pc.Version
		if version == "" {
			version = "latest"
		}

		fmt.Printf("Installing plugin %s...\n", pc.Name)
		downloader := plugin.NewDownloader(pluginDir)
		if err := downloader.Download(pc.Source, version); err != nil {
			return fmt.Errorf("failed to install plugin %s: %w", pc.Name, err)
		}
		fmt.Printf("Installed plugin %s\n", pc.Name)
		installedCount++
	}

	if installedCount == 0 {
		fmt.Println("All plugins already installed")
	}

	return nil
}

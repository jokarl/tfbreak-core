package cli

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/config"
	"github.com/jokarl/tfbreak-core/plugin"
)

// runPluginInit handles the --init flag for downloading plugins.
func runPluginInit() error {
	// Load configuration
	cfg, err := config.Load("", ".")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get plugins that need installation (have source and version)
	var toInstall []plugin.InstallConfig
	for _, pc := range cfg.Plugins {
		if pc.Source == "" || pc.Version == "" {
			continue
		}
		// Skip disabled plugins
		if pc.Enabled != nil && !*pc.Enabled {
			continue
		}
		toInstall = append(toInstall, plugin.InstallConfig{
			Name:       pc.Name,
			Source:     pc.Source,
			Version:    pc.Version,
			SigningKey: pc.SigningKey,
		})
	}

	if len(toInstall) == 0 {
		fmt.Println("No plugins configured for automatic installation.")
		fmt.Println("Add plugins with 'source' and 'version' to .tfbreak.hcl:")
		fmt.Println("")
		fmt.Println("  plugin \"azurerm\" {")
		fmt.Println("    enabled = true")
		fmt.Println("    source  = \"github.com/jokarl/tfbreak-ruleset-azurerm\"")
		fmt.Println("    version = \"0.1.0\"")
		fmt.Println("  }")
		return nil
	}

	// Determine plugin directory
	pluginDir := cfg.GetPluginDir()
	if pluginDir == "" {
		pluginDir = plugin.GetDefaultPluginDir()
	}

	fmt.Printf("Installing plugins to %s\n\n", pluginDir)

	// Install each plugin
	var hasErrors bool
	for _, ic := range toInstall {
		fmt.Printf("  - %s v%s: ", ic.Name, ic.Version)

		result, err := ic.Install(pluginDir)
		if err != nil {
			fmt.Printf("ERROR\n    %v\n", err)
			hasErrors = true
			continue
		}

		if result.Installed {
			fmt.Printf("Installed\n")
		} else {
			fmt.Printf("Already installed\n")
		}
	}

	fmt.Println()

	if hasErrors {
		return fmt.Errorf("some plugins failed to install")
	}

	fmt.Println("Plugin installation complete.")
	return nil
}


package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jokarl/tfbreak-core/internal/config"
)

var forceFlag bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create starter .tfbreak.hcl configuration",
	Long: `Create a new .tfbreak.hcl configuration file in the current directory
with documented default settings.

The generated configuration includes comments explaining each option,
making it easy to customize for your project's needs.`,
	Args: cobra.NoArgs,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVar(&forceFlag, "force", false, "Overwrite existing configuration file")
}

func runInit(cmd *cobra.Command, args []string) error {
	configPath := filepath.Join(".", ".tfbreak.hcl")

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		if !forceFlag {
			return fmt.Errorf("configuration file already exists: %s (use --force to overwrite)", configPath)
		}
	}

	// Write the default configuration
	content := config.DefaultConfigHCL()
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("Created %s\n", configPath)
	return nil
}

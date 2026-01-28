package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print the version, commit, and build date of tfbreak.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tfbreak version %s\n", versionStr)
		if commitStr != "none" && commitStr != "" {
			fmt.Printf("  commit: %s\n", commitStr)
		}
		if dateStr != "unknown" && dateStr != "" {
			fmt.Printf("  built:  %s\n", dateStr)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

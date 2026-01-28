package cli

import (
	"github.com/spf13/cobra"
)

var (
	versionStr string
	commitStr  string
	dateStr    string
)

// SetVersionInfo sets the version information for the CLI
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
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags will be added here
}

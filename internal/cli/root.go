package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	versionStr string
	commitStr  string
	dateStr    string
)

// Global flags
var initFlag bool

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
	// Handle --init flag before running any subcommand
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// --init is handled specially - it runs and exits
		if initFlag {
			if err := runPluginInit(); err != nil {
				return err
			}
			os.Exit(0)
		}
		return nil
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global --init flag for plugin installation (tflint-aligned)
	rootCmd.PersistentFlags().BoolVar(&initFlag, "init", false, "Download plugins configured in .tfbreak.hcl")

	// Custom usage template to show --init prominently
	rootCmd.SetUsageTemplate(usageTemplate())
}

// usageTemplate returns a custom usage template that highlights --init.
func usageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}

Plugin Installation:
  {{.CommandPath}} --init    Download plugins configured in .tfbreak.hcl{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
}

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jokarl/tfbreak-core/internal/loader"
	"github.com/jokarl/tfbreak-core/internal/output"
	"github.com/jokarl/tfbreak-core/internal/rules"
	"github.com/jokarl/tfbreak-core/internal/types"
)

var (
	formatFlag  string
	outputFlag  string
	failOnFlag  string
	colorFlag   string
	quietFlag   bool
	verboseFlag bool
)

var checkCmd = &cobra.Command{
	Use:   "check <old_dir> <new_dir>",
	Short: "Compare directories and evaluate policy",
	Long: `Compare an "old" and "new" Terraform configuration directory,
extract structural signatures, and report changes that would break
callers or destroy state.`,
	Args: cobra.ExactArgs(2),
	RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Flags().StringVar(&formatFlag, "format", "text", "Output format: text, json")
	checkCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Write output to file instead of stdout")
	checkCmd.Flags().StringVar(&failOnFlag, "fail-on", "BREAKING", "Fail on severity: BREAKING, RISKY, INFO")
	checkCmd.Flags().StringVar(&colorFlag, "color", "auto", "Color mode: auto, always, never")
	checkCmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress non-error output")
	checkCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose output")
}

func runCheck(cmd *cobra.Command, args []string) error {
	oldDir := args[0]
	newDir := args[1]

	// Parse fail-on severity
	failOn, err := types.ParseSeverity(failOnFlag)
	if err != nil {
		return fmt.Errorf("invalid --fail-on value: %w", err)
	}

	// Load old config
	oldSnapshot, err := loader.Load(oldDir)
	if err != nil {
		return fmt.Errorf("failed to load old config: %w", err)
	}

	// Load new config
	newSnapshot, err := loader.Load(newDir)
	if err != nil {
		return fmt.Errorf("failed to load new config: %w", err)
	}

	// Create and run engine
	engine := rules.NewDefaultEngine()
	result := engine.Check(oldDir, newDir, oldSnapshot, newSnapshot, failOn)

	// Determine output writer
	var writer *os.File
	if outputFlag != "" {
		f, err := os.Create(outputFlag)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		writer = f
	} else {
		writer = os.Stdout
	}

	// Skip output if quiet and no findings
	if !quietFlag || result.Result == "FAIL" {
		// Determine color mode
		colorEnabled := shouldUseColor(writer)

		// Create renderer and output
		format := output.Format(formatFlag)
		renderer := output.NewRenderer(format, colorEnabled)
		if err := renderer.Render(writer, result); err != nil {
			return fmt.Errorf("failed to render output: %w", err)
		}
	}

	// Set exit code based on result
	if result.Result == "FAIL" {
		os.Exit(1)
	}

	return nil
}

func shouldUseColor(f *os.File) bool {
	switch colorFlag {
	case "always":
		return true
	case "never":
		return false
	default: // auto
		// Check if stdout is a terminal
		stat, err := f.Stat()
		if err != nil {
			return false
		}
		return (stat.Mode() & os.ModeCharDevice) != 0
	}
}

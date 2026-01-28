package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jokarl/tfbreak-core/internal/rules"
)

var explainCmd = &cobra.Command{
	Use:   "explain <rule_id>",
	Short: "Show rule documentation",
	Long: `Show detailed documentation for a specific rule, including:
- Rule ID and name
- Default severity
- Description
- Example code (before and after)
- Remediation guidance

Example:
  tfbreak explain BC001`,
	Args: cobra.ExactArgs(1),
	RunE: runExplain,
}

func init() {
	rootCmd.AddCommand(explainCmd)
}

func runExplain(cmd *cobra.Command, args []string) error {
	ruleID := strings.ToUpper(args[0])

	doc := rules.GetDocumentation(ruleID)
	if doc == nil {
		fmt.Fprintf(os.Stderr, "Error: unknown rule ID: %s\n", ruleID)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Available rules:")
		for _, id := range rules.DefaultRegistry.IDs() {
			r, _ := rules.DefaultRegistry.Get(id)
			fmt.Fprintf(os.Stderr, "  %s  %s\n", id, r.Name())
		}
		os.Exit(2)
	}

	// Print the documentation
	fmt.Printf("%s: %s\n", doc.ID, doc.Name)
	fmt.Printf("Severity: %s\n", doc.DefaultSeverity)
	fmt.Println()
	fmt.Println(doc.Description)
	fmt.Println()

	if doc.ExampleOld != "" || doc.ExampleNew != "" {
		fmt.Println("Example:")
		fmt.Println()
		if doc.ExampleOld != "" {
			fmt.Println("Old configuration:")
			fmt.Println(indent(doc.ExampleOld, "  "))
			fmt.Println()
		}
		if doc.ExampleNew != "" {
			fmt.Println("New configuration:")
			fmt.Println(indent(doc.ExampleNew, "  "))
			fmt.Println()
		}
	}

	if doc.Remediation != "" {
		fmt.Println("Remediation:")
		fmt.Println(indent(doc.Remediation, "  "))
	}

	return nil
}

// indent adds a prefix to each line of text
func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

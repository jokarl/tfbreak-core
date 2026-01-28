package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jokarl/tfbreak-core/internal/annotation"
	"github.com/jokarl/tfbreak-core/internal/config"
	"github.com/jokarl/tfbreak-core/internal/loader"
	"github.com/jokarl/tfbreak-core/internal/output"
	"github.com/jokarl/tfbreak-core/internal/pathfilter"
	"github.com/jokarl/tfbreak-core/internal/rules"
	"github.com/jokarl/tfbreak-core/internal/types"
)

var (
	// Output flags
	formatFlag  string
	outputFlag  string
	colorFlag   string
	quietFlag   bool
	verboseFlag bool

	// Policy flags
	failOnFlag    string
	enableFlag    []string
	disableFlag   []string
	severityFlags []string

	// Path flags
	configFlag  string
	includeFlag []string
	excludeFlag []string

	// Annotation flags
	noAnnotationsFlag bool
	requireReasonFlag bool
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

	// Output flags
	checkCmd.Flags().StringVar(&formatFlag, "format", "", "Output format: text, json")
	checkCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Write output to file instead of stdout")
	checkCmd.Flags().StringVar(&colorFlag, "color", "", "Color mode: auto, always, never")
	checkCmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress non-error output")
	checkCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose output")

	// Policy flags
	checkCmd.Flags().StringVar(&failOnFlag, "fail-on", "", "Fail on severity: BREAKING, RISKY, INFO")
	checkCmd.Flags().StringSliceVar(&enableFlag, "enable", nil, "Enable rules (comma-separated)")
	checkCmd.Flags().StringSliceVar(&disableFlag, "disable", nil, "Disable rules (comma-separated)")
	checkCmd.Flags().StringSliceVar(&severityFlags, "severity", nil, "Override rule severity (RULE=SEV)")

	// Config and path flags
	checkCmd.Flags().StringVarP(&configFlag, "config", "c", "", "Path to config file")
	checkCmd.Flags().StringSliceVar(&includeFlag, "include", nil, "Include patterns (overrides config)")
	checkCmd.Flags().StringSliceVar(&excludeFlag, "exclude", nil, "Exclude patterns (overrides config)")

	// Annotation flags
	checkCmd.Flags().BoolVar(&noAnnotationsFlag, "no-annotations", false, "Disable annotation processing")
	checkCmd.Flags().BoolVar(&requireReasonFlag, "require-reason", false, "Require reason in annotations")
}

func runCheck(cmd *cobra.Command, args []string) error {
	oldDir := args[0]
	newDir := args[1]

	// Load configuration
	cfg, err := config.Load(configFlag, oldDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply CLI flag overrides
	applyFlagOverrides(cfg)

	// Parse fail-on severity
	failOn, err := types.ParseSeverity(cfg.Policy.FailOn)
	if err != nil {
		return fmt.Errorf("invalid fail_on value: %w", err)
	}

	// Create path filter
	filter := pathfilter.New(cfg.Paths.Include, cfg.Paths.Exclude)

	// Load old config with path filtering
	oldSnapshot, err := loader.LoadWithFilter(oldDir, filter)
	if err != nil {
		return fmt.Errorf("failed to load old config: %w", err)
	}

	// Load new config with path filtering
	newSnapshot, err := loader.LoadWithFilter(newDir, filter)
	if err != nil {
		return fmt.Errorf("failed to load new config: %w", err)
	}

	// Create and configure engine
	engine := rules.NewDefaultEngine()
	configureEngine(engine, cfg)

	// Run rules
	result := engine.Check(oldDir, newDir, oldSnapshot, newSnapshot, failOn)

	// Process annotations if enabled
	if cfg.IsAnnotationsEnabled() && !noAnnotationsFlag {
		if err := processAnnotations(newDir, filter, cfg, result); err != nil {
			// Log warning but don't fail
			if verboseFlag {
				fmt.Fprintf(os.Stderr, "Warning: failed to process annotations: %v\n", err)
			}
		}
	}

	// Recompute result after annotation processing
	result.Compute()

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
		colorEnabled := shouldUseColor(writer, cfg.Output.Color)

		// Create renderer and output
		format := output.Format(cfg.Output.Format)
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

// applyFlagOverrides applies CLI flags to the config, with flags taking precedence
func applyFlagOverrides(cfg *config.Config) {
	// Output overrides
	if formatFlag != "" {
		cfg.Output.Format = formatFlag
	}
	if colorFlag != "" {
		cfg.Output.Color = colorFlag
	}

	// Policy overrides
	if failOnFlag != "" {
		cfg.Policy.FailOn = failOnFlag
	}

	// Path overrides (replace entirely, don't merge)
	if len(includeFlag) > 0 {
		cfg.Paths.Include = includeFlag
	}
	if len(excludeFlag) > 0 {
		cfg.Paths.Exclude = excludeFlag
	}

	// Annotation overrides
	if requireReasonFlag {
		cfg.Annotations.RequireReason = true
	}
}

// configureEngine applies config settings to the rules engine
func configureEngine(engine *rules.Engine, cfg *config.Config) {
	// Apply rule configurations from config file
	for _, rc := range cfg.Rules {
		if rc.Enabled != nil {
			if *rc.Enabled {
				engine.EnableRule(rc.ID)
			} else {
				engine.DisableRule(rc.ID)
			}
		}
		if rc.Severity != nil {
			sev, err := types.ParseSeverity(*rc.Severity)
			if err == nil {
				ruleCfg := engine.GetConfig(rc.ID)
				if ruleCfg != nil {
					ruleCfg.Severity = sev
					engine.SetConfig(rc.ID, ruleCfg)
				}
			}
		}
	}

	// Apply CLI enable/disable flags (these take precedence)
	for _, ruleID := range enableFlag {
		ruleID = strings.TrimSpace(strings.ToUpper(ruleID))
		engine.EnableRule(ruleID)
	}
	for _, ruleID := range disableFlag {
		ruleID = strings.TrimSpace(strings.ToUpper(ruleID))
		engine.DisableRule(ruleID)
	}

	// Apply CLI severity overrides
	for _, s := range severityFlags {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			continue
		}
		ruleID := strings.TrimSpace(strings.ToUpper(parts[0]))
		sevStr := strings.TrimSpace(strings.ToUpper(parts[1]))
		sev, err := types.ParseSeverity(sevStr)
		if err != nil {
			continue
		}
		ruleCfg := engine.GetConfig(ruleID)
		if ruleCfg != nil {
			ruleCfg.Severity = sev
			engine.SetConfig(ruleID, ruleCfg)
		}
	}
}

// processAnnotations parses annotations and matches them to findings
func processAnnotations(dir string, filter *pathfilter.Filter, cfg *config.Config, result *types.CheckResult) error {
	var allAnnotations []*annotation.Annotation
	blockStarts := make(map[string]map[int]string)

	// Create a resolver from the rules registry
	resolver := annotation.NewRegistryResolver(rules.DefaultRegistry.NameToIDMap())
	parser := annotation.NewParser(resolver)

	// Parse annotations from all files
	err := filter.WalkDir(dir, func(path string, d os.DirEntry) error {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		anns, err := parser.ParseFile(path, src)
		if err != nil {
			return err
		}
		allAnnotations = append(allAnnotations, anns...)

		blocks, err := annotation.FindBlockStarts(path, src)
		if err == nil {
			blockStarts[path] = blocks
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Create matcher
	matcher := annotation.NewMatcher(allAnnotations, blockStarts)

	// Create governance config
	govCfg := annotation.GovernanceConfig{
		Enabled:       true,
		RequireReason: cfg.Annotations.RequireReason,
		AllowRuleIDs:  cfg.Annotations.AllowRuleIDs,
		DenyRuleIDs:   cfg.Annotations.DenyRuleIDs,
	}

	// Match annotations to findings
	for _, finding := range result.Findings {
		matchResult := matcher.Match(finding)
		if !matchResult.Matched {
			continue
		}

		ann := matchResult.Annotation

		// Check governance
		violation := annotation.CheckGovernance(ann, govCfg)
		if violation != nil {
			// Add governance violation as a warning to the finding
			finding.Detail = fmt.Sprintf("%s (governance: %s)", finding.Detail, violation.Message)
			continue
		}

		// Check if annotation is expired
		if ann.IsExpired() {
			continue
		}

		// Mark finding as ignored
		finding.Ignored = true
		finding.IgnoreReason = ann.Reason
		if finding.IgnoreReason == "" && ann.Ticket != "" {
			finding.IgnoreReason = ann.Ticket
		}
	}

	return nil
}

func shouldUseColor(f *os.File, colorMode string) bool {
	switch colorMode {
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

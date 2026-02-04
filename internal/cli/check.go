package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/jokarl/tfbreak-core/internal/annotation"
	"github.com/jokarl/tfbreak-core/internal/config"
	"github.com/jokarl/tfbreak-core/internal/git"
	"github.com/jokarl/tfbreak-core/internal/loader"
	"github.com/jokarl/tfbreak-core/internal/output"
	"github.com/jokarl/tfbreak-core/internal/pathfilter"
	"github.com/jokarl/tfbreak-core/internal/rules"
	"github.com/jokarl/tfbreak-core/internal/types"
	"github.com/jokarl/tfbreak-core/plugin"
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
	onlyFlag      []string

	// Path flags
	configFlag    string
	includeFlag   []string
	excludeFlag   []string
	filterFlag    string
	recursiveFlag bool

	// Annotation flags
	noAnnotationsFlag bool
	requireReasonFlag bool

	// Output enhancement flags
	includeRemediationFlag bool

	// Git ref flags
	baseFlag string
	headFlag string
	repoFlag string
)

var checkCmd = &cobra.Command{
	Use:   "check [flags] [old_dir] [new_dir]",
	Short: "Compare directories and evaluate policy",
	Long: `Compare an "old" and "new" Terraform configuration directory,
extract structural signatures, and report changes that would break
callers or destroy state.

Git ref comparison modes:
  tfbreak check --base <ref[:path]> [new_dir]       Compare working dir against local ref
  tfbreak check --base <ref[:path]> --head <ref[:path]>    Compare two local refs
  tfbreak check --repo <url> --base <ref[:path]> --head <ref[:path]>  Compare two remote refs

The ref:path syntax (like git show) specifies a subdirectory within the ref.
If no path is specified, the repository root is used.

Examples:
  tfbreak check ./old ./new                    Directory mode
  tfbreak check --base main ./                 Compare ./ against main branch
  tfbreak check --base main:modules/vpc ./     Compare modules/vpc at main vs ./
  tfbreak check --base v1.0.0 --head v2.0.0    Compare two local tags
  tfbreak check --base v1:src --head v2:src    Compare src/ directory between tags
  tfbreak check --repo https://github.com/org/mod --base v1 --head v2  Remote mode`,
	Args: validateCheckArgs,
	RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)

	// Output flags
	checkCmd.Flags().StringVar(&formatFlag, "format", "", "Output format: text, json, compact, checkstyle, junit, sarif")
	checkCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Write output to file instead of stdout")
	checkCmd.Flags().StringVar(&colorFlag, "color", "", "Color mode: auto, always, never")
	checkCmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress non-error output")
	checkCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose output")

	// Policy flags
	checkCmd.Flags().StringVar(&failOnFlag, "minimum-failure-severity", "", "Minimum severity to fail: ERROR, WARNING, NOTICE")
	checkCmd.Flags().StringSliceVar(&enableFlag, "enable-rule", nil, "Enable rules by ID or name (can be repeated)")
	checkCmd.Flags().StringSliceVar(&disableFlag, "disable-rule", nil, "Disable rules by ID or name (can be repeated)")
	checkCmd.Flags().StringSliceVar(&severityFlags, "severity", nil, "Override rule severity (RULE=SEV)")
	checkCmd.Flags().StringSliceVar(&onlyFlag, "only", nil, "Run only these rules by ID or name (can be repeated)")

	// Config and path flags
	checkCmd.Flags().StringVarP(&configFlag, "config", "c", "", "Path to config file")
	checkCmd.Flags().StringSliceVar(&includeFlag, "include", nil, "Include patterns (overrides config)")
	checkCmd.Flags().StringSliceVar(&excludeFlag, "exclude", nil, "Exclude patterns (overrides config)")
	checkCmd.Flags().StringVar(&filterFlag, "filter", "", "Limit scan to specific directory")
	checkCmd.Flags().BoolVar(&recursiveFlag, "recursive", false, "Scan subdirectories containing .tf files")

	// Annotation flags
	checkCmd.Flags().BoolVar(&noAnnotationsFlag, "no-annotations", false, "Disable annotation processing")
	checkCmd.Flags().BoolVar(&requireReasonFlag, "require-reason", false, "Require reason in annotations")

	// Output enhancement flags
	checkCmd.Flags().BoolVar(&includeRemediationFlag, "include-remediation", false, "Include remediation guidance for each finding")

	// Git ref flags
	checkCmd.Flags().StringVar(&baseFlag, "base", "", "Git ref for old configuration (branch, tag, or commit)")
	checkCmd.Flags().StringVar(&headFlag, "head", "", "Git ref for new configuration (requires --base)")
	checkCmd.Flags().StringVar(&repoFlag, "repo", "", "Remote repository URL (requires --base)")
}

// checkMode represents the comparison mode
type checkMode int

const (
	modeDirectory    checkMode = iota // Two directory arguments
	modeLocalRef                      // --base with working directory
	modeTwoLocalRefs                  // --base and --head (local)
	modeRemoteRefs                    // --repo with --base and --head
	modeMixed                         // --repo with --base and local new_dir
)

// validateCheckArgs validates the command arguments based on flags
func validateCheckArgs(cmd *cobra.Command, args []string) error {
	hasBase := baseFlag != ""
	hasHead := headFlag != ""
	hasRepo := repoFlag != ""

	// --head requires --base
	if hasHead && !hasBase {
		return errors.New("--head requires --base to be specified")
	}

	// --repo requires --base
	if hasRepo && !hasBase {
		return errors.New("--repo requires --base to be specified")
	}

	// Determine expected arguments based on mode
	if hasRepo {
		if hasHead {
			// --repo --base --head: no positional args needed
			if len(args) > 0 {
				return errors.New("no positional arguments expected with --repo --base --head")
			}
		} else {
			// --repo --base: need new_dir
			if len(args) != 1 {
				return errors.New("exactly one positional argument (new_dir) required with --repo --base")
			}
		}
	} else if hasBase {
		if hasHead {
			// --base --head: no positional args needed
			if len(args) > 0 {
				return errors.New("no positional arguments expected with --base --head")
			}
		} else {
			// --base only: optional new_dir (defaults to .)
			if len(args) > 1 {
				return errors.New("at most one positional argument (new_dir) expected with --base")
			}
		}
	} else {
		// Directory mode: need exactly 2 args
		if len(args) != 2 {
			return errors.New("exactly two directory arguments required: <old_dir> <new_dir>")
		}
	}

	return nil
}

// determineMode returns the check mode based on flags
func determineMode() checkMode {
	hasBase := baseFlag != ""
	hasHead := headFlag != ""
	hasRepo := repoFlag != ""

	if hasRepo {
		if hasHead {
			return modeRemoteRefs
		}
		return modeMixed
	}
	if hasBase {
		if hasHead {
			return modeTwoLocalRefs
		}
		return modeLocalRef
	}
	return modeDirectory
}

func runCheck(cmd *cobra.Command, args []string) error {
	mode := determineMode()

	// For git modes, run pre-flight checks
	if mode != modeDirectory {
		if err := runPreflightChecks(mode); err != nil {
			// Exit code 2 for git-related errors
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(2)
		}
	}

	// Get old and new directories based on mode
	oldDir, newDir, cleanup, err := resolveDirectories(mode, args)
	if err != nil {
		if mode != modeDirectory {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(2)
		}
		return err
	}

	// Set up signal handling for cleanup
	if cleanup != nil {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			cleanup()
			os.Exit(130) // 128 + SIGINT
		}()
		defer cleanup()
	}

	// Apply --filter to narrow scan directory
	scanOldDir := oldDir
	scanNewDir := newDir
	if filterFlag != "" {
		scanOldDir = filepath.Join(oldDir, filterFlag)
		scanNewDir = filepath.Join(newDir, filterFlag)
	}

	// Handle recursive mode
	if recursiveFlag {
		return runRecursiveCheck(cmd, scanOldDir, scanNewDir, cleanup)
	}

	return runSingleCheck(scanOldDir, scanNewDir)
}

// runSingleCheck performs a check on a single directory pair
func runSingleCheck(oldDir, newDir string) error {
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

	// Run rules with options
	checkOpts := rules.CheckOptions{
		IncludeRemediation: includeRemediationFlag,
	}
	result := engine.CheckWithOptions(oldDir, newDir, oldSnapshot, newSnapshot, failOn, checkOpts)

	// Execute plugin rules if any plugins are configured
	if err := executePluginRules(cfg, oldDir, newDir, result, verboseFlag); err != nil {
		if verboseFlag {
			fmt.Fprintf(os.Stderr, "Warning: plugin execution error: %v\n", err)
		}
	}

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

// runRecursiveCheck finds all module directories and runs checks on each
func runRecursiveCheck(_ *cobra.Command, oldDir, newDir string, _ func()) error {
	// Find all directories with .tf files in newDir
	modules := findModuleDirs(newDir)
	if len(modules) == 0 {
		return fmt.Errorf("no directories containing .tf files found in %s", newDir)
	}

	// Load configuration once for common settings
	cfg, err := config.Load(configFlag, oldDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	applyFlagOverrides(cfg)

	failOn, err := types.ParseSeverity(cfg.Policy.FailOn)
	if err != nil {
		return fmt.Errorf("invalid fail_on value: %w", err)
	}

	// Aggregate results from all modules
	aggregatedResult := types.NewCheckResult(oldDir, newDir, failOn)
	filter := pathfilter.New(cfg.Paths.Include, cfg.Paths.Exclude)

	for _, modulePath := range modules {
		relPath, err := filepath.Rel(newDir, modulePath)
		if err != nil {
			relPath = modulePath
		}
		oldModulePath := filepath.Join(oldDir, relPath)

		// Skip if old module doesn't exist
		if _, err := os.Stat(oldModulePath); os.IsNotExist(err) {
			if verboseFlag {
				fmt.Fprintf(os.Stderr, "Skipping %s (not found in old directory)\n", relPath)
			}
			continue
		}

		if verboseFlag {
			fmt.Fprintf(os.Stderr, "Checking module: %s\n", relPath)
		}

		// Load snapshots for this module
		oldSnapshot, err := loader.LoadWithFilter(oldModulePath, filter)
		if err != nil {
			if verboseFlag {
				fmt.Fprintf(os.Stderr, "Warning: failed to load old config for %s: %v\n", relPath, err)
			}
			continue
		}

		newSnapshot, err := loader.LoadWithFilter(modulePath, filter)
		if err != nil {
			if verboseFlag {
				fmt.Fprintf(os.Stderr, "Warning: failed to load new config for %s: %v\n", relPath, err)
			}
			continue
		}

		// Create and configure engine for this module
		engine := rules.NewDefaultEngine()
		configureEngine(engine, cfg)

		// Run rules
		checkOpts := rules.CheckOptions{
			IncludeRemediation: includeRemediationFlag,
		}
		result := engine.CheckWithOptions(oldModulePath, modulePath, oldSnapshot, newSnapshot, failOn, checkOpts)

		// Add findings to aggregated result
		for _, finding := range result.Findings {
			aggregatedResult.AddFinding(finding)
		}
	}

	// Recompute aggregated result
	aggregatedResult.Compute()

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
	if !quietFlag || aggregatedResult.Result == "FAIL" {
		colorEnabled := shouldUseColor(writer, cfg.Output.Color)
		format := output.Format(cfg.Output.Format)
		renderer := output.NewRenderer(format, colorEnabled)
		if err := renderer.Render(writer, aggregatedResult); err != nil {
			return fmt.Errorf("failed to render output: %w", err)
		}
	}

	if aggregatedResult.Result == "FAIL" {
		os.Exit(1)
	}

	return nil
}

// findModuleDirs finds all directories containing .tf files under root
func findModuleDirs(root string) []string {
	var dirs []string
	seen := make(map[string]bool)

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip directories we can't access
		}
		if d.IsDir() {
			return nil
		}
		// Check if this is a .tf file
		if strings.HasSuffix(path, ".tf") {
			dir := filepath.Dir(path)
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
		return nil
	})

	return dirs
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

// resolveRuleID converts a rule identifier (ID or name) to its canonical ID.
// Examples: "BC001" -> "BC001", "required-input-added" -> "BC001"
func resolveRuleID(identifier string) string {
	identifier = strings.TrimSpace(identifier)

	// Try as ID first (uppercase)
	upper := strings.ToUpper(identifier)
	if _, ok := rules.DefaultRegistry.Get(upper); ok {
		return upper
	}

	// Try as name (case-insensitive kebab-case)
	lower := strings.ToLower(identifier)
	if rule, ok := rules.DefaultRegistry.GetByName(lower); ok {
		return rule.ID()
	}

	// Return as-is uppercase (will fail gracefully in engine)
	return upper
}

// configureEngine applies config settings to the rules engine
func configureEngine(engine *rules.Engine, cfg *config.Config) {
	// If --only is specified, disable all rules first, then enable only the specified ones
	if len(onlyFlag) > 0 {
		engine.DisableAllRules()
		for _, identifier := range onlyFlag {
			ruleID := resolveRuleID(identifier)
			engine.EnableRule(ruleID)
		}
		return // Skip other enable/disable logic
	}

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
	for _, identifier := range enableFlag {
		ruleID := resolveRuleID(identifier)
		engine.EnableRule(ruleID)
	}
	for _, identifier := range disableFlag {
		ruleID := resolveRuleID(identifier)
		engine.DisableRule(ruleID)
	}

	// Apply CLI severity overrides
	for _, s := range severityFlags {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			continue
		}
		ruleID := resolveRuleID(parts[0])
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

// refSpec represents a parsed ref:path specification
type refSpec struct {
	Ref  string
	Path string // Empty means repo root
}

// parseRefSpec parses a ref:path specification like "main:modules/vpc"
// If no path is specified, Path is empty (meaning repo root)
func parseRefSpec(spec string) refSpec {
	// Find the first colon that's not part of a Windows drive letter
	// and not part of a URL scheme (e.g., https://)
	idx := strings.Index(spec, ":")

	// Skip URL-like patterns (e.g., https://...)
	if idx > 0 && idx < len(spec)-2 && spec[idx+1] == '/' && spec[idx+2] == '/' {
		// This looks like a URL, no path component
		return refSpec{Ref: spec, Path: ""}
	}

	// Skip Windows drive letters (single letter before colon)
	if idx == 1 && len(spec) > 2 && (spec[0] >= 'A' && spec[0] <= 'Z' || spec[0] >= 'a' && spec[0] <= 'z') {
		// Looks like C:\... - no path component in ref sense
		return refSpec{Ref: spec, Path: ""}
	}

	if idx == -1 {
		return refSpec{Ref: spec, Path: ""}
	}

	return refSpec{
		Ref:  spec[:idx],
		Path: spec[idx+1:],
	}
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

// runPreflightChecks performs pre-flight validation for git modes
func runPreflightChecks(mode checkMode) error {
	// 1. Check if git is installed
	if !git.Available() {
		return fmt.Errorf(`Error: git is not installed or not in PATH

tfbreak requires git 2.5 or later for git ref comparison.
Install git: https://git-scm.com/downloads`)
	}

	// 2. Check git version (need 2.5 for worktree support)
	if mode != modeRemoteRefs && mode != modeMixed {
		if err := git.CheckVersion(2, 5); err != nil {
			var versionErr *git.ErrVersionTooOld
			if errors.As(err, &versionErr) {
				return fmt.Errorf(`Error: git version %s is below minimum required 2.5

tfbreak requires git 2.5 or later for worktree support.
Please upgrade git: https://git-scm.com/downloads`, versionErr.Current)
			}
			return fmt.Errorf("failed to check git version: %w", err)
		}
	}

	// 3. Check if we're in a git repository (not required for --repo mode)
	if mode == modeLocalRef || mode == modeTwoLocalRefs {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		if !git.IsGitRepository(cwd) {
			return fmt.Errorf(`Error: not a git repository (or any parent up to /)

The --base flag requires running from within a git repository.
Either:
  - Run from inside a git repository
  - Use --repo to compare remote repositories`)
		}
	}

	// 4. Validate refs exist (parse ref:path specs to extract just the ref)
	baseSpec := parseRefSpec(baseFlag)
	headSpec := parseRefSpec(headFlag)

	if mode == modeLocalRef || mode == modeTwoLocalRefs {
		cwd, _ := os.Getwd()
		repoRoot, err := git.FindGitRoot(cwd)
		if err != nil {
			return err
		}

		// Check base ref
		if _, err := git.ResolveRef(repoRoot, baseSpec.Ref); err != nil {
			return formatRefNotFoundError(baseSpec.Ref, repoRoot, err)
		}

		// Check head ref if specified
		if headFlag != "" {
			if _, err := git.ResolveRef(repoRoot, headSpec.Ref); err != nil {
				return formatRefNotFoundError(headSpec.Ref, repoRoot, err)
			}
		}
	}

	// For remote mode, validate remote refs
	if mode == modeRemoteRefs || mode == modeMixed {
		// Validate base ref exists remotely
		if _, _, err := git.ResolveRemoteRef(repoFlag, baseSpec.Ref); err != nil {
			return formatRemoteRefNotFoundError(baseSpec.Ref, repoFlag, err)
		}

		// Validate head ref if specified
		if headFlag != "" {
			if _, _, err := git.ResolveRemoteRef(repoFlag, headSpec.Ref); err != nil {
				return formatRemoteRefNotFoundError(headSpec.Ref, repoFlag, err)
			}
		}
	}

	return nil
}

// formatRefNotFoundError formats a user-friendly error for missing local refs
func formatRefNotFoundError(ref, repoDir string, originalErr error) error {
	isShallow, _ := git.IsShallowClone(repoDir)
	if isShallow {
		return fmt.Errorf(`Error: ref '%s' not found in repository

This may be because the repository is a shallow clone.
To fix, fetch the required ref:

  git fetch origin %s

Or fetch full history:

  git fetch --unshallow

For CI pipelines, configure full checkout depth:
  - GitHub Actions: actions/checkout with fetch-depth: 0
  - GitLab CI: GIT_DEPTH: 0`, ref, ref)
	}

	return fmt.Errorf(`Error: ref '%s' not found in repository

%v

Check that the ref exists:
  git rev-parse --verify %s`, ref, originalErr, ref)
}

// formatRemoteRefNotFoundError formats a user-friendly error for missing remote refs
func formatRemoteRefNotFoundError(ref, url string, err error) error {
	// Check if it's an authentication/access error
	if git.IsAuthError(err) {
		return fmt.Errorf(`Error: cannot access repository '%s'

Git error: %v

Troubleshooting:
  - Check the URL is correct
  - For private repos, ensure credentials are configured
  - Test manually: git ls-remote %s`, url, err, url)
	}

	return fmt.Errorf(`Error: ref '%s' not found in '%s'

Available tags can be listed with:
  git ls-remote --tags %s

Available branches:
  git ls-remote --heads %s`, ref, url, url, url)
}

// validateSubdirPath checks that a subdirectory path exists within a root directory.
// Returns a user-friendly error if the path doesn't exist.
func validateSubdirPath(rootDir, subPath, ref string) error {
	fullPath := filepath.Join(rootDir, subPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path '%s' does not exist at ref '%s'", subPath, ref)
		}
		return fmt.Errorf("cannot access path '%s' at ref '%s': %w", subPath, ref, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path '%s' at ref '%s' is not a directory", subPath, ref)
	}
	return nil
}

// resolveDirectories resolves old and new directories based on mode
// Returns directories and a cleanup function (may be nil)
func resolveDirectories(mode checkMode, args []string) (oldDir, newDir string, cleanup func(), err error) {
	// Parse ref:path specs
	baseSpec := parseRefSpec(baseFlag)
	headSpec := parseRefSpec(headFlag)

	switch mode {
	case modeDirectory:
		return args[0], args[1], nil, nil

	case modeLocalRef:
		// new_dir is args[0] or "."
		if len(args) > 0 {
			newDir = args[0]
		} else {
			newDir = "."
		}

		// Create worktree for base ref
		cwd, _ := os.Getwd()
		repoRoot, err := git.FindGitRoot(cwd)
		if err != nil {
			return "", "", nil, err
		}

		worktree, err := git.CreateWorktree(repoRoot, baseSpec.Ref)
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to create worktree for %s: %w", baseSpec.Ref, err)
		}

		// Apply path within worktree if specified
		oldDir = worktree.Path
		if baseSpec.Path != "" {
			if err := validateSubdirPath(worktree.Path, baseSpec.Path, baseSpec.Ref); err != nil {
				worktree.Remove()
				return "", "", nil, err
			}
			oldDir = filepath.Join(worktree.Path, baseSpec.Path)
		}

		return oldDir, newDir, func() { worktree.Remove() }, nil

	case modeTwoLocalRefs:
		cwd, _ := os.Getwd()
		repoRoot, err := git.FindGitRoot(cwd)
		if err != nil {
			return "", "", nil, err
		}

		// Create worktree for base ref
		baseWorktree, err := git.CreateWorktree(repoRoot, baseSpec.Ref)
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to create worktree for %s: %w", baseSpec.Ref, err)
		}

		// Create worktree for head ref
		headWorktree, err := git.CreateWorktree(repoRoot, headSpec.Ref)
		if err != nil {
			baseWorktree.Remove()
			return "", "", nil, fmt.Errorf("failed to create worktree for %s: %w", headSpec.Ref, err)
		}

		cleanup = func() {
			baseWorktree.Remove()
			headWorktree.Remove()
		}

		// Apply paths within worktrees if specified
		oldDir = baseWorktree.Path
		if baseSpec.Path != "" {
			if err := validateSubdirPath(baseWorktree.Path, baseSpec.Path, baseSpec.Ref); err != nil {
				cleanup()
				return "", "", nil, err
			}
			oldDir = filepath.Join(baseWorktree.Path, baseSpec.Path)
		}
		newDir = headWorktree.Path
		if headSpec.Path != "" {
			if err := validateSubdirPath(headWorktree.Path, headSpec.Path, headSpec.Ref); err != nil {
				cleanup()
				return "", "", nil, err
			}
			newDir = filepath.Join(headWorktree.Path, headSpec.Path)
		}

		return oldDir, newDir, cleanup, nil

	case modeRemoteRefs:
		// Clone both refs from remote
		baseClone, headClone, err := git.CloneForComparison(repoFlag, baseSpec.Ref, headSpec.Ref)
		if err != nil {
			return "", "", nil, err
		}

		cleanup = func() {
			baseClone.Remove()
			headClone.Remove()
		}

		// Apply paths within clones if specified
		oldDir = baseClone.Path
		if baseSpec.Path != "" {
			if err := validateSubdirPath(baseClone.Path, baseSpec.Path, baseSpec.Ref); err != nil {
				cleanup()
				return "", "", nil, err
			}
			oldDir = filepath.Join(baseClone.Path, baseSpec.Path)
		}
		newDir = headClone.Path
		if headSpec.Path != "" {
			if err := validateSubdirPath(headClone.Path, headSpec.Path, headSpec.Ref); err != nil {
				cleanup()
				return "", "", nil, err
			}
			newDir = filepath.Join(headClone.Path, headSpec.Path)
		}

		return oldDir, newDir, cleanup, nil

	case modeMixed:
		// Clone base ref from remote, use local new_dir
		newDir = args[0]

		baseClone, err := git.ShallowClone(repoFlag, baseSpec.Ref)
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to clone %s from %s: %w", baseSpec.Ref, repoFlag, err)
		}

		// Apply path within clone if specified
		oldDir = baseClone.Path
		if baseSpec.Path != "" {
			if err := validateSubdirPath(baseClone.Path, baseSpec.Path, baseSpec.Ref); err != nil {
				baseClone.Remove()
				return "", "", nil, err
			}
			oldDir = filepath.Join(baseClone.Path, baseSpec.Path)
		}

		return oldDir, newDir, func() { baseClone.Remove() }, nil
	}

	return "", "", nil, errors.New("unknown mode")
}

// executePluginRules discovers, loads, and executes plugin rules.
// Plugin findings are added to the result.
// Returns an error if configured plugins are missing (user should run tfbreak init).
func executePluginRules(cfg *config.Config, oldDir, newDir string, result *types.CheckResult, verbose bool) error {
	// Check for missing plugins before attempting to load
	missing := plugin.GetMissingPlugins(cfg)
	if len(missing) > 0 {
		var names []string
		for _, p := range missing {
			names = append(names, p.Name)
		}
		return fmt.Errorf("plugin(s) not installed: %s\n\nRun 'tfbreak --init' to install configured plugins", strings.Join(names, ", "))
	}

	// Create plugin manager
	mgr := plugin.NewManager(cfg)
	defer mgr.Close()

	// Discover and load plugins (no auto-download)
	count, loadErrs := mgr.DiscoverAndLoad()
	if len(loadErrs) > 0 && verbose {
		for _, err := range loadErrs {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	// If no plugins loaded, nothing to do
	if count == 0 {
		return nil
	}

	if verbose {
		summaries := mgr.GetLoadedPlugins()
		for _, s := range summaries {
			fmt.Fprintf(os.Stderr, "Loaded plugin: %s v%s (%d rules)\n", s.Name, s.Version, s.RuleCount)
		}
	}

	// Load HCL files for plugins
	oldFiles, err := plugin.LoadHCLFiles(oldDir)
	if err != nil {
		return fmt.Errorf("failed to load old HCL files: %w", err)
	}

	newFiles, err := plugin.LoadHCLFiles(newDir)
	if err != nil {
		return fmt.Errorf("failed to load new HCL files: %w", err)
	}

	// Execute plugin rules
	findings, execErrs := mgr.ExecuteRules(oldFiles, newFiles)
	if len(execErrs) > 0 && verbose {
		for _, err := range execErrs {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	// Add plugin findings to result
	for _, f := range findings {
		result.AddFinding(f)
	}

	return nil
}

package config

// Default returns the default configuration
func Default() *Config {
	enabled := true
	renameDisabled := false
	defaultThreshold := DefaultSimilarityThreshold
	return &Config{
		Version: 1,
		ConfigBlock: &ConfigBlockConfig{
			PluginDir: "",
		},
		Paths: &PathsConfig{
			Include: []string{"**/*.tf"},
			Exclude: []string{".terraform/**"},
		},
		Output: &OutputConfig{
			Format: "text",
			Color:  "auto",
		},
		Policy: &PolicyConfig{
			FailOn:                "ERROR",
			TreatWarningsAsErrors: false,
		},
		Annotations: &AnnotationsConfig{
			Enabled:       &enabled,
			RequireReason: false,
			AllowRuleIDs:  []string{},
			DenyRuleIDs:   []string{},
		},
		RenameDetection: &RenameDetectionConfig{
			Enabled:             &renameDisabled,
			SimilarityThreshold: &defaultThreshold,
		},
		Rules:   []*RuleConfig{},
		Plugins: []*PluginConfig{},
	}
}


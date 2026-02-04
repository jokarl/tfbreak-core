package cli

import (
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/jokarl/tfbreak-core/internal/config"
	"github.com/jokarl/tfbreak-core/internal/rules"
	"github.com/jokarl/tfbreak-core/internal/types"
)

func TestValidateCheckArgs(t *testing.T) {
	// Save and restore global flag state
	saveFlags := func() (string, string, string) {
		return baseFlag, headFlag, repoFlag
	}
	restoreFlags := func(b, h, r string) {
		baseFlag = b
		headFlag = h
		repoFlag = r
	}

	tests := []struct {
		name      string
		base      string
		head      string
		repo      string
		args      []string
		wantError bool
		errSubstr string
	}{
		{
			name:      "directory mode with two args",
			base:      "",
			head:      "",
			repo:      "",
			args:      []string{"./old", "./new"},
			wantError: false,
		},
		{
			name:      "directory mode with one arg",
			base:      "",
			head:      "",
			repo:      "",
			args:      []string{"./old"},
			wantError: true,
			errSubstr: "exactly two directory arguments",
		},
		{
			name:      "directory mode with no args",
			base:      "",
			head:      "",
			repo:      "",
			args:      []string{},
			wantError: true,
			errSubstr: "exactly two directory arguments",
		},
		{
			name:      "--head requires --base",
			base:      "",
			head:      "dev",
			repo:      "",
			args:      []string{},
			wantError: true,
			errSubstr: "--head requires --base",
		},
		{
			name:      "--repo requires --base",
			base:      "",
			head:      "",
			repo:      "https://github.com/org/repo",
			args:      []string{},
			wantError: true,
			errSubstr: "--repo requires --base",
		},
		{
			name:      "--base only with new_dir",
			base:      "main",
			head:      "",
			repo:      "",
			args:      []string{"./new"},
			wantError: false,
		},
		{
			name:      "--base only with no args",
			base:      "main",
			head:      "",
			repo:      "",
			args:      []string{},
			wantError: false,
		},
		{
			name:      "--base only with too many args",
			base:      "main",
			head:      "",
			repo:      "",
			args:      []string{"./old", "./new"},
			wantError: true,
			errSubstr: "at most one positional argument",
		},
		{
			name:      "--base --head with no args",
			base:      "main",
			head:      "dev",
			repo:      "",
			args:      []string{},
			wantError: false,
		},
		{
			name:      "--base --head with extra args",
			base:      "main",
			head:      "dev",
			repo:      "",
			args:      []string{"./new"},
			wantError: true,
			errSubstr: "no positional arguments expected with --base --head",
		},
		{
			name:      "--repo --base --head with no args",
			base:      "v1.0.0",
			head:      "v2.0.0",
			repo:      "https://github.com/org/repo",
			args:      []string{},
			wantError: false,
		},
		{
			name:      "--repo --base --head with extra args",
			base:      "v1.0.0",
			head:      "v2.0.0",
			repo:      "https://github.com/org/repo",
			args:      []string{"./new"},
			wantError: true,
			errSubstr: "no positional arguments expected with --repo --base --head",
		},
		{
			name:      "--repo --base with new_dir",
			base:      "v1.0.0",
			head:      "",
			repo:      "https://github.com/org/repo",
			args:      []string{"./new"},
			wantError: false,
		},
		{
			name:      "--repo --base without new_dir",
			base:      "v1.0.0",
			head:      "",
			repo:      "https://github.com/org/repo",
			args:      []string{},
			wantError: true,
			errSubstr: "exactly one positional argument (new_dir) required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origBase, origHead, origRepo := saveFlags()
			defer restoreFlags(origBase, origHead, origRepo)

			baseFlag = tt.base
			headFlag = tt.head
			repoFlag = tt.repo

			cmd := &cobra.Command{}
			err := validateCheckArgs(cmd, tt.args)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDetermineMode(t *testing.T) {
	saveFlags := func() (string, string, string) {
		return baseFlag, headFlag, repoFlag
	}
	restoreFlags := func(b, h, r string) {
		baseFlag = b
		headFlag = h
		repoFlag = r
	}

	tests := []struct {
		name     string
		base     string
		head     string
		repo     string
		wantMode checkMode
	}{
		{
			name:     "directory mode",
			base:     "",
			head:     "",
			repo:     "",
			wantMode: modeDirectory,
		},
		{
			name:     "local ref mode",
			base:     "main",
			head:     "",
			repo:     "",
			wantMode: modeLocalRef,
		},
		{
			name:     "two local refs mode",
			base:     "main",
			head:     "dev",
			repo:     "",
			wantMode: modeTwoLocalRefs,
		},
		{
			name:     "remote refs mode",
			base:     "v1.0.0",
			head:     "v2.0.0",
			repo:     "https://github.com/org/repo",
			wantMode: modeRemoteRefs,
		},
		{
			name:     "mixed mode",
			base:     "v1.0.0",
			head:     "",
			repo:     "https://github.com/org/repo",
			wantMode: modeMixed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origBase, origHead, origRepo := saveFlags()
			defer restoreFlags(origBase, origHead, origRepo)

			baseFlag = tt.base
			headFlag = tt.head
			repoFlag = tt.repo

			got := determineMode()
			if got != tt.wantMode {
				t.Errorf("determineMode() = %v, want %v", got, tt.wantMode)
			}
		})
	}
}

func TestApplyFlagOverrides(t *testing.T) {
	saveFlags := func() (string, string, string, bool, []string, []string) {
		return formatFlag, colorFlag, failOnFlag, requireReasonFlag, includeFlag, excludeFlag
	}
	restoreFlags := func(f, c, fo string, rr bool, inc, exc []string) {
		formatFlag = f
		colorFlag = c
		failOnFlag = fo
		requireReasonFlag = rr
		includeFlag = inc
		excludeFlag = exc
	}

	t.Run("no overrides", func(t *testing.T) {
		origFormat, origColor, origFailOn, origRequireReason, origInclude, origExclude := saveFlags()
		defer restoreFlags(origFormat, origColor, origFailOn, origRequireReason, origInclude, origExclude)

		formatFlag = ""
		colorFlag = ""
		failOnFlag = ""
		requireReasonFlag = false
		includeFlag = nil
		excludeFlag = nil

		cfg := &config.Config{
			Output: &config.OutputConfig{
				Format: "text",
				Color:  "auto",
			},
			Policy: &config.PolicyConfig{
				FailOn: "ERROR",
			},
			Paths: &config.PathsConfig{
				Include: []string{"*.tf"},
				Exclude: []string{".terraform"},
			},
			Annotations: &config.AnnotationsConfig{
				RequireReason: false,
			},
		}

		applyFlagOverrides(cfg)

		if cfg.Output.Format != "text" {
			t.Errorf("Format = %q, want %q", cfg.Output.Format, "text")
		}
		if cfg.Output.Color != "auto" {
			t.Errorf("Color = %q, want %q", cfg.Output.Color, "auto")
		}
		if cfg.Policy.FailOn != "ERROR" {
			t.Errorf("FailOn = %q, want %q", cfg.Policy.FailOn, "ERROR")
		}
	})

	t.Run("all overrides", func(t *testing.T) {
		origFormat, origColor, origFailOn, origRequireReason, origInclude, origExclude := saveFlags()
		defer restoreFlags(origFormat, origColor, origFailOn, origRequireReason, origInclude, origExclude)

		formatFlag = "json"
		colorFlag = "never"
		failOnFlag = "WARNING"
		requireReasonFlag = true
		includeFlag = []string{"modules/**/*.tf"}
		excludeFlag = []string{"test/**"}

		cfg := &config.Config{
			Output: &config.OutputConfig{
				Format: "text",
				Color:  "auto",
			},
			Policy: &config.PolicyConfig{
				FailOn: "ERROR",
			},
			Paths: &config.PathsConfig{
				Include: []string{"*.tf"},
				Exclude: []string{".terraform"},
			},
			Annotations: &config.AnnotationsConfig{
				RequireReason: false,
			},
		}

		applyFlagOverrides(cfg)

		if cfg.Output.Format != "json" {
			t.Errorf("Format = %q, want %q", cfg.Output.Format, "json")
		}
		if cfg.Output.Color != "never" {
			t.Errorf("Color = %q, want %q", cfg.Output.Color, "never")
		}
		if cfg.Policy.FailOn != "WARNING" {
			t.Errorf("FailOn = %q, want %q", cfg.Policy.FailOn, "WARNING")
		}
		if !cfg.Annotations.RequireReason {
			t.Error("RequireReason should be true")
		}
		if len(cfg.Paths.Include) != 1 || cfg.Paths.Include[0] != "modules/**/*.tf" {
			t.Errorf("Include = %v, want [modules/**/*.tf]", cfg.Paths.Include)
		}
		if len(cfg.Paths.Exclude) != 1 || cfg.Paths.Exclude[0] != "test/**" {
			t.Errorf("Exclude = %v, want [test/**]", cfg.Paths.Exclude)
		}
	})
}

func TestConfigureEngine(t *testing.T) {
	saveFlags := func() ([]string, []string, []string) {
		return enableFlag, disableFlag, severityFlags
	}
	restoreFlags := func(en, dis, sev []string) {
		enableFlag = en
		disableFlag = dis
		severityFlags = sev
	}

	t.Run("config file enables and disables rules", func(t *testing.T) {
		origEnable, origDisable, origSeverity := saveFlags()
		defer restoreFlags(origEnable, origDisable, origSeverity)
		enableFlag = nil
		disableFlag = nil
		severityFlags = nil

		engine := rules.NewDefaultEngine()

		enabled := true
		disabled := false
		warningStr := "WARNING"
		cfg := &config.Config{
			Rules: []*config.RuleConfig{
				{ID: "TFB001", Enabled: &disabled},
				{ID: "TFB002", Severity: &warningStr},
				{ID: "TFB003", Enabled: &enabled},
			},
		}

		configureEngine(engine, cfg)

		tfb001Config := engine.GetConfig("TFB001")
		if tfb001Config != nil && tfb001Config.Enabled {
			t.Error("TFB001 should be disabled")
		}
		tfb003Config := engine.GetConfig("TFB003")
		if tfb003Config != nil && !tfb003Config.Enabled {
			t.Error("TFB003 should be enabled")
		}
		tfb002Config := engine.GetConfig("TFB002")
		if tfb002Config != nil && tfb002Config.Severity != types.SeverityWarning {
			t.Errorf("TFB002 severity = %v, want WARNING", tfb002Config.Severity)
		}
	})

	t.Run("CLI flags override config", func(t *testing.T) {
		origEnable, origDisable, origSeverity := saveFlags()
		defer restoreFlags(origEnable, origDisable, origSeverity)

		enableFlag = []string{"TFB001"}
		disableFlag = []string{"TFB003"}
		severityFlags = []string{"TFB002=NOTICE"}

		engine := rules.NewDefaultEngine()
		cfg := &config.Config{}

		configureEngine(engine, cfg)

		tfb001Config := engine.GetConfig("TFB001")
		if tfb001Config != nil && !tfb001Config.Enabled {
			t.Error("TFB001 should be enabled via CLI flag")
		}
		tfb003Config := engine.GetConfig("TFB003")
		if tfb003Config != nil && tfb003Config.Enabled {
			t.Error("TFB003 should be disabled via CLI flag")
		}
		tfb002Config := engine.GetConfig("TFB002")
		if tfb002Config != nil && tfb002Config.Severity != types.SeverityNotice {
			t.Errorf("TFB002 severity = %v, want NOTICE", tfb002Config.Severity)
		}
	})

	t.Run("invalid severity flag format is ignored", func(t *testing.T) {
		origEnable, origDisable, origSeverity := saveFlags()
		defer restoreFlags(origEnable, origDisable, origSeverity)

		enableFlag = nil
		disableFlag = nil
		severityFlags = []string{"invalid-no-equals", "TFB001=INVALID_SEVERITY"}

		engine := rules.NewDefaultEngine()
		cfg := &config.Config{}

		// This should not panic
		configureEngine(engine, cfg)
	})
}

func TestShouldUseColor(t *testing.T) {
	tests := []struct {
		name      string
		colorMode string
		want      bool
	}{
		{
			name:      "always returns true",
			colorMode: "always",
			want:      true,
		},
		{
			name:      "never returns false",
			colorMode: "never",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp file (not a terminal)
			f, err := createTempFile(t)
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer f.Close()

			got := shouldUseColor(f, tt.colorMode)
			if got != tt.want {
				t.Errorf("shouldUseColor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatRefNotFoundError(t *testing.T) {
	// Just verify it doesn't panic and returns an error
	err := formatRefNotFoundError("nonexistent", "/tmp", nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention ref name, got: %v", err)
	}
}

func TestFormatRemoteRefNotFoundError(t *testing.T) {
	err := formatRemoteRefNotFoundError("v1.0.0", "https://github.com/org/repo", nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !contains(err.Error(), "v1.0.0") {
		t.Errorf("error should mention ref name, got: %v", err)
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func createTempFile(t *testing.T) (*os.File, error) {
	t.Helper()
	return os.CreateTemp("", "test")
}

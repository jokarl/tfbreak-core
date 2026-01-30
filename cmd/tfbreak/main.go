package main

import (
	"os"
	"runtime/debug"

	"github.com/jokarl/tfbreak-core/internal/cli"
)

// Version information (set via ldflags during build, or read from build info)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// If version wasn't set via ldflags, try to get it from build info
	// This works when installed via `go install module@version`
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}

	// Try to get commit from build info VCS settings
	if commit == "none" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					if len(setting.Value) >= 7 {
						commit = setting.Value[:7]
					} else {
						commit = setting.Value
					}
				case "vcs.time":
					date = setting.Value
				}
			}
		}
	}

	cli.SetVersionInfo(version, commit, date)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

package main

import (
	"os"

	"github.com/jokarl/tfbreak-core/internal/cli"
)

// Version information (set via ldflags during build)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.SetVersionInfo(version, commit, date)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

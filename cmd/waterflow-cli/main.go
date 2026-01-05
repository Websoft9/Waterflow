package main

import (
	"fmt"
	"os"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/cmd"
)

var (
	// Version information injected at build time
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	cmd.SetVersionInfo(Version, Commit, BuildTime)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

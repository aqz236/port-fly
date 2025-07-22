package main

import (
	"os"

	"github.com/aqz236/port-fly/cli/cmd"
)

var (
	// Build information set by ldflags
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Set build information
	cmd.SetBuildInfo(Version, BuildTime)
	
	// Execute root command
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

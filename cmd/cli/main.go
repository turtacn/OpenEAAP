package main

import (
	"fmt"
	"os"

	"github.com/openeeap/openeeap/internal/api/cli"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/pkg/config"
)

var (
	// Version is the application version
	Version = "dev"
	// BuildTime is the build timestamp
	BuildTime = "unknown"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := logging.NewLogger(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	// Set version information
	cli.SetVersionInfo(Version, BuildTime, GitCommit)

	// Execute CLI
	if err := cli.Execute(cfg, logger); err != nil {
		logger.Error("CLI execution failed", "error", err)
		os.Exit(1)
	}
}

//Personal.AI order the ending

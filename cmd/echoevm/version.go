package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

// These variables are intended to be set via -ldflags during build, e.g.:
// go build -ldflags "-X main.GitCommit=$(git rev-parse --short HEAD) -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.Version=v0.1.0"
// They default to "dev" when not provided.
// Version is the current version of the application.
// It is set at build time or defaults to the hardcoded value here.
var Version = "v0.0.17"

var (
	GitCommit = "dev"
	BuildDate = "dev"
)

type versionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

func newVersionCmd() *cobra.Command {
	var outputJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show build version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := versionInfo{
				Version:   Version,
				GitCommit: GitCommit,
				BuildDate: BuildDate,
				GoVersion: runtime.Version(),
				Platform:  runtime.GOOS + "/" + runtime.GOARCH,
			}
			if info.BuildDate == "dev" { // attempt friendly fallback if unset
				info.BuildDate = time.Now().UTC().Format(time.RFC3339)
			}
			if outputJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "echoevm %s (commit %s, built %s) %s %s\n", info.Version, info.GitCommit, info.BuildDate, info.Platform, info.GoVersion)
			return nil
		},
		Example: "echoevm version --json",
	}
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")
	return cmd
}

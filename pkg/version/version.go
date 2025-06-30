// Package version provides build version and metadata information.
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version of the application.
	// Set during build with -ldflags "-X github.com/d-kuro/claude-code-mcp/pkg/version.Version=v1.0.0"
	Version = "dev"

	// GitCommit is the git commit hash.
	// Set during build with -ldflags "-X github.com/d-kuro/claude-code-mcp/pkg/version.GitCommit=abcdef"
	GitCommit = "unknown"

	// BuildDate is the build timestamp.
	// Set during build with -ldflags "-X github.com/d-kuro/claude-code-mcp/pkg/version.BuildDate=2024-01-01T00:00:00Z"
	BuildDate = "unknown"
)

// Info contains version and build information.
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetVersion returns the current version information.
func GetVersion() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string.
func (i Info) String() string {
	return fmt.Sprintf("claude-code-mcp %s (%s) built with %s on %s",
		i.Version, i.GitCommit, i.GoVersion, i.Platform)
}

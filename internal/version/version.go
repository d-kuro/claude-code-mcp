// Package version provides build version and metadata information.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"time"
)

// Info contains version and build information.
type Info struct {
	Version    string    `json:"version"`
	GitCommit  string    `json:"git_commit"`
	BuildDate  time.Time `json:"build_date"`
	GoVersion  string    `json:"go_version"`
	Platform   string    `json:"platform"`
	ModulePath string    `json:"module_path"`
}

// Format returns a formatted version string for display.
func (i Info) Format() string {
	return i.String()
}

// GetVersion returns the current version information.
func GetVersion() Info {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Info{
			Version:   "(devel)",
			GitCommit: "unknown",
			GoVersion: runtime.Version(),
			Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		}
	}

	// Extract version from build info
	version := info.Main.Version
	if version == "" || version == "(devel)" {
		// If no version is set, try to get it from module
		version = "(devel)"
	}

	// Extract build settings
	var (
		vcsRevision string
		vcsTime     time.Time
		vcsModified bool
	)

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			vcsRevision = setting.Value
		case "vcs.time":
			if t, err := time.Parse(time.RFC3339, setting.Value); err == nil {
				vcsTime = t
			}
		case "vcs.modified":
			vcsModified = setting.Value == "true"
		}
	}

	// Add modified flag to revision if applicable
	if vcsModified && vcsRevision != "" {
		vcsRevision += "-modified"
	}

	return Info{
		Version:    version,
		GitCommit:  vcsRevision,
		BuildDate:  vcsTime,
		GoVersion:  runtime.Version(),
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		ModulePath: info.Main.Path,
	}
}

// String returns a formatted version string.
func (i Info) String() string {
	// Format version string
	versionStr := i.Version
	if versionStr == "(devel)" {
		versionStr = "development"
	} else if len(versionStr) > 0 && versionStr[0] == 'v' {
		// Already has 'v' prefix
	} else {
		versionStr = "v" + versionStr
	}

	// Format git commit
	gitStr := i.GitCommit
	if gitStr == "" || gitStr == "unknown" {
		gitStr = "unknown"
	} else if len(gitStr) > 7 {
		// Show short commit hash
		gitStr = gitStr[:7]
	}

	// Build multi-line output
	output := fmt.Sprintf("claude-code-mcp %s\n", versionStr)

	if gitStr != "unknown" {
		output += fmt.Sprintf("  Commit:     %s\n", gitStr)
	}

	if !i.BuildDate.IsZero() {
		output += fmt.Sprintf("  Built:      %s\n", i.BuildDate.Format("2006-01-02 15:04:05"))
	}

	output += fmt.Sprintf("  Go version: %s\n", i.GoVersion)
	output += fmt.Sprintf("  Platform:   %s", i.Platform)

	return output
}

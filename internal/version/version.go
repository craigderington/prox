package version

import (
	"fmt"
	"runtime"
)

// Version information set at build time via -ldflags
var (
	Version   = "dev"     // Version tag (e.g., v1.2.3)
	Commit    = "unknown" // Git commit hash
	BuildTime = "unknown" // Build timestamp
	GoVersion = runtime.Version()
)

// GetVersion returns a formatted version string
func GetVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s, go: %s)",
		Version, Commit, BuildTime, GoVersion)
}

// GetShortVersion returns just the version tag
func GetShortVersion() string {
	return Version
}

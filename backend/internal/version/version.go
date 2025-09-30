// Package version provides build-time version information for the application.
package version

var (
	// Version is the application version (e.g., git tag or "dev")
	Version = "dev"
	// Commit is the git commit hash
	Commit = "dev"
	// BuildTime is the build timestamp
	BuildTime = "unknown"
)

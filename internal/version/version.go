// Package version holds build metadata injected at release time via
// goreleaser ldflags.
package version

// These are overridden by goreleaser at build time:
//
//	-ldflags "-X .../internal/version.Version=... -X .../internal/version.Commit=... -X .../internal/version.Date=..."
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

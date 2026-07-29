// Package scan discovers git repositories below a root directory and streams
// their paths as they are found.
package scan

// DefaultExcludes returns the directory basenames that are never descended
// into: dependency trees that plausibly contain stray .git directories but
// never repositories the user wants to clean. Matching is exact-basename
// (no globs) in v0.1. Config-provided excludes are merged on top.
func DefaultExcludes() map[string]bool {
	return map[string]bool{
		"node_modules": true,
		"vendor":       true,
		".terraform":   true,
		".venv":        true,
		"__pycache__":  true,
		".cache":       true,
	}
}

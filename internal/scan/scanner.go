package scan

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Walk recursively discovers git repositories below root and streams their
// paths on the returned channel, which is closed when the walk finishes or
// ctx is cancelled. It validates root synchronously; everything after that
// is best-effort: unreadable directories are skipped, not fatal (REQ-A4).
//
// Discovered repositories are not descended into: their worktrees are
// enumerated precisely later via `git worktree list`, and nested working
// trees (linked worktrees, submodules) must not be double-counted. Symbolic
// links are never followed, so link loops cannot occur.
func Walk(ctx context.Context, root string, excludes map[string]bool) (<-chan string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("scan root %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan root %s is not a directory", root)
	}

	ch := make(chan string)
	go func() {
		defer close(ch)
		//nolint:errcheck // the only error surfaced is ctx cancellation, which the caller initiated.
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				// Unreadable entry: skip whatever it was and continue.
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			if path != root && excludes[d.Name()] {
				return filepath.SkipDir
			}
			switch classifyDir(path) {
			case repository:
				select {
				case ch <- path:
				case <-ctx.Done():
					return ctx.Err()
				}
				return filepath.SkipDir
			case linkedWorktree, submodule:
				// Belongs to some repository; never counted on its own and
				// never descended into (REQ-P3).
				return filepath.SkipDir
			case notGit:
			}
			return nil
		})
	}()
	return ch, nil
}

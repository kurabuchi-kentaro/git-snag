package gitcli

import "context"

// IsDirty reports whether the worktree at wtPath has uncommitted changes,
// including untracked files. It must run against the worktree's own path,
// not the repository root (each worktree has its own HEAD and index).
func (c *Client) IsDirty(ctx context.Context, wtPath string) (bool, error) {
	out, err := c.run(ctx, wtPath, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

package gitcli

import (
	"context"
	"strings"
)

// DeleteBranch deletes a local branch. Without force it uses `-d`, which git
// rejects for unmerged branches; with force it uses `-D`.
func (c *Client) DeleteBranch(ctx context.Context, repoPath, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := c.run(ctx, repoPath, "branch", flag, name)
	return err
}

// MergedBranches returns the set of local branches merged into target,
// resolved with a single git invocation per repository.
func (c *Client) MergedBranches(ctx context.Context, repoPath, target string) (map[string]bool, error) {
	out, err := c.run(ctx, repoPath, "branch", "--merged", target, "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	merged := make(map[string]bool)
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			merged[line] = true
		}
	}
	return merged, nil
}

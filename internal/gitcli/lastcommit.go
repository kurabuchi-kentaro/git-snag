package gitcli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// LastCommitTime returns the committer time of the worktree's HEAD commit.
func (c *Client) LastCommitTime(ctx context.Context, wtPath string) (time.Time, error) {
	out, err := c.run(ctx, wtPath, "log", "-1", "--format=%ct")
	if err != nil {
		return time.Time{}, err
	}
	sec, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing commit timestamp %q: %w", out, err)
	}
	return time.Unix(sec, 0), nil
}

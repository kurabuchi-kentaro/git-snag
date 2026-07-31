package gitcli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

// branchesFormat packs everything branch mode needs into one for-each-ref
// call: name, committer date, upstream divergence, and the worktree the
// branch is checked out in. Tab-separated — git forbids tabs (and all
// whitespace) in ref names, and %(upstream:track) is the only field with
// inner spaces.
var branchesFormat = strings.Join([]string{
	"%(refname:short)",
	"%(committerdate:unix)",
	"%(upstream:track)",
	"%(worktreepath)",
}, "\t")

// Branches lists the repository's local branches with the status a single
// `git for-each-ref refs/heads` supplies (ADR 0014). Merged flags are not
// set here — the enricher owns the merged set.
func (c *Client) Branches(ctx context.Context, repoPath string) ([]domain.Branch, error) {
	out, err := c.run(ctx, repoPath, "for-each-ref", "refs/heads", "--format="+branchesFormat)
	if err != nil {
		return nil, err
	}
	var branches []domain.Branch
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			return nil, fmt.Errorf("parsing for-each-ref line %q: want 4 fields, got %d", line, len(fields))
		}
		unix, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing committer date %q: %w", fields[1], err)
		}
		track := fields[2]
		branches = append(branches, domain.Branch{
			Name:           fields[0],
			LastCommitTime: time.Unix(unix, 0),
			UpstreamGone:   track == "[gone]",
			UnpushedCount:  aheadCount(track),
			WorktreePath:   fields[3],
		})
	}
	return branches, nil
}

// aheadCount extracts N from an %(upstream:track) value like "[ahead 3]" or
// "[ahead 3, behind 1]"; anything else (empty, behind-only, gone) is 0.
func aheadCount(track string) int {
	_, rest, found := strings.Cut(track, "ahead ")
	if !found {
		return 0
	}
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	n, err := strconv.Atoi(rest[:end])
	if err != nil {
		return 0
	}
	return n
}

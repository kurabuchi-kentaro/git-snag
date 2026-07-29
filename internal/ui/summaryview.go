package ui

import (
	"fmt"
	"strings"
)

// applyResults removes successfully deleted worktrees from the model and
// records the results for the summary screen.
func (m *Model) applyResults(msg DeleteResultsMsg) {
	m.results = msg.Results
	deleted := map[string]bool{}
	for _, r := range m.results {
		if !r.Skipped && r.WorktreeErr == nil {
			deleted[r.Item.Worktree.Path] = true
		}
	}
	for i := range m.repos {
		repo := &m.repos[i]
		kept := repo.Worktrees[:0]
		for _, w := range repo.Worktrees {
			if !deleted[w.Path] {
				kept = append(kept, w)
			}
		}
		repo.Worktrees = kept
	}
	m.selection = map[string]bool{}
	m.rebuildRows()
	m.phase = phaseSummary
}

// viewSummary renders the per-item outcome of the batch (REQ-A8: worktree
// and branch results reported separately).
func (m Model) viewSummary() string {
	var b strings.Builder
	b.WriteString("Deletion summary\n\n")
	for _, r := range m.results {
		name := r.Item.Worktree.Branch
		if r.Item.Worktree.Detached() {
			name = leafLabel(r.Item)
		}
		switch {
		case r.Skipped:
			fmt.Fprintf(&b, "– %s: skipped\n", name)
		case r.WorktreeErr != nil:
			fmt.Fprintf(&b, "✗ %s: worktree removal failed: %v\n", name, r.WorktreeErr)
		default:
			fmt.Fprintf(&b, "✓ %s: worktree removed\n", name)
			switch {
			case !r.BranchAttempted:
				// Branch kept on purpose or detached: nothing to report.
			case r.BranchErr != nil:
				fmt.Fprintf(&b, "  ✗ branch deletion failed: %v\n", r.BranchErr)
			case r.BranchForced:
				b.WriteString("  ✓ branch force-deleted (had unmerged commits)\n")
			default:
				b.WriteString("  ✓ branch deleted\n")
			}
		}
	}
	b.WriteString("\n" + styleDim.Render("press any key to continue"))
	return b.String()
}

package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// applyResults removes successfully deleted worktrees and branches from the
// model and records the results for the summary screen.
func (m *Model) applyResults(msg DeleteResultsMsg) {
	m.results = msg.Results
	deletedWt := map[string]bool{}
	deletedBranch := map[string]bool{}
	for _, r := range m.results {
		if r.Skipped {
			continue
		}
		if r.Item.BranchOnly() {
			if r.BranchErr == nil {
				deletedBranch[branchKey(r.Item.RepoPath, r.Item.Branch)] = true
			}
			continue
		}
		if r.WorktreeErr == nil {
			deletedWt[r.Item.Worktree.Path] = true
			if r.BranchAttempted && r.BranchErr == nil {
				deletedBranch[branchKey(r.Item.RepoPath, r.Item.Worktree.Branch)] = true
			}
		}
	}
	for i := range m.repos {
		repo := &m.repos[i]
		keptWt := repo.Worktrees[:0]
		for _, w := range repo.Worktrees {
			if !deletedWt[w.Path] {
				keptWt = append(keptWt, w)
			}
		}
		repo.Worktrees = keptWt
		keptBr := repo.Branches[:0]
		for _, b := range repo.Branches {
			if deletedBranch[branchKey(repo.Path, b.Name)] {
				continue
			}
			if deletedWt[b.WorktreePath] {
				// The branch survived its worktree (deletion refused or
				// kept on purpose): it is bare now.
				b.WorktreePath = ""
			}
			keptBr = append(keptBr, b)
		}
		repo.Branches = keptBr
	}
	m.selection = map[string]bool{}
	m.rebuildRows()
	m.phase = phaseSummary
}

// summaryLines renders the per-item outcome of the batch (REQ-A8: worktree
// and branch results reported separately): ✓ green, ✗ red, – faint.
func (m Model) summaryLines(width int) []string {
	var lines []string
	add := func(mark string, style lipgloss.Style, text string) {
		lines = append(lines, style.Render(truncate(mark+" "+text, max(width, 1))))
	}
	for _, r := range m.results {
		if r.Item.BranchOnly() {
			name := r.Item.Branch
			switch {
			case r.Skipped:
				add("–", styleDim, name+": skipped")
			case r.BranchErr != nil:
				add("✗", styleBad, fmt.Sprintf("%s: branch deletion failed: %v", name, r.BranchErr))
			case r.BranchForced:
				add("✓", styleGood, name+": branch force-deleted (had unmerged commits)")
			default:
				add("✓", styleGood, name+": branch deleted")
			}
			continue
		}
		name := r.Item.Worktree.Branch
		if r.Item.Worktree.Detached() {
			name = leafLabel(r.Item)
		}
		switch {
		case r.Skipped:
			add("–", styleDim, name+": skipped")
		case r.WorktreeErr != nil:
			add("✗", styleBad, fmt.Sprintf("%s: worktree removal failed: %v", name, r.WorktreeErr))
		default:
			add("✓", styleGood, name+": worktree removed")
			switch {
			case !r.BranchAttempted:
				// Branch kept on purpose or detached: nothing to report.
			case r.BranchErr != nil:
				add(" ✗", styleBad, fmt.Sprintf("branch deletion failed: %v", r.BranchErr))
			case r.BranchForced:
				add(" ✓", styleGood, "branch force-deleted (had unmerged commits)")
			default:
				add(" ✓", styleGood, "branch deleted")
			}
		}
	}
	return lines
}

// viewSummary renders the closing act of the delete modal: the batch
// outcome in the same frame the confirm opened.
func (m Model) viewSummary() string {
	w := m.actWidth()
	lines := m.summaryLines(w)
	h := m.actHeight(len(lines) + modalChromeLines)
	bodyH := max(h-modalChromeLines, 1)

	offset := m.summaryScroll
	if offset > len(lines)-bodyH {
		offset = len(lines) - bodyH
	}
	offset = max(offset, 0)

	pos := ""
	hints := styleDim.Render("press any key to close")
	if len(lines) > bodyH {
		pos = styleDim.Render(fmt.Sprintf("%d-%d/%d", offset+1, offset+bodyH, len(lines)))
		hints = hint("j/k", "scroll") + styleDim.Render("  ·  ") + styleDim.Render("any other key to close")
	}

	title := lipgloss.NewStyle().Bold(true).Render("Deletion summary")
	out := []string{padLine(title, w-lipgloss.Width(pos)) + pos, "", ""}
	out = append(out, windowLines(lines, offset, bodyH)...)
	out = append(out, "", hints)
	for i := range out {
		out[i] = padLine(out[i], w)
	}
	return strings.Join(out, "\n")
}

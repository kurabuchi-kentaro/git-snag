package tree

import (
	"testing"
	"time"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

func br(name string, opts ...func(*domain.Branch)) domain.Branch {
	b := domain.Branch{
		Name:           name,
		LastCommitTime: time.Unix(1_700_000_000, 0),
	}
	for _, o := range opts {
		o(&b)
	}
	return b
}

func brTime(t time.Time) func(*domain.Branch) {
	return func(b *domain.Branch) { b.LastCommitTime = t }
}

func brMerged() func(*domain.Branch) { return func(b *domain.Branch) { b.Merged = true } }
func brGone() func(*domain.Branch)   { return func(b *domain.Branch) { b.UpstreamGone = true } }

func TestBuildBranches_defaultBranchAnchorsTheTree(t *testing.T) {
	t.Parallel()
	root, anchorID := BuildBranches([]domain.Branch{
		br("main"), br("feature/x"), br("feature/y"), br("solo"),
	}, "main")
	if anchorID != "main" {
		t.Errorf("anchorID = %q, want main", anchorID)
	}

	if len(root.Children) != 1 {
		t.Fatalf("root children = %v, want the anchor only", childNames(root))
	}
	anchor := root.Children[0]
	if anchor.Name != "main" || anchor.Branch == nil || anchor.IsGroup() {
		t.Fatalf("anchor = %+v, want the main branch leaf", anchor)
	}
	got := childNames(anchor)
	want := []string{"feature", "solo"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("anchor children = %v, want %v", got, want)
	}
	feature := findChild(t, anchor, "feature")
	if feature.ID != "feature" {
		t.Errorf("group ID = %q, want un-prefixed %q", feature.ID, "feature")
	}
	if leaf := findChild(t, feature, "x"); leaf.ID != "feature/x" {
		t.Errorf("leaf ID = %q, want the full branch name", leaf.ID)
	}
}

func TestBuildBranches_withoutDefaultBranchHangsOffRoot(t *testing.T) {
	t.Parallel()
	root, anchorID := BuildBranches([]domain.Branch{br("a"), br("b")}, "")
	if anchorID != "" {
		t.Errorf("anchorID = %q, want empty without a default branch", anchorID)
	}
	got := childNames(root)
	if len(got) != 2 {
		t.Fatalf("root children = %v, want the branches directly", got)
	}
}

func TestBuildBranches_flattenDescendsThroughAnchor(t *testing.T) {
	t.Parallel()
	root, _ := BuildBranches([]domain.Branch{br("main"), br("feature/x")}, "main")
	rows := Flatten(root, nil)
	if len(rows) != 3 { // main, feature/, x
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	if rows[0].Node.Name != "main" || rows[0].Depth() != 0 {
		t.Errorf("row 0 = %+v, want the anchor at depth 0", rows[0].Node.Name)
	}
	if rows[2].Node.Name != "x" || rows[2].Depth() != 2 {
		t.Errorf("row 2 = %s depth %d, want leaf x at depth 2", rows[2].Node.Name, rows[2].Depth())
	}
	// Collapsing the anchor hides the whole subtree.
	collapsed := Flatten(root, map[string]bool{"main": true})
	if len(collapsed) != 1 {
		t.Errorf("collapsed rows = %d, want the anchor only", len(collapsed))
	}
}

func TestBuildFlatBranches_staleFirstAcrossNames(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_700_000_000, 0)
	root := BuildFlatBranches([]domain.Branch{
		br("fresh", brTime(now)),
		br("old", brTime(now.Add(-100*time.Hour))),
	}, domain.StaleFirst)
	got := childNames(root)
	if got[0] != "old" || got[1] != "fresh" {
		t.Errorf("order = %v, want oldest first", got)
	}
}

func TestFilterBranches_mergedOnlyIncludesGoneUpstreams(t *testing.T) {
	t.Parallel()
	out := FilterBranches([]domain.Branch{
		br("merged", brMerged()),
		br("squashed", brGone()),
		br("active"),
	}, "", true)
	if len(out) != 2 {
		t.Fatalf("filtered = %v, want merged and squashed", out)
	}
	for _, b := range out {
		if b.Name == "active" {
			t.Error("active must be filtered out")
		}
	}
}

func TestFilterBranches_queryMatchesNameAndWorktreePath(t *testing.T) {
	t.Parallel()
	wt := br("checked-out")
	wt.WorktreePath = "/work/somewhere"
	out := FilterBranches([]domain.Branch{wt, br("other")}, "somewhere", false)
	if len(out) != 1 || out[0].Name != "checked-out" {
		t.Errorf("filtered = %v, want checked-out via its path", out)
	}
}

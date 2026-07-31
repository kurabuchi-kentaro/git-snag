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

func brProtected() func(*domain.Branch) { return func(b *domain.Branch) { b.Protected = true } }

func TestBuildBranches_slashHierarchyWithProtectedPinnedFirst(t *testing.T) {
	t.Parallel()
	root := BuildBranches([]domain.Branch{
		br("feature/x"), br("feature/y"), br("solo"), br("main", brProtected()),
	})

	got := childNames(root)
	want := []string{"main", "feature", "solo"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("root children = %v, want %v (protected first, then alphabetical)", got, want)
	}
	if main := root.Children[0]; main.Branch == nil || len(main.Children) != 0 {
		t.Errorf("main should be a plain leaf sibling, got %+v", main)
	}
	feature := findChild(t, root, "feature")
	if feature.ID != "feature" {
		t.Errorf("group ID = %q, want un-prefixed %q", feature.ID, "feature")
	}
	if leaf := findChild(t, feature, "x"); leaf.ID != "feature/x" {
		t.Errorf("leaf ID = %q, want the full branch name", leaf.ID)
	}
}

func TestBuildBranches_compactsSingleChildNamespaces(t *testing.T) {
	t.Parallel()
	root := BuildBranches([]domain.Branch{
		br("main", brProtected()),
		br("feature/only-one"),
		br("chore/deep/nested"),
		br("fix/a"), br("fix/b"),
	})

	got := childNames(root)
	want := []string{"main", "chore/deep/nested", "feature/only-one", "fix"}
	if len(got) != len(want) {
		t.Fatalf("root children = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("root children = %v, want %v (single-child namespaces folded)", got, want)
		}
	}
	if leaf := findChild(t, root, "feature/only-one"); leaf.IsGroup() || leaf.ID != "feature/only-one" {
		t.Errorf("folded leaf = %+v, want the branch leaf keeping its ID", leaf)
	}
	if fix := findChild(t, root, "fix"); !fix.IsGroup() || len(fix.Children) != 2 {
		t.Errorf("fix/ has two children and must stay a group: %+v", fix)
	}
}

func TestBuild_compactsSingleChildNamespaces(t *testing.T) {
	t.Parallel()
	root := Build([]domain.Worktree{wt("feature/only"), wt("fix/a"), wt("fix/b")})
	got := childNames(root)
	want := []string{"feature/only", "fix"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("root children = %v, want %v", got, want)
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

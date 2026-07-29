package tree

import (
	"testing"
	"time"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

func wt(branch string, opts ...func(*domain.Worktree)) domain.Worktree {
	w := domain.Worktree{
		Path:           "/work/" + branch,
		Branch:         branch,
		HeadSHA:        "0123456789abcdef0123456789abcdef01234567",
		LastCommitTime: time.Unix(1_700_000_000, 0),
	}
	for _, o := range opts {
		o(&w)
	}
	return w
}

func withTime(t time.Time) func(*domain.Worktree) {
	return func(w *domain.Worktree) { w.LastCommitTime = t }
}

func withMerged() func(*domain.Worktree)   { return func(w *domain.Worktree) { w.Merged = true } }
func withMain() func(*domain.Worktree)     { return func(w *domain.Worktree) { w.IsMain = true } }
func withDetached() func(*domain.Worktree) { return func(w *domain.Worktree) { w.Branch = "" } }

func childNames(n *Node) []string {
	names := make([]string, 0, len(n.Children))
	for _, c := range n.Children {
		names = append(names, c.Name)
	}
	return names
}

func findChild(t *testing.T, n *Node, name string) *Node {
	t.Helper()
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("child %q not found among %v", name, childNames(n))
	return nil
}

// --- 4.1 Build ---

func TestBuild_slashPrefixesGroupUnderSharedNode(t *testing.T) {
	t.Parallel()
	root := Build([]domain.Worktree{wt("feature/rate-limit"), wt("feature/old-migration")})

	feature := findChild(t, root, "feature")
	if !feature.IsGroup() {
		t.Fatal("feature should be a group node")
	}
	if len(feature.Children) != 2 {
		t.Fatalf("feature has %d children, want 2: %v", len(feature.Children), childNames(feature))
	}
	for _, c := range feature.Children {
		if c.IsGroup() {
			t.Errorf("child %s should be a leaf", c.Name)
		}
	}
}

func TestBuild_doubleNestedBranchCreatesTwoGroupLevels(t *testing.T) {
	t.Parallel()
	root := Build([]domain.Worktree{wt("feature/fix/login-redirect")})

	feature := findChild(t, root, "feature")
	fix := findChild(t, feature, "fix")
	leaf := findChild(t, fix, "login-redirect")
	if leaf.IsGroup() {
		t.Error("login-redirect should be a leaf")
	}
	if leaf.Worktree == nil || leaf.Worktree.Branch != "feature/fix/login-redirect" {
		t.Errorf("leaf worktree = %+v, want full branch name preserved", leaf.Worktree)
	}
}

func TestBuild_slashlessBranchIsRootLevelLeaf(t *testing.T) {
	t.Parallel()
	root := Build([]domain.Worktree{wt("main", withMain())})

	leaf := findChild(t, root, "main")
	if leaf.IsGroup() {
		t.Error("main should be a leaf directly under the root")
	}
}

func TestBuild_detachedWorktreeGetsShortShaLabelAtRoot(t *testing.T) {
	t.Parallel()
	root := Build([]domain.Worktree{wt("", withDetached())})

	if len(root.Children) != 1 {
		t.Fatalf("got %d root children, want 1", len(root.Children))
	}
	leaf := root.Children[0]
	if leaf.IsGroup() {
		t.Error("detached worktree should be a root-level leaf")
	}
	if want := "(detached: 0123456)"; leaf.Name != want {
		t.Errorf("leaf name = %q, want %q", leaf.Name, want)
	}
}

func TestBuild_noWorktreesYieldsEmptyTree(t *testing.T) {
	t.Parallel()
	root := Build(nil)
	if len(root.Children) != 0 {
		t.Errorf("got %d children, want 0", len(root.Children))
	}
}

// --- 4.2 sorting ---

func TestBuild_treeModeSortsAlphabeticallyPerLevel(t *testing.T) {
	t.Parallel()
	root := Build([]domain.Worktree{
		wt("zeta"), wt("feature/b"), wt("alpha"), wt("feature/a"),
	})

	if got := childNames(root); got[0] != "alpha" || got[1] != "feature" || got[2] != "zeta" {
		t.Errorf("root order = %v, want [alpha feature zeta]", got)
	}
	feature := findChild(t, root, "feature")
	if got := childNames(feature); got[0] != "a" || got[1] != "b" {
		t.Errorf("feature order = %v, want [a b]", got)
	}
}

func TestBuild_groupContainingProtectedLeafSortsFirst(t *testing.T) {
	t.Parallel()
	root := Build([]domain.Worktree{
		wt("aaa/one"),
		wt("zzz/protected-inside", withMain()),
		wt("bbb"),
	})

	if got := childNames(root); got[0] != "zzz" {
		t.Errorf("root order = %v, want zzz (contains protected) first", got)
	}
}

func TestBuildFlat_staleFirstSortsOldestFirst(t *testing.T) {
	t.Parallel()
	old := time.Unix(1_000_000_000, 0)
	mid := time.Unix(1_500_000_000, 0)
	fresh := time.Unix(2_000_000_000, 0)
	root := BuildFlat([]domain.Worktree{
		wt("mid", withTime(mid)), wt("fresh", withTime(fresh)), wt("old", withTime(old)),
	}, domain.StaleFirst)

	if got := childNames(root); got[0] != "old" || got[1] != "mid" || got[2] != "fresh" {
		t.Errorf("order = %v, want [old mid fresh]", got)
	}
	for _, c := range root.Children {
		if c.IsGroup() {
			t.Errorf("flat mode should have no groups, got %s", c.Name)
		}
	}
}

func TestBuildFlat_freshFirstSortsNewestFirst(t *testing.T) {
	t.Parallel()
	old := time.Unix(1_000_000_000, 0)
	fresh := time.Unix(2_000_000_000, 0)
	root := BuildFlat([]domain.Worktree{
		wt("old", withTime(old)), wt("fresh", withTime(fresh)),
	}, domain.FreshFirst)

	if got := childNames(root); got[0] != "fresh" {
		t.Errorf("order = %v, want fresh first", got)
	}
}

func TestBuildFlat_mergedFirstThenStalenessTiebreak(t *testing.T) {
	t.Parallel()
	old := time.Unix(1_000_000_000, 0)
	fresh := time.Unix(2_000_000_000, 0)
	root := BuildFlat([]domain.Worktree{
		wt("unmerged-old", withTime(old)),
		wt("merged-fresh", withTime(fresh), withMerged()),
		wt("merged-old", withTime(old), withMerged()),
	}, domain.MergedFirst)

	want := []string{"merged-old", "merged-fresh", "unmerged-old"}
	got := childNames(root)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestBuildFlat_usesFullBranchNamesAndPinsProtected(t *testing.T) {
	t.Parallel()
	old := time.Unix(1_000_000_000, 0)
	root := BuildFlat([]domain.Worktree{
		wt("feature/x", withTime(old)),
		wt("main", withMain()),
	}, domain.StaleFirst)

	got := childNames(root)
	if got[0] != "main" {
		t.Errorf("order = %v, want protected main pinned first", got)
	}
	if got[1] != "feature/x" {
		t.Errorf("flat leaf name = %q, want full branch name feature/x", got[1])
	}
}

// --- 4.3 Flatten / Filter ---

func TestFlatten_collapsedGroupHidesDescendants(t *testing.T) {
	t.Parallel()
	root := Build([]domain.Worktree{wt("feature/a"), wt("feature/b"), wt("solo")})
	feature := findChild(t, root, "feature")

	rows := Flatten(root, map[string]bool{feature.ID: true})
	for _, r := range rows {
		if !r.Node.IsGroup() && r.Node.Name != "solo" {
			t.Errorf("leaf %s should be hidden under collapsed group", r.Node.Name)
		}
	}
}

func TestFlatten_isLastInfoDrivesConnectors(t *testing.T) {
	t.Parallel()
	root := Build([]domain.Worktree{wt("feature/a"), wt("feature/b"), wt("solo")})

	rows := Flatten(root, nil)
	// Sorted order: feature(group) > a, b; then solo.
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4: %+v", len(rows), rows)
	}
	feature, a, b, solo := rows[0], rows[1], rows[2], rows[3]
	if feature.Node.Name != "feature" || feature.IsLast[0] {
		t.Errorf("feature should be a non-last root row: %+v", feature)
	}
	if a.Node.Name != "a" || a.IsLast[1] {
		t.Errorf("a should be a non-last child: %+v", a)
	}
	if b.Node.Name != "b" || !b.IsLast[1] {
		t.Errorf("b should be the last child: %+v", b)
	}
	if solo.Node.Name != "solo" || !solo.IsLast[0] {
		t.Errorf("solo should be the last root row: %+v", solo)
	}
}

func TestFilter_branchSubstringKeepsLeafAndDropsEmptyGroups(t *testing.T) {
	t.Parallel()
	all := []domain.Worktree{wt("feature/rate-limit"), wt("chore/cleanup")}

	got := Filter(all, "rate", false)
	if len(got) != 1 || got[0].Branch != "feature/rate-limit" {
		t.Fatalf("Filter = %+v, want only feature/rate-limit", got)
	}
	root := Build(got)
	if len(root.Children) != 1 || root.Children[0].Name != "feature" {
		t.Errorf("groups without matching leaves should vanish: %v", childNames(root))
	}
}

func TestFilter_pathSubstringMatches(t *testing.T) {
	t.Parallel()
	target := wt("feature/x")
	target.Path = "/somewhere/special-place/x"
	all := []domain.Worktree{target, wt("feature/y")}

	got := Filter(all, "special-place", false)
	if len(got) != 1 || got[0].Path != target.Path {
		t.Errorf("Filter = %+v, want only the path match", got)
	}
}

func TestFilter_mergedOnlyKeepsMergedLeaves(t *testing.T) {
	t.Parallel()
	all := []domain.Worktree{wt("a", withMerged()), wt("b")}

	got := Filter(all, "", true)
	if len(got) != 1 || got[0].Branch != "a" {
		t.Errorf("Filter = %+v, want only merged a", got)
	}
}

func TestFilter_noMatchYieldsEmpty(t *testing.T) {
	t.Parallel()
	got := Filter([]domain.Worktree{wt("a")}, "no-such-thing", false)
	if len(got) != 0 {
		t.Errorf("Filter = %+v, want empty", got)
	}
}

func TestFilter_emptyQueryKeepsEverything(t *testing.T) {
	t.Parallel()
	got := Filter([]domain.Worktree{wt("a"), wt("b")}, "", false)
	if len(got) != 2 {
		t.Errorf("Filter = %+v, want both", got)
	}
}

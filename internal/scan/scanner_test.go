package scan

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kurabuchi-kentaro/git-snag/internal/testutil"
)

// collect drains the channel into a sorted-agnostic set of paths.
func collect(t *testing.T, ch <-chan string) map[string]bool {
	t.Helper()
	got := map[string]bool{}
	for p := range ch {
		got[p] = true
	}
	return got
}

func walkAll(t *testing.T, root string) map[string]bool {
	t.Helper()
	ch, err := Walk(t.Context(), root, DefaultExcludes())
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	return collect(t, ch)
}

func TestWalk_findsRepositoriesInTree(t *testing.T) {
	t.Parallel()
	root := testutil.TempDir(t)
	repoA := testutil.NewRepoAt(t, filepath.Join(root, "team", "alpha"))
	repoB := testutil.NewRepoAt(t, filepath.Join(root, "beta"))

	got := walkAll(t, root)
	if len(got) != 2 || !got[repoA.Dir] || !got[repoB.Dir] {
		t.Errorf("got %v, want exactly {%s, %s}", got, repoA.Dir, repoB.Dir)
	}
}

func TestWalk_skipsDefaultExcludedDirectories(t *testing.T) {
	t.Parallel()
	root := testutil.TempDir(t)
	testutil.NewRepoAt(t, filepath.Join(root, "node_modules", "stray"))
	visible := testutil.NewRepoAt(t, filepath.Join(root, "app"))

	got := walkAll(t, root)
	if len(got) != 1 || !got[visible.Dir] {
		t.Errorf("got %v, want only %s", got, visible.Dir)
	}
}

func TestWalk_linkedWorktreeInsideRootIsNotDoubleCounted(t *testing.T) {
	t.Parallel()
	root := testutil.TempDir(t)
	repo := testutil.NewRepoAt(t, filepath.Join(root, "repo"))
	wtPath := filepath.Join(root, "repo-wt")
	repo.Git("worktree", "add", "-q", "-b", "feature/x", wtPath)
	// A marker directory inside the worktree proves we do not recurse into it.
	if err := os.MkdirAll(filepath.Join(wtPath, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.NewRepoAt(t, filepath.Join(wtPath, "sub", "nested"))

	got := walkAll(t, root)
	if len(got) != 1 || !got[repo.Dir] {
		t.Errorf("got %v, want only the main repository %s", got, repo.Dir)
	}
}

func TestWalk_submoduleIsNotDetectedAsRepository(t *testing.T) {
	t.Parallel()
	root := testutil.TempDir(t)
	sub := testutil.NewRepoAt(t, filepath.Join(root, "lib"))
	sub.Commit("content")
	super := testutil.NewRepoAt(t, filepath.Join(root, "app"))
	// file:// protocol needs an explicit allowance in recent git.
	super.Git("-c", "protocol.file.allow=always", "submodule", "add", "-q", sub.Dir, "vendored")
	super.Git("commit", "-q", "-m", "add submodule")

	got := walkAll(t, root)
	if got[filepath.Join(super.Dir, "vendored")] {
		t.Errorf("submodule working tree detected as repository: %v", got)
	}
	if len(got) != 2 || !got[sub.Dir] || !got[super.Dir] {
		t.Errorf("got %v, want exactly {%s, %s}", got, sub.Dir, super.Dir)
	}
}

func TestWalk_configuredExcludesAreRespected(t *testing.T) {
	t.Parallel()
	root := testutil.TempDir(t)
	testutil.NewRepoAt(t, filepath.Join(root, "dist", "generated"))
	visible := testutil.NewRepoAt(t, filepath.Join(root, "src"))

	excludes := DefaultExcludes()
	excludes["dist"] = true
	ch, err := Walk(t.Context(), root, excludes)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	got := collect(t, ch)
	if len(got) != 1 || !got[visible.Dir] {
		t.Errorf("got %v, want only %s", got, visible.Dir)
	}
}

func TestWalk_streamsResultsAndHonorsCancellation(t *testing.T) {
	t.Parallel()
	root := testutil.TempDir(t)
	// "0-repo" sorts first in the lexical walk; the rest is filler the walk
	// would still be busy with when the first result arrives.
	testutil.NewRepoAt(t, filepath.Join(root, "0-repo"))
	for i := range 300 {
		if err := os.MkdirAll(filepath.Join(root, "zz-filler", string(rune('a'+i%26)), string(rune('a'+i/26))), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithCancel(t.Context())
	ch, err := Walk(ctx, root, DefaultExcludes())
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	select {
	case first, ok := <-ch:
		if !ok {
			t.Fatal("channel closed before first result")
		}
		if filepath.Base(first) != "0-repo" {
			t.Errorf("first result = %s, want the 0-repo repository", first)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no result streamed within timeout")
	}
	cancel()
	// Drain; cancellation must close the channel promptly or the test
	// itself would hang and time out.
	drained := 0
	for range ch {
		drained++
	}
	t.Logf("drained %d results after cancellation", drained)
}

func TestWalk_noRepositoriesYieldsEmptyResult(t *testing.T) {
	t.Parallel()
	root := testutil.TempDir(t)
	if err := os.MkdirAll(filepath.Join(root, "just", "dirs"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := walkAll(t, root)
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestWalk_unreadableDirectoryIsSkippedNotFatal(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	root := testutil.TempDir(t)
	locked := filepath.Join(root, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	visible := testutil.NewRepoAt(t, filepath.Join(root, "open"))
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	got := walkAll(t, root)
	if len(got) != 1 || !got[visible.Dir] {
		t.Errorf("got %v, want only %s", got, visible.Dir)
	}
}

func TestWalk_symlinkLoopIsNotFollowed(t *testing.T) {
	t.Parallel()
	root := testutil.TempDir(t)
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(nested, "loop")); err != nil {
		t.Fatal(err)
	}
	visible := testutil.NewRepoAt(t, filepath.Join(root, "repo"))

	done := make(chan map[string]bool, 1)
	go func() { done <- walkAll(t, root) }()
	select {
	case got := <-done:
		if len(got) != 1 || !got[visible.Dir] {
			t.Errorf("got %v, want only %s", got, visible.Dir)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("walk did not terminate; symlink loop was probably followed")
	}
}

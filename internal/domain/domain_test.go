package domain

import (
	"testing"
)

func TestWorktree_protectedWhenMainOrCurrent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		wt   Worktree
		want bool
	}{
		{"main worktree", Worktree{IsMain: true}, true},
		{"current worktree", Worktree{IsCurrent: true}, true},
		{"main and current", Worktree{IsMain: true, IsCurrent: true}, true},
		{"plain linked worktree", Worktree{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.wt.Protected(); got != tc.want {
				t.Errorf("Protected() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWorktree_detachedWhenBranchEmpty(t *testing.T) {
	t.Parallel()
	if !(Worktree{}).Detached() {
		t.Error("worktree without branch should be detached")
	}
	if (Worktree{Branch: "main"}).Detached() {
		t.Error("worktree with branch should not be detached")
	}
}

func TestSortMode_nextCyclesThroughAllModes(t *testing.T) {
	t.Parallel()
	order := []SortMode{TreeView, StaleFirst, FreshFirst, MergedFirst, TreeView}
	for i := 0; i < len(order)-1; i++ {
		if got := order[i].Next(); got != order[i+1] {
			t.Errorf("SortMode(%d).Next() = %d, want %d", order[i], got, order[i+1])
		}
	}
}

func TestSortMode_onlyTreeViewIsNotFlat(t *testing.T) {
	t.Parallel()
	if TreeView.Flat() {
		t.Error("TreeView should not be flat")
	}
	for _, m := range []SortMode{StaleFirst, FreshFirst, MergedFirst} {
		if !m.Flat() {
			t.Errorf("SortMode(%d) should be flat", m)
		}
	}
}

# git-snag

> A TUI to snag your dead worktrees.

In forestry, a *snag* is a standing dead tree — often one left behind by
fire. Your projects accumulate them too: worktrees for branches long merged,
experiments abandoned months ago, directories you `rm -rf`'d but git still
remembers. git-snag finds them all and lets you clear them out.

**Status: pre-release, under active development. Not yet functional.**

## What it does

- Scans below the current directory for git repositories and lists every
  worktree across all of them, `tree`-command style.
- Shows what makes each worktree safe (or unsafe) to delete: merged into the
  default branch, uncommitted changes, unpushed commits, locked, or already
  gone from disk (prunable).
- Multi-select (including vim-style visual range selection), one confirmation,
  batch delete — worktree and branch together.
- Deletes go out with a bang. (Explosion animation adapted from
  [lazygit](https://github.com/jesseduffield/lazygit)'s nuke animation —
  thanks!)

## Install

Not yet released. Once v0.1 is out:

```sh
brew install kurabuchi-kentaro/tap/git-snag
# or
go install github.com/kurabuchi-kentaro/git-snag/cmd/git-snag@latest
```

Installed on PATH it works both as `git-snag` and as `git snag`.

## License

[MIT](LICENSE)

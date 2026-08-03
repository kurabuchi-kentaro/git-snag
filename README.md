# git-snag

> A TUI to snag your dead worktrees.

In forestry, a *snag* is a standing dead tree — often one left behind by
fire. Your projects accumulate them too: worktrees for branches long merged,
experiments abandoned months ago, directories you `rm -rf`'d but git still
remembers. git-snag finds them all and lets you clear them out — worktree
and branch together, across every repository under your current directory.

<!-- TODO: demo GIF (vhs/asciinema recording of browse → select → boom) -->

## What it does

- **Scans recursively** below the current directory (or a path you give it)
  and streams every repository it finds into one view. Linked worktrees and
  submodules are recognized structurally and never double-counted.
- **Renders branches as a tree**: `feature/rate-limit` and
  `feature/old-migration` nest under `feature/`, `tree`-command style —
  including accidents like `feature/fix/login-redirect`. Groups collapse,
  and select as a unit.
- **Shows what makes a worktree deletable** at a glance:

  | tag (nerd / unicode) | color | meaning |
  |---|---|---|
  | ` merged` / `✓ merged` | green | merged into the default branch (origin/HEAD) |
  | ` dirty` / `± dirty` | yellow | uncommitted changes |
  | `↑3` | cyan | commits not pushed to the upstream (with count) |
  | ` locked` / `⊘ locked` | blue | locked via `git worktree lock` |
  | `󰊠 gone` / `† gone` | red | directory gone; only git metadata remains (prunable) |
  | ` ` / `⌂ ●` | faint | main worktree / the one you're in — never selectable |

  Tags render with Nerd Font glyphs by default; set `icons: unicode` in the
  config (or pass `--icons unicode`) for plain-Unicode symbols if your
  terminal font is not patched.

- **Sorts to find candidates**: cycle tree view → oldest-first →
  newest-first → merged-first. Non-tree modes render flat with full branch
  names. `/` filters by branch or path; `m` shows merged only.
- **Selects in bulk**: space toggles a worktree, a group, or a whole
  repository (tri-state indicators); `v` starts a vim-style visual range.
- **Deletes with consent**: one confirmation screen lists every selected
  item with its warnings and lets you keep individual branches. Dirty,
  locked, or unpushed worktrees are force-handled *after* you confirm —
  never blocked, never silent. Failures don't abort the rest of the batch,
  and the summary reports worktree and branch outcomes separately.
- **Goes out with a bang.** Deletion plays a short ASCII explosion that
  scales with the batch size, adapted from
  [lazygit](https://github.com/jesseduffield/lazygit)'s nuke animation —
  thanks! Skip it with any key, or disable it entirely.

## Install

```sh
brew install kurabuchi-kentaro/tap/git-snag
# or
go install github.com/kurabuchi-kentaro/git-snag/cmd/git-snag@latest
```

On PATH it works both as `git-snag` and as `git snag`.

Requires the `git` binary. Linux and macOS are supported; Windows works via
WSL.

## Usage

```sh
git snag                  # scan below the current directory
git snag ~/projects       # scan below a specific root
git snag --no-animation   # calm mode
git snag --icons unicode  # no Nerd Font required
```

### Keybindings

| key | action |
|---|---|
| `j`/`k`, arrows | move |
| `h`/`l` | collapse / expand (on a leaf, `h` jumps to its parent) |
| `space` | select worktree / group / repository |
| `v` | visual range select (`v` confirms, `esc` cancels) |
| `d` / `enter` | delete selected… |
| `y` / `n` | …confirm / cancel |
| `/` | filter by branch or path |
| `m` | merged-only toggle |
| `s` | cycle sort: tree → oldest → newest → merged-first |
| `?` | help |
| `q` | quit |

## Configuration

Optional, at `~/.config/git-snag/config.yaml`:

```yaml
scan:
  excludes: [dist, build] # extra directory names to skip while scanning
animation:
  enabled: false          # same effect as --no-animation
ui:
  icons: unicode          # "nerd" (default) or "unicode"; --icons wins
```

## Known limitations

- Bare repositories (`git clone --bare` + worktrees) are not discovered in
  v0.1 — see `docs/adr/0011`.
- Merge detection needs an `origin` remote; local-only repositories simply
  show no merged tags.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Design decisions live in
[`docs/adr/`](docs/adr/).

## License

[MIT](LICENSE)

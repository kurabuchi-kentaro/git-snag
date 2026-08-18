# git-snag — Project Guide

git-snag is a TUI tool for cleaning up `git worktree`s (and their associated
branches) across multiple git repositories discovered under the current
directory. It is delete-only by design: no worktree creation or management.

## Architecture: three layers, one hard rule

| Layer | Packages | Tested how |
|---|---|---|
| A. Git integration (touches disk / spawns `git`) | `internal/gitcli`, `internal/scan`, `internal/enrich`, `internal/action` | Real throwaway repos in `t.TempDir()`, real `git` subprocesses. Never mocked. |
| B. Pure data transforms (no I/O) | `internal/domain`, `internal/tree`, `internal/config` | Plain table-driven unit tests. |
| C. Bubble Tea UI | `internal/ui/*` | Construct a `Model`, feed synthetic `tea.Msg` values, assert on resulting state. No real terminal. |

**Hard rule: `internal/ui` must never import `internal/gitcli` or `os/exec`.**
All mutation goes through `internal/action`; the UI only returns a `tea.Cmd`
that calls `action.Execute(...)` and wraps the result in a `tea.Msg`. Do not
break this as a shortcut — it is what keeps every `Update()` path testable.

Everything non-`main` lives under `internal/` (no `pkg/`): this is a tool, not
a library.

## Testing policy (TDD, classicist)

- Development is test-first: write the failing test, make it pass, refactor.
- Classicist (Detroit school) style: use real collaborators (real git binary,
  real filesystem, real `Model`), assert on state and output. No mock
  frameworks, no interaction-based verification.
- Git-layer tests build real repositories via `internal/testutil`
  (`git init` / `git worktree add` under `t.TempDir()`).
- UI tests inject synthetic `tea.KeyPressMsg` / tick messages into `Update()`
  and assert on the model — never spawn a terminal.

## Build / test / lint commands

Defined as mise tasks in `mise.toml` (`mise tasks` lists them):

```sh
mise run check          # build + vet + lint + test (what CI runs)
mise run test           # go test ./...
mise run lint           # golangci-lint run (config: .golangci.yaml)
mise run fmt            # golangci-lint fmt (gofumpt etc.)
mise run install        # go install ./cmd/git-snag (updates `git snag`)
mise run snapshot       # goreleaser build --snapshot --clean
```

The underlying go/golangci-lint commands work directly as well.

## Hooks and secret scanning

CI (`.github/workflows/ci.yml`) is the only gate that applies to everyone:
build + vet + lint + test + gitleaks secret scan. Local hooks are a
convenience layer on top:

- **Contributors working via plain git**: run `lefthook install` once
  (config: `lefthook.yml`). pre-commit runs fast checks (format + vet);
  pre-push runs the heavy gate (test + lint + `gitleaks git --staged`).
- **Maintainer working via jj (Jujutsu, colocated)**: jj does not fire git
  hooks. Use instead:
  - `jj fix` with `golangci-lint fmt --stdin` registered as a fix tool
    (formats every mutable revision). Note `--stdin` ignores path-based
    exclusions — keep generated-code exclusion on `generated: strict` or
    jj-side glob patterns.
  - `golangci-lint run --fix` manually for lint autofixes (it rewrites files
    in place and needs whole-package type info, so it cannot be a jj fix
    tool), then `jj absorb` to distribute the fixes into the right commits.

See `docs/adr/0012-hooks-lefthook-gitleaks-and-jj-fix-workflow.md` for the
full rationale.

## Releases

Releases are automated by tagpr: merging the release PR it keeps open against
`main` tags the commit and runs goreleaser. Nothing in the tree records the
version — no version file, no `CHANGELOG.md` — so never hand-edit one. See
`docs/adr/0015-tagpr-release-prs-with-goreleaser-via-workflow-call.md`.

## Conventions

- All project-facing text is English: README, CONTRIBUTING, ADRs, commit
  messages, code comments.
- Significant design decisions are recorded as ADRs in `docs/adr/`
  (Context / Decision / Consequences). Do not write a monolithic design doc.
- Commit messages: imperative mood, concise subject line.

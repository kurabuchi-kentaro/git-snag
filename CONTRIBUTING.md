# Contributing to git-snag

Thanks for your interest! Issues and pull requests are welcome.

## Development setup

Requirements: Go (see `go.mod`), git, and ideally [mise](https://mise.jdx.dev)
for the dev tools.

```sh
git clone https://github.com/kurabuchi-kentaro/git-snag
cd git-snag
mise install        # golangci-lint, lefthook, gitleaks, goreleaser
lefthook install    # git hooks: fast checks on commit, full gate on push
```

Day-to-day commands are defined as mise tasks (`mise tasks` lists them):

```sh
mise run check     # build + vet + lint + test (what CI runs)
mise run test      # test suite only
mise run fmt       # format
mise run install   # install into GOBIN so `git snag` picks up your changes
```

The raw commands (`go test ./...`, `golangci-lint run`, ...) work too.

## Architecture in one minute

Read [AGENT.md](AGENT.md) — it is the canonical project guide. The one rule
that must never break: `internal/ui` never imports `internal/gitcli` or
`os/exec`. All mutation goes through `internal/action` as a `tea.Cmd`.

Significant design decisions are recorded in [`docs/adr/`](docs/adr/). If
your change makes a decision a future contributor would ask "why?" about,
add an ADR (Context / Decision / Consequences).

## Testing policy

Test-first, classicist style:

- Anything touching git or the filesystem is tested against **real
  throwaway repositories** built with `internal/testutil` under
  `t.TempDir()`. Never mock the git layer.
- Bubble Tea logic is tested by constructing a `Model` and feeding it
  synthetic `tea.Msg` values — no terminal needed.
- Test names are self-describing
  (`TestListWorktrees_lockedWorktreeIsFlagged`); no ID references.

CI runs build, vet, lint, tests (Linux + macOS), and a gitleaks scan on
every PR — it is the gate that counts.

## Note for jj users

This repository is maintained in a jj/git colocated setup. jj does not fire
git hooks; the maintainer workflow uses `jj fix` (with
`golangci-lint fmt --stdin` as a fix tool) plus a manual
`golangci-lint run --fix` distributed by `jj absorb`. Details in
[`docs/adr/0012`](docs/adr/0012-hooks-lefthook-gitleaks-and-jj-fix-workflow.md).

## Pull requests

- Keep commits focused; write imperative-mood messages in English.
- Add or update tests for behavior changes.
- `go test ./...` and `golangci-lint run` must pass.

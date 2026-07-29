# 0012. Hooks: lefthook + gitleaks, with a jj fix workflow for the maintainer

Date: 2026-07-30

## Context

The repository is maintained in a Jujutsu (jj) + git colocated setup. jj
operates on the git object store directly and never fires git hooks
(`.git/hooks/*`), so any hook-based quality gate silently does not apply to
the maintainer's own commits. External contributors, on the other hand, work
via plain git and benefit from fast local feedback before CI.

We also want secret scanning (gitleaks) from the very first commit.

## Decision

Three complementary layers:

1. **CI is the only universal gate.** `.github/workflows/ci.yml` runs build,
   vet, lint, tests, and a gitleaks scan on every push/PR. Nothing relies on
   local hooks for correctness.
2. **lefthook for git-based contributors** (`lefthook.yml`, enabled via
   `lefthook install`): pre-commit runs fast checks (format diff + vet);
   pre-push runs the heavy gate (tests + lint + gitleaks full scan).
3. **jj fix + jj absorb for the maintainer.** `jj fix` requires tools that
   read file contents on stdin and write the fixed contents to stdout.
   `golangci-lint fmt --stdin` satisfies that contract and is registered as a
   fix tool, applying all configured formatters across mutable revisions.
   `golangci-lint run --fix` does *not* fit the contract (it rewrites files
   in place, prints diagnostics to stdout, and needs whole-package type
   information), so lint autofixes are applied manually in the working copy
   and distributed to the right commits with `jj absorb`.

Because `--stdin` deprives golangci-lint of file paths, path-based exclusions
do not apply in the jj fix path. Generated-code exclusion therefore uses
`generated: strict` (source-comment detection) in `.golangci.yaml` instead of
path patterns.

## Consequences

- The maintainer's commits are still fully checked — by CI, not by local
  hooks. Local hook breakage can never block jj workflows.
- Contributors get fast pre-commit feedback and a heavier pre-push gate
  without CI round-trips.
- Two parallel local workflows must be documented (AGENT.md, CONTRIBUTING),
  and formatter configuration must stay stdin-compatible.

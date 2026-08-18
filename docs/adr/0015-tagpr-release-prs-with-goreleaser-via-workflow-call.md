# 0015. tagpr for release PRs, with goreleaser chained via workflow_call

Date: 2026-08-18

## Context

Releases were cut by hand: tag `vX.Y.Z` locally, push it, and let
`.github/workflows/release.yml` (`on: push tags`) run `goreleaser release`.
That works but leaves the release decision undocumented — nothing shows what
is unreleased on `main`, and the version bump is whatever the maintainer
types.

[tagpr](https://github.com/Songmu/tagpr) turns that into a merge button: it
keeps a "next release" pull request in sync with `main`, and tags the merge
commit when the PR is merged. Three details of this repository shape how it
is wired in:

1. **There is no version file.** The version reaches the binary through
   goreleaser ldflags into `internal/version`; git tags are the only source
   of truth.
2. **goreleaser already owns the GitHub Release** (archives, checksums, the
   Homebrew tap update via `HOMEBREW_TAP_TOKEN`). tagpr can create Releases
   too, which would collide.
3. **A tag pushed with `GITHUB_TOKEN` does not trigger other workflows.**
   tagpr pushes the tag with the token from its `GITHUB_TOKEN` environment
   variable (injected as an `http.<host>.extraheader`), so `release.yml`'s
   `on: push tags` trigger would simply never fire. The upstream workaround
   is to hand tagpr a personal access token or a GitHub App token instead.

## Decision

**Chain goreleaser off tagpr's `tag` output instead of introducing a token
with write access.** `.github/workflows/tagpr.yml` runs on pushes to `main`;
its `release` job is gated on `needs.tagpr.outputs.tag != ''` and invokes
`.github/workflows/release.yml` through `workflow_call`, passing the new tag
as the checkout `ref` and `secrets: inherit` so the tap token still reaches
goreleaser.

The existing `HOMEBREW_TAP_TOKEN` is deliberately *not* reused for this: it
is issued for the `homebrew-tap` repository, and widening a tap-publishing
credential into a tag-pushing one for this repository is a scope increase
that the output-chaining approach makes unnecessary. No new secret is needed
at all.

`release.yml` keeps its `on: push tags: ["v*"]` trigger. A tag pushed by hand
carries the maintainer's own credentials and fires it normally, so manual
releases remain available as a fallback; tagpr's tag push cannot fire it, so
there is no double release.

Configuration in `.tagpr`:

- `versionFile = -` — the documented value for "rely on git tags only and
  skip file updates". The next version comes from `tagpr:minor` /
  `tagpr:major` labels on the release PR, defaulting to a patch bump.
- `vPrefix = true` — matches the existing `v0.1.0` / `v0.1.1` tags.
- `release = false` — tagpr tags and stops; goreleaser creates the Release.
- `changelog = false` — no `CHANGELOG.md`. Release notes have a single
  source, goreleaser's `changelog: use: git`; a tagpr-maintained changelog
  would generate the same content a second way and make every release PR
  touch a conflict-prone file.

`.github/release.yml` is committed up front (excluding the `tagpr` label)
because tagpr creates it on its first run otherwise. It only feeds tagpr's
release PR body; goreleaser ignores it.

Permissions are per job: `contents: write` + `pull-requests: write` +
`issues: read` for tagpr (the last one to read labels of merged PRs),
`contents: write` for the release job, and `contents: read` at the workflow
level.

## Consequences

- Releasing is merging the tagpr PR. The PR itself is the record of what is
  about to ship, and the version is adjusted by labelling it rather than by
  typing a tag.
- The repository setting *Settings > Actions > General > Allow GitHub Actions
  to create and approve pull requests* must be enabled, or tagpr cannot open
  its PR.
- The release runs as a downstream job of the `tagpr` workflow, so its logs
  live under that workflow run rather than under a tag push. Anyone looking
  for "the release run" has to know that.
- `release.yml` has two entry points and must keep working from both; the
  checkout ref is `inputs.ref || github.ref` for exactly that reason.
- If tagpr is ever given a PAT or GitHub App token, the chaining job can be
  dropped and the plain tag trigger takes over again.

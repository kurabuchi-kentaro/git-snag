# 0008. One global XDG config file, no per-project config

Date: 2026-07-30

## Context

git-snag scans *across* repositories from an arbitrary starting directory,
so there is no single "project" whose local config file would naturally
apply — honoring per-repository config files would mean merging an
unpredictable number of them per run, with ordering questions and surprising
overrides. The tool must also work with no configuration at all.

## Decision

Configuration lives in exactly one optional file:
`$XDG_CONFIG_HOME/git-snag/config.yaml` (via `os.UserConfigDir`). It
currently holds extra scan excludes and the animation toggle. CLI flags
(`--config` for an alternate path, `--no-animation`) always win over the
file. A missing file is normal; a present-but-broken file aborts startup
loudly (REQ-A6) rather than silently running with defaults.

## Consequences

- Zero-config by default; one obvious place to look when customizing.
- Per-project exclude needs must be expressed globally (or via the scan
  root argument). If real demand for project-level config appears, it can
  be added later with explicit precedence rules in a new ADR.
- YAML was chosen over TOML for consistency with the Go CLI ecosystem's
  prevailing convention (gh, k9s).

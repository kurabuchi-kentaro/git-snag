# 0001. Record architecture decisions

Date: 2026-07-30

## Context

git-snag's requirements were settled in an extensive design session before
any code was written. Publishing that raw requirements document in the repo
would age badly: it would start drifting from the implementation at the first
commit. Contributors still need a way to learn *why* non-obvious decisions
were made.

## Decision

Record individually significant design decisions as Architecture Decision
Records (ADRs) in `docs/adr/`, numbered sequentially, using this minimal
format: Context / Decision / Consequences. Write an ADR when a decision would
otherwise make a future contributor ask "why is it like this?".

## Consequences

- Decisions are documented in small, durable units instead of one monolithic
  design doc that would rot.
- Writing an ADR is a small extra step when landing a significant change.
- ADRs are immutable history: superseded decisions get a new ADR that
  references the old one rather than editing it in place.

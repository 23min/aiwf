---
id: ADR-0055
title: Keep aiwf-repo-only rules out of the shipped aiwf check
status: proposed
---
> **Date:** 2026-09-26 · **Decided by:** human/peter

## Context

`aiwf check` is compiled into the binary every consumer installs:
`go list -deps ./cmd/aiwf` lists `internal/check`. Two of its rules have a subject that
exists only in this repository. `skill-body-id` and `skill-body-claude-md-section`
read aiwf's own embedded skill trees and are inert in a consumer repo. The shipped
`aiwf-check` skill still documents them to consumers, in rows that name this
repository's source paths, so the rules meant to stop aiwf-only references from
shipping are themselves shipped aiwf-only references.

`internal/policies` is not compiled into the binary; `go list -deps ./cmd/aiwf` does
not list it. Its tests run in `make check-fast`, `make ci` and CI.

Two alternatives were considered:

- **Keep repo-only rules in `aiwf check`,** so they fire at the pre-push hook. This is
  rejected because it ships code and documentation that mean nothing downstream, and
  the documentation is itself a leak.
- **Run a repo-only rule from this repository's own pre-push hook.** This is not
  needed. The policy suite already runs locally in `make check-fast`, and nothing
  reaches a consumer before a release tag, by which point CI has run.

## Decision

- A rule whose subject exists only in this repository lives in `internal/policies`,
  not in `internal/check`. That covers its source tree, its embedded content, its own
  CLAUDE.md and its own history.
- A rule that means something in a consumer's repository ships in `aiwf check`, and
  is not duplicated as a policy test (D-0081).

## Consequences

- `skill-body-id` and `skill-body-claude-md-section` leave `internal/check` (M-0355).
  The repo-only gate takes over what they check (M-0354).
- A repo-only rule fires in the policy suite rather than at the pre-push hook. A
  violation can reach `main`, where CI catches it before a release tag.
- The shipped `aiwf-check` skill documents only rules a consumer's repository can
  trip.
- CLAUDE.md's "What's enforced and where" lists `skill-body-id` among the pre-push
  `aiwf check` rules, so that line changes with M-0355.

## Validation

The decision holds while no rule in `internal/check` is inert in a consumer
repository. Revisit it if a repo-only rule must block before a push rather than
before a release.

## References

- E-0096, M-0354, M-0355
- D-0081 — a rule that applies in a consumer's repository is not duplicated as a
  policy test
- G-0538, G-0548

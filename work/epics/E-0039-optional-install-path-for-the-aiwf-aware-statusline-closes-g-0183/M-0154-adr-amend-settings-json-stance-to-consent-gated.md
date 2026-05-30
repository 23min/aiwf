---
id: M-0154
title: 'ADR: amend settings.json stance to consent-gated'
status: draft
parent: E-0039
tdd: none
---
# M-0154 — ADR: amend settings.json stance to consent-gated

## Goal

Record the decision to amend aiwf's documented "never edits settings.json"
stance to "never edits **without explicit per-invocation consent**," so the
consent-gated wiring milestone (M-0156) builds on a ratified decision rather
than an ad-hoc relaxation.

## Context

`internal/cli/doctor/doctor.go` (the marketplace-overlap comment) and CLAUDE.md
both state aiwf never edits `settings.json`. Shipping a statusline that can wire
itself in — even with consent — revises that invariant. Per CLAUDE.md's
"Authoring an ADR" rule, the decision is recorded as a choice; *when* to act on
it stays in the planning surface, not in the ADR body.

## Acceptance criteria

<!-- Formal ACs added at start-milestone via `aiwf add ac M-0154`. Intended shape: -->

An ADR exists under `docs/adr/` with `## Context` / `## Decision` /
`## Consequences`, naming the consent mechanism (interactive `[y/N]` on a TTY or
explicit `--wire-settings`) and the `settings.local.json` default for project
scope; CLAUDE.md and the `doctor.go` comment are updated to the amended stance;
a structural assertion pins the ADR's named sections and the CLAUDE.md change.

## Constraints

- The ADR body carries **no gate language** ("ratify after X") — decision is
  decision, per CLAUDE.md's ADR-authoring rule.
- Mechanical evidence is a structural section assertion (this milestone is
  `tdd: none` — a doc deliverable, not red-green code).

## Design notes

- Leaning a full ADR (not a lighter `decision` entity) because it revises a
  documented invariant. The FSM `proposed → accepted` via `aiwf promote` is the
  ratification surface; no bespoke status-pinning test.

## Out of scope

- Implementing the wiring (M-0156). This milestone records the decision only.

## Dependencies

- None.

## References

- [E-0039](epic.md) · `internal/cli/doctor/doctor.go` · CLAUDE.md · ADR-0014 (embed precedent)

---

## Work log

- (pending)

## Decisions made during implementation

- (none)

## Validation

- (pending)

## Deferrals

- (none)

## Reviewer notes

- (none)

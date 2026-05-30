---
id: M-0154
title: 'ADR: amend settings.json stance to consent-gated'
status: in_progress
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
scope. This milestone is the **sole owner** of the prose stance amendment: it
updates every surface that today states "aiwf never edits settings.json" — the
CLAUDE.md operator-setup line, the `doctor.go` marketplace-overlap comment, and
the `doctor.go` user-facing "aiwf will not edit your settings.json" string — to
the amended "not without explicit per-invocation consent." (M-0156, the wiring
milestone, does **not** touch this prose.) Mechanical evidence is a structural
assertion scoped to the named ADR sections and to each amended surface — asserted
within its section/string, not via a loose whole-file grep, per CLAUDE.md's
"substring assertions are not structural assertions" rule.

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

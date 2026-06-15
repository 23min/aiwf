---
id: M-0163
title: Embed and materialize the guidance fragment
status: in_progress
parent: E-0040
tdd: required
acs:
    - id: AC-1
      title: Materialized guidance file contains every full-set rule and the id-shape line
      status: met
      tdd_phase: done
    - id: AC-2
      title: Materialized guidance file declares the aiwf version it was generated from
      status: open
      tdd_phase: red
    - id: AC-3
      title: init and update materialize the gitignored guidance file idempotently
      status: met
      tdd_phase: done
    - id: AC-4
      title: Guidance fragment stays within its per-turn line budget
      status: open
      tdd_phase: red
---
# M-0163 — Embed and materialize the guidance fragment

## Goal

Ship the version-pinned aiwf guidance fragment as an embedded artifact that
`aiwf init` and `aiwf update` materialize to `.claude/aiwf-guidance.md`
(gitignored), without yet wiring the consumer's `CLAUDE.md`. After this milestone
the guidance file exists in a consumer tree but is not yet imported — a coherent
intermediate state.

## Context

E-0040 reaches the consumer's per-turn surface in two steps: materialize the
content (here), then wire the import line (the next milestone). The fragment
content and the rule for what may enter it are settled in ADR-0018. The shipping
mechanism is the embed-and-materialize manifest established by ADR-0014 / E-0038;
this milestone adds one more materialized artifact to that set.

## Acceptance criteria

### AC-1 — Materialized guidance file contains every full-set rule and the id-shape line

The fragment carries each rule named in ADR-0018's inclusion set — per-action
gate discipline; never suggest the user pause; collision resolves via
`aiwf reallocate` not `git mv`; AC promotion requires mechanical evidence; Q&A
one decision at a time; finish in context rather than papering over — plus the
id-shape one-liner. Asserted against the embedded bytes via a path constant (the
G-0182 pattern), not a flat grep.

### AC-2 — Materialized guidance file declares the aiwf version it was generated from

The fragment carries a version marker; the test asserts it equals the binary's
own version (`version.Current()`), so a stale materialized copy is detectable.

### AC-3 — init and update materialize the gitignored guidance file idempotently

A fixture-tree test drives `aiwf init` and `aiwf update`, asserting
`.claude/aiwf-guidance.md` is written and the gitignore entry is present; a second
run produces no diff.

### AC-4 — Guidance fragment stays within its per-turn line budget

A line-count assertion holds the fragment under a fixed budget so it stays terse
enough to re-anchor on every turn.

## Constraints

- The materialized file lives in-repo under `.claude/` — never `~/.claude/` or an
  absolute path (ADR-0018).
- The fragment obeys ADR-0018's inclusion principle and its hard boundary (it
  says nothing about the consumer's own code).
- Authored in the embedded snapshot (ADR-0014 / ADR-0016), not a retired
  upstream; AC tests assert against the embedded bytes.

## Design notes

- ADR-0018 — the consent decision and the inclusion principle that govern this
  fragment's content.
- Reuses the embed-and-materialize manifest mechanism (E-0038); the guidance file
  joins the unconditionally-refreshed artifact set (unlike the scaffold-once
  statusline).

## Surfaces touched

- `internal/skills/embedded-rituals/` — the authored fragment source.
- `internal/skills/` — the materialization manifest / writer.

## Out of scope

- The marker-wrapped `CLAUDE.md` import line and its consent flow — the wiring
  milestone.
- The `aiwf doctor` unwired-guidance finding — the doctor-finding milestone.

## Dependencies

- None. (ADR-0018 is `accepted`; the embed-and-materialize mechanism already
  exists.)

## References

- ADR-0018 — risk-calibrated consent + the fragment inclusion principle.
- G-0243 — the gap E-0040 closes.
- ADR-0014 / E-0038 — the embed-and-materialize precedent.
- G-0182 — the embedded-path-constant test pattern.

---

## Work log

<!-- One entry per AC or unit of work; append-only. -->

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)

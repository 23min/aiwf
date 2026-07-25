---
id: M-0278
title: Extract BeginVerbDiag and migrate the verb diagnostic block
status: draft
parent: E-0072
tdd: advisory
acs:
    - id: AC-1
      title: BeginVerbDiag encapsulates the verb diagnostic lifecycle
      status: open
    - id: AC-2
      title: All diagnostic-block verbs route through BeginVerbDiag
      status: open
    - id: AC-3
      title: Migrated verb diagnostics and JSON envelopes are byte-identical
      status: open
---
# M-0278 — Extract BeginVerbDiag and migrate the verb diagnostic block

## Goal

Introduce a `cliutil.BeginVerbDiag(...)` helper and route the ~30 verbs that hand-roll the diagnostic-logging wiring block through it, so the block lives in one place instead of being copy-pasted into every command.

## Context

Every mutating verb opens with the same ~11-line block — `ResolveLogger` → `defer closeDiagLog` → run-id / `WithVerb` setup → `defer EmitVerbOutcome` — which also drags `log/slog` + `os` + `logger` imports into each verb solely to host it. This is the highest-LOC concentration of the G-0447 convergent-duplication cleanup and the first milestone of E-0072. No prior milestone is required.

## Acceptance criteria

<!-- ACs created via `aiwf add ac`; contracts filled below. -->

### AC-1 — BeginVerbDiag encapsulates the verb diagnostic lifecycle

### AC-2 — All diagnostic-block verbs route through BeginVerbDiag

### AC-3 — Migrated verb diagnostics and JSON envelopes are byte-identical

## Constraints

- Behavior-preserving: emitted diagnostic events (run-id, the `verb.completed` / `verb.failed` outcome and its fields) are byte-identical per verb before and after migration.
- The helper must capture the verb's *named* `code` / `sha` returns so the deferred `EmitVerbOutcome` reports the same outcome the inline `defer` did.
- Land the helper plus one pilot verb first and review it before the bulk migration; the bulk step is then a mechanical, diff-shaped change.
- Extraction lands in today's `cliutil`; no anticipatory restructuring for the G-0227 split.

## Out of scope

- The `ResolveRoot → ResolveActor` prelude (its own milestone).
- The regression-guard structural test (its own milestone).
- Any change to what the diagnostic events contain — this is a relocation, not a redesign.

## Dependencies

- None. First milestone of E-0072.

## References

- E-0072 — parent epic.
- G-0447 — the convergent-duplication cleanup this seam completes.

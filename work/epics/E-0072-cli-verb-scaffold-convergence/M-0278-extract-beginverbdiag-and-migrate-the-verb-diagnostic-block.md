---
id: M-0278
title: Extract BeginVerbDiag and migrate the verb diagnostic block
status: in_progress
parent: E-0072
tdd: advisory
acs:
    - id: AC-1
      title: BeginVerbDiag encapsulates the verb diagnostic lifecycle
      status: met
    - id: AC-2
      title: All diagnostic-block verbs route through BeginVerbDiag
      status: met
    - id: AC-3
      title: Migrated verb diagnostics and JSON envelopes are byte-identical
      status: met
---
# M-0278 — Extract BeginVerbDiag and migrate the verb diagnostic block

## Goal

Introduce a `cliutil.BeginVerbDiag(...)` helper and route the ~30 verbs that hand-roll the diagnostic-logging wiring block through it, so the block lives in one place instead of being copy-pasted into every command.

## Context

Every mutating verb opens with the same ~11-line block — `ResolveLogger` → `defer closeDiagLog` → run-id / `WithVerb` setup → `defer EmitVerbOutcome` — which also drags `log/slog` + `os` + `logger` imports into each verb solely to host it. This is the highest-LOC concentration of the G-0447 convergent-duplication cleanup and the first milestone of E-0072. No prior milestone is required.

## Acceptance criteria

<!-- ACs created via `aiwf add ac`; contracts filled below. -->

### AC-1 — BeginVerbDiag encapsulates the verb diagnostic lifecycle

`cliutil.BeginVerbDiag(...)` resolves the logger, sets up the run-id / `WithVerb` context, and returns a `finish`-closure that captures the verb's named `code` / `sha` returns and fires the deferred close + `EmitVerbOutcome`. A single pilot verb, migrated onto the helper, emits diagnostics (run-id, outcome event and its fields) identical to what its inline block emitted — established by an independent review of the pilot before the bulk migration.

### AC-2 — All diagnostic-block verbs route through BeginVerbDiag

Every verb that previously hand-rolled the diagnostic block now calls `BeginVerbDiag`; none reconstructs the block inline. Each such verb no longer imports `log/slog`, `os`, or `logger` where those imports existed only to host the block. (Reference-phrased against the set of verbs carrying the block today, not a fixed count.)

### AC-3 — Migrated verb diagnostics and JSON envelopes are byte-identical

Across the migrated verbs, the emitted diagnostic events and the `--format=json` envelopes are unchanged from pre-migration — verified by the existing verb-metadata and integration JSON suites staying green, with no assertion changes that would mask a diff.

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

## Work log

### AC-1 — BeginVerbDiag helper + promote pilot

Extracted `cliutil.BeginVerbDiag`, returning a finish closure that captures the verb's named `code` / `sha` returns by pointer and fires `EmitVerbOutcome` then close in the inline block's LIFO order; migrated the `promote` pilot and dropped its now-redundant `slog` / `os` / `logger` imports. Independently reviewed before the bulk migration. · commit e0a2b433 · tests: `beginverbdiag_test.go` + the integration diag suite green.

### AC-2 / AC-3 — bulk migration

Migrated the remaining diagnostic-block verbs through the helper family. Extracted a shared `beginVerbDiagCore` engine and added `BeginReadVerbDiag` for the six verbs with no `--actor` flag (`show`, `list`, `check`, `history`, `contract-verify`, `worktree`) that resolve a best-effort actor lazily inside the `Enabled` guard, preserving the git-config exec-avoidance when logging is off (the default). `upgrade` remains a deliberate non-member (non-`"verb"` prefix, non-deferred emit). Net −304 lines across 28 files. · commit cfc1db49 · byte-identical verified by the integration diag suite staying green.

### Corrective

Restored `move`'s diagnostic-entity rationale comment that the bulk-migration script dropped. · commit 74941aa7

## Validation

- `make check-fast` (vet + full `golangci-lint` set + tests): exit 0.
- `make coverage-gate` (diff-scoped branch-coverage + firing-fixture gates): green — every changed line covered.
- `make lint`: 0 issues.
- Byte-identical net: the `internal/cli/integration` diagnostic suite (`wired_verbs_diag_test.go` et al.) pins every migrated verb's emitted events and JSON envelopes and stays green.
- All `internal/cli/...`, `internal/verb`, `internal/policies`, `internal/check` tests green.

## Reviewer notes

- Three independent fresh-context reviews — the AC-1 pilot, the AC-2 bulk migration (exhaustive per-verb), and the wrap design-lens pass over the helper family — all returned APPROVE.
- Design decision (two entrypoints over one): a single eager helper would add an unconditional `git config` subprocess to every default-config (logging-off) read-verb call. The shared-core + lazy-actor-callback shape preserves that exec-avoidance and single-sources both the eager (mutating) and lazy (actorless) shapes.
- `BeginReadVerbDiag` naming: the real axis is "no `--actor` flag / best-effort lazy actor" rather than "read verb" (`worktree-add` is git-plumbing, not a read). The name was kept; the doc comment states the real axis. Non-blocking nit.
- Pre-existing, not introduced here: `internal/cli/integration/correlation_id_test.go` carries a stale comment claiming `rename` has no diagnostic call site — already false before this milestone. Left for a future touch of that file rather than slipping an unrelated file into this wrap.

## Deferrals

None. The remaining E-0072 seams — the `ResolveRoot → ResolveActor` prelude and the regression-guard structural test — are their own milestones (M-0279, M-0280), not deferrals of this one.

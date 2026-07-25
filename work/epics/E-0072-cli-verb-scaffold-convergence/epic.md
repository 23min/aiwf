---
id: E-0072
title: CLI verb-scaffold convergence
status: proposed
---

# E-0072 — CLI verb-scaffold convergence

## Goal

Single-source the two copy-pasted per-verb scaffolds in `internal/cli` — the diagnostic-logging wiring block and the `ResolveRoot → ResolveActor` prelude — so the verb layer stops retyping the same setup at every command. This is the largest and last seam of G-0447's convergent-duplication cleanup.

## Context

G-0447 catalogued convergent duplication (same job, textually-divergent code) across the four core packages and has been worked in independently-shippable seams; six have shipped (trailer-triple, JSON envelope, single-entity write tail, `slices.Contains`, `eachActiveMilestone`, `areaFS`), and three decision/risk-bearing sub-seams were split into their own gaps. What remains is the `internal/cli` verb-scaffold duplication — the highest-LOC concentration, deferred to its own epic because it introduces real abstractions and touches ~30 verbs, beyond a single patch's scope.

Two scaffolds are in play:

- The ~11-line diagnostic-logging wiring block — `ResolveLogger` → `defer closeDiagLog` → run-id / `WithVerb` setup → `defer EmitVerbOutcome` — is copy-pasted into ~30 verbs, and it forces `log/slog` + `os` + `logger` imports into each verb solely to host the block.
- The `ResolveRoot → ResolveActor` prelude, with its identical usage-error arm, is duplicated across ~23 verbs.

The read-verb JSON-envelope duplication that G-0447 also listed in this package is already resolved (the `OKEnvelope` / `FindingsEnvelope` / `ErrorEnvelope` seam).

## Scope

### In scope

- A `cliutil.BeginVerbDiag(...)` helper returning a `finish`-closure that captures the verb's named `code` / `sha` returns and fires the deferred `EmitVerbOutcome`, replacing the hand-rolled diag block at every verb.
- Dropping the `log/slog` / `os` / `logger` imports that existed only to host the inlined block.
- A shared `ResolveRoot → ResolveActor` prelude helper (with the identical usage-error arm) replacing the per-verb copies.
- A structural regression-guard test under `internal/policies` asserting no verb hand-rolls either scaffold.

### Out of scope

- The G-0227 `cliutil` split (into `cliidentity` / `clioutput` / `cligitstate` / `cliflagsupport`). This epic extracts the helpers into today's `cliutil` in place; G-0227 relocates them later via its gofmt-aware import rewrites.
- The deferred G-0447 sub-seams tracked separately in G-0453 (SHA-abbrev helpers), G-0454 (entity id-shape parsers), and G-0455 (`body.go` heading-walkers).

## Constraints

- Behavior-preserving: emitted diagnostic events and `--format=json` envelopes stay byte-identical per verb; `make ci` green at every milestone boundary.
- Branch-coverage hard rule holds for each new shared helper; the extraction pins the helper's behavior once and deletes the copies against it.
- Each milestone is independently shippable and mergeable; no big-bang single commit.
- Extraction lands in today's `cliutil` — no anticipatory restructuring for the G-0227 split (YAGNI; G-0227 owns that move).

## Success criteria

- [ ] The diagnostic-logging wiring block exists in exactly one place, and no verb reconstructs it inline.
- [ ] The `ResolveRoot → ResolveActor` prelude exists in exactly one place, and no verb reconstructs it inline.
- [ ] No verb imports `log/slog` / `os` / `logger` solely to host the diagnostic block.
- [ ] The regression-guard test fails if a future verb re-inlines either scaffold.
- [ ] Verb diagnostic output and JSON envelopes are unchanged (behavior-preserving), verified by the existing verb and integration suites staying green.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the `finish`-closure cleanly capture every verb's return shape, or do a few verbs have an idiosyncratic diag setup that resists the common helper? | no | Surfaced during the first milestone's pilot-verb step; any outliers are enumerated and either adapted or documented as intentional non-members. |
| Where do the extracted helpers ultimately live once G-0227 splits `cliutil`? | no | Not this epic's concern; G-0227's gofmt-aware rewrites relocate them. Landing them in `cliutil` now is deliberate. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| A silent behavior change in the diagnostic/outcome emission across ~30 verbs (e.g. a dropped run-id or a changed outcome event). | med | Pilot one verb first and review it independently before the bulk migration; rely on the existing verb-metadata and integration JSON suites, which pin the emitted shapes. |
| The bulk migration's size makes review shallow. | med | Stage the first milestone internally (helper + pilot reviewed, then mechanical bulk migration reviewed as a diff-shape); regression-guard test in the final milestone backstops re-drift. |

## Milestones

<!-- Ids are allocated by aiwfx-plan-milestones; this list is refined there. -->

- BeginVerbDiag helper + migrate the ~30 verbs + drop the now-redundant `slog` / `os` / `logger` imports (internally staged: helper + pilot verb reviewed first, then the bulk migration) · depends on: —
- Shared `ResolveRoot → ResolveActor` prelude helper + migrate the ~23 verbs · depends on: the BeginVerbDiag milestone
- `internal/policies` structural regression-guard test asserting no verb hand-rolls either scaffold · depends on: the prelude milestone

## References

- G-0447 — the convergent-duplication cleanup this epic executes the remaining scope of; kept open until this epic wraps, then closed.
- G-0227 — the layering/cohesion refactor that will split `cliutil`; this epic is sequence-aware of it (Option A: extract in place now, relocate later).
- G-0453 / G-0454 / G-0455 — the deferred G-0447 sub-seams split out of scope.
- `wf-codebase-health` — the rubric (A2 low-coupling, C1 single-source-of-truth) this convergence cleanup serves.

# Epic wrap — E-0036

**Date:** 2026-05-26
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0036-reconcile-legal-workflow-spec
**Merge commit:** 3b2e0461

## Milestones delivered

- M-0138 — Introduce typed CodedError; convert existing unstructured legality errors
- M-0139 — Refuse cancel of parents with non-terminal children/ACs via coded errors
- M-0140 — Classify legality finding codes; close AC-5 bidirectional arm
- M-0141 — Enforce three-edge scope reachability at verb-time
- M-0142 — Rename gap-resolved-has-resolver to match the gap FSM vocabulary
- M-0143 — Surface Coded error codes in the JSON envelope

## Summary

The epic made E-0033's legal-workflow spec a verified source of truth by retiring the `deferredImplErrorCodes` IOU list. M-0138 introduced the typed `entity.Coded` pattern (ADR-0012) and piloted it on `fsm-transition-illegal`; M-0139 and M-0140 brought the cancel-refusal codes and the legality classifier online, closing the AC-5 bidirectional drift arm so every legality-classed impl code is now pinned to a spec rule. M-0142 renamed `gap-resolved-has-resolver` to match the gap FSM, and M-0143 surfaced `Coded` refusals in the `--format=json` envelope (unifying the legality exit code at `1`). M-0141 was the reviewed-reconcile add-on: it narrowed scope reachability — at both the verb-time and check-time enforcement sites — from the full reference graph to D-0006's exact three-edge tree, closing a scope-leak through governance edges and emitting a structured out-of-scope code. The deferred allowlist now holds only `ac-evidence-missing` (D-0005), which was carved out of this epic by design.

Scope shift mid-flight: M-0141's formal-model arm (making `scope-reach` an executable spec predicate + reclassifying its code to legality) proved to be ADR-worthy greenfield spec-schema design — D-0006 deferred that encoding and never settled it — so per the epic's open-question-1 it was split to G-0171 (recommended as its own epic). M-0141 shipped the behavior fix in full; the formal certification follows.

## ADRs ratified

- ADR-0012 — typed Coded error pattern for legality-pertinent verb refusals (M-0138)

(The five spec decisions ratified at epic start — D-0002, D-0003, D-0004, D-0006, D-0007 — were promoted proposed→accepted during `aiwfx-start-epic`.)

## Decisions captured

- D-0011 — Classify codes with a typed Code descriptor carrying a Class field (M-0138)
- D-0012 — Rename gap-resolved-has-resolver to gap-addressed-has-resolver (M-0142)
- D-0013 — Surface Coded verb refusals as a status:error envelope object, exit 1 (M-0143)
- D-0014 — Narrow scope reachability to D-0006 three edges; split formal-model arm (M-0141)

## Follow-ups carried forward

- G-0169 — Wire `--format=json` into non-FinishVerb verbs and read-display commands
- G-0170 — Apply rollback discards pre-existing uncommitted worktree edits at touched paths
- G-0171 — Executable `scope-reach` global precondition + legality classification (M-0141's formal-model arm; recommend its own epic)

## Handoff

The legal-workflow spec is now bidirectionally drift-checked for the legality class, and verb-time legality refusals are machine-readable on par with `check` findings. The one open structural arm is G-0171: `scope-reach` remains a documented-but-unimplemented spec predicate, and `provenance-authorization-out-of-scope` stays `codes.ClassStructural` until that arm lands (the two are inseparable — a legality code must round-trip through a spec rule). G-0169 and G-0170 are independent quality follow-ups (JSON wiring for the non-chokepoint verbs; a transactional-rollback data-loss fix). D-0005 / the `--evidence` gate remains deliberately out of scope pending the philosophy walk-back.

---
id: D-0008
title: fsm-history-consistent subcodes disjoint; audit-only only on manual-edit
status: proposed
relates_to:
    - M-0130
---
## Sources

- M-0130 spec body §3 (per-subcode definitions): defines `illegal-transition`, `forced-untrailered`, `manual-edit` without explicitly pinning whether the predicates are disjoint or whether audit-only suppression applies per-subcode.
- M-0130/AC-2 review conversation: the disjointness ambiguity surfaced during reading-comprehension audit after AC-2 landed. The literal predicates as stated in the spec body overlap (a commit with illegal transition + no force + no verb matches both illegal-transition and manual-edit). The audit-only question was raised in the same review.
- Class: FP-only — neither Pass A's audit catalog nor Pass B's first-principles catalog enumerated subcode disjointness rules or audit-only-per-subcode policy. This decision pins both ahead of AC-3/AC-4 impl.

## Resolution

Two related design choices for the `fsm-history-consistent` check rule:

**Choice 1: Subcodes are disjoint by construction, not by precedence.**

Each subcode's predicate explicitly carries all the conditions under which it fires, including the negations of conditions other subcodes catch. The three predicates:

- `illegal-transition`: transition NOT in `entity.AllowedTransitions(kind, prior)` AND no `aiwf-force` trailer
- `forced-untrailered`: transition IN FSM AND sovereign-act-shape AND no `aiwf-force` trailer
- `manual-edit`: transition IN FSM AND NOT sovereign-act-shape AND no `aiwf-verb` trailer

By construction, at most one subcode fires per (entity, commit) pair. No precedence logic at emit time; the predicates themselves enforce mutual exclusion.

**Choice 2: Audit-only suppression is restricted to `manual-edit` only.**

Subcode-by-subcode:

- `illegal-transition`: **not** suppressed by audit-only. The FSM is a load-bearing invariant; audit-only can't retroactively make a bypass legal.
- `forced-untrailered`: **not** suppressed by audit-only. The force trailer records the *sovereign nature* of the act; audit-only records *provenance* (who did it). They are orthogonal metadata.
- `manual-edit`: **suppressed** by audit-only, inheriting the existing pattern from `provenance-untrailered-entity-commit` (the warning manual-edit's FSM-specific framing layers on top of).

## Reasoning

**Choice 1 rationale.** The kernel principle *"framework correctness must not depend on the LLM's behavior"* extends to subcode logic. Mechanical, self-contained predicates beat order-dependent precedence rules: a future reader of `forcedUntraileredFindings()` shouldn't need to know about ordering relative to `illegalTransitionFindings()` or `manualEditFindings()` to understand its emission domain. Disjoint-by-construction makes each predicate self-documenting.

The alternative — precedence-at-emit-time (e.g., illegal-transition > forced-untrailered > manual-edit) — would require a coordinating layer that knows the order. That's an implicit ordering invariant the kernel would have to maintain. Disjoint predicates avoid the invariant by structurally preventing the overlap.

**Choice 2 rationale.** Audit-only's semantic domain is **provenance backfill**, narrowly defined: `aiwf <verb> --audit-only --reason "..."` records *"who did this and why, even though I didn't route through a verb at the time."* It does NOT claim *"the change was legal under the FSM"* or *"the change was an authorized sovereign act."*

- The FSM is the kernel's load-bearing invariant. Audit-only can't change what the FSM allows; it can only record provenance for changes the FSM either admitted (legal) or didn't (illegal, but happened anyway). For illegal-transition, the operator's recourse is `--force --reason` going forward, or revert + redo.
- The force trailer records a *kind of act* (sovereign override), not a *backfilled witness statement*. Audit-only and force are orthogonal metadata for orthogonal purposes; conflating them would muddy the principal × agent × scope model.
- Manual-edit is the only subcode whose entire semantic is *"untrailered provenance"* — exactly what audit-only is designed to retroactively record. Inheriting the existing `provenance-untrailered-entity-commit` suppression mechanism keeps the model consistent.

The spec body's silence is consistent with this reading: it explicitly notes manual-edit's overlap with `provenance-untrailered-entity-commit` (the audit-only-suppressible warning) but says nothing about audit-only for the other two subcodes.

## Implementation

What this means for the M-0130 sub-ACs:

- **AC-2 (`illegal-transition`)** — predicate is `(NOT in FSM) AND (no force)`. No audit-only suppression.
- **AC-3 (`forced-untrailered`)** — predicate must be `(IN FSM) AND (sovereign-act-shape) AND (no force)`. The `IN FSM` guard is implicit since sovereign-act-shape transitions are by definition a subset of legal FSM transitions, but the predicate should state it explicitly for self-documentation. No audit-only suppression.
- **AC-4 (`manual-edit`)** — predicate must be `(IN FSM) AND (NOT sovereign-act-shape) AND (no verb)`. The two guards in front of the `no verb` check are the load-bearing disjointness guards. Audit-only suppression applies (mirror the existing `provenance-untrailered-entity-commit` mechanism).

## Follow-up

A future ADR may want to generalize *"subcodes within a check rule are disjoint by construction, not by precedence"* as a kernel-wide design rule, once a second check rule encounters the same design choice. For now this decision is rule-specific.

The sovereign-act-shape enumeration that AC-3 needs is itself an implementation decision (which transitions count as sovereign-act-shape?) and may warrant its own D-NNNN. ADR-0007 (epic ratification) provides the canonical example (epic `proposed → active`); whether others exist is for AC-3's implementer to audit.

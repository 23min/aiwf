---
id: G-0150
title: 'fsm-history-consistent: per-subcode merge-commit observation policy'
status: addressed
prior_ids:
    - G-0148
discovered_in: M-0130
addressed_by:
    - M-0130
---
## What's missing

M-0130/AC-1 (redo) shipped a DAG-aware walker that emits one observation per (entity, commit, parent) tuple where the entity's status differs between commit and parent. For single-parent commits this is straightforward — one observation per status change. For **multi-parent (merge) commits** the walker emits one observation per parent where the status differs, which means a single logical status change that originated on a feature branch will be observed twice in the audit stream:

1. Once at the original commit on the feature branch (parent = the branch's prior commit, real status change).
2. Once at the merge commit on the integration branch (parent = the OTHER branch's tip at merge time, observed as "the merged side brought in this status change").

The walker deliberately does not bake merge semantics into its emission — the function comment explicitly defers this to the AC-2/3/4 predicates. But the per-subcode predicates haven't decided yet, and the design choice is non-trivial. This gap captures that AC-2/3/4 (and any future check rule subcode that consumes walker observations) must explicitly decide its merge-commit policy, and the decision must be documented in the predicate's spec.

## Why it matters

The same logical event being audited twice from two angles produces noisy findings. But filtering merge commits unconditionally is also wrong — a merge that integrates an illegal transition does deserve a finding (the operator chose to bring the illegal state into the trunk). The right answer is per-subcode:

- **illegal-transition (AC-2):** likely SHOULD fire on merge commits. If the merge's resulting state is FSM-illegal vs the integration-target's parent, the trunk now carries an illegal state — that's auditable regardless of how it got there. But the *same* illegal transition firing once at the feature-branch commit and once at the merge is double-counting; deduplication or per-commit-once semantics may be wanted.

- **forced-untrailered (AC-3):** likely SHOULD NOT fire on merge commits. The kernel's sovereign-act rule applies to the *original act* (the commit that performed the FSM-legal-but-sovereign transition), not to the merge that integrates it. A merge commit lacking `aiwf-force` is normal git behavior; firing forced-untrailered on it would force-trailer every integration of a feature-branch promote.

- **manual-edit (AC-4):** likely SHOULD NOT fire on merge commits. Merges don't carry `aiwf-verb` trailers because they aren't routed verbs; firing manual-edit on every merge would produce constant noise. The original manual edit (if any) on the feature branch is the real signal.

So each subcode's predicate needs explicit handling — accept-merge-observation, reject-merge-observation, or accept-with-dedup-against-feature-branch-counterpart. Three subcodes × three possible policies × interaction with audit-only suppression × deduplication across the (commit, parent) pair set = a design space that benefits from being pinned before code lands.

## Resolution paths

- **Per-subcode policy field on observation.** Walker emits `IsMergeCommit bool` (already a candidate field on `statusChange`); each predicate checks the flag and applies its policy. Simple; matches the disjoint-predicates discipline from D-0008. Recommended.

- **Walker-level merge filtering with opt-in via predicate.** Walker emits merge observations behind a separate slice / flag; predicates pull from one or both streams. More flexible but adds API surface.

- **Defer to AC-2/3/4 spec body.** Each AC's wrap-time spec explicitly states the merge-commit policy in its predicate documentation, and the implementation matches. This is the minimum bar regardless of which mechanical option (above) wins — the user-facing semantic must be documented per subcode.

- **A small dedicated D-NNNN (a kernel-vocabulary decision).** If the merge-policy answer turns out to apply uniformly to *all* sub-FSM check rules (a kernel design rule rather than M-0130-specific), capture it as a D-NNNN that future check rules cite. Defer the D-NNNN until a second check rule with subcode observations encounters the same choice.

## Implementation

What this means for the open M-0130 sub-ACs:

- **AC-2 (`illegal-transition`):** the spec body needs to explicitly state the merge policy. Recommendation per the reasoning above: fire on merge commits (the resulting trunk state is the audit target), with per-(commit) dedup so the same trunk SHA only emits once even if multiple parents disagree.

- **AC-3 (`forced-untrailered`):** spec body explicitly skips merge commits. Tests assert no forced-untrailered finding fires on a merge that integrates a sovereign-act-shape transition from a feature branch — the finding fires on the *original* commit, not the merge.

- **AC-4 (`manual-edit`):** spec body explicitly skips merge commits. The walker's per-parent observation for a merge that brings in a manual-edit-shaped change is silently dropped (the original commit's observation is what produces the finding, if any).

- **`statusChange` struct:** add `IsMergeCommit bool` field, populated by the walker from `len(parents) > 1`. Predicates consult the flag; the walker stays merge-policy-free.

## Class

Design-completeness gap on the M-0130 sub-AC predicate semantics. Discovered during AC-1 redo. Affects AC-2/3/4 implementation; resolution before AC-2 lands is desirable so each subcode's first commit reflects the right semantic.

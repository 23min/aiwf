---
id: D-0009
title: 'fsm-history-consistent merge-commit policy: per-subcode, no dedup'
status: accepted
relates_to:
    - G-0148
---
## Sources

- M-0130/AC-1 redo (commit `649bdc5a`): the DAG-aware walker emits one observation per (entity, commit, parent) tuple where status differs. Multi-parent (merge) commits naturally produce per-parent observations — up to one per parent where the status differs. The walker deliberately stays merge-policy-free; the function comment defers per-subcode decisions to AC-2/3/4.
- [G-0148](work/gaps/G-0148-fsm-history-consistent-per-subcode-merge-commit-observation-policy.md) filed alongside AC-1 redo as the open design question this decision now closes.
- Class: design-completeness gap. Discovered during AC-1 redo's confidence audit; affects AC-2/3/4 predicate semantics.

## Resolution

Three related design choices for the `fsm-history-consistent` check rule:

**Choice 1: The walker exposes merge-commit status on every observation.**

`statusChange` gains an `IsMergeCommit bool` field, populated by the walker from `len(parents) > 1` at observation-emit time. The walker stays policy-free — it reports the structural fact, predicates decide what to do with it.

**Choice 2: Each subcode's predicate carries an explicit merge-commit policy.**

- `illegal-transition` (AC-2): **fires on merge observations.** The predicate ignores `IsMergeCommit`; any (kind, prior, next) tuple outside `entity.AllowedTransitions` produces a finding regardless of whether the commit is a merge.
- `forced-untrailered` (AC-3): **skips merge observations.** The predicate checks `IsMergeCommit` and short-circuits to no-fire when true. Sovereign-act-shape transitions are *acts*; merges *integrate* acts.
- `manual-edit` (AC-4): **skips merge observations.** Same rationale as AC-3 — merges aren't manual edits in the per-status-flip sense; the original commit (if any) on the integrated branch is where the manual edit happened.

**Choice 3: No deduplication.**

The kernel does not deduplicate observations across (commit, parent) pairs. If the same logical illegal transition surfaces both at the original commit and at a merge that integrated it, both findings fire. Both are real audit points pointing at real commits in the entity's reachable history. Dedup would require a synthetic equivalence relation across SHAs that the kernel has no other use for; the cost (extra code, ambiguity over which SHA to report) exceeds the benefit (one fewer finding in a rare scenario).

## Reasoning

**Choice 1 rationale.** The walker's job is enumeration; per-subcode interpretation is the predicates'. Exposing `IsMergeCommit` on the observation matches the kernel principle from D-0008: each predicate self-documents its emission domain, including merge handling. A predicate's body literally states `if o.IsMergeCommit { continue }` (or doesn't) — a future reader sees the policy by reading the predicate, not by tracing the walker's filtering logic.

**Choice 2 rationale (per-subcode).** The three subcodes target semantically distinct shapes of acts:

- *Illegal-transition* is about the trunk's reachable history being FSM-consistent. An illegal status that lands in trunk via a merge is just as much a violation of the FSM invariant as one that lands via a direct commit. The audit is location-agnostic: "is there a (parent → child) edge in main's history that's illegal?" If yes, fire. The exotic but real case where a merge *resolves* to a new status illegal vs both parents (a conflict-resolution that picks neither side's status and lands a third) would be silent under merge-skipping; firing on merges catches it.

- *Forced-untrailered* is about the *sovereign nature* of an act. Sovereign acts are *original gestures* — a human (or `--force`-wielding actor) committing to a sovereign-act-shape transition. A merge that integrates a feature-branch promote-to-active didn't perform the sovereign act; the original promote did. The merge is integration ceremony, not a sovereign event. Firing forced-untrailered on every merge that integrates a sovereign-act-shape commit would produce constant noise — every routine merge of a feature branch with a promote would surface a finding pointing at the merge, not the act.

- *Manual-edit* is about *untrailered provenance*. Manual edits are direct frontmatter manipulations the kernel doesn't know how to attribute (no `aiwf-verb` trailer). Merges don't carry `aiwf-verb` trailers either, but for a categorically different reason: they're a routine git operation, not a status change. Treating every merge as a "manual edit" would constantly fire on integration ceremony.

**Choice 3 rationale (no dedup).** The kernel's audit shape is *per-commit per-finding*. Each finding names a specific commit and a specific (prior, next) edge. When an illegal transition exists both at the original commit and at a merge that brings it into trunk, both are accurate: the original commit performed the illegal change; the merge committed to landing it in main. Reporting both gives the operator full audit visibility.

The alternative (synthetic dedup keyed by, e.g., (entity, prior, next)) would have to pick a "canonical" SHA to report — earliest? latest? — and would suppress the other from the operator's view. That's information loss without a clear benefit. In practice, the double-count case requires (a) an illegal transition on a feature branch, (b) the branch being merged in, and (c) the merge resolution adopting the illegal state. The combination is rare; when it occurs, both findings are correct, both are useful.

## Implementation

What this means for the open M-0130 sub-ACs:

- **statusChange struct (internal/check/fsm_history_consistent.go):** add `IsMergeCommit bool` field. Walker (`walkOneEntity`) populates it from `len(parents) > 1` before emitting each observation. Lands as part of AC-2's commit (the predicate AC-2 builds is the first consumer).

- **AC-2 (`illegal-transition`):** predicate fires on every observation where `(prior, next) ∉ entity.AllowedTransitions(kind, prior)` AND `aiwf-force` trailer is absent. `IsMergeCommit` is not part of the predicate. Tests include: an illegal transition on main (fires), an illegal transition on a feature branch + merge (fires twice — once at the original commit, once at the merge), a merge that resolves to an FSM-legal state (no fire), a merge that resolves to an FSM-illegal state vs at least one parent (fires).

- **AC-3 (`forced-untrailered`):** predicate fires on observations where `IsSovereignActShape(kind, prior, next)` AND `aiwf-force` trailer is absent AND `IsMergeCommit` is false. Tests include: sovereign-act-shape transition on main without `--force` (fires), sovereign-act-shape transition with `--force` (no fire), sovereign-act-shape transition on a feature branch + merge into main (fires once on the original commit, no fire on the merge).

- **AC-4 (`manual-edit`):** predicate fires on observations where transition is FSM-legal AND not sovereign-act-shape AND `aiwf-verb` trailer is absent AND `IsMergeCommit` is false. Audit-only suppression applies per D-0008 (same shape as `provenance-untrailered-entity-commit`).

The disjointness invariant from D-0008 holds for non-merge observations: at most one subcode fires per (entity, commit, parent) pair. For merge observations, AC-2 may fire and AC-3/AC-4 will not — still at most one subcode per pair. D-0008's structural invariant survives the merge policy.

## Follow-up

If a future check rule encounters the same "what about merges?" design question, the answer above generalizes: walker exposes `IsMergeCommit`; predicates make per-subcode policy choices in their own bodies. A possible future ADR may codify *"check-rule predicates that consume DAG observations declare their merge policy explicitly, per subcode, in their predicate body"* as a kernel-wide design rule. Defer the ADR until a second check rule encounters the choice.

The `IsMergeCommit` flag's population is purely structural (parent count from `git log`); no extension is needed if future kernel work redefines what "merge" means (e.g., octopus merges with >2 parents — they're still `IsMergeCommit == true`, and per-parent observations still emit correctly).

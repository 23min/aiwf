---
id: D-0010
title: 'fsm-history-consistent: skip merges in all subcodes (supersedes D-0009)'
status: proposed
relates_to:
    - D-0009
---
## Sources

- D-0009 (superseded): the first attempt at pinning the per-subcode merge-commit policy for `fsm-history-consistent`. Chose "AC-2 fires on merges; AC-3 and AC-4 skip merges." Ratified before AC-2's implementation landed.
- M-0130/AC-2 implementation work: with D-0009's policy in code, the kernel's own `aiwf check` (run via the worktree-scoped session binary against the live repo at `/workspaces/aiwf-M-0130-fsm-history`) emitted **44 errors**, all of the same structural shape: merge commits integrating feature-branch milestone wraps into main. None corresponded to a real operator-error illegal transition.
- M-0130/AC-1 (commit `649bdc5a`): the DAG-aware walker emits per-parent observations on merge commits; this decision pins what AC-2/3/4 do with them.
- Class: design-correctness fix. D-0009's reasoning overweighted the exotic conflict-resolution case; empirical run on the live repo surfaced the cost.

## Resolution

**Supersedes D-0009.** Two related design choices for the `fsm-history-consistent` check rule:

**Choice 1 (unchanged from D-0009): The walker exposes merge-commit status on every observation.**

`statusChange` carries an `IsMergeCommit bool` field, populated by the walker from `len(parents) > 1` at observation-emit time. The walker stays policy-free — predicates decide.

**Choice 2 (revised from D-0009): All three subcodes skip merge-commit observations.**

- `illegal-transition` (AC-2): **skips merge observations.** Predicate short-circuits when `IsMergeCommit` is true.
- `forced-untrailered` (AC-3): **skips merge observations.** (Unchanged from D-0009.)
- `manual-edit` (AC-4): **skips merge observations.** Audit-only suppression per D-0008 still applies. (Unchanged from D-0009.)

**Choice 3 (unchanged from D-0009): No deduplication.**

The kernel does not synthesize equivalence relations across SHAs. For non-merge observations there is no double-count by construction.

## Reasoning

**What D-0009 got wrong.** D-0009's "AC-2 fires on merges" argument was: fire on merges to catch the exotic case of a merge resolving to a status illegal vs both parents (a conflict-resolution that adopts a third state). The reasoning made a cost-benefit claim — *"the cost (extra finding in a rare scenario) exceeds the benefit (one fewer finding in a rare scenario)"* — that turned out to be wrong empirically. The "extra findings in a rare scenario" weren't rare at all: every feature-branch milestone wrap that merges into main produces a finding, because main's pre-merge view of the milestone is at an earlier FSM state (e.g., `draft`) and the merge resolves to the feature branch's final state (e.g., `done`). The first-parent edge looks like `draft → done` and the FSM doesn't admit it.

This isn't a forbidden transition — it's the *normal* feature-branch workflow viewed through one of the merge's two parent edges. The actual progression on the feature branch was `draft → in_progress → done` (all FSM-legal), and those individual commits already emit their own (non-merge, legal) observations. The merge integration is integration, not a transition.

The exotic case D-0009 was protecting against is genuinely rare. In aiwf's workflow, status fields are managed by verbs (`aiwf promote`), not by manual merge-conflict resolution. An operator would have to (a) reach a real merge conflict on a status field and (b) manually edit the resolution to a third state and (c) commit it. Each of those steps is a deliberate human gesture. If it happens, the resulting commit will be caught by other checks (the manual frontmatter edit will be missing the `aiwf-verb` trailer; if the operator wanted to land an illegal state intentionally, they'd use `aiwf <verb> --force --reason "..."` and the force trailer would exempt the finding regardless of merge handling).

**What the revised policy preserves.** AC-2's purpose is to catch operators (or aiwf bugs) that produce FSM-illegal transitions. Every non-merge commit's status change is audited by per-parent comparison. A direct hand-edit, an aiwf verb bug, an attempted skip-ahead promote on a feature branch — all are caught on the original commit. The merge that integrates a legal feature-branch wrap is silent, which matches the operator's mental model.

**What the revised policy gives up.** A merge commit that resolves a status-field conflict to a third state that's FSM-illegal vs both parents is no longer caught by AC-2. This is the trade-off; the cost is the routine-noise that D-0009 produced.

**Consistency across subcodes.** The revised policy makes all three subcodes uniform: each consults `IsMergeCommit` and skips merges. The walker-vs-predicate separation from D-0008 holds — the walker enumerates structurally, the predicates apply per-subcode policy. The disjointness invariant from D-0008 also holds vacuously: for merge observations, no predicate fires.

## Implementation

What this means for the M-0130 sub-AC code:

- **statusChange struct (internal/check/fsm_history_consistent.go):** `IsMergeCommit bool` field stays. (Already landed in AC-2's commit-in-progress.)

- **AC-2 (`illegal-transition`) — change from D-0009:** `illegalTransitionFindings` predicate now starts with `if o.IsMergeCommit { continue }`. The (commit, parent) edge for merge commits is no longer audited; the routine feature-branch-integration noise vanishes.

- **AC-3 (`forced-untrailered`) — unchanged from D-0009:** predicate skips when `IsMergeCommit` is true.

- **AC-4 (`manual-edit`) — unchanged from D-0009:** predicate skips when `IsMergeCommit` is true; audit-only suppression per D-0008 still applies.

- **Tests:** the AC-2 fixture test that previously asserted "fires twice on illegal-then-merged per D-0009 no-dedup" is updated to assert one finding (the original commit on the branch; the merge integration is silent). A new positive test pins "an FSM-illegal merge resolution to a third state is NOT caught by AC-2 — the operator's accountability is via the force trailer or other checks." That non-detection is the explicit trade-off recorded here.

- **Live-repo verification:** `aiwf check` against the kernel's own tree returns 0 errors. The 44 D-0009 false positives go silent. (Pre-existing warnings remain.)

## Follow-up

If a future operator-experience report surfaces a real instance of "a merge resolved to an FSM-illegal state and the kernel missed it," this decision may need revisiting. The expected mitigation in that scenario is operator discipline (use `aiwf <verb> --force --reason "..."` for intentional sovereign acts), not changes to the check rule. If discipline isn't enough and the case becomes recurring, a future decision can re-introduce per-merge auditing with a narrower scope (e.g., "fire only when the merge's status is unreachable from any parent in the FSM").

The cost of *not* having merge audit today: missed catching of an exotic class of operator-introduced FSM violations on merge resolution. The benefit: the kernel's own `aiwf check` passes; the rule is usable.

---
id: G-0355
title: Wrap declared-sequence bundles the trunk merge with downstream bookkeeping
status: open
---
## What's missing

The wrap rituals' terminal **declared-sequence gate** (`wf-patch` step 9; the
`aiwfx-wrap-milestone` / `aiwfx-wrap-epic` terminal sequences) bundles three
actions under a single approval: the **merge** into the integration target, the
**tracker/status closure** (`aiwf promote G-NNNN addressed` / `aiwf promote E-NN
done`), and **cleanup** (branch + worktree removal).

The merge is the only member of that set whose success depends on
integration-target state that concurrent sessions can mutate between the
reconcile check and the merge — it can fail or require reconciliation. The
promote and cleanup are bookkeeping that *presume the merge already landed*.
Bundling a fragile, state-dependent action with steps that assume its success is
safe only if the sequence **hard-aborts the instant the merge doesn't cleanly
land** — a property that today lives entirely in operator discipline, with no
mechanical backstop and no structural separation in the ritual bodies.

## Why it matters

When the integration target diverges between the ancestry check and the merge,
an `--ff-only` merge refuses (correct) — but nothing stops the operator (human
or LLM) from proceeding to the promote, closing the tracker by a commit not yet
in trunk. This happened live during the G-0346 wrap: `main` advanced under the
patch branch during `make ci`, the `--ff-only` merge refused, and the batched
promote ran onto the diverged trunk anyway, recording `addressed_by_commit` for
a commit `main` did not yet contain. Recovery required merging trunk back into
the branch and re-running the gate.

Filing the merge under "wrap" also frames the single riskiest, state-dependent
step as routine wrap-up bookkeeping, which is what obscured the risk in the
first place. Integration is not bookkeeping.

## Direction

Split the merge out of the wrap declared-sequence into **its own gate**
(operator preference: separate gate), across all three ritual bodies:

- The merge to the integration target becomes its own approval — it is the
  integration act, and the step that surfaces divergence, not bookkeeping.
- The reconcile-first ancestry check (shipped as prose in G-0346) moves *inside*
  that merge step, run immediately before the merge — not as an earlier
  precondition a concurrent push can invalidate.
- The remaining bookkeeping (tracker closure + cleanup) may stay a
  declared-sequence gate, since those are deterministic once the merge lands.
- Update `wf-patch`, `aiwfx-wrap-milestone`, `aiwfx-wrap-epic` and their
  referencing structural tests under `internal/policies/`.

Stronger complementary follow-up (inherited from G-0346, now untracked since
that gap closed): a **mechanical wrap preflight** that asserts `git merge-base
--is-ancestor <target> <branch>` and blocks the merge step when the target has
diverged. The separate gate is the ritual-structure fix; the preflight is the
mechanical enforcement the framework's "correctness must not depend on LLM
behavior" principle argues for. This session is direct evidence the prose-only
form is insufficient — the reconcile guidance shipped and was violated in the
same wrap that shipped it.

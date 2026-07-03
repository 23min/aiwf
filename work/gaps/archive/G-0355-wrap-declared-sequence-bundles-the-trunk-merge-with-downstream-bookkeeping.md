---
id: G-0355
title: Wrap declared-sequence bundles the trunk merge with downstream bookkeeping
status: addressed
addressed_by_commit:
    - b2f7d7e7
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

The load-bearing fix is a **mechanical backstop, not a new human gate.** A
tracker closure carrying `--by-commit` already runs `validateAddressedByCommit`
(G-0186, `internal/verb/promote.go`) — but that only asserts each SHA is a *real
commit*, not that it is *reachable from the integration target*. G-0346 slipped
through exactly there: the patch commit was real (it lived on the patch branch)
but was not on `main` after the `--ff-only` refused.

- **Strengthen the promote-time check from existence to reachability.** On the
  normal path, a gap's `addressed_by_commit` SHAs must be ancestors of the
  mainline/trunk tip (`git merge-base --is-ancestor <sha> <trunk>`); a closure
  recorded for a commit trunk does not contain is **refused**, not merely warned
  after the fact by `gap-addressed-has-resolver`. Keep the `--force` / `--reason`
  escape the adjacent existence check already carries for its documented
  exceptions (a cross-repo reference, a commit on an unmerged fixing branch).
  This is the guarantee that does not depend on the operator honoring an abort —
  the framework's "correctness must not depend on LLM behavior" principle.
  Belt-and-suspenders: mirror it as a pre-push `aiwf check` finding so the
  closure is caught before it leaves the machine regardless of which branch the
  promote ran on.

- **Keep the wrap declared-sequence gate bundled — do *not* split the merge into
  its own gate.** Once the reachability check refuses a promote onto a commit
  trunk lacks, splitting adds a human gate without adding a guarantee: a separate
  *prose* gate is exactly as bypassable as a bundled one — G-0346 barreled
  straight through the abort-on-deviation clause the bundled gate already
  carried. The gate-split is optional framing polish, explicitly not the fix.
  (This reverses the earlier "operator preference: separate gate.")

- **Correct and relocate the reconcile-refresh, run immediately before the
  merge** (inside the merge step, not as an earlier precondition a concurrent
  push invalidates):
  - Compare against the **actual integration target** — local `main` for
    `wf-patch` / `aiwfx-wrap-epic` (**not `origin/main`**: the G-0346 divergence
    was *local* `main` advancing under a concurrent session), the epic branch for
    `aiwfx-wrap-milestone`.
  - `git fetch` first and fast-forward the local target to its remote
    counterpart, folding **both** divergence axes into the target before the
    ancestry check: the *local* axis (concurrent local commits — already in local
    `main`) and the *origin* axis (another clone pushed — visible only after a
    fetch). You merge into local `main` but push to `origin`; a stale target on
    either axis re-creates the "resolve on mainline mid-merge, no gate validated
    it" failure one step downstream at the push.
  - `git merge-base --is-ancestor <target> <branch>`; if false, **abort the
    sequence**, integrate the target into the branch, re-run the full local CI
    gate on the reconciled branch, then re-present the wrap gate — the target may
    have moved again during CI, which is why the check lives immediately before
    the merge.

- Fix the shipped example ref from `origin/main` to local `main` (made current
  via fetch), and apply the reconcile-refresh + bundled-gate shape across
  `wf-patch`, `aiwfx-wrap-milestone`, `aiwfx-wrap-epic` and their referencing
  structural tests under `internal/policies/`.

The reachability check and the reconcile-refresh are complementary
defense-in-depth: the check is the mechanical guarantee, the reconcile-refresh
minimizes how often it has to fire.

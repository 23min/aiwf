---
id: G-081
title: aiwf rename does not pre-flight trunk-collision check
status: open
discovered_in: E-21
---

# G-081 — aiwf rename does not pre-flight trunk-collision check

## What's missing

`aiwf rename <id> <new-slug>` performs the rename and writes its commit unconditionally. The next `aiwf check` invocation may then surface an `ids-unique/trunk-collision` error if the entity was already known to trunk under a different slug. The block lands *after* the mutation, leaving the branch in an error state until either trunk also adopts the new slug or the rename is reverted.

The information needed to refuse the rename pre-mutation is already in scope at verb-invocation time: the trunk reference is configured, the entity's path on trunk is computable, the proposed new path on the branch is known. Running the existing `ids-unique/trunk-collision` rule against the post-rename hypothetical tree state would catch the collision exactly when the operator can act on it — before the commit lands — instead of after.

## Why it matters

The current shape violates two patterns the kernel otherwise honours:

1. **Verbs refuse mutations that would put the tree into an error state.** Other verbs preflight invariants before mutating (e.g., `aiwf promote` validates FSM transitions; `aiwf add` rejects duplicate ids). `aiwf rename` is anomalous here.
2. **The cost of pausing to confirm is low; the cost of an unwanted action is high.** When this fired during E-21 milestone planning on 2026-05-08, the rename produced one commit; the title hand-edit produced another; an audit-only backfill produced a third. Reverting required `git reset --hard` to drop all three. A pre-flight refuse would have replaced that whole sequence with a one-line "refuse + hint" message at rename time.

The friction is bounded — most renames don't cross trunk-collision boundaries — but when it fires, it's a sequence of cleanup commits or a destructive reset, neither of which is good. And the fix is small: one call from the rename verb into the existing checker rule.

## Reproducer

On a ritual branch where the entity exists on trunk under slug `old`:

```bash
git checkout epic/E-NN-old
aiwf rename E-NN new-slug   # succeeds, commits
aiwf check                   # error ids-unique/trunk-collision
```

The collision exists because trunk's tree still has `E-NN-old/`; the branch now has `E-NN-new-slug/`. Same id, two paths from the kernel's per-branch view. The error clears only when trunk adopts the new slug (merge / cherry-pick) or the branch reverts.

## Possible resolution shapes

- **Refuse-by-default, opt-out flag.** `aiwf rename` calls into the trunk-collision rule pre-mutation; refuses with the same finding text the post-check produces. Hint: *"rename trunk-resident entities on trunk first"* or *"use `--allow-trunk-divergence` if the per-branch divergence is intentional."*
- **Warn but allow.** Same pre-flight; emit the finding to stderr but proceed. Less paternalistic; loses the chokepoint guarantee.
- **Lean: refuse-by-default with the opt-out flag.** Matches the kernel's *"errors are findings, not parse failures"* stance — the finding already exists in checker form; the verb just consults it. The opt-out covers the rare case where the operator intentionally wants the per-branch divergence.

## Out of scope

- Generalising *"verbs pre-flight against checker rules"* across every mutating verb. That's a bigger pattern; this gap names the specific case where it bit.
- Detecting per-branch slug divergence as a kernel feature — that's the opt-out flag's job, not this gap's.
- Auto-applying the rename to trunk from the branch, or cherry-picking the rename commit. Workflow choice; not the verb's responsibility.

## References

- E-21 milestone planning conversation, 2026-05-08 — surface event. Reverted via `git reset --hard` since the rename commits were local-only and not pushed.
- `internal/check/...` (or wherever the `ids-unique/trunk-collision` rule lives) — the existing checker the rename verb would call into.
- G-072 (depends_on writer verb — closed via E-22/M-076), G-065 (no `aiwf retitle` verb — closed via E-22/M-077) — adjacent verb-asymmetry gaps that closed during the E-21 planning session that surfaced this one. This gap is shape-different: the rename verb exists; it just doesn't pre-flight against the trunk-collision rule.
- CLAUDE.md *Engineering principles* §"Errors are findings, not parse failures" — informs the resolution lean.

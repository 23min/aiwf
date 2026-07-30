---
id: G-0475
title: FSM history walker skips a commit that both renames and changes status
status: open
discovered_in: E-0075
---
## What's missing

`internal/check/fsm_history_walker.go` skips any commit that both renames an entity
file and changes its status. The walker seeds a path-to-entity map from the tree's
current paths and adds a rename's source path as it processes the touch, so older
commits resolve at the pre-rename path — but for the renaming commit itself the
parent holds the file at the source path while the rule reads the parent at the new
path, which does not exist there. The pair is skipped and no status observation is
recorded.

The code states this and states why it is acceptable: "pure renames don't change
status, so no observation is lost on the typical path."

That premise no longer holds. G-0466 makes a rename carry arbitrary frontmatter,
including `status:`, because a move-shaped verb commits the on-disk bytes of every
path it moves. Measured: a gap hand-edited from a terminal status back to `open` —
an illegal transition under the kind's FSM — then committed by `aiwf retitle` or
`aiwf rename`, passes `aiwf check` with no findings. The same edit committed by hand
raises `fsm-history-consistent/illegal-transition` at error severity.

## Why it matters

`illegal-transition` is a blocking error, and the FSM's one-function-per-kind
legality rule is the first of aiwf's committed guarantees. A route that lands an
illegal transition and passes the check inverts what the rule promises: the more
the operator routes work through verbs, as all the guidance instructs, the less
this particular rule sees.

The walker's assumption was reasonable when written and is now false, which makes
it the kind of defect that survives review — the comment reads as a considered
trade-off rather than an open hole, so a reader checking whether renames are
handled finds an answer and stops.

E-0075's precondition would incidentally mask this by refusing the commit that
produces it, but the rule stays wrong for any other route to a rename-plus-status
commit: a future verb that legitimately writes both, a squash that combines them, or
a history rewrite. The walker should be correct independently of whether one
operator sequence is blocked upstream.

## Options

1. **Resolve the parent at the source path for a rename touch.** The walker already
   knows the source path — it adds it to the map in the same pass. Reading the
   parent blob at the source path rather than the new one makes the pair observable
   with no change to the walk's shape. Smallest fix, and it removes the exception
   rather than documenting it.
2. **Emit a finding for the skipped pair** rather than resolving it, so an
   unobservable rename-plus-status commit is reported instead of silently dropped.
   Honest and cheap, but it converts a correctness gap into an operator-visible
   warning without answering the question the rule exists to answer.
3. **Leave it and rely on E-0075's precondition.** Cheapest, and wrong for the
   reason above: it makes one rule's correctness depend on another epic's guard
   holding at every route, forever.

Option 1 is the lean. The information the walk needs is already in hand at the point
it gives up.

## Scope

Independent of E-0075 and should not wait on it. Surfaced while reviewing E-0075,
by asking which rules the laundering defect actually evades.

Note for whoever picks this up: the provenance audit is scope-gated —
`ResolveUntrailedRange` short-circuits without an upstream ref or `--since`, so a
disposable-repo reproduction needs `--since <sha>` or the check reports a skip
rather than a result.

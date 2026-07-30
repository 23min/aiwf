---
id: ADR-0038
title: Refuse verb writes over HEAD-divergent entity content
status: accepted
---
## Context

A mutating verb can commit entity content no verb computed, by two distinct
mechanisms, and a guard closing one leaves the other open.

Serializing verbs read the entity from the loaded tree — which parsed the working
copy — re-serialize its frontmatter, and read its body straight off disk. Both
ride into the next commit under that verb's trailer. Measured: with a gap's
`priority` hand-edited and a body line rewritten in the working tree, the
following `aiwf set-priority` commit carried the field change *and* the body
rewrite, `aiwf history` showed no `edit-body` event for the body, and
`aiwf check` reported no errors.

Move-shaped verbs launder through a second path. `gatherCommitOps` walks a moved
directory's destination recursively and commits whatever is on disk for every
file inside, so renaming an epic commits every nested milestone's bytes. No verb
names those entities, so a guard keyed on the verb's target cannot see them.

Two blocking rules are defeated, not one. The untrailered-entity-commit audit
skips any commit carrying a non-empty `aiwf-verb:` trailer and derives touched
entities from changed *paths* rather than from frontmatter, so which fields moved
is invisible to it. And the FSM history walker deliberately skips a commit that
both renames and changes status, on the stated reasoning that pure renames do not
change status — a premise this defect falsifies.

`checkStagedConflict` already refuses this exact operator sequence when the edit
is *staged*, naming the overlapping paths and pointing at `git restore --staged`
or `git stash`. The unstaged working-copy edit is the open hole, and the shape of
its answer is constrained by that precedent.

## Decision

### Seam

Two seams, both inside `internal/verb`.

The precondition runs at two seams, sharing one comparison function:

- **`verb.Apply`** — the commit-side guard. Every write route already passes
  through it, `checkStagedConflict` already occupies that position, and
  `gatherCommitOps` computes the authoritative committed path set there. The
  guard's path set *is* the commit's path set.
- **A shared `noOp()` constructor** — the claim-side guard. `Apply` cannot see
  this window: a same-state guard returns from the verb body before any plan
  exists, so `Apply` is never reached. The literal `Result{NoOp: true}`
  constructions collapse into one constructor that performs the check.

Both seams live in `internal/verb` because the layering runs `cmd → cli → verb`;
a guarantee about what a commit contains must belong to the layer that makes the
commit, or the verb package's own tests and any future caller bypass it.

A top-of-verb helper was considered and rejected: it would have to *predict* the
paths the verb will touch, which for a directory-moving verb means a second
implementation of `gatherCommitOps`'s recursive walk — duplicated path logic free
to drift. Placing the guard where the path set is already computed removes the
prediction. A CLI-layer seam was rejected on the layering grounds above.

### Path scope

Committed-path at `Apply`; target-scoped at the NoOp seam.

At `Apply` the comparison covers the full committed path set — `gatherCommitOps`'
`removes` plus `writes` — not the verb-named `op.Path` / `op.NewPath`.
Verb-named-only would miss every nested entity under a directory move, which is
the vector that defeats the FSM walker. Cost is not a counter-argument:
`gatherCommitOps` already holds the working-copy bytes for every path, and
`gitops.BlobReader` batches the HEAD side into one `cat-file` subprocess
regardless of path count.

At the NoOp seam the comparison covers the target entity only. A NoOp writes
nothing and so cannot launder; what it can do is make a false claim, and that
claim is about the target's own state.

### Field scope

Whole-file at `Apply`; frontmatter-scoped at the NoOp seam.

At `Apply` the honest question is what the commit will carry, and it carries
whole files — so the comparison is whole-file. Frontmatter-only would leave the
measured body defect open.

At the NoOp seam the question is whether the no-change claim is trustworthy. That
claim rests on parsed frontmatter, so a dirty body cannot make it wrong, and
whole-file there would refuse to converge a true status claim merely because the
body was mid-edit.

**Surgical commits were considered and deferred.** Rather than refusing, a verb
could commit HEAD's content for everything it does not own and its own computed
content for what it does, leaving the operator's edit uncommitted on disk. This
is feasible — `CommitVerbChange` takes explicit `PathWrite.Content`, so commit
content need not equal disk. It is not adopted here because it requires a
per-verb map of which fields each verb owns, which G-0480 already identifies as
the bulk of that rule's work, and because it would change `gatherCommitOps`'
contract underneath every verb at once. It is the shape a later epic should reach
for; this decision is not the end state.

### Verdict

Refuse, at both seams.

Divergence refuses. It does not warn.

The precedent decides it: `checkStagedConflict` refuses the *staged* form of this
condition. If the unstaged form only warned, staging an edit — the more
deliberate act — would attract the harsher treatment, and one condition would
carry two differently-shaped responses.

Warning also fails on the merits. The commit still lands, still carries content no
verb computed, still shows no `edit-body` event. On the sharp half it is worse
than cosmetic: a laundered `status` on a path-changing route bypasses
`fsm-history-consistent/illegal-transition`, so warning would let a blocking check
be defeated by a printed line. At the NoOp seam refusing is the more conservative
choice for a different reason — the alternative reports "nothing to change" while
silently discarding the mutation the operator asked for, losing work rather than
merely annoying someone.

The refusal reuses `checkStagedConflict`'s error shape and exit code, so the two
conditions are indistinguishable to a caller, and names the overlapping paths the
same way.

### Escape hatch

None.

No `--force`, and no repair verb.

The precedent has no override, and adding one would reproduce the staged/unstaged
asymmetry a notch further. The escape already exists at no cost: bless the edit
(`aiwf edit-body <id>`), set it aside (`git stash`), or drop it (`git restore`).
Measured, only `promote` and `cancel` expose `--force` among the routes in scope,
so a hatch would mean adding the flag to eleven more verbs, each with completion
wiring, while giving a flag that already means *sovereign act* and
*born-complete-body bypass* a third meaning.

A `--force` would also not fix what it appears to fix. The commit would still
carry the wrong verb's name; the misattribution this decision exists to remove
would be re-authorized rather than corrected.

## Consequences

**Both measured misbehaviors of a loaded-only comparison are reached.** The
`Apply` seam eliminates the empty-diff commit — requesting HEAD's value while the
disk diverges. The NoOp seam eliminates the false "already set; nothing to
change" that drops the operator's mutation. Neither seam alone covers both, which
is why there are two.

**The guard fires during the bless workflow, by design.** Editing an entity's
body in the working tree, reading the diff, then committing it with
`aiwf edit-body` is the rhythm the shipped guidance recommends, and under
whole-file scope at `Apply` a structured-state verb on that same entity refuses
inside that window. This is correct — today that window silently commits the body
edit under the other verb's trailer — but it is a behavioral change operators
will meet routinely, not a rare-mistake guard. The blast radius is bounded by path
overlap: an unblessed edit to one entity does not block verbs on another. It does
mean renaming an epic refuses while any nested milestone carries an unblessed
body edit, which is precisely the nested case this decision exists to close.

**Repair of a malformed entity is unaffected.** Measured: when HEAD carries an
unparseable entity, every mutating verb refuses cleanly with `unknown id` /
`not found`, writing nothing — so no verb can corrupt a stub further. The
recovery is the one the `load-error` hint already names: repair the frontmatter by
hand, commit it, then acknowledge the hand-commit via
`aiwf acknowledge illegal <sha> --for-entity <id> --reason "..."`. A hand-commit
never passes through `Apply`, so this guard never fires on the repair itself. What
it does prevent is hand-repairing and then letting the next verb sweep the fix in
under that verb's trailer.

**The NoOp constructor becomes a chokepoint.** Literal `Result{NoOp: true}`
construction is forbidden outside it, enforced by an `internal/policies/` rule in
the shape `atomic_write_chokepoint` and `logging_chokepoint` already use. Without
that rule the claim-side guard is a convention a new verb can forget.

**E-0074 is unblocked.** Its `PromoteACPhase` work waits on the seam decision
because that verb writes frontmatter.

**This does not fix the FSM history walker's blind spot.** The precondition
incidentally masks it, but the walker stays wrong for any other route to a commit
that both renames and changes status. G-0475 remains open on its own terms.

**After-the-fact detection stays necessary.** This is a precondition; it cannot
see commits that predate it, hand-commits, or merge commits — which the
untrailered audit skips by design. G-0480 covers that half and is not closed by
this decision.

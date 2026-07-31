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

Two seams, both inside `internal/verb`, sharing one comparison primitive but
scoped differently — each covers a window the other structurally cannot see:

- **The guard block at the top of `verb.Apply`** — the commit-side guard,
  alongside `checkStagedConflict` and *before* Phase 1. Position is load-bearing:
  by the time `gatherCommitOps` runs, Phase 1 moves and Phase 2 writes have
  already mutated disk, so a comparison there reads the verb's own freshly
  written bytes and answers the wrong question in both directions — it refuses
  every content-writing verb on a clean tree, and skips every move-shaped one,
  because a move-discovered write is keyed by a destination path that has no
  blob at HEAD. The comparison must name the **pre-mutation** working copy.
- **The verb's own prelude, ahead of its same-state comparison** — the claim-side
  guard. `Apply` cannot see this window: a same-state guard returns from the verb
  body before any plan exists, so `Apply` is never reached. It runs *before* the
  comparison rather than inside the NoOp return, so no verb ever classifies
  against disputed bytes.

Both seams live in `internal/verb` because the layering runs `cmd → cli → verb`;
a guarantee about what a commit contains must belong to the layer that makes the
commit, or the verb package's own tests and any future caller bypass it.
`verb.Apply` is the sole *commit* seam — `gitops.CommitVerbChange` has exactly one
production caller, and `PolicyCommitConstructionSingleSeam` keeps the commit
primitives behind that one exported entry point. That, not "the only path that
writes to disk", is the property the seam choice rests on: `initrepo`,
`config.Write` and the renderers all write without committing. The policy's scan
covers `internal/verb` and `internal/gitops`, so a second caller appearing in
another package would not be caught by it — the single-caller property is
current fact plus a partial guard, not a total one.

The pre-mutation path set is derived from `p.Ops` by **prefix matching**, not by
walking the filesystem. A directory `OpMove` covers every nested path by prefix,
which is precisely what `stagedPathConflicts` already does for the staged twin,
for the reason its own comment records. So a pre-mutation guard needs no
prediction and no second implementation of `gatherCommitOps`' recursive walk.

The dirty set comes from `gitops.DirtyPaths` — tracked-modified plus
untracked-not-ignored, two subprocesses, no per-path blob reads. It already
exists and is already called from `internal/verb`.

A CLI-layer seam was rejected on the layering grounds above.

### Path scope

Committed-path at `Apply`; claim-scoped at the NoOp seam.

At `Apply` the comparison covers every path the plan will touch: each
`OpWrite.Path`, each `OpMove.Path` and `NewPath`, **and every path prefixed by
either**. Verb-named-only would miss every nested entity under a directory move,
which is the vector that defeats the FSM walker; the prefix rule covers those by
construction. It also avoids depending on `gatherCommitOps`' return shape: its
`removes` and `writes` slices are index-aligned only across the move-derived
prefix, because the later `OpWrite` loop appends to `writes` alone, so the
old-path/new-path correspondence is not recoverable from them in general.

At the NoOp seam the comparison covers what the claim actually asserts about,
which is usually the target entity's file — but not always. Three claims are
scoped to `aiwf.yaml` rather than an entity (`contract bind`, `contract recipe`,
`rename-area`); two are whole-tree sweeps whose claim is derived from every
entity's status or id width (`archive`, `rewidth`), so their scope is the set the
sweep selected, computed after selection; and one (`acknowledge illegal`) already
compares against git history rather than the working copy, so it needs no guard
at all. Scoping these to "the target entity" would make the guard a silent
pass-through exactly where three of them splice a working-copy `aiwf.yaml`.

### Field scope

Whole-file at both seams.

At `Apply` the honest question is what the commit will carry, and it carries
whole files. Frontmatter-only would leave the measured body defect open.

At the NoOp seam the comparison is whole-file too, because a no-change claim is
not confined to frontmatter. Measured, five sites already compare a second
surface and say so in their own comments: `retitle` compares the stored title
*and* the body H1; the AC-level `retitle` and `rename` compare the
`### AC-N — <title>` heading; `move`'s claim spans the `parent:` field *and* the
file's location; `promote --superseded-by` consults a *second* entity's
reciprocal link. A frontmatter-only comparison would pass an entity whose H1 the
operator has hand-repaired on disk, and the verb would then converge — dropping
the very repair CLAUDE.md says a drifted heading must receive. "That claim rests
on parsed frontmatter" is simply false at those sites.

The cost is that a true status claim can be refused while the body is mid-edit.
That is a deliberate trade, not a free choice: "nothing to change" is not a true
statement about the record when the body is uncommitted — there is something to
change, just not through that verb.

**What happens to a path no verb named is deferred to M-0283.** A path that
enters the commit only because it is nested under an `OpMove` was named by no
verb, so the verb owns none of its bytes, and taking HEAD's blob there rather
than refusing is an attractive refinement: it would prevent the laundering
without stopping the operator, and would close a measured leak where `aiwf
rename` git-adds a never-tracked file left in an epic directory and
`aiwf archive --apply` commits a stray one into the archive.

It is not adopted here because prototyping showed it is not a prose-level
decision. Implemented, the substitution duplicated subtrees under `retitle`,
`move` and `reallocate` — visible only in a fresh clone, not to a local
`aiwf check` — dropped a milestone from a `rewidth --apply` commit while
reporting success, and left a flat-file move's own destination uncovered, since
that path is neither nested under a move nor computed by the verb. It also
interacts with link-rewrite `OpWrite`s so that two sibling milestones under one
sweep behave oppositely depending on whether their body happens to carry a link.

So the decision here is the conservative one — **refuse on divergence for every
path the guard covers** — and the refinement is deferred to M-0283, where a
prototype can settle it against measurement rather than argument.

**Full surgical commits stay deferred too.** Committing HEAD's content for every
field a verb does not own needs a per-verb map of field ownership, which G-0480
identifies as the bulk of that rule's work. This decision is not the end state.

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

The refusal reuses `checkStagedConflict`'s error *shape* and names the
overlapping paths the same way.

Its exit code is an open question left to M-0283, not settled here. Measured, a
staged conflict exits **3**, because `cliutil` maps every `Apply` error to
`ExitInternal`. Exit 3 means "internal" in this CLI's contract, so an
operator-fixable precondition reported that way is a misclassification, and
propagating it to the far more frequent unstaged case would multiply the error
rather than inherit it. Correcting it to `ExitUsage` (2) is the obvious
direction — but the mapping lives in `internal/cli/cliutil`, outside the layer
this decision confines itself to, and it would reclassify the pre-existing staged
case at the same time. That blast radius is measured in M-0283 before the change
is made.

**`edit-body` bless mode is exempt, structurally.** Bless mode's entire
precondition is that the working copy diverges from HEAD — it refuses when they
are equal — so a guard refusing divergence would block the one verb whose job is
to commit it, and would make this decision's own recommended recovery
unreachable. The exemption is a per-op declaration that the write adopts the
working copy, verified at the guard by asserting the op's content equals the
bytes on disk. It is per-op rather than per-plan so it cannot over-exempt a
move's nested subtree, and self-verifying so it cannot smuggle anything but the
disk bytes it declares. It is not a sovereign override and needs no `--reason`:
bless mode already refuses any frontmatter divergence of its own accord, so the
exemption cannot become a laundering route.

### Escape hatch

None.

No `--force`, and no repair verb.

The precedent has no override, and adding one would reproduce the staged/unstaged
asymmetry a notch further. The escape already exists at no cost: bless the edit
(`aiwf edit-body <id>`), set it aside (`git stash`), or drop it (`git restore`).

`--force` is the wrong lever regardless of how many verbs would need it. Among
the routes in scope it already appears on `promote`, `cancel`, `contract bind`
and `contract recipe install`, and elsewhere on `authorize` and `add` — carrying
*sovereign act* in one place and *born-complete-body bypass* in another. A third
meaning would not extend a consistent lever; it would overload an already
ambiguous one, and it would have to be added to the remaining routes with their
completion wiring besides.

A `--force` would also not fix what it appears to fix. The commit would still
carry the wrong verb's name; the misattribution this decision exists to remove
would be re-authorized rather than corrected.

The bless-mode exemption recorded under *Verdict* is not an escape hatch and does
not weaken this. An escape hatch lets an operator commit content the verb did not
compute; the exemption declares that a specific write *is* the working copy, and
is refused unless that is literally true. The distinction is the same one that
separates a repair verb from `--force`: what the commit's trailer ends up
claiming.

## Consequences

**Both measured misbehaviors of a loaded-only comparison are reached.** The
`Apply` seam eliminates the empty-diff commit — requesting HEAD's value while the
disk diverges. The claim-side seam eliminates the false "already set; nothing to
change" that drops the operator's mutation. Neither seam alone covers both, which
is why there are two.

**The guard fires during the bless workflow, by design.** Editing an entity's
body in the working tree, reading the diff, then committing it with
`aiwf edit-body` is the rhythm the shipped guidance recommends, and a verb
touching that same path refuses inside that window. This is correct — today the
window silently commits the body edit under the other verb's trailer — but it is
a behavioural change operators will meet routinely, not a rare-mistake guard.

**Multi-entity routes are blocked by any one dirty participant.** The refusal is
scoped to the paths a plan touches, so an unblessed edit to an unrelated entity
never interferes. But `rename-area` writes every tagged entity, `rewidth --apply`
rewrites tree-wide, `archive` writes every linking entity, and `rename` /
`retitle` / `reallocate` emit link-rewrite writes for referencing entities — so
for those, one dirty participant refuses the whole operation. A stray untracked
file inside a moved directory has the same effect, and its recovery is the
weakest: it cannot be blessed with `aiwf edit-body`, since it is not an entity.
That cost is the direct reason the carry-along refinement is worth settling in
M-0283 rather than dropped.

**A NoOp constructor still becomes a chokepoint, but it is not where the check
runs.** Literal `Result{NoOp: true}` construction is confined to a shared
constructor, enforced by an `internal/policies/` rule in the shape
`atomic_write_chokepoint` and `logging_chokepoint` already use — that is what
stops a new verb forgetting the convention. The precondition itself runs earlier,
in the verb's prelude, because a check inside the constructor guards the wrong
instant: by then the verb has already compared and classified against disputed
bytes, and the constructor only suppresses the resulting message. Measured, that
is not hypothetical — `cancel` classified an entity as terminal whose status at
HEAD was `open`.

Two limits on that chokepoint are worth stating rather than discovering. The
existing NoOp policy scans *exported* entry points only, so the unexported
composite branches (`promoteAC`, `cancelAC`, `renameAC`, `retitleAC`) are
invisible to it and need their own tests. And a same-shaped NoOp construction
exists outside `internal/verb` on a different type, so the rule's scope has to be
decided rather than assumed.

**Existing verb tests will fail, and mostly should.** Fixtures that write an
entity file and then run a verb without committing it first are exercising
exactly the sequence this decision refuses. The count is not estimated here;
M-0283 reports its own measured set rather than inheriting a figure.

**Four questions are deferred to M-0283, with a prototype rather than argument.**
How the compared path set is finally derived; whether carry-along substitution is
adopted and under what corrections; the `ExitUsage` change's blast radius across
`cliutil`; and which tests break. Each was reached for during this decision and
each turned out to need measurement — the defects that surfaced while settling
them were all invisible to reading.

**Repair of a malformed entity is unaffected.** Measured: when HEAD carries an
unparseable entity, every mutating verb refuses cleanly with `unknown id` /
`not found`, writing nothing — so no verb can corrupt a stub further. The
`load-error` finding's hint names the first step — repair the frontmatter by hand
and re-run `aiwf check` — and the `provenance-untrailered-entity-commit` hint
names the rest, since the hand-commit that lands the repair trips it: acknowledge
it with `aiwf acknowledge illegal <sha> --for-entity <id> --reason "..."`. A hand-commit
never passes through `Apply`, so this guard never fires on the repair itself. What
it does prevent is hand-repairing and then letting the next verb sweep the fix in
under that verb's trailer.

**E-0074 is unblocked.** Its `PromoteACPhase` work waits on the seam decision
because that verb writes frontmatter.

**This does not fix the FSM history walker's blind spot.** The precondition
incidentally masks it, but the walker stays wrong for any other route to a commit
that both renames and changes status. G-0475 remains open on its own terms.

**After-the-fact detection stays necessary.** This is a precondition; it cannot
see commits that predate it, hand-commits, or merge commits — which the
untrailered audit skips by design. G-0480 covers that half and is not closed by
this decision.

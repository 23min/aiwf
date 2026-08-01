---
id: M-0284
title: Land the claim-side precondition and record each sweep's call
status: in_progress
parent: E-0075
depends_on:
    - M-0283
tdd: required
acs:
    - id: AC-1
      title: A no-change claim is never made against HEAD-divergent state
      status: met
      tdd_phase: done
    - id: AC-2
      title: Every NoOp site compares at the scope its own claim asserts
      status: met
      tdd_phase: done
    - id: AC-3
      title: Each multi-entity sweep carries a recorded in-or-out call the guard matches
      status: met
      tdd_phase: done
    - id: AC-4
      title: The measured cancel-classifies-terminal defect no longer occurs
      status: open
      tdd_phase: done
    - id: AC-5
      title: The commit-side guard reads the record, not git's dirty report
      status: open
---

## Goal

Put the precondition ahead of every same-state comparison, so no verb classifies
against disputed bytes, and give each multi-entity sweep an explicit recorded
call.

## Context

M-0283 lands the commit-side guard. This milestone covers the window that guard
structurally cannot see: a same-state comparison returns from the verb body
before any plan exists, so `verb.Apply` is never reached.

The failure there is worse than laundering rather than milder. Laundering
commits bytes under a wrong trailer; a false no-change claim reports success at
exit 0 and silently drops the mutation the operator asked for. Measured: with a
gap at `status: open` in HEAD and `wontfix` hand-edited onto disk,
`aiwf cancel` reports the entity already terminal and exits 0.

Placement matters as much as presence. A check inside the NoOp return guards the
wrong instant — by then the verb has already compared and classified against
dirty bytes, and suppressing the message does not undo the classification. The
precondition therefore runs *before* the comparison.

The two seams share one comparison. ADR-0038 records that, but M-0283 had no
second seam to share with, so the commit-side guard got a private input — git's
report of what the operator changed, patched once for ignored files. Building the
second seam here is what turns the shared comparison from a recorded intent into
the code both seams call, and it is the last cheap moment to do so: M-0285 makes
the seam mechanical, and a chokepoint around an input that asks the wrong
question is harder to move than the input itself.

## Approach

One precondition, called from each verb's prelude between resolution and the
same-state comparison, scoped to whatever that verb's claim actually asserts
about. Then the literal `Result{NoOp: true}` constructions collapse into a shared
constructor, which is what M-0285 later makes mechanical.

The 22 sites are not one shape, so the scoping is per-claim rather than uniform:
sixteen are target-scoped to an entity file (an AC-level claim reads its parent
milestone's file); three are scoped to `aiwf.yaml`; two are whole-tree sweeps
with no target, scoped per candidate rather than per verb; and one compares
against git history rather than the working copy and needs no guard at all.

The comparison those preludes call asks one question — does HEAD's blob for a
path equal what is on disk. The two seams differ only in which paths they hand
it: at the claim side the set is small and known, usually the one entity file the
claim asserts about; at the commit side it is the plan's own paths under the
prefix rule M-0283 established. One primitive, two path sets.

## Acceptance criteria

### AC-1 — A no-change claim is never made against HEAD-divergent state

A verb no longer reports "already set; nothing to change" while HEAD disagrees
with the working copy, silently discarding the operator's requested mutation.

Whole-file, not frontmatter-only. Five sites already compare a second surface and
say so in their own comments: `retitle` compares the stored title *and* the body
H1; the AC-level `retitle` and `rename` compare the `### AC-N — <title>` heading;
`move`'s claim spans the `parent:` field *and* the file's location;
`promote --superseded-by` consults a second entity's reciprocal link. A
frontmatter-only comparison would pass an entity whose H1 the operator had
hand-repaired, and the verb would converge — dropping the repair a drifted
heading is supposed to receive.

### AC-2 — Every NoOp site compares at the scope its own claim asserts

Each site's guard is scoped to what that site claims about, and a site needing no
guard carries a recorded reason rather than an omission.

The three scopes are distinguishable and none is a silent pass-through: target
entity file, `aiwf.yaml`, and none — with the sweeps scoped per candidate inside
their own planner rather than by a scope name. Scoping everything
to "the target entity" was measured to make the guard inert at exactly the three
sites that splice a working-copy `aiwf.yaml`.

### AC-3 — Each multi-entity sweep carries a recorded in-or-out call the guard matches

`rename-area`, `rewidth --apply`, `import --on-collision update` and `archive`
each carry an explicit recorded decision about whether the precondition applies,
and the guard's behaviour matches that decision.

What this forbids is a sweep whose treatment is accidental — covered because it
happened to route through a seam, or exempt because nobody looked. `archive` is
the one most likely to be exempt and the one whose exemption most needs writing
down, since it moves files without rewriting their content.

### AC-4 — The measured cancel-classifies-terminal defect no longer occurs

The specific reproduction: a gap at `status: open` in HEAD, hand-edited to
`wontfix` on disk, then `aiwf cancel` — which today reports the entity already at
a terminal status and exits 0, having classified against bytes no verb committed.

It earns its own criterion because it shows the defect is not only a dropped
mutation but a wrong *classification*: the FSM consult itself ran against
disputed state. That is what makes placement before the comparison load-bearing
rather than stylistic.

### AC-5 — The commit-side guard reads the record, not git's dirty report

`verb.Apply`'s guard derives its divergence set by comparing HEAD's blobs against
disk for the paths a plan would carry, rather than by intersecting those paths
with git's report of what the operator changed.

A path git declines to report is still a path the commit carries, and the dirty
set has no way to say so. Two instances were measured separately, each initially
read as its own limit: an ignored file beneath a moved directory, which M-0283
closed by adding a third query, and an `assume-unchanged` milestone under a
parent-epic rename, which stayed reachable after that fix shipped. The blob
comparison closes both as one property — `.gitignore` never enters it, and the
index bits live on a side neither half of it reads.

This is a re-point, not a second implementation: the primitive is the one AC-1
introduces, and the ignored-path query retires with the dirty set it was
patching.

## Constraints

- The precondition runs before the same-state comparison, not inside the NoOp
  return. A guard that only suppresses the message leaves the classification
  already made.
- No site is scoped by convenience. A pass-through needs a recorded reason, and
  "the target entity" is not a default.
- The shared constructor is introduced here; the chokepoint that forbids the
  literal form elsewhere is M-0285's. Landing the constructor without that rule
  leaves a convention a new verb can forget — which is why the two milestones are
  ordered this way rather than merged.
- Whatever lands must not be read as fixing the FSM walker's rename-plus-status
  blind spot. G-0475 stays open on its own terms.
- ADR-0038's *Seam* section names `gitops.DirtyPaths` as the guard's input, "two
  subprocesses, no per-path blob reads". AC-5 supersedes that, so the ADR takes an
  in-place amendment naming the superseded statement — the shape its escape-hatch
  subsection already uses, since the `accepted` text is in other clones.

## Design notes

- `acknowledge illegal` needs no guard: `ackAlreadyRecorded` walks git history,
  so its baseline is already the record rather than the working copy. Recording
  that as a reasoned exemption is part of AC-2, not an omission from it.
- `archive` and `rewidth` have no target entity by construction — their claims are
  derived from the whole tree, and a selected set is empty exactly when their NoOp
  fires, so a selection-scoped guard has nothing to look at. `archive` compares per
  candidate move instead, declining the ones whose verdict rests on a mid-edit
  file; `rewidth` needs none, because its body scan is independent of the rename
  set and re-emits a masked rewrite on the next run.
- The existing NoOp policy scans *exported* entry points only, so the unexported
  composite branches (`promoteAC`, `cancelAC`, `renameAC`, `retitleAC`) are
  invisible to it and need their own tests here rather than relying on M-0285.
- The blob comparison is index-independent by construction: HEAD's side is read
  from the object database and the working copy's from disk, so the bits that hid
  a path from the dirty set — `assume-unchanged`, `skip-worktree` — are on neither
  side of it, and `.gitignore` governs neither. That is why AC-5 closes G-0492 as
  a class rather than closing its third instance.
- Whether AC-5 also closes G-0487 turns on what the guard does with a path present
  in HEAD and absent from disk, which is the sparse-checkout shape. That call is
  AC-5's to make and measure; the promotion is a wrap-time question, not an
  assumption to carry in.

## Out of scope

- The commit-side guard's seam, verdict, and the nested-path vector — M-0283.
  Only its input is in scope here, as AC-5.
- The `internal/policies/` chokepoint forbidding literal `Result{NoOp: true}`
  construction — M-0285.
- After-the-fact detection of laundering already in history (G-0480).

## Dependencies

- M-0283 — the shared comparison and the mechanics its spike settles.
- M-0282 — ADR-0038, which decides that the precondition precedes the comparison.

## Work log

### AC-1 — A no-change claim is never made against HEAD-divergent state

`guardClaim` refuses in each verb's prelude, over `gitops.DivergentPaths`'
HEAD-blob-against-disk comparison, wired at the fifteen target-scoped sites
including the four unexported composite branches. Twenty new test functions
across `internal/verb` and `internal/gitops`; mutation probe 7/7 caught, no
survivors · commit `5704f2493`

### AC-2 — Every NoOp site compares at the scope its own claim asserts

The three `aiwf.yaml`-scoped claims — `contract bind`, `recipe install`,
`rename-area` — now guard on that file rather than on a target entity they do
not have. `PolicyNoOpClaimScope` derives every converging function from the
source and requires each to carry one of the recorded scopes, with a reason,
failing closed when the scan finds nothing. Twelve new test functions;
mutation probe 8/8 caught, no survivors · commit `88f949150`

### AC-3 — Each multi-entity sweep carries a recorded in-or-out call the guard matches

`rename-area` is in, scoped to `aiwf.yaml` (AC-2). `archive` is in per candidate
move: each is declined when a file its verdict rests on is mid-edit — the
entity's own file, anything a directory move carries beneath it, or an entity
whose committed body links into the move — and each declined move is reported
through the channel skipped epics already use. `rewidth` is out, on measured
self-healing. `import`'s zero-plan NoOp is not a same-state claim and sits
outside `internal/verb`, so it is recorded in a companion list the scan cannot
derive. Fifteen new test functions; mutation probe 8/8 caught, no survivors ·
commit `6f476ec73`

## Decisions made during implementation

### The sweeps are scoped per candidate, not per verb

`archive` and `rewidth` have no target entity, and the set a sweep selects is
empty exactly when its NoOp fires — so a guard scoped to that selection guards
nothing at the only moment the claim is made. Refusing the whole verb instead
would block a sweep, `--dry-run` included, on any unrelated draft.

`archive` therefore compares per candidate move. Measured, the alternative is
not merely inert: with a referrer part-way through a reword, the move lands
without its link rewrite and HEAD carries a link to a path that does not exist,
which no later run repairs because an archived target leaves the scan for good.
The referrer comparison reads HEAD's body rather than the working copy's, since
a draft that dropped the link is precisely what a working-copy scan cannot see.

`rewidth` needs no comparison: `planRewidthRewrites` rescans every active
markdown independently of the rename set, so a masked rewrite is re-emitted by
the next run — the cost is a rewrite deferred, not lost.

The scope name `sweep-selection` went with that reasoning. A closed-set value
with no member is a decision nothing in the code implements.

### The refusal is emitted in the prelude, not at the converge point

`guardClaim` refuses as soon as it observes divergence, rather than recording
the observation and acting on it only where a same-state claim is made. The
narrower placement is unsafe at `promote`, the one guarded site where anything
consequential runs between those two points.

`--superseded-by` reads the superseding ADR to decide whether the reciprocal
back-link is already stored. With that link hand-edited onto disk the read
concludes it is, no op is emitted for that file, and the plan therefore never
names it — so the commit-side guard, keyed on the plan's paths, cannot see it
either. Measured: a one-sided supersession committed at exit 0 with
`aiwf check` silent, because the working copy that caused it also satisfies
`adr-supersession-mutual`.

The resolver re-point refusal fires before any plan exists, so no plan-aware
guard can reach it. Measured: with a gap at `open` and no resolver in HEAD,
and `addressed` plus a resolver hand-edited onto disk,
`aiwf promote <gap> addressed --by <adr>` reports the gap already addressed
and already carrying a resolver, and recommends `--force` — a sovereign act
solicited on a false premise.

Both are pinned by regression tests. What holds is ADR-0038's own statement:
the guard runs before the comparison.

## References

- E-0075 — the parent epic
- ADR-0038 — the claim-side seam and its scope
- G-0466 — a verb commits frontmatter it does not own
- G-0475 — the FSM walker blind spot this must not be read as fixing
- G-0480 — after-the-fact detection; the backstop this does not replace
- G-0492 — the guard reads git's dirty report, not the bytes a plan would carry
- G-0487 — git's hidden-state bits make a dirty path invisible to a dirty-set guard
- `internal/policies/verb_result_noop_invariant.go` — the exported-only scan
- CLAUDE.md — "Same-state convergence — resolve, then converge"

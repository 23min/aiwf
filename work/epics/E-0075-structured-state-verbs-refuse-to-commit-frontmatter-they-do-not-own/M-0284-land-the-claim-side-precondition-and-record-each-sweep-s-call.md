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
      status: met
      tdd_phase: done
    - id: AC-5
      title: The commit-side guard reads the record, not git's dirty report
      status: met
      tdd_phase: done
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
about. The literal `Result{NoOp: true}` construction stays as it is at every
site; see the Constraints below.

The converging sites are not one shape, so the scoping is per-claim rather than
uniform. Twenty functions in `internal/verb` converge, each carrying a recorded
scope: 13 target-scoped to an entity file (an AC-level claim reads its parent
milestone's file), 3 scoped to `aiwf.yaml`, 1 sweep scoped per candidate rather
than per verb, and 3 needing no comparison for a recorded reason. They hold 22
literal sites between them, since `set-area` and `set-priority` each converge on
two branches. One further site sits outside `internal/verb` altogether, where the
scan cannot derive it, so a companion list records it.

The comparison those preludes call asks one question — would the working copy at
this path store as what the record holds. The two seams differ only in which
paths they hand it: at the claim side the set is small and known, usually the one
entity file the claim asserts about; at the commit side it is every path the plan
would carry, enumerated from both the working tree and the record. One primitive,
two path sets.

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
- No shared NoOp constructor is introduced, here or later. The literal form is
  the house style at all 23 sites; see the decision below.
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
through the channel skipped epics already use.

The link half of that decline is narrower than this entry first claimed. The
referrer *bodies* are compared against HEAD, but the referrer *candidate list* is
drawn from the loaded tree, so a referrer present at HEAD and absent from that
list — deleted on disk, hand-renamed, or carrying momentarily unparseable
frontmatter — is never consulted, and its move lands without the link rewrite.
Measured after this milestone shipped; the dangling reference is permanent and
`aiwf check` reports no error on it. G-0499 carries the measurement and its three
neighbours, M-0286 closes them. What holds here is the decline machinery, its
reporting channel, and the record-derived comparison feeding it. `rewidth` is out, on measured
self-healing. `import`'s zero-plan NoOp is not a same-state claim and sits
outside `internal/verb`, so it is recorded in a companion list the scan cannot
derive. Fifteen new test functions; mutation probe 8/8 caught, no survivors ·
commit `6f476ec73`

### AC-4 — The measured cancel-classifies-terminal defect no longer occurs

A regression pin; AC-1's prelude guard already refuses the reproduction, so no
implementation changed and the phase ladder's red is vacuous by construction.
Two arms, each with its committed-state control: `wontfix` hand-edited onto a gap
at `open` in HEAD, which absent the guard converges at exit 0 before any plan
exists for the commit-side guard to see; and an unrecognized status, which falls
past the terminal check into the FSM consult, whose refusal names bytes no verb
wrote. The discriminating evidence is the mutation probe rather than a
red-to-green transition: 3/3 caught, and the converge-point guard this milestone
rejected passes the first arm while failing the second. Four new test functions ·
commit `61912a311`

### AC-5 — The commit-side guard reads the record, not git's dirty report

`verb.Apply` derives its divergence set from `planCarriedPaths` +
`gitops.DivergentPaths` — the record's object id against the working copy's, for
the paths the plan carries, at both ends of a move. Both vectors were measured on a real binary
first: an `assume-unchanged` milestone rode into a parent epic's rename commit
under that rename's trailer, and a `skip-worktree` path absent from disk split
the epic directory, stranding one milestone at the old path while its siblings
moved. `gitops.IgnoredPathsUnder` retires with the dirty set it patched.
`UncommittedConflictError` carries a third bucket for a path the record holds
and the working tree lacks, whose remedy is `git restore` rather than a
destructive one. Ten new test functions · commit `9c6ebe566`

Corrected across two review rounds. The first measured four defects here — a
space or a newline in a carried path broke or silently desynchronised the
comparison, a path existing nowhere produced a phantom verdict, and a tracked
symlink blocked every directory move on a clean tree — and one in AC-1's
carve-out. The comparison reads unfiltered object ids over the argument vector,
paths are inspected before they are read, and the entity-scoped carve-out is
scoped apart from the config one · commit `d65cffcb0`

A fifth defect in that round, `core.autocrlf` locking out every verb, is
diagnosed rather than corrected: the lockout is accurate, because the verb commit
path stores bytes verbatim and would rewrite every line ending. Its cause is that
commit path, and G-0498 carries the decision.

The second round measured a regression the first correction introduced: comparing
a symlink by its target string is right about git and wrong about this commit
path, which dereferences the link and stores the result at mode 100644. A clean
tree therefore lost the refusal that had been holding — measured, an epic rename
replaced a `120000` link with the linked file's body, and a link pointing outside
the repo carried that file's contents into git history. Carried links are now
refused outright · commit `d61565e6e`

Twenty-one test functions across the two rounds; mutation probes 8 run, 7 caught,
the survivor investigated and pinned, plus 6 against the symlink guard

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

### A move's carried set is enumerated from the record as well as from disk

`gatherCommitOps` builds a move's writes by walking disk, so a path the record
carries and the working tree lacks is never re-written at the destination — and
never removed from the source either. The commit lands a split directory: the
epic and its other children at the new path, that one stranded at the old.
Measured, `aiwf check` reports zero errors on the result, so the guard is the
only place it is caught. That is what decided the milestone's open question in
favour of comparing both sides rather than the disk walk alone.

The destination is enumerated for the same reason the source is: `os.Rename`
onto an existing file replaces it, so a plan landing on an occupied path would
destroy content no verb named.

### The comparison is batched over the argument vector, not over stdin

A per-path query costs two subprocesses per carried file, which is a plan's count
rather than a constant — measured at 319ms for 51 paths, and a directory move
carries every file beneath it. So the comparison is batched.

*How* it batches is the load-bearing part. A batch fed over stdin is delimited,
and a path is not a token: measured, a space truncated a whitespace-split
response into a malformed-header failure, and a newline split one request into
two, shifting every later answer onto the wrong path with a nil error. Batching
over the argument vector has no delimiter to break, so any byte a filesystem
permits in a name survives. Chunking keeps the command line bounded while the
subprocess count stays proportional to the set rather than to each path.

### The archive sweep's per-candidate decision moved to the same input

AC-3's `dirtyPathsUnderMoves` asked git what the operator changed, so a candidate
whose file is ignored, `assume-unchanged`, or omitted by a sparse checkout read
as clean and was swept. The commit-side guard refuses that, so nothing is
laundered — but a partial sweep degrades into a whole-verb refusal that names no
candidate, which is the behaviour AC-3 exists to avoid. Both seams now route
through one enumeration (`addCarriedUnder`), so they cannot drift on what a move
is considered to carry.

### The comparison is unfiltered object ids, matching what the commit path stores

The primitive compares object ids computed over raw bytes, not the bytes
themselves and not filtered ids. Each half of that is forced by a measurement.

Raw bytes alone are not comparable across a path git materialised: under
`core.autocrlf` — the Git-for-Windows installer default — a checkout smudges the
working copy away from its blob, and a byte comparison then reports every path in
the repo as divergent. Measured, every mutating verb refuses on a tree `git
status` calls clean, with a false message and three remedies that do nothing.

Applying git's clean filter to the working-copy side is equally wrong, and less
obviously so. The verb commit path stores content verbatim, so a filtered
comparison measures against a convention these commits do not follow: measured, a
path whose bytes HEAD already held was reported divergent because the disk side
had been normalised and the record had not. `--no-filters` reproduces HEAD's
recorded id exactly.

What remains is a real incompatibility rather than a comparison bug, and it is
the commit path's: a repo whose blobs came from `git add` and whose working copy
was smudged on checkout genuinely would be rewritten by any verb. The guard is
the surface that makes that visible; G-0498 carries the decision about which
convention aiwf adopts.

Paths ride on git's argument vector for the same reason the ids are compared at
all. A line-oriented batch protocol cannot carry a path containing a space
(measured: a malformed-header failure at exit 3 where a refusal was due) or a
newline (measured: one request became two, every later answer landed on the wrong
path, and untouched committed files were reported as absent from HEAD — with a
nil error, and into the one divergence kind the claim guard skips).

### The untracked exemption is the config scope's, not the entity scope's

A path absent from HEAD carries no record to contradict, so both seams exempt it.
Applied to a target entity that reasoning is wrong, because an entity's file can
move without passing a verb: after a plain `mv` the record still sits at the
original path, so the exemption fires exactly where the record most needs
consulting.

Measured, both halves lost work. `aiwf cancel` over a moved file reported the
entity already terminal at exit 0 while HEAD said `open` — this milestone's own
AC-4 reproduction, reachable through a path its pin did not cover. And a write
landed beside HEAD's untouched copy, putting one id at two paths in the record
while a local `aiwf check` stayed clean, so the pre-push hook passed it.

`guardClaimConfig` keeps the exemption for `aiwf.yaml`, which needs it: `aiwf
init` leaves that file uncommitted by design, and a config-scoped claim refusing
it would make every verb that rewrites it unreachable until someone committed it.
aiwf.yaml is not an entity and cannot move out from under an id.

### The shared NoOp constructor is dropped, not deferred

The plan carried a shared constructor for `Result{NoOp: true}` plus a chokepoint
forbidding the literal, on the argument that together they stop a new verb
forgetting the convention. Neither is built, and the direction is closed rather
than handed on.

The convention is already mechanical from a different direction.
`PolicyNoOpClaimScope` fails a converging verb that carries no recorded claim
scope, so a verb that forgets is caught today — which was the whole benefit the
constructor was to provide.

Worse, the two work against each other. That policy derives its set of converging
functions by matching the literal `Result{NoOp: true}` composite in the source.
A constructor removes exactly the syntax the check keys on, so adopting one
breaks the guarantee currently in place unless the re-key preserves per-function
attribution — paying real cost to weaken a check that already works.

The literal form is therefore the house style at all 23 sites. What the ADR
decides about this seam is placement: the precondition runs in the prelude, ahead
of the comparison, and never at the point where the NoOp is built.

### A carried symlink is refused, not compared

The comparison primitive answers whether a link's target still matches the
record, and does so exactly. That answer is not the one the guard needs, because
this commit path cannot store a link at all: `gatherCommitOps` reads every
carried path with `os.ReadFile`, which follows links, and `CommitTree` writes
each blob at mode `100644`. A link whose target is untouched — clean by every
measure git offers — is still replaced in the record by a copy of what it points
at.

So the refusal is unconditional rather than keyed on divergence. Measured, the
alternative is not a missed refusal but an active rewrite: an epic rename turned
a `120000` entry into an `100644` blob holding the linked milestone's body under
`aiwf-verb: rename`, and a link pointing outside the repository carried that
file's contents into git history. Both left the working tree reporting a type
change with no remedy that clears it.

Recording links faithfully is the fix this defers to, and it belongs with the
commit path rather than with the guard. Until then a refusal is the honest
answer, since the guard's only other options are to permit a silent rewrite or to
claim a prediction it does not make.

## Validation

`make ci` green — race suite, both coverage gates, the profile-driven policies,
and the 29-step self-check. `make lint` 0 issues. `aiwf check` 0 errors.

The diff-scoped coverage gate is clean against HEAD and against the merge-base
with `main`, so the epic to mainline merge is not blocked.

Every fixed defect was re-measured end-to-end on a real binary against a
disposable repo, not asserted from the test suite. The `--no-filters` correction
was found by that final re-measurement after the isolated check had already
passed.

## Reviewer notes

An independent five-lens review ran before closure — four code-quality slices
(the `Apply` seam, the `gitops` primitive, the claim-guard wiring, the archive
sweep plus the claim-scope ledger) and one design pass over the two-seam
architecture. Every reviewer was briefed adversarially and instructed to verify
by measuring; every finding below was then reproduced independently before being
acted on.

Nine defects were confirmed by measurement. Eight are fixed here; the ninth is
the archive referrer defect, split to M-0286 with G-0499, because it needs a
different fix in a different subsystem and bundling it is what made this
milestone hard to review.

Two things the review established that are worth keeping:

- The architecture is sound. The two-seam factoring was independently judged
  forced rather than chosen — the claim seam reaches vectors no plan-keyed guard
  can see, and the commit seam reaches nested paths no verb named. The carried
  set was instrumented against `gatherCommitOps` across three packages with zero
  misses in `internal/verb`.
- The defects clustered in the new comparison surface's handling of inputs the
  fixtures never produced: content filters, spaces and newlines in paths,
  symlinks, and moved paths. The mutation probes run before review returned 8/8
  because every fixture committed its files first, used ordinary names, and never
  moved a path — three whole input classes were invisible to the entire suite.
  Probe design, not probe count, was the gap.

AC-1, AC-4 and AC-5 stayed `met` rather than being re-opened. Their claims hold at
the corrective commit — more robustly than before — and the AC FSM offers `met`
only `deferred` and `cancelled`, both meaning "off the contract", which would
assert work remains where none does. The correction lives here and in
`aiwf history`; `--force` is sovereign and was not reached for.

AC-3's phase ladder ran once, before the per-candidate rework, and was not
re-opened afterwards. AC-4's `red` was vacuous by construction, since AC-1 had
already shipped the behaviour it pins.

Two reported findings were judged and deliberately not acted on. The `none` claim
scope carries two obligations — "nothing could contradict this" and "something
could, but recoverably" — and both reviewers independently recommended against
splitting it: the only contingent entry is `Rewidth`, and its contingency is
already pinned by a named test, which a fourth enum value could not do. And
`archive --dry-run` measured slower on a large tree; the cost sits in a function
M-0286 rewrites, so tuning it here would be work discarded there.

Carried forward, unfixed and tracked: G-0498 (commit-path filter blindness),
G-0499 and M-0286 (the archive referrer class), plus a pre-existing `no-silent-
fallback` policy defect that keys on a switch's field name rather than its type,
leaving `switch op.Type` unenforced repo-wide.

## Deferrals

- G-0498 — verb commits bypass git's content filters. Discovered here, decided
  elsewhere: it is about which convention aiwf stores blobs under, and the
  comparison follows that decision rather than leading it.
- G-0499 / M-0286 — the archive sweep's referrer and destination gaps.
- E-0075 owes a `CHANGELOG.md` entry at epic wrap. Every structured-state verb
  now refuses over a HEAD-divergent working copy with a new error class and new
  remedy text; neither M-0283 nor M-0284 has an `## [Unreleased]` subsection.

## References

- E-0075 — the parent epic
- ADR-0038 — the claim-side seam and its scope
- G-0466 — a verb commits frontmatter it does not own
- G-0475 — the FSM walker blind spot this must not be read as fixing
- G-0480 — after-the-fact detection; the backstop this does not replace
- G-0492 — the guard reads git's dirty report, not the bytes a plan would carry
- G-0487 — git's hidden-state bits make a dirty path invisible to a dirty-set guard
- G-0498 — verb commits bypass git's content filters, discovered here
- G-0499 — the archive referrer class this milestone recorded as closed
- M-0286 — the milestone that closes it
- `internal/policies/verb_result_noop_invariant.go` — the exported-only scan
- CLAUDE.md — "Same-state convergence — resolve, then converge"

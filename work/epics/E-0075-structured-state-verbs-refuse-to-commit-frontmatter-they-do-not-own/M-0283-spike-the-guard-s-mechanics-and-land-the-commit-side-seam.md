---
id: M-0283
title: Spike the guard's mechanics and land the commit-side seam
status: in_progress
parent: E-0075
depends_on:
    - M-0282
tdd: required
acs:
    - id: AC-1
      title: Unstaged HEAD-divergent content is never committed silently
      status: met
      tdd_phase: done
    - id: AC-2
      title: The measured priority-through-retitle laundering no longer succeeds
      status: met
      tdd_phase: done
    - id: AC-3
      title: A verb over a dirty disk never commits a tree identical to its parent
      status: met
      tdd_phase: done
    - id: AC-4
      title: Every verb entry point has a stated guard decision or a reasoned exemption
      status: met
      tdd_phase: done
    - id: AC-5
      title: The measured nested laundering through a parent rename no longer succeeds
      status: met
      tdd_phase: done
    - id: AC-6
      title: edit-body bless mode still commits a working-copy edit
      status: met
      tdd_phase: done
---

## Goal

Settle the guard's remaining mechanics by building a throwaway prototype, then
land the commit-side guard at `verb.Apply` on what the prototype proved.

## Context

ADR-0038 settles the decisions answerable by reading the code: two seams both
inside `internal/verb`, refuse rather than warn, whole-file scope at both, no
`--force` and no repair verb, and a structural exemption for `edit-body` bless
mode. It deliberately defers the mechanics, because every defect found while
drafting it was an implementation discovery rather than a reading one — that a
comparison at `gatherCommitOps` reads the verb's own freshly-written bytes, that
a carry-along substitution duplicates subtrees and drops a milestone from a
`rewidth` commit, that a flat-file move's destination falls outside a
nested-only taxonomy. None of those is visible in prose.

So this milestone inverts the usual order: prototype first, decide from
measurement, then implement under TDD. The prototype is throwaway and never
committed; only the matrix, the answers, and the final implementation land.

## Approach

**The prototype leads.** Every defect found while settling this design was
visible only by running code, and none was found by classifying it. The
classification below exists to drive the prototype — a checklist of what to try —
not to discover anything on its own.

Build the guard in a scratch copy at the top of `verb.Apply`, before Phase 1,
where disk still holds the operator's state. Two inputs decide each path.

**One bit, because that is all the instrument yields.** The guard's dirty set
comes from `gitops.DirtyPaths` — `git diff --name-only HEAD` unioned with
`git ls-files --others --exclude-standard` — and staged paths are already refused
by `checkStagedConflict` earlier in the same function. Between them those two
queries distinguish three classes of path, not the eight a HEAD/index/disk
tri-state would suggest, so the decidable question per path is simply: *is it
dirty?* Anything finer is a distinction the tooling erases.

**Five roles, because that is what actually discriminates.** What separates the
measured defects is not a path's git state but its role in the plan:

| role | derived from | the decision |
|---|---|---|
| named write | `OpWrite.Path` | refuse if dirty, unless the op declares it adopts the working copy |
| move source | `OpMove.Path` | refuse if dirty |
| move destination | `OpMove.NewPath` | absent by construction, except a flat-file move onto an existing path |
| nested under a move | prefix of `OpMove.Path` / `NewPath` | the open question — no verb named it |
| not in the plan | — | nothing; `Apply` never touches it |

Roles crossed with one bit is five to eight real decisions, all reachable. The
two-axis alternative was measured at roughly one part in seven load-bearing, and
it could not express the nested-milestone defect as a decision distinct from the
hand-edited-body one, because both land in the same cell with different answers.

**Then the second layer, which is where the unsettled questions live.** Per-path
verdicts have to compose into one plan-level outcome — refuse if any path is
dirty, or only for paths whose content the verb computed. That is a rule, not a
table, and carry-along substitution is one candidate answer to it.

Drive the prototype across the role/bit grid for each verb class, record what
happens, decide from the results, then implement test-first and discard it.

**Three limits are known before the prototype starts, and the guard must state
them rather than imply completeness.**

- **A dirty path can be invisible.** `assume-unchanged`, `skip-worktree` and
  sparse checkout make both of `DirtyPaths`' queries answer "clean" for a path
  whose disk content differs. Measured, a nested milestone's `tdd:` laundered
  through an epic rename that way. G-0487.
- **A clean path can already be corrupt, and the guard then makes it worse.** A
  directory move flattens symlinks and forces mode `100644`, after which those
  paths read as modified forever — so this guard would refuse every later verb on
  that directory with no aiwf-side recovery, while a bare `chmod +x` would be
  refused although it can launder nothing. G-0486.
- **Some unowned writes involve no divergence at all.** The loader normalizes as
  it reads, and the next write-verb commits that normalization under its own
  trailer with disk and HEAD in agreement throughout. G-0488.

Those three also correct the framing this milestone inherited. The root cause is
not that verbs compare against a projection of disk; it is that a verb rewrites a
**whole file re-serialized from a lossy in-memory model** rather than editing the
fields it owns. HEAD-divergence is one way that goes wrong. The guard addresses
that one, and the surgical-commit approach ADR-0038 defers is the shape that
would address the rest.

## Acceptance criteria

### AC-1 — Unstaged HEAD-divergent content is never committed silently

A verb run against a path whose working-copy content differs from HEAD does not
commit that difference without saying so. Whole-file, not frontmatter-only: the
measured defect covers an unblessed body edit as well as a hand-edited field.

The reproduction to pin: a gap's body rewritten in the working tree, then
`aiwf set-priority` run, after which the commit carries both the priority change
and the body rewrite while `aiwf history` shows no `edit-body` event.

### AC-2 — The measured priority-through-retitle laundering no longer succeeds

`aiwf retitle` no longer carries `-priority: high / +priority: low` into a commit
trailered `aiwf-verb: retitle`.

`retitle` earns its own criterion because it sits in both mechanisms at once — it
builds an `OpMove` and an `OpWrite`, so it is a serializing route and a
move-shaped one. A guard that covers it covers the overlap.

### AC-3 — A verb over a dirty disk never commits a tree identical to its parent

The empty-diff direction of a loaded-only comparison: asking for HEAD's value
while the working copy diverges commits a tree byte-identical to its parent — the
class M-0281 existed to eliminate.

The other direction — a false "already set; nothing to change" that drops the
operator's mutation — is claim-side and belongs to M-0284. Both are real; they
are separated because they are caught at different seams.

### AC-4 — Every verb entry point has a stated guard decision or a reasoned exemption

The mechanical half is coverage of the axis that drifts. Every exported
`(*Result, error)` entry point under `internal/verb`, plus the unexported
composite branches an AST scan of exported functions cannot see, either has a
recorded decision about how the guard treats it or a reviewed allowlist entry
giving its specific reason. The assertion is derived from the source rather than
from a hand-authored list, in the shape `verb_result_noop_invariant.go` already
uses — so adding a verb without deciding its treatment fails, which is the only
drift a policy can actually catch.

State its reach honestly: it proves every route is *named*. It cannot prove an
answer is correct, or that it was measured rather than reasoned. Those are read
at the wrap review, which is where this project puts judgment a check cannot
carry. Claiming more would repeat the mistake M-0282 recorded — a chokepoint that
reads as enforcing and does not.

The non-mechanical half is the milestone's actual output: each question ADR-0038
defers is answered from the prototype rather than by argument — how the compared
path set is derived, how per-path verdicts compose into a plan-level outcome,
whether carry-along substitution is adopted and under what corrections, what the
`ExitUsage` change costs across `cliutil`, and which existing tests break. An
answer with no measurement behind it is not an answer, and the wrap review is
where that is judged.

Two answers the prototype is expected to produce, flagged so they are not
mistaken for new scope. The sweeps' claim scope: `archive` and `rewidth` return
their NoOp precisely when the selected set is *empty*, so scoping their guard to
that selection would give it nothing to look at. And `import`: its NoOp is
constructed outside `internal/verb` on a different type, so no site inventory
contains it, yet it is in E-0075's scope and needs a decision like any other
route.

Where an answer contradicts an ADR-0038 decision rather than refining it, the ADR
is superseded rather than quietly rewritten.

### AC-5 — The measured nested laundering through a parent rename no longer succeeds

`tdd:` hand-edited on a milestone, then `aiwf rename` on the parent epic, no
longer produces a commit that attributes the change to the epic while
`aiwf history` on the milestone shows no event for it.

The nested vector is commit-side, which is why it sits here rather than with the
claim-side work. It is also the vector that defeats a blocking check, since the
FSM history walker skips a commit that both renames and changes status.

Coverage follows the code rather than any prose route list: the vector belongs to
every `OpMove` whose source is a directory, and to a file move's own destination.
A taxonomy scoped to "nested under a move" was measured to leave flat-file
renames laundering freely.

### AC-6 — edit-body bless mode still commits a working-copy edit

Bless mode's precondition is that the working copy diverges from HEAD, so a guard
refusing divergence would block the one verb whose job is to commit it — and
would make the recovery ADR-0038 recommends unreachable.

The exemption is verified, not merely declared, and it covers both of the verb's
modes — explicit mode needs it for the same reason bless mode does, since the
write-then-route and declarative-revert flows both hand the verb a working copy
that diverges from HEAD.

Two conditions carry the verification. The working copy's own frontmatter must
still equal HEAD's, so no hand-edited field rides in. And the write's content must
carry nothing beyond that working copy: equal to it outright, as bless mode's
verbatim bytes are, or equal to it re-serialized through the loaded entity model,
as explicit mode's are. Both comparisons are field-based, because re-serializing
canonicalizes field order without changing what is declared.

An adopting write may still change the body freely — that is the exemption's
entire purpose, and the declarative revert depends on it.

## Decisions made during implementation

**The four questions ADR-0038 deferred, answered from the prototype.**

- *How the compared path set is derived.* From the plan's ops, using the match
  rule `stagedPathConflicts` already applied to its staged twin: an `OpWrite`
  matches its path exactly, an `OpMove` matches its source, its destination, and
  anything nested under either. Both guards now resolve a path through one shared
  function, so they cannot drift on which paths a plan is considered to touch.
- *How per-path verdicts compose.* Refuse if any compared path is dirty. Driven
  across the role grid, no scenario needed a finer rule.
- *Whether carry-along substitution is adopted.* No. The rule above closes every
  measured vector without it, and its costs were already measured: duplicated
  subtrees under `retitle` / `move` / `reallocate`, a milestone dropped from a
  `rewidth --apply` commit, flat-file destinations uncovered.
- *What the `ExitUsage` change costs.* One typed error and one `errors.As` arm in
  `internal/cli/cliutil`. The refusal exits 2; the staged twin still exits 3,
  which is an asymmetry this milestone leaves standing rather than widening its
  scope to the pre-existing guard.

**One carve-out the measurements forced.** An untracked path the plan names as an
`OpWrite` destination is not compared. It has no committed version to contradict,
so the write creates the record rather than laundering one. Without it every verb
writing `aiwf.yaml` is refused in a freshly-initialised repo, where `aiwf init`
itself leaves that file uncommitted by design. Untracked paths nested under a move
still refuse, which is what keeps the untracked-scratch-file vector closed.

**An unasked-for discovery.** `git diff HEAD` exits 128 on an unborn HEAD, and a
verb's own commit is routinely a repo's first. Unguarded, that accounted for 735
of the prototype's 842 failing tests.

**A fourth carried-file vector, found at review.** Both halves of the dirty set
omit ignored paths by construction, while a directory move carries whatever sits
beneath it — so a `.gitignore`d file inside an epic directory was committed by
`aiwf rename` with git reporting a clean tree throughout, and became tracked from
that commit onward. The guard now collects ignored paths beneath a move's own
prefixes as a third source. It is scoped to those prefixes rather than the
repository, because only the files a plan is about to carry are relevant and
ignored files are otherwise numerous.

That vector is worth stating alongside the three limits above for what it says
about the instrument: the guard asks git what the operator changed, while the
commit carries what the filesystem holds, and those are not the same set. G-0487
is the same shape reached by a different route. A comparison against HEAD's blobs
for every path a plan would carry would close both as one class; it is not what
this milestone built.

**The exemption's verification was corrected mid-milestone.** ADR-0038 specified
that the guard assert the exempted write's content equals the bytes on disk.
Measured, that test is unsound in both directions: it refuses a declarative revert,
and it accepts a write over hand-edited frontmatter — the laundering the epic
exists to stop. The shipped verification is the two-condition form recorded under
AC-6, and ADR-0038's *Escape hatch* subsection was amended to match rather than
left contradicting the code.

**AC-4's phase ladder is not ordering evidence.** The policy and its tests were
authored together, so no moment existed in which the test was written and the
implementation was not. The recorded `red` was produced afterwards by emptying the
ledger and observing the live assertion fail with one violation per entry point.
That demonstrates the assertion can fire — which this repo already pins
permanently through `firing_fixture_presence` — but it is not the test-first
ordering `aiwf history` is read as showing. Recorded here because re-staging the
ladder now would be the actual back-stamp.

## Constraints

- The prototype is throwaway. It is never committed, and no implementation commit
  may depend on it existing.
- The matrix is measured, not predicted. A cell filled by reasoning is not a
  measurement, and the numbers this milestone reports are its own.
- `checkStagedConflict` is the precedent for the message and the position, so the
  operator meets one message shape for one condition rather than two.
- Every route the guard covers is exercised *through* the guard by a test.
  Compiling onto a shared helper is not evidence that a route reaches it.
- Nothing here may be read as fixing the FSM walker's rename-plus-status blind
  spot. The precondition incidentally masks it; G-0475 stays open on its own
  terms.

## Design notes

- The comparison must name the *pre-mutation* working copy. By the time
  `gatherCommitOps` runs, Phase 1 moves and Phase 2 writes have already mutated
  disk, so a comparison there reads the verb's own bytes — measured to refuse
  every content-writing verb on a clean tree while skipping every move-shaped
  one.
- A pre-mutation guard prefix-matches rather than predicts. `stagedPathConflicts`
  already derives the nested set that way for the staged twin, with a comment
  recording why it is equivalent to walking the filesystem.
- Carry-along substitution — taking HEAD's blob for a path no verb named — is a
  candidate refinement, not a settled decision. Prototyped, it duplicated
  subtrees under `retitle` / `move` / `reallocate`, dropped a milestone from a
  `rewidth --apply` commit, and left flat-file renames uncovered; it also
  interacts with link-rewrite `OpWrite`s in a way that makes two sibling
  milestones behave oppositely. Adopt it only with those resolved, and record the
  reasoning either way.
- `ExitUsage` for the precondition needs a change in `internal/cli/cliutil`,
  which maps every `Apply` error to `ExitInternal`. That sits outside
  `internal/verb` and has to be scoped rather than assumed.

## Out of scope

- The claim-side precondition and the 22 NoOp sites — M-0284.
- Each multi-entity sweep's recorded in-or-out call — M-0284.
- The `internal/policies/` invariant keeping new routes on the seam — M-0285.
- After-the-fact detection of laundering already in history (G-0480).

## Dependencies

- M-0282 — ADR-0038, which settles the decisions this milestone implements and
  names the questions it answers.

## Work log

### AC-1 — Unstaged HEAD-divergent content is never committed silently
Guard landed at the top of `verb.Apply`, before Phase 1 · commit 51d04c88 · the
refusal reports exit 2 through a typed error rather than the internal-failure class.

### AC-2 — The measured priority-through-retitle laundering no longer succeeds
Closed by the same guard; `retitle` builds both an `OpMove` and an `OpWrite`, so it
exercises the overlap between the two mechanisms · commit 51d04c88.

### AC-3 — A verb over a dirty disk never commits a tree identical to its parent
Closed by the same guard, and the operator's edit survives the refusal · commit
51d04c88.

### AC-4 — Every verb entry point has a stated guard decision or a reasoned exemption
`internal/policies/verb_write_guard_coverage.go`: 28 entry points derived by AST
scan, each with a recorded treatment; fail-closed on an empty scan; stale entries
detected · commit 51d04c88. Extended at review to pin `AdoptsWorkingCopy`'s sole
setter, which the ledger could not see · commit dfeea4ee.

### AC-5 — The measured nested laundering through a parent rename no longer succeeds
Closed for hand-edited, untracked, and — after review measured a third route —
ignored files beneath a moved directory · commits 51d04c88, dfeea4ee.

### AC-6 — edit-body bless mode still commits a working-copy edit
Exemption extended to both modes, verified two ways rather than declared ·
commits 51d04c88, dfeea4ee. Explicit mode gained the frontmatter refusal bless
mode already had, which closes G-0463.

## Validation

`make ci` exit 0 on the implementation commit: `go vet` across the untagged,
`stress` and `testpins` tag sets, the full `golangci-lint` set, `go test -race`
over 72 packages with zero failures, `aiwf doctor --self-check` passing 29 steps,
and `govulncheck` clean.

After the corrective commit: `make check-fast` exit 0, the diff-scoped coverage
gate and firing-fixture meta-gate both exit 0 against this milestone's base, and
`aiwf check --since origin/main` reports no findings.

Each guard behaviour was additionally measured end-to-end against a real binary
in disposable repositories, not only through the Go tests.

## Reviewer notes

Two independent review rounds each found a live defect in the fix itself, and
both were invisible to reading — they took a running reproduction.

The first found that the `AdoptsWorkingCopy` exemption was itself a laundering
route: the guard verified the working copy but never the write's own content, so
a plan could commit fabricated frontmatter under any verb's trailer. The second
found that ignored files beneath a moved directory were committed while git
reported a clean tree, and that the refusal's own advice — `git restore` — errors
on an untracked path and destroys work irrecoverably on a tracked one.

Two limits are deliberate and stated where they are met rather than only here.
The exemption constrains frontmatter and says nothing about the body, because
changing the body is the exemption's purpose and the declarative revert depends
on it. And the guard is bounded by git's own reporting of the working tree, which
G-0492 records as a class rather than a list of instances.

One design alternative was considered and not taken: having explicit-mode
`edit-body` compose HEAD's frontmatter with the requested body, which would make
the exemption unforgeable by construction and delete the verification entirely.
It would also stop explicit mode re-canonicalising frontmatter, which a shipped
test pins deliberately. Recorded so the next reader need not re-derive it.

ADR-0038 was amended in place rather than superseded. Four of its five decisions
are untouched; only the escape hatch's verification mechanism was measured
unsound, and the amendment names the superseded statement rather than replacing
it silently.

## Deferrals

- G-0492 — the guard reads git's dirty report rather than the bytes a plan would
  carry; subsumes G-0487 and the ignored-file route closed here.
- G-0493 — `edit-body`'s two modes judge frontmatter divergence by different
  rules while sharing one message.
- G-0494 — refusals reach machine callers as prose, and the staged twin still
  exits 3 where this one exits 2.
- The duplicated entry-point AST scan shared with `verb_result_noop_invariant.go`
  belongs to E-0077's convergent-duplication sweep.
- G-0486's permanent-dirty interaction is now reachable: a directory move that
  flattens a mode leaves paths reading modified forever, and every later
  move-shaped verb on that directory refuses. Non-move verbs still work.

## References

- E-0075 — the parent epic
- ADR-0038 — the decisions settled, and the mechanics deferred here
- G-0466 — a verb commits frontmatter it does not own
- G-0463 — the `edit-body --body-file` instance
- G-0475 — the FSM walker blind spot this must not be read as fixing
- M-0281 — the same-state convergence work whose empty-diff class AC-3 protects
- `internal/verb/apply.go` — `checkStagedConflict` and `stagedPathConflicts`
- `internal/gitops/gitops.go` — `DirtyPaths`, the dirty-set primitive

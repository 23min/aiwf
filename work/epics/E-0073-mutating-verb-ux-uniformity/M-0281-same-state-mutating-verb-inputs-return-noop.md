---
id: M-0281
title: Same-state mutating-verb inputs return NoOp
status: in_progress
parent: E-0073
tdd: required
acs:
    - id: AC-1
      title: promote to current status returns NoOp instead of an error
      status: met
      tdd_phase: done
    - id: AC-2
      title: cancel of an already-terminal entity returns NoOp
      status: met
      tdd_phase: done
    - id: AC-3
      title: move to the current parent epic returns NoOp
      status: met
      tdd_phase: done
    - id: AC-4
      title: acknowledge-illegal on an acknowledged SHA is NoOp, appends no duplicate commit
      status: met
      tdd_phase: done
    - id: AC-5
      title: rename to current slug and retitle to current title return NoOp
      status: met
      tdd_phase: done
    - id: AC-6
      title: verb_result_noop_invariant policy pins same-state NoOp across mutating verbs
      status: met
      tdd_phase: done
    - id: AC-7
      title: Five field-mutation verbs converge to NoOp on same-state input
      status: met
      tdd_phase: done
    - id: AC-8
      title: edit-body --body-file converges to NoOp when the body is already committed
      status: met
      tdd_phase: done
    - id: AC-9
      title: Composite promote and cancel converge; cancel stops bypassing the AC FSM
      status: met
      tdd_phase: done
---

## Goal

Extend the NoOp-on-same-state convention already shipped in `archive`,
`rewidth`, `contract bind`, and `contract recipe install` to the mutating verbs
that still return a Go error — or silently commit nothing — when the requested
change already equals current state. An operator who re-runs
`aiwf promote M-… done` — interactively, or from a forgotten script — should get
a clean "already done" no-op at exit 0, matching the kernel's atomic /
single-commit / FSM-policed safety, not a confusing second-run error. The
tradeoff between fail-loud and idempotent resolves toward convergence here,
consistent with the four verbs that already behave this way.

Scope was set by the six verbs the source scan named up front (`promote`,
`cancel`, `move`, `rename`, `retitle`, `acknowledge-illegal`) and grew to
twelve: AC-6's policy surfaced five more the scan had missed, AC-7 converted
those with a same-state notion, and AC-8 added `edit-body --body-file`. AC-9
then extended `promote` and `cancel` to their composite-id paths, so the twelve
operator-facing verbs are sixteen entry points in the code. The growth is the premise being confirmed, not
scope creep — a convention that four verbs honored and the rest did not was
exactly the "half-rolled-out discipline rots" condition this milestone was filed
against.

`acknowledge-illegal` is additionally a **correctness** fix, not only a UX one:
re-running it against an already-acknowledged SHA appended a duplicate empty
audit commit — the "re-running creates duplicates" smell the scorecard's C2 noted.

The convergence is codified as a policy invariant (AC-6) so it cannot rot back to
one-of as new verbs land. For the field-mutation verbs (`move`, `rename`,
`retitle`, `acknowledge-illegal`) this completes a pattern already live in four
verbs. For the FSM-transition verbs (`promote`, `cancel`) it is a kernel-semantics
change — same-status is reclassified from FSM refusal to a first-class NoOp,
recorded in ADR-0036 — because the one-directional FSM encoded "same-status is
refused" across four correctness surfaces (the FSM, the legal-workflow spec, the
negative-driver, the stresstest oracle) that this milestone updates to model NoOp.

**Design note — the `promote` wrinkle.** `promote` already has a legitimate
same-status *mutation*: when a resolver flag is supplied it sets a resolver
pointer without changing status, deliberately skipping `ValidateTransition`
(`internal/verb/promote.go`). The NoOp guard must therefore fire only when the
target status equals current **and** no other field is changing, so it never
swallows a resolver-pointer write.

**Design note — the seam.** `verb.Result.NoOp` maps to exit 0 with `NoOpMessage`
on stdout; the CLI layer already surfaces NoOp for `archive`/`rewidth`. Each verb
is covered at the CLI seam (drive `run([]string{"<verb>", …})`), not just at the
verb layer, so a verb-layer NoOp that the CLI wiring drops would be caught.

## Acceptance criteria

### AC-1 — promote to current status returns NoOp instead of an error

`promote <id> <current-status>` with no other field changing returns
`Result.NoOp == true` (exit 0, descriptive message), not an
`FSMTransitionError`. The same-status path *with* a resolver flag continues to
perform the resolver-pointer mutation — the guard distinguishes "nothing is
changing" from "status is unchanged but a pointer is being set."

### AC-2 — cancel of an already-terminal entity returns NoOp

`cancel <id>` on an entity already at a terminal status returns
`Result.NoOp == true` instead of the current "already at terminal" error.

### AC-3 — move to the current parent epic returns NoOp

`move <M-id> --epic <epic>` where the milestone is already under that epic returns
`Result.NoOp == true` instead of the current "already under epic" error.

### AC-4 — acknowledge-illegal on an acknowledged SHA is NoOp, appends no duplicate commit

`acknowledge illegal <sha>` against a SHA already acknowledged returns
`Result.NoOp == true` and writes **no** commit — closing the duplicate-empty-
audit-commit path. The assertion checks both the NoOp result and that the commit
count is unchanged.

### AC-5 — rename to current slug and retitle to current title return NoOp

`rename <id> <current-slug>` and `retitle <id> <current-title>` each return
`Result.NoOp == true` instead of their current "matches the current slug" /
"title already" errors.

### AC-6 — verb_result_noop_invariant policy pins same-state NoOp across mutating verbs

`internal/policies/verb_result_noop_invariant.go` asserts, at the AST level, that
every exported `internal/verb/` entry point — a function returning
`(*Result, error)` — has at least one test under `internal/verb/` that both
drives it and asserts on `Result.NoOp`, unless it carries an allowlist entry with
a reason.

The two signals must be connected by *dataflow*, not merely co-occur: the
identifier a call's `*Result` is bound to must be the identifier whose `NoOp`
field is read. Scoping to a single test function is not enough on its own —
fixture setup and the assertion live in the same function, so a verb called only
to build a fixture would be credited by a NoOp assertion about some other verb's
result. On the live tree that was not hypothetical: a text scan credited `Add`
from 18 test functions and `Promote` from three of `Cancel`'s. The binding is
what separates them. The entry-point set is derived from the AST rather than
hardcoded, so a newly-added verb is picked up with no list to maintain.

The allowlist holds verbs with no same-state input to converge on — the bar is
"can a caller supply input that already equals current state?" Two kinds qualify
by design: purely additive verbs, which allocate a fresh id every call (`Add`,
`AddAC`, `AddACBatch`, `Reallocate`), and verbs that already compare and refuse
in their own body, so a same-state input writes nothing (`ContractUnbind` and
`RecipeRemove` refusing an absent target as a referential-integrity error).

Every reason states behavior verified by running the real binary and reading the
verb's source, never inferred from the verb's shape. That discipline is
load-bearing rather than pedantic: an inferred reason is how an allowlist entry
comes to excuse a live defect. The specific trap is the assumption that a
byte-identical write produces an empty diff aiwf rejects — it does not, so a verb
that merely looks additive can still be appending commits.

Six entries are marked `OPEN`: they record a deferred decision, not a settled
property, and each names its gap (`PromoteACPhase` → G-0458; the five
event-shaped verbs whose repeats append duplicate records → G-0459, with
`Authorize` additionally blocked on G-0460). Keeping them legible as holes rather
than dressing them as by-design exemptions is the point — an allowlist that
excuses a live defect is worse than no chokepoint, because it manufactures
confidence.

Granularity is structural, not semantic: it verifies such a test exists, not that
it drives genuinely same-state input. What it catches is the failure mode that
actually recurred here — a verb with no same-state NoOp coverage at all.

### AC-7 — Five field-mutation verbs converge to NoOp on same-state input

`set-area`, `set-priority`, `rename-area`, `milestone tdd`, and
`milestone depends-on` each return `Result.NoOp == true` on input that already
equals current state, closing the field-mutation holes AC-6's policy surfaced:

- `set-area <id> <current-member>` and `set-area <id> --clear` on an untagged
  entity (two guards).
- `set-priority <id> <current-level>` and `set-priority <id> --clear` on an
  unset priority (two guards).
- `rename-area <name> <same-name>`.
- `milestone tdd <M-id> --policy <current-policy>` and
  `milestone depends-on <M-id> --on <identical-list>` / `--clear` on an empty
  list — both **correctness** fixes, not only UX. Neither verb had any
  same-state guard, so a re-run wrote byte-identical content and still landed a
  commit with an empty diffstat, growing history on every repeat. Their
  assertions check the commit count, not just the result.

The empty-diff commits are possible because aiwf does not reject one: `Apply`'s
guard refuses only a plan with *zero* file ops, and `git commit-tree` has no
same-tree refusal, so a byte-identical write is one op that commits cleanly. The
guard therefore has to live in each verb. `depends-on` compares order-sensitively,
since `--on` is replace-not-append and a reordered list is a real change.

Verbs deliberately excluded, each carrying an `OPEN` allowlist entry in the AC-6
policy rather than a by-design reason: `promote --phase` (G-0458), and the five
event-shaped verbs whose repeats append duplicate records (G-0459), one of which
also leaves two active scopes (G-0460).

### AC-8 — edit-body --body-file converges to NoOp when the body is already committed

`aiwf edit-body <id> --body-file <content>` landed a commit unconditionally.
Handed content byte-identical to what was already committed, it produced a commit
whose diff was empty, and did so on every repeat — measured taking a repo from 2
to 5 commits across three runs, each with zero files changed, leaving three
indistinguishable rows in `aiwf history`.

This is the shape AC-7 treated as a correctness fix rather than UX polish for
`milestone tdd` and `milestone depends-on`: a verb with no same-state guard
silently polluting history. `edit-body` is a hotter path than either.

The verb's other mode keeps refusing. Bless mode (no `--body-file`) commits
whatever edit is already in the working copy, and when the working copy equals
HEAD it reports "no changes to commit". The two outcomes are not an
inconsistency to unify — they follow from whether the verb can check the
operator's target against reality. Explicit mode is handed a target, so it can
truthfully say the target is already met. Bless mode's input *is* the current
state, so it cannot distinguish "I meant to change nothing" from "my editor did
not save"; the refusal is the only honest answer to a premise it cannot falsify.

Convergence requires the serialized content to equal **both** the committed
bytes at HEAD and the bytes on disk. Neither comparison alone is correct:

- HEAD alone reports "already matches" while a dirty working copy holds
  different content, stranding an operator's revert and lying about the state.
- Disk alone converges when the working copy already carries the requested
  content uncommitted — never landing the commit that was the point of the call,
  which is exactly the agent-writes-then-routes-through-the-verb flow the
  guidance encourages.

The comparison is on serialized output, not body bytes: a byte-identical body
over non-canonical frontmatter still needs a real write, because
`entity.Serialize` re-canonicalizes.

#### Acceptance

- A clean tree plus byte-identical content returns a NoOp at exit 0 and the
  commit count is unchanged.
- A working copy carrying the requested content uncommitted still commits.
- A byte-identical body over non-canonical frontmatter still commits.
- Bless mode continues to refuse on a clean tree.
- `EditBody` leaves the `verb_result_noop_invariant` allowlist, and the policy
  is satisfied by a real NoOp assertion rather than an exemption.

### AC-9 — Composite promote and cancel converge; cancel stops bypassing the AC FSM

Two defects, one root cause. `cancelAC` decided for itself which AC statuses mean
"nothing left to cancel", hardcoding a comparison against `cancelled`, while
`promoteAC` asked `entity.IsLegalACTransition`. The FSM holds two terminal
statuses, so the verb's private answer was wrong — and because it never consulted
the FSM at all, it also wrote states the FSM forbids.

**The correctness half.** `aiwf cancel <M-NNNN>/AC-N` on a `deferred` AC exited 0
and transitioned it to `cancelled`, though that edge does not exist —
`acTransitions["deferred"]` is empty. The same edge asked of `promote` was
refused. Nothing caught the write afterwards: `aiwf check` reported zero errors,
the commit carried ordinary `cancel` trailers with no `aiwf-force:`, and the JSON
envelope publishes `from: deferred, to: cancelled` — an edge that is not in the
kernel. No `acs-transition` check rule exists. A code comment and two design
docs named one as the enforcement point; this milestone removed the comment, and
both docs now record that the rule was never shipped. A junk status
reaches the same write: an AC hand-edited to an unrecognized status is refused by
`promote` and laundered into `cancelled` by `cancel`.

**The convergence half.** Composite ids sat outside the convention this milestone
established, so one semantic family answered with three exit codes: `retitle` and
`rename` converge at 0, `promote` to the current status refuses at 1, and `cancel`
of an already-cancelled AC refuses at 2 through a bespoke error that never
reaches the FSM.

Converging `cancel` on a terminal AC is sound because **both AC terminals are
removal-class**: `deferred` and `cancelled` each mean "off the milestone's
contract", and neither claims the criterion succeeded. There is no success-terminal
to conflate with, which is what makes this narrower than the entity-level rule —
where `cancel` of a `done` entity converges and does absorb a success outcome. The
reasoning is a fact about `acTransitions`, not about terminality in general; a
future success-class terminal AC status would have to revisit it.

`met` is deliberately **not** terminal. An AC is a claim inside a contract that is
still being rescoped, so a met criterion can legitimately be descoped while its
parent milestone runs; an epic is a closed unit of work. `cancel` on a `met` AC
therefore keeps doing real work.

The verb consults the FSM rather than a policy asserting the FSM's shape. An
assertion that every non-terminal AC status can reach `cancelled` would contradict
the entity-level commitment already encoded in `PolicyFSMInvariants` — where
`ADR.accepted` is non-terminal with no cancel target, explicitly permitted — and
could not cover junk statuses anyway, since it quantifies over the declared set
while the defect arrives from disk.

#### Acceptance

- `cancel <M-NNNN>/AC-N` on a terminal AC (`deferred` or `cancelled`) returns a
  NoOp at exit 0, appends no commit, and names the actual status; it fires
  regardless of `--force`.
- `cancel <M-NNNN>/AC-N` on an AC whose status the FSM does not recognize is
  refused rather than written, and `--force` remains the sanctioned override.
- `cancel <M-NNNN>/AC-N` on a `met` AC still transitions.
- `promote <M-NNNN>/AC-N <current-status>` returns a NoOp instead of the FSM
  refusal, regardless of `--force`.
- The concurrent-milestone-race oracle judges promote actors by commits landed
  rather than by success count, so an AC NoOp is not read as a duplicate winner.


## Decisions made during implementation

- **ADR-0036** — Same-status FSM transitions converge to NoOp, not refusal.
  Surfaced while implementing AC-1: the one-directional FSM classified a
  same-status promote as illegal-and-refused across four correctness surfaces, so
  making it a NoOp is a kernel-semantics decision, not a local guard.
- **ADR-0037** — Retitle re-derives the slug only while it tracks the title.
  Surfaced at the final review: `aiwf rename` sets a slug independently of the
  title, so re-deriving unconditionally left rename's effect lasting only until
  the next retitle. Which surfaces a verb owns is a contract question, not a
  guard detail, so it is recorded rather than absorbed.

## Work log

### AC-1 — promote same-status returns NoOp
NoOp guard in `verb.Promote` gated on no resolver flag; same-status reclassified
from FSM refusal to a first-class NoOp and the four FSM-legality oracles updated
to model it. commit b0ea0a17 · tests: verb + CLI-seam + negative-driver +
stresstest classify, all green.

### AC-2 — cancel of a terminal entity returns NoOp
`verb.Cancel`'s terminal guard flips from a coded FSM refusal to a NoOp (Option A:
any terminal, not only cancel's cancel-class terminal). The concurrent-milestone-
race oracle now models the cancel NoOp — the "one cancel wins" invariant moves
from ok-count to commit-count. commit 8533f85f · tests: verb (2 new, 2 updated) +
CLI-seam + race oracle (scenario 4×) + classify unit, all green.

### AC-3 — move to the current parent epic returns NoOp
`verb.Move`'s `already under epic` guard flips to a NoOp. No oracle cascade: move
mutates a field rather than status, and the verb-sequence walk always targets the
*alternate* epic, never the current parent. commit 7387f474 · tests: verb +
CLI-seam, full suite green.

### AC-4 — acknowledge-illegal on an acknowledged SHA is a NoOp
The duplicate-empty-audit-commit path is closed. "Already acknowledged" is
answered by the check package's own ack walkers, so the verb's notion matches the
rules' exactly; the blanket per-SHA and per-(SHA, entity) shapes stay independent,
and an unwalkable history fails open to recording. Adds
`gitops.ResolveCommitSHA` so short and full SHA spellings compare equal.
commit dfcabfab · tests: verb (duplicate suppression asserted by commit count,
blanket-vs-entity-bound independence, orphan SHA under an unborn HEAD) + gitops
resolver + CLI-seam.

`check.resolveFullSHA` keeps its own resolver rather than routing through the new
gitops primitive. The reason first recorded here — that seven audit rules depend
on its unverified 40-hex fast path — did not survive checking: converging the two
in a scratch copy left every affected package green. The real difference is cost,
one `git rev-parse` per trailer on a walk that runs over every commit in HEAD's
history. That is a performance argument, and a weaker one than the original
claim, so the convergence stays available rather than foreclosed.

### AC-5 — rename and retitle to the current value return NoOp
Four guards converge, not two: the composite-id (AC) variants of both verbs
carried the same same-title refusal, so `rename`/`retitle` now behave uniformly
whether the target is an entity or one of its ACs. Three tests pinning the old
refusals are folded into the new same-state tests. commit d0d5b561 · tests: verb
(entity + both AC variants) + CLI-seam driving both commands.

### AC-6 — the convergence chokepoint, and what it immediately found
The policy earned its keep on first run: it flagged five verbs still outside the
convention (`SetArea`, `SetPriority`, `RenameArea`, `MilestoneTDD`,
`PromoteACPhase`) — evidence for the "half-rolled-out discipline rots" premise
the milestone was filed on.

Auditing the allowlist then found more, because the first draft's reasons were
*inferred* rather than measured. Re-running every exempt verb against a freshly
built binary and reading each implementation falsified six of thirteen entries:
`MilestoneDependsOn` (no guard at all), and the five event-shaped verbs
(`AcknowledgeMistag`, `Authorize`, and the three audit-only modes) whose repeats
each append a duplicate record. `EditBody`'s entry was false in both directions:
the byte comparison it cited exists only in bless mode, and the explicit
`--body-file` path had no comparison at all, so a byte-identical re-run landed
an empty-diff commit every time. AC-8 converges it and the entry is gone.

The false inference was one claim reused across entries: that aiwf rejects an
empty diff. It does not — `Apply` refuses only a plan with zero file ops, and
`git commit-tree` has no same-tree refusal, so a byte-identical write commits
cleanly. Every allowlist reason now states measured behavior, and the list's
header records that trap so a future entry cannot repeat it.

`MilestoneDependsOn` was converted under AC-7. The remaining six carry `OPEN`
entries naming their gap (G-0458, G-0459, G-0460) rather than a by-design reason,
so the chokepoint's known holes are legible instead of excused.

A firing fixture covers the violation branch, and its negative control proves the
check is not satisfied by the mere existence of a test — only a `Result.NoOp`
assertion in the same function counts.

### AC-7 — Five field-mutation verbs converge to NoOp on same-state input
`SetArea` (2 guards), `SetPriority` (2), `RenameArea` (1), `MilestoneTDD` (1) and
`MilestoneDependsOn` (2) converge. The last two are correctness fixes, not UX
polish: neither had any same-state guard, so each re-run landed an empty-diffstat
commit — the duplicate-commit shape AC-4 closed for `acknowledge-illegal`, and
worse than an error because it polluted history silently. Both assert commit
counts. `depends-on` compares order-sensitively, with a control test proving a
reordered list still commits.

Six pre-existing refusal assertions across the verb and CLI layers were retargeted
rather than deleted where they were the natural home for the seam coverage: the
`set-area` / `set-priority` CLI subtests now assert exit 0 + message + no commit.

Process note: the phase ladders for AC-6 and AC-7 were stamped after the fact
rather than live. The tests were genuinely written before their implementations,
but `aiwf history` cannot distinguish that from back-stamping — the ladder is
weaker evidence here than on AC-1 through AC-5. Measured commit times put the
AC-6/AC-7 ladder rungs in a 12-second burst over an hour *before* their
implementation commits, so for those two ACs the ladder is not evidence at all.

### AC-8 — edit-body --body-file converges when the body is already committed
Implemented in `fc4c1709`. `editBodyExplicit` gained a guard converging when the
serialized entity equals both the committed bytes and the bytes on disk, and
`EditBody` left the allowlist — the policy is now satisfied by a real NoOp
assertion rather than an exemption. Four verb-layer tests plus a CLI-seam test;
three mutants (HEAD-only, disk-only, body-bytes-instead-of-serialized) each kill
a test, so all three comparison choices are load-bearing rather than incidental.

The ladder ran live here, unlike AC-6 and AC-7: red was stamped before the tests
were written, and the primary case failed for the expected reason before the
guard existed. The other three tests passed from the start by design — they are
regression guards on behavior that was already correct and that the fix must not
break.

Two corrections rode this AC because they were false statements in text it was
already editing: AC-6's same-function claim (function scoping does not prevent
fixture-setup credit; the dataflow binding does), and the policy's citation of
ADR-0036 as authority for a convention spanning entry points that ADR explicitly
disclaims.

### AC-9 — Composite promote and cancel converge; cancel stops bypassing the AC FSM
Implemented in `54360c30`. `IsTerminalACStatus` derives AC terminality from
`acTransitions`; `cancelAC` converges on a terminal AC and otherwise consults the
FSM; `promoteAC` converges on a same-status request above its existing consult.

The convergence half was the filed finding; the correctness half was found while
investigating it. `cancel` on a `deferred` AC wrote an edge the FSM does not
contain, and a junk status was laundered into `cancelled` the same way — both
invisible to `aiwf check`, `aiwf history` and the JSON envelope, because the
`acs-transition` rule those surfaces named as the enforcement point was never
shipped. `promoteAC` had asked the FSM all along; `cancelAC` was the lone
outlier.

Two review rounds changed this materially rather than confirming it. A proposed
policy asserting every non-terminal AC status can reach `cancelled` was dropped:
it contradicts `PolicyFSMInvariants`, which permits exactly that shape at kind
level, and it could not have covered the junk-status case anyway since it
quantifies over the declared set while the defect arrives from disk. And the
promote half of the concurrent-milestone-race oracle broke deterministically
under the change — its actors drive a composite id — which the review predicted
and measurement confirmed before the fix landed.

`met` stays non-terminal, so cancelling a met AC still does real work. The
reasoning now lives beside the FSM data it constrains: an AC is a claim inside a
contract that can still be rescoped, an epic is a closed unit of work.

### Wrap rounds — remediation, and the record of what the commits claimed

Three commits closed review rounds four and five (`f7c37f16`, `65a91ee1`,
`888b3393`); four more closed round six (`a9f3051d`, `d04b4a8e`, `14a1a8ed`,
`df7478e3`), alongside ADR-0037.

Round six reviewed the round-four and round-five commits themselves — the one
part of the milestone no independent pass had covered. Its code lens planted 27
mutants and killed 22. Every conjunct in every changed guard proved
load-bearing, the policy fails closed on an empty entry set, and the race oracle
detects both a byte-identical re-write from cancel's NoOp branch (4 of 5
attempts) and an unclaimed `--allow-empty` commit (5 of 6). Three of the five
surviving mutants are provably equivalent rather than untested.

It also found a genuine hole in that policy: `:=` replaced a name's binding set
unconditionally, so a second one in the same scope transferred credit to the
verb it named. A verb with no NoOp assertion of its own could then satisfy the
chokepoint on a neighbour's assertion. No verb depended on the shape, so the
hole was latent rather than live; `a9f3051d` closes it.

Six claims in the earlier commit messages are not supported by the commits that
carry them. They are stated here as measured, rather than rewritten in history:

- `65a91ee1` says all six non-primary guard conjuncts gained false-arm tests.
  There are five such conjuncts, not six, and all five gained one: `move`'s path
  comparison, `retitle`'s path and H1 comparisons, and the AC heading comparison
  on each of `retitleAC` and `renameAC`. No conjunct is untested.
- The same commit reports roughly 200 comment lines removed. It removes 143 and
  adds 130: a net of 13 across the commit.
- It reports the empty-diff premise consolidated into the policy that depends on
  it. All three restatements survived it; the consolidation lands in `14a1a8ed`.
- It calls the deep-copy guard it discarded unfalsifiable. With the verb sets
  shared, a closure's `=` reaches whichever sibling the walk visits next, so two
  programs identical but for sibling order return different answers. The guard
  changed no outcome only because no test exercised `=` inside a closure;
  `a9f3051d` restores the copy and adds that test.
- It describes the round-four scoping defect as a nested scope costing the outer
  scope its credit. A scope's credit is decided before the walk descends into
  it, so the loss landed on a sibling.
- `f7c37f16` says convergence-above-force is tested in both verbs. `promote` was
  tested at both granularities from AC-1 onward; `cancel` only at the AC
  granularity. The entity-level `Cancel` force arm gained its test in
  `d04b4a8e`.

`f7c37f16`'s message was amended once, to drop two fixes it claimed but did not
contain. The amendment left standing a summarizing claim that it fixed the whole
set, which is equally unsupported — those two fixes landed in `65a91ee1`.

Round six additionally surfaced a contract conflict this milestone made visible
rather than created. `aiwf rename` sets a slug independently of the title while
`retitle` re-derived one unconditionally, so a rename lasted only until the next
retitle. Measured against this repo's own tree, 44 of 902 entities carry a slug
their title does not derive; widening the narrow ids embedded in those slugs
reconciles 17, leaving 27 deliberate short paths. ADR-0037 records the resolution — retitle
re-derives only while the slug still tracks the title — and `d04b4a8e`
implements it, with `df7478e3` closing a slug-warning test hole the restructure
exposed in a path that predates this milestone.

## Validation

Every result below was taken after the commit it describes. `make coverage-gate`
compares committed HEAD against the merge-base, so a pre-commit run measures the
wrong diff — that trap cost two false greens earlier in this milestone.

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test -count=1 ./...` — every package passes.
- `make lint` — 0 issues.
- `make coverage-gate` — exit 0.
- `aiwf check` — 0 errors. Two warnings stand, both pre-existing and unrelated:
  an active epic with no drafted milestones, and the provenance audit skipping
  for want of an upstream ref.

The concurrent-milestone-race scenario was additionally driven against a built
binary during round six, with planted mutants confirming the oracle fires rather
than merely passing.

## Deferrals

Each carries a gap, so none of it depends on this spec being read again.

- **G-0458** — `promote --phase` refuses a same-phase input rather than
  converging. The phase ladder is audit-bearing evidence and the verb carries a
  test payload, so convergence needs a deliberate carve-out, not a mechanical
  repeat.
- **G-0459** — five event-shaped verbs append a duplicate record on an identical
  re-run.
- **G-0460** — a repeat `authorize` leaves two simultaneously-active scopes on
  one entity with no check finding. It settles the invariant G-0459's
  `Authorize` entry waits on.
- **G-0461** — `acknowledge illegal --for-entity <composite>` emits at full
  composite width while the check walkers look up the rolled-up key, so the flag
  suppresses nothing.
- **G-0462** — intermittent suite failures from two distinct causes: ETXTBSY on
  exec of a just-written file, and a repo-lock budget that holds only on an idle
  machine.
- **G-0463** — `edit-body --body-file` re-serializes from the loaded entity, so
  a working-copy frontmatter edit rides into the commit. The verb is not
  body-only in practice, despite its doc.
- **G-0464** — three `internal/check` predicates skip `cancelled` but not
  `deferred` when deciding whether an AC is out of scope, though the FSM makes
  both terminal.
- **G-0465** — no chokepoint catches a shipped surface drifting from the verb
  behavior it describes. Three separate review rounds each caught more of them by
  reading, which is the argument for the gap: reading is the only detector, and
  it does not scale.

## Reviewer notes

Thirteen independent fresh-context reviewers ran across six rounds: four at the
first wrap attempt over `base..HEAD` (three code-quality slices — verb guards,
correctness oracles, the AC-6 policy — plus a design-quality pass), two more on
each of the AC-8 and AC-9 designs before they were implemented, then a narrow
sixth round over the remediation commits themselves, split into a code lens and
a prose lens. Every round returned request-changes. Every finding was verified by
measurement — real binary against disposable repos, or production mutants in
scratch copies — not by reading.

The rounds separate cleanly by what they caught. Rounds one through five found
defects in the code; round six found one latent defect in the code and nine in
the prose describing it, six of those in commit messages. That distribution is
itself a finding: the mechanical gates cover the code, and nothing mechanical
reads a claim about the code back against it, which is what G-0465 is filed
for.

**Every blocking finding is resolved, and every non-blocking one except those
named at the end.** The first round falsified three ACs outright, and each was
reconciled by making the code true rather than by narrowing the claim: AC-1 by
converging the resolver arm, AC-4 by converging composite acknowledgments, and
AC-6 by rewriting the credit relation and removing a false allowlist entry.
Choosing the other direction — editing the criterion to match what shipped —
would have left three ACs that passed by definition.

The later rounds paid for themselves twice over. The AC-8 review caught a guard
that would have shipped a false NoOp — reporting "already matches" while a dirty
working copy held different content, stranding an operator's revert. The AC-9
review predicted, and measurement confirmed, that converging composite promotes
would break the concurrent-milestone-race oracle deterministically in
`go test ./...`; it also falsified a policy assertion this milestone had planned
to add, by reaching a branch that assertion assumed unreachable.

### Resolved

Round one returned seventeen blocking findings across three code-quality slices
and a design pass: verb guards (V1–V6), correctness oracles (O1–O4), and the
AC-6 policy (P1–P7). All are resolved, and the design they produced is stated in
the acceptance criteria and work log above rather than reconstructed from the
findings here.

Two changed the design rather than fixing a defect, and both have a permanent
home elsewhere. V5 forced AC-1's guard from "no resolver flag was supplied" to
"nothing would be written", which is what ADR-0036's resolver bullet now records.
P1 forced the chokepoint's credit relation from a body-text scan to intra-function
AST dataflow, which AC-6's body states. The remaining fifteen were defects: they
left no trace beyond the code that no longer has them.

### Non-blocking

Closed except where noted inline.

- An eighth inferred claim, disproved: the work log says `check.resolveFullSHA`
  cannot converge with `gitops.ResolveCommitSHA` because audit rules depend on its
  unverified 40-hex fast path. Converging them in a scratch copy left every
  affected package green. The real difference is one `git rev-parse` per trailer on
  a hot path — a performance argument. Restate it, and file a gap for the
  convergence.
- Stale docstrings still documenting removed refusals: `setarea.go`,
  `setpriority.go`, `renamearea.go`, `cancel.go`, `move.go`, `retitle.go`.
- `acks.go` and `acks_helper_lift.go` docs claim a single/four-consumer set that
  the new verb call site invalidates; that helper's own doc names same-commit
  updating as the chokepoint.
- NoOp surface inconsistency across the converging verbs: `promote` omits the
  `"; nothing to …"` clause, `move` alone quotes the id, all messages echo the raw
  rather than canonicalized id, and none set `Result.Metadata` — so a JSON
  consumer distinguishes no-op from mutation only by prose in `result.subject`.
- `verb_sequence_classify_test.go` asserts violation counts only; a mutant
  swapping the two same-status arms survives the suite. The sibling race classify
  test already uses `wantSubstrings`.
- `shaAckable` and `ResolveCommitSHA` issue two `git rev-parse --verify …^{commit}`
  subprocesses for one question; the latter subsumes the former.
- The policy's OPEN entries have no ratchet, no gap-shape check, and no
  stale-entry check, so a resolved hole survives as a false exemption. A typed
  state field plus those three assertions is the sibling `grandfatherDark` idiom.
- AC-6 body lines carry drafting history inside the criterion text; keep the
  reasoning, drop the framing.

Those that are **not** closed are recorded under
"Deliberately not closed here" below, with the reasoning for each.

### Deliberately not closed here

The work this milestone found but did not do is enumerated under `## Deferrals`
above, each with its gap. Three items are recorded here instead, because for
them the decision is to not act rather than to act later. A fourth, at the end,
is a live residual small enough to carry no gap of its own.

- `Result.Metadata` stays unset on a NoOp. `metadata.commit_sha` is already the
  machine-readable discriminator — present on a mutation, absent on a NoOp — and
  the race oracle reconciles against it, so a second signal would be a second
  source of truth for one fact.
- The race oracle reconciles reported successes against the observed commit
  delta across both actor groups, but only the cancel group also reconciles
  per-group. The promote group's own bound is counted from git order and catches
  a zero as readily as a two, and the cross-group reconciliation catches an
  unclaimed commit from either side, so a per-group promote mirror would add no
  shape this scenario can produce. It is available if a future scenario needs
  one.
- NoOp messages echo the operator's spelling rather than the canonical id. The
  `; nothing to …` clause and `move`'s id quoting were aligned across the verbs;
  the raw echo was left, because the message names what the operator typed and
  that is the more useful thing to read back.

The allowlist's `OPEN` entries still carry no ratchet, no gap-shape check and no
stale-entry check, so a resolved hole would survive as a false exemption. The
sibling `grandfatherDark` idiom is the shape to copy. This one is a genuine
residual rather than a decision, and it is small enough to ride the next change
to that file.

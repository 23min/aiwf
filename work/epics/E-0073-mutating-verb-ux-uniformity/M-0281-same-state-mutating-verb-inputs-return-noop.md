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
      status: open
      tdd_phase: red
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
`cancel`, `move`, `rename`, `retitle`, `acknowledge-illegal`) and then grew to
eleven: AC-6's policy surfaced five more the scan had missed, and AC-7 converted
those with a same-state notion. The growth is the premise being confirmed, not
scope creep — a convention that four verbs honored and the rest did not was
exactly the "half-rolled-out discipline rots" condition this milestone was filed
against.

`acknowledge-illegal` is additionally a **correctness** fix, not only a UX one:
re-running it against an already-acknowledged SHA today appends a duplicate empty
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

Both signals must come from the *same* test function: a file-level co-occurrence
would credit any verb merely used as fixture setup alongside an unrelated NoOp
assertion. The entry-point set is derived from the AST rather than hardcoded, so
a newly-added verb is picked up with no list to maintain.

The allowlist holds verbs with no same-state input to converge on — the bar is
"can a caller supply input that already equals current state?" Two kinds qualify
by design: purely additive verbs, which allocate a fresh id every call (`Add`,
`AddAC`, `AddACBatch`, `Reallocate`), and verbs that already compare and refuse
in their own body, so a same-state input writes nothing (`EditBody`'s
working-copy-vs-HEAD byte comparison, `ContractUnbind` and `RecipeRemove`
refusing an absent target as a referential-integrity error).

Every reason states behavior verified by running the real binary and reading the
verb's source. That discipline is load-bearing rather than pedantic: the first
draft inferred its reasons instead, and six of thirteen were wrong — most reusing
a single false claim, that an identical write is an empty diff aiwf rejects.

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

`aiwf edit-body <id> --body-file <content>` lands a commit unconditionally. Handed
content byte-identical to what is already committed, it produces a commit whose
diff is empty, and it does so on every repeat — measured taking a repo from 2 to
5 commits across three runs, each with zero files changed, leaving three
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

## Acceptance

- A clean tree plus byte-identical content returns a NoOp at exit 0 and the
  commit count is unchanged.
- A working copy carrying the requested content uncommitted still commits.
- A byte-identical body over non-canonical frontmatter still commits.
- Bless mode continues to refuse on a clean tree.
- `EditBody` leaves the `verb_result_noop_invariant` allowlist, and the policy
  is satisfied by a real NoOp assertion rather than an exemption.

## Decisions made during implementation

- **ADR-0036** — Same-status FSM transitions converge to NoOp, not refusal.
  Surfaced while implementing AC-1: the one-directional FSM classified a
  same-status promote as illegal-and-refused across four correctness surfaces, so
  making it a NoOp is a kernel-semantics decision, not a local guard.

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
gitops primitive: its `len(sha)==40` fast path returns unverified input as-is and
seven audit rules depend on that, so converging them is a separate concern.

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
each append a duplicate record. `EditBody`'s conclusion held but its stated
mechanism was wrong.

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

## Reviewer notes

Four independent fresh-context reviewers ran at wrap over `base..HEAD` (three
code-quality slices — verb guards, correctness oracles, the AC-6 policy — plus a
design-quality pass on the policy module). All four returned request-changes.
Findings below are open unless marked resolved; every one was verified by
measurement (real binary against disposable repos, or production mutants in
scratch copies), not by reading.

Three ACs are `met` on claims the review falsified: **AC-1** (V5), **AC-4** (V2),
**AC-6** (P5, P6). The milestone must not wrap until these are reconciled — either
the code converges or the AC text and allowlist state the narrower truth.

### Blocking — verb guards

- **V1 — the committed tree fails `-race`.** All five CLI-seam NoOp tests carry
  `t.Parallel()` plus `testutil.CaptureStdout`, which swaps the process-global
  `os.Stdout`. The whole `internal/cli/integration` package fails from commit
  `8533f85f` onward, cascading into unrelated `TestAuthorize_*`; `-count=20`
  yields 23 `write |1: file already closed`. The fix (drop `t.Parallel()`, add the
  serial rationale required by that package's `setup_test.go` skip-list) exists in
  the working tree and must be committed. The base is clean, so the regression is
  wholly this milestone's.
- **V2 — a composite `--for-entity` ack never converges; AC-4 is false.** The verb
  emits `aiwf-entity` at full composite width; `check.WalkAcknowledgedSHAEntities`
  stores it verbatim; `ackAlreadyRecorded` looks up the `CompositeRoot` rollup.
  The keys never match, so a repeat lands a duplicate audit commit — the exact
  path AC-4 claims to close. The rollup is correct in
  `verifySHATouchesEntity` (a diff resolves to a milestone path) and wrong in the
  lookup. The inverse case also misreports: after a bare-id ack, the composite
  form claims "already acknowledged for `M-NNNN/AC-N`", naming a binding it never
  found. Fix: consult both the composite and rolled-up keys. Separately, the
  emit-vs-lookup asymmetry makes `--for-entity <composite>` inert against the
  `provenance-untrailered-entity-commit` rule it exists to suppress — pre-existing,
  wider than this milestone, wants its own gap.
- **V3 — `rename-area`'s guard precedes its validation.** It returns a NoOp for an
  undeclared member and for the reserved `global` sentinel, asserting success for
  state that cannot exist, where the verb previously refused. `SetArea` places the
  equivalent guard after validation; this is the outlier. Move it below the
  declared-member check.
- **V4 — width-sensitive comparison in `move` and `milestone depends-on`.** Both
  compare raw operator input against stored values, but `ByID` accepts legacy
  narrow widths, so a legal spelling of the same state is missed — and the
  resulting write degrades a canonical id to narrow width, against ADR-0008.
  `entity.Canonicalize` on both sides of each comparison.
- **V5 — the resolver arm leaves AC-1 half-met.** AC-1 says "target equals current
  **and no other field is changing**"; the guard implements "no resolver flag was
  supplied". A re-run of `promote <gap> addressed --by-commit <sha>` — the
  tracker-closure command this repo's own gate discipline names as routine — still
  refuses. Either extend the guard to "status equals current and
  `applyResolverFlags` would be a no-op", or narrow AC-1's wording and record the
  carve-out as an `OPEN` allowlist entry plus a gap.
- **V6 — composite ids still refuse, so ADR-0036 over-claims.** One semantic family
  yields three exit codes: entity same-status 0, AC same-status 1, AC
  already-cancelled 2. ADR-0036's Decision carries no AC carve-out, and G-0458
  covers `tdd_phase` only — neither AC status nor `cancel <composite>`. Converge,
  or narrow the ADR and file the gap. Per-function policy granularity cannot see
  branch-level holes like this.

### Blocking — correctness oracles

- **O1 — the concurrent-milestone-race oracle lost a cross-check it could have
  kept.** Replacing the (genuinely unsound) `cancelOKCount > 1` invariant dropped
  the per-actor "reported success implies exactly one commit" check instead of
  re-deriving it. A NoOp is machine-distinguishable — a real cancel's JSON
  metadata carries `commit_sha`, a NoOp's does not — but `verbEnvelope.Metadata`
  never parses that field. Consequence, measured: a mutant that lands
  `git commit --allow-empty` from cancel's NoOp branch (a real violation of the
  one-commit-per-mutation commitment) leaves **`make stress` green**. Two further
  shapes went undetected: two cancel "ok"s with one commit, and one "ok" with
  zero. The `-1` sentinel is now dead code. Fix (~8 lines): parse `commit_sha`,
  carry it on `raceActorOutcome`, assert real-success count equals cancel-commit
  count and that every NoOp carries an empty `commit_sha`.
- **O2 — the spec-cell justification is false.** `terminalIllegal`'s new comment
  claims the spec "carries no target axis". `Rule.Preconditions` is that axis and
  is already live (`self.target-state`, with `Op: "!="` in the vocabulary, and
  `cellKey` folding `preconditionSignature` into cell identity). The fix declared
  impossible is available with no schema change, so the false reason forecloses
  it. Either add the precondition, or record the NoOp via an `AntiRule`, and state
  the real constraint (cell-count churn) instead.
- **O3 — dead OR-arm and stale cross-references.** `"is already at terminal
  status"` in `errorSubstringsFor` is now unreachable as a refusal; if any future
  path re-emits that phrasing the arm would bless it. `promote.go`'s
  `fsmTransitionIllegalError` doc still names Cancel's already-terminal pre-check
  as a construction site — Cancel no longer constructs the type.
- **O4 — the fourth outcome shape has no real-binary seam test**, and
  `verb_sequence_test.go` still documents three. The NoOp shape is pinned only
  against fabricated envelopes and seed-dependent walk coverage.

### Blocking — the AC-6 policy

- **P1 — the credit relation admits three false greens** (both the policy and
  design reviewers converged here independently, and one prototyped the fix). A
  `.NoOp` mention in a **comment** credits the verb; so does a **negative**
  assertion (`if res.NoOp { t.Fatal(...) }`); so does **fixture setup** — on the
  live tree `Add` is credited by 17–18 test functions that merely call it as
  setup. Replace the body-text scan with intra-function AST dataflow: bind the
  identifier receiving the call, require `<ident>.NoOp` to be read. Verified to
  keep all 15 non-exempt verbs green, drop every spurious credit, and need no test
  changes.
- **P2 — the policy fails OPEN.** An empty entry set yields zero violations, so a
  rename of `internal/verb/` leaves it green while scanning nothing. Sibling
  `firing_fixture_presence.go` fails closed for exactly this class. Note the
  enumerate-and-slice block is now its third copy, so the defect is tripled.
- **P3 — branch-coverage rule violated; `make coverage-gate` fails.** Five changed
  lines untested and unannotated, confirmed with the repo's own engine. The
  notable pair is `containsBareCall`'s continuation and `isIdentByte`'s
  `return true` — the only subtle logic in the file, advertised as load-bearing,
  never traversed. The AST rewrite (P1) deletes both. `make coverage-gate` has
  never been run for this milestone.
- **P4 — `containsBareCall`'s rationale is mechanically false and its helper's doc
  contradicts the behavior callers depend on.** The needle includes `(`, so
  `Promote(` cannot match inside `PromoteAuditOnly(` — the described collision is
  impossible. What the guard actually rejects is a qualified or suffixed call
  (`harness.Promote(`, `mustPromote(`), and it works only because `isIdentByte`
  treats `.` as an identifier byte while its doc says a period cannot appear in an
  identifier. A maintainer "correcting" that turns every `pkg.Verb(` into a false
  credit — on an untested line.
- **P5 — `EditBody` is a seventh false allowlist entry.** `bytes.Equal` exists only
  in `editBodyBless`; `editBodyExplicit` has no equality comparison and emits one
  `OpWrite` unconditionally. Measured: `edit-body --body-file <byte-identical>`
  exits 0 and lands an empty-diffstat commit, repeatably — the same shape AC-7
  treated as a correctness fix elsewhere. Not covered by G-0459 (scoped to
  event-shaped verbs). Pick one: guard `editBodyExplicit`, or reclassify the entry
  as `OPEN` with a new gap. The AC-6 body and the work log need the same
  correction.
- **P6 — the same-function claim is false in both the code comment and the AC-6
  body.** Function scoping was presented as preventing fixture-setup credit; setup
  lives in the same function, so it does not. State the real limit.
- **P7 — ADR-0036 scope overreach.** The policy cites ADR-0036 as its authority but
  enforces the convention across all 28 entry points, while the ADR scopes itself
  to `promote`/`cancel`. Widen the ADR or cite the convention instead.

### Non-blocking, carried

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
- NoOp surface inconsistency across the eleven verbs: `promote` omits the
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


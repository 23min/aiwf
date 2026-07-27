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
weaker evidence here than on AC-1 through AC-5.


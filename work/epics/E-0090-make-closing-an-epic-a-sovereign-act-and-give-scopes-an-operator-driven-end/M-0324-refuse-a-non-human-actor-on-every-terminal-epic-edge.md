---
id: M-0324
title: Refuse a non-human actor on every terminal epic edge
status: in_progress
parent: E-0090
depends_on:
    - M-0323
tdd: required
acs:
    - id: AC-1
      title: Promoting an epic to done with a non-human actor is refused before any write
      status: met
      tdd_phase: done
    - id: AC-2
      title: Cancelling an epic with a non-human actor is refused at the verb, not the audit
      status: met
      tdd_phase: done
    - id: AC-3
      title: Every commit the widened audit fires on is ratified and check reports no error
      status: met
      tdd_phase: done
    - id: AC-4
      title: The static audit catches a scripted aiwf cancel of a sovereign edge
      status: met
      tdd_phase: done
    - id: AC-5
      title: The audit catalogue names every transition in the sovereign closed set
      status: met
      tdd_phase: done
    - id: AC-6
      title: The legal-workflow spec models sovereignty, derived from the closed set
      status: met
      tdd_phase: done
---
## Goal

Refuse a non-human actor on every edge into a terminal epic status, at the verb, before anything is written — and ratify the historical acts the widened audit reaches.

## Closes

- G-0646 — every terminal epic edge gated, with the `cancel` call site that makes the two cancel entries enforceable at the verb.

## Context

`sovereignActShapes` holds one entry, epic `proposed → active`. Measured in a fixture, `aiwf promote <epic> done --actor ai/claude --principal human/fixture` exits 0, flips the status, writes no force trailer, and leaves a tree `aiwf check` calls clean. Commit `c030cb926` is that act, already in this repo's history.

ADR-0047 rules every edge into a terminal epic status sovereign, so three entries join the set: `active → done`, `active → cancelled`, and `proposed → cancelled`.

Adding `active → done` is a one-line change, because `promote` already calls the gate. The two cancel edges are not: `requireHumanActorForSovereignAct` has a single call site, so set entries alone would leave `cancel` silent while the history audit fired on the landed commit — refusal after the act, which is the record ADR-0040 exists to prevent. The audit sees a cancel because it compares an entity's `status:` field across a commit and its parent rather than reading the verb's trailers, and a cancel commit carries no `aiwf-to:` at all.

## Acceptance criteria

### AC-1 — Promoting an epic to done with a non-human actor is refused before any write

`aiwf promote <epic> done` with a non-human actor returns an error naming the required `human/` actor, `HEAD` is unmoved, and the entity file is unchanged. A `human/` actor on the same transition still succeeds, so the gate is scoped to the actor rather than the edge.

### AC-2 — Cancelling an epic with a non-human actor is refused at the verb, not the audit

`aiwf cancel <epic>` with a non-human actor is refused before anything is written, by the same predicate, from both `active` and `proposed`. The distinguishing assertion is *where*: the refusal comes from the verb, with `HEAD` unmoved — not from a later `aiwf check` over a commit that already landed.

### AC-3 — Every commit the widened audit fires on is ratified and check reports no error

After the closed set widens, `aiwf check` over the tree reports no error-severity `fsm-history-consistent` finding. The historical acts the audit now reaches carry a ratification recorded by a human with a written reason.

Stated as a property of the tree rather than as a count, because widening the set is what determines which commits qualify, and a number written now would be a forecast.

The evidence is the shipped `fsm-history-consistent/forced-untrailered` rule, which fires on exactly this class and blocks the pre-push hook, together with the acknowledgment commit that clears it. No policy test asserts the same property a second time: D-0081 records why, with the measurements that decided it. The claim is pinnable and is deliberately left unpinned, which is the reason it carries a decision rather than a silent omission.

### AC-4 — The static audit catches a scripted aiwf cancel of a sovereign edge

The audit keys its patterns on `(prefix, To)`, so it matches only the `aiwf promote <id> <to>` spelling. It emits a pattern for the `aiwf cancel <id>` spelling too — one per distinct id prefix in the closed set — and fires on a line carrying it without `--force`.

One regex per kind is all the cancel spelling can express, because it names no status and so cannot discriminate on the from-state the way the promote form does. That makes it wider than the closed set for any kind whose sovereign edges are not all cancel-reachable: a scripted cancel of such a kind would be reported though the kernel permits it. No kind is in that position, and the audit's subject is automation-shaped source, where the finding names a file and a line and is cheap to answer.

Fails if a widened set adds a prefix whose cancel spelling the audit cannot see.

### AC-5 — The audit catalogue names every transition in the sovereign closed set

Every entry in `entity.SovereignActShapes()` is named by the sovereign-acts section of `docs/design/legal-workflows-audit.md`, with the expectation derived from the closed set rather than written as a literal — so widening the set without touching the catalogue turns the check red and names the missing transition.

It pins that the transitions are *named*, not that what the rows say about them is true. R-RULE-001's Note is false in a way this check does not reach; content correctness in the catalogue stays held at review.

### AC-6 — The legal-workflow spec models sovereignty, derived from the closed set

`spec.GlobalRules()` carries a cross-cutting rule for the sovereignty gate, so the legality encoding states that a human is required to open or close an epic. Before this milestone it stated that for no edge at all, the shipped `proposed → active` included: every epic cell was bare `OutcomeLegal`, and the only preconditioned ones were the child-cascade pair.

The rule's subject names the kernel's closed set — `sovereign-act-shape` — rather than enumerating the entries it holds today. Nothing therefore needs re-syncing as the set widens, and there is no second copy of the set to drift. The alternative, listing the four transitions and deriving the expected list in a test, would have bought a drift check by first creating the drift it detects; single-source-of-truth says make the drift impossible instead.

That choice is what the AC can honestly claim. A symbolic rule cannot go stale, so no test asserts that it tracks a widening — there is nothing to track. What is asserted is that exactly one such rule exists, that it is `OutcomeIllegal` at `RejectionLayerVerbTime` with `BlockingStrict` (matching where the refusal happens, which ADR-0040 requires of a prevention rule), that its preconditions match subject, operator and value exactly — the value axis carries the meaning, since `force == "true"` inverts the rule and an actor-role compared against anything but `human` refuses the wrong population — and that its authorizing record resolves through the loader at status `accepted`. The residual risk is stated rather than hidden: the subject is a string the spec and the kernel each spell independently, so a rule renamed on one side and not the other is caught by review rather than by a test.

The rule carries no `ExpectedErrorCode`, because the refusal has none to name. That keeps it outside the spec's two code-oriented drift arms, which skip any rule with an empty code. `TestM0123_AC2_IllegalImpliesErrorCode` iterates `Rules()` and not `GlobalRules()`, so the empty code is schema-legal; G-0649 carries the underlying question.

The citation check is this AC's own, not a reuse. `TestM0123_AC6_RuleDecisionSourcesResolve` resolves `Sources.Decision` for `Rules()` cells but requires the target to be a decision entity, while the cross-cutting rules cite ADRs — so it could not cover this rule without redefining what that field admits.

## Constraints

- Sovereign-act shape is a property over legal transitions (D-0008); every entry added is FSM-legal and `TestSovereignActShapes_AllFSMLegal` stays green.
- A closed-set entry and the call site that enforces it land together (ADR-0040). Neither edge is added ahead of its verb-time refusal.
- `--force` stays human-only; the coherence guard at `verb.Apply` is not modified.
- The ratification is a sovereign act, so it is performed by a human and cannot be delegated.

## Design notes

The refusal message names only the human-run path. Offering `--force` there would be wrong every time it appeared: the message is reachable only for a non-human actor, and `verb.Apply` refuses that actor's force trailer anyway.

Two other consumers read the same closed set. The history audit in `internal/check/fsm_history_consistent.go` widens with no edit. The static audit in `internal/policies/aiwf_promote_epic_active_audit.go` picks up the new entries for the `promote` spelling with no edit, and AC-4 extends it to the `cancel` spelling. That pattern cannot discriminate on `From`, so for a kind where only some from-states were sovereign it would over-match; no such kind exists — for epics both from-states reach `cancelled` — and the builder says so where it makes the choice.

Six rows of `docs/design/legal-workflows-audit.md` scope sovereignty to epic activation: R-AUDIT-0050, R-AUDIT-0113, R-AUDIT-0115, R-RULE-001, R-RULE-002 and R-RULE-078. AC-5 brings them onto the widened set. Two carry defects predating this milestone, corrected in passing: R-AUDIT-0050 cites `auditUnforcedEpicActivate`, a function that does not exist, and R-RULE-001's Note requires `--force --reason` for a transition a human reaches with no flag. R-AUDIT-0115 is the one row this milestone makes true rather than stale — it claims `cancel` carries the same sovereign rules as promote, which the missing call site falsifies today.

The two pins in `m0293_force_enforcement_surfaces_test.go` assert phrasing, so a row rewritten to cover every terminal edge keeps them green without checking that it did. AC-5's check is derived from the closed set instead.

The ratification burden falls on the `done` edge alone: every epic cancel in this repo's history was run by a human actor, so the audit's non-human predicate excludes them all.

The stress catalogue's `verb-sequence` walk is unaffected by the widening: it runs as a human actor, so it never reaches the gate, and its oracle already treats an FSM-legal transition refused by an orthogonal rule as legitimate. AC-6's own account of what the spec modelled beforehand is in its AC body.

## Out of scope

- Any change to the automatic scope-end at terminal promote.
- The identity substrate. The gate keys on a self-declared actor: an invocation that omits `--actor` inherits the human identity from `git config` and passes through. That property is shared with the shipped activation gate and is not addressed here.

## Work log

### AC-1 — Epic active → done gated at the verb

One closed-set entry, refusal proved at the binary seam with HEAD unmoved,
and a human control so the gate is scoped to the actor rather than the edge
· commit 15f6360e6 · A mutation probe found the actor predicate's
prefix boundary unconstrained — `HasPrefix(actor, "human/")` replaced by
`Contains(actor, "human")` left the whole suite green — and the pin that
closes it was watched red under the mutant and green on correct code.

### AC-2 — Both cancel edges gated at the verb, not the audit

Two closed-set entries plus the `cancel` call site that makes them
enforceable, placed ahead of the cascade guards · commit 811f208d9 · tests
5/5. The gate now takes the verb name, because a message reading `aiwf
promote` after `aiwf cancel` sends the operator to a command that did not
refuse them. Relocating the call after the cascade guards turns all three
refusal cases red, so the ordering is pinned rather than only commented.

### AC-3 — The act the widened audit reaches is ratified

`aiwf acknowledge illegal c030cb926` · commit ba39a9b2c. The phase ladder
records the kernel rule's own output rather than a Go test: `aiwf check`
reported the finding before the acknowledgment and reports none after. No
Go test exists for this AC, by the judgment recorded in D-0081.

Observation, re-runnable by a later reader:

| Field | Value |
|---|---|
| Command | `aiwf check` at the repo root |
| Environment | devcontainer (Linux), binary built from this branch via `make diag-aiwf` |
| Expected | no error-severity finding once the acknowledgment lands |
| Before | `1 findings (1 errors, 0 warnings)` — `fsm-history-consistent/forced-untrailered` on `c030cb92`, E-0029 `active → done` |
| After | `1 findings (0 errors, 1 warnings)` — the remaining warning is `provenance-untrailered-scope-undefined`, which reports that an unpushed branch has no upstream to audit against and clears on first push |

### AC-4 — The static audit sees the cancel spelling

The regex builder emits a cancel-form pattern per distinct id prefix in
the closed set · commit 3970cc754, narrowed at review in d48a4f586 ·
covered by the count-and-spelling assertion in
`TestSovereignActPromoteRegexes_TracksKernelClosedSet`.

It first restricted emission to cancel-reachable entries, deriving
reachability from `entity.CancelTarget`, with the builder's core extracted
so that rule could be driven with fabricated shape lists. The compression
lens at wrap costed it: against any closed set whose entries share one
kind, the rule produces byte-identical output, so it bought a partial
guarantee — over-match is still accepted for a kind with mixed edges — for
roughly seventy-five lines. Cut, with the residual over-match documented
where the emission happens.

### AC-5 — The catalogue names every sovereign transition

Six rows brought onto the widened set, checked by a test that derives the
expected transitions from the closed set · commit ec874f5f1 ·
watched failing on all four transitions beforehand and on a row with one
transition removed. Two defects predating this milestone were corrected in
passing: R-AUDIT-0050 cited a function that does not exist, and
R-RULE-001's Note required `--force --reason` for an edge a human reaches
with no flag.

### AC-6 — The legal-workflow spec models sovereignty

One cross-cutting `GlobalRules` entry, symbolic in the closed set rather
than enumerating it · commit e5ccbe809 · A mutation dropping
the `actor-role` precondition and downgrading the rejection layer to
check-time turns it red on both counts.

## Validation

Run on the milestone branch at the readiness pass, in the devcontainer
(Linux, `go test` unwrapped):

| Gate | Command | Result |
|---|---|---|
| Full CI-parity gate | `make ci` | exit 0; self-check passed (29 steps) |
| Full linter set | (within `make ci`) | `0 issues.` |
| Diff-scoped coverage | `AIWF_COVERAGE_BASE=epic/E-0090-… make coverage-gate` | `ok internal/policies` |
| Kernel check | `aiwf check` | 0 errors, 1 warning |

The one warning is `provenance-untrailered-scope-undefined`: a branch with
no upstream gives the provenance audit no range to walk. It clears on the
first push and is not a finding about this work.

`make ci` was run rather than `make check-fast` because this milestone
changed Go across four packages and the epic branch it merges into is
bound for a push.

## Decisions made during implementation

- D-0081 — the ratification's evidence is the shipped kernel rule plus the
  acknowledgment commit, not a policy test duplicating it. Measured: the
  duplicate costs 100 seconds, scoping the walk to the closed set's kinds
  recovers none of it, and the suite's wall time would go from roughly 40
  seconds to 140 on every push.

## Reviewer notes

**Recurring obligation.** Two standing rules land here, and both bind future
change rather than this one. `m0324_audit_catalogue_test.go` requires every
entry in the sovereign closed set to be named in the audit catalogue's
sovereign-acts section; its owner is whoever widens that set, nothing
retires it, and it carries an unstated second obligation — the section
heading string is now load-bearing, so renaming that section turns it red.
`m0324_spec_sovereignty_test.go` requires exactly one `GlobalRules` entry
carrying the sovereignty precondition, at a fixed outcome, layer and
strictness, citing an accepted record; its owner is whoever edits that
accessor, and **G-0649 retires it** — giving the refusal a finding code lets
the rule move into `Rules()` and be bound by the drift arms that already
cover every other illegal cell. The regex-builder assertions are behavior
pins rather than mandates: they cost nothing per future entry, because the
count is re-derived from the closed set.

**Deletions.** Two genuine retirements. `TestPromote_EpicActive_OtherTransitionsUnaffected`
pinned that the rule fired on `proposed → active` and no other epic
transition; ADR-0047 leaves that claim with no subject, since all four legal
epic transitions are now sovereign. Its three subtests and three rows of the
`IsSovereignActShape` false-cases table went with it. Kind-scoping survives
separately and is still a live claim. A third retirement came from review:
the cancel-reachability rule and the extraction built to constrain it.

**Same-outcome clusters.** One cluster of five — "a non-human actor is
refused at a sovereign epic edge" — across the verb layer and the seam. An
independent reviewer established distinctness by deleting each closed-set
entry singly and recording which tests died; no two members share a kill
set. The one overlap worth naming is that the verb-layer cancel test and the
seam's `from proposed` case both die on the guard-ordering mutation, leaving
the unit test's marginal value as its `res == nil` assertion and its
package's coverage. Both are real but small.

**Three lenses ran, not two.** Code-quality was sliced in two by concern.
Design-quality was skipped deliberately and the reason is worth recording,
since a later reader will notice its absence: `wf-rethink` triggers on a new
module boundary, core abstraction, or data model, and this milestone widened
an existing closed set, added a second consumer of an existing gate, and
added one row to an existing spec table. The third lens was a trial —
whether the same result could be had with materially less logic. It found a
helper duplicating one in the same package, two tests contributing zero
incremental coverage, and the reachability rule above; it also declined two
available cuts, citing this repo's seam-and-layer rule and its
comment-content mandate. Acting on it removed about a hundred lines net. A
gap filed on trunk proposes making that question a standing fourth item in
the wrap review's shape-measurement step.

**What the reviews found, and the pattern in it.** Both correctness lenses
returned request-changes. Between them the blocking findings were: an AC test
that compared predicate operators but never their values, so the spec rule
passed while inverted; two behaviours asserted in comments and pinned by
nothing; a vacuous subtest; a catalogue row understating the promote verb by
two edges; and a decision resting on a measurement that was wrong.

Every one of those is a claim that could only be read rather than run. The
claims a command could check — guard ordering, the ratification count, which
transitions are legal, the exit code, whether a human still succeeds on all
four edges — were right the first time and survived independent
re-derivation. That is the same asymmetry M-0323 recorded, holding a second
time, and it is now the strongest argument this epic has for keeping the
review independent rather than making it another pass by the author.

**The sharpest instance.** Two comments claimed the allow-rule refuses an
unauthorized non-human actor before the sovereign gate is consulted, and used
that to justify a fixture's design and a test's layer. Measured false: the
unforced gate returns before a plan exists, so it is reached with any actor
and no scope at all. The claim was true of the neighbouring `--force`
fixture, whose own doc comment says so correctly, and was carried across to a
case where force is absent and the reasoning does not hold.

**Declined, with the evidence.** `TestFSMHistoryConsistent_PerfBudget`
asserts a ten-second wall-clock budget and runs on every push. It was
observed failing at 10.29s during this milestone and passing at 4.82s alone
on the same machine minutes later, no code change between — it measures
runner load, which is the oracle shape the stress-lane guidance refuses for
scenarios. Not filed: one observation, and the fix is a real decision
(benchmark, subprocess-count assertion, or a different lane) rather than a
correction. Recorded here so the next reader meets a judgment rather than a
blank. This milestone doubled that fixture's findings from 50 to 100 by
widening the closed set; finding construction is small next to the walk and
the isolated run keeps roughly twofold headroom, so that is context, not
cause.

**Not fixed, deliberately.** `aiwfx-wrap-epic` does not mention that
`aiwf promote <epic> done` is now sovereign, while its sibling
`aiwfx-start-epic` documents the analogous activation in three places. The
ritual is unaffected in practice — it runs with no `--actor`, so identity
resolves to the operator's own — but an agent invoked under a delegated
scope can no longer close an epic and no shipped surface says so. Left for
the epic's own wrap, where the ritual surfaces are already in scope.

**A shipped error message changed.** The sovereign refusal read
`aiwf promote epic done: …`; it now names the verb the operator ran and both
states. Calling the gate from `cancel` made the old text name a command the
reader had not run, and with several edges sovereign the destination alone no
longer identifies which was refused. The exit code is unchanged, and no
consumer read the old text.

**Where a manual audit failed and a gate caught it.** The branch-coverage
walk is agent-performed, and mine cleared a refusal branch that no test
reached: the integration test covering it drives the binary as a subprocess,
so it contributes nothing to `internal/verb`'s coverage profile. The
diff-scoped gate named the line. Worth knowing that the seam-and-layer rule
has a coverage consequence, not only a correctness one.

## Deferrals

- G-0649 — the sovereign-act refusal carries no finding code, so AC-6's spec rule can be declared but not bound through the drift arms that pair every other illegal rule against the impl. Giving it a code changes the shipped activation gate's exit from 2 to 1, which needs its own decision.

## Dependencies

- M-0323 — produced ADR-0047, which rules the cancel edges sovereign and requires the call site to land with them.

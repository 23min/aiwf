---
id: M-0323
title: Settle what closing an epic is
status: in_progress
parent: E-0090
tdd: none
acs:
    - id: AC-1
      title: An accepted ADR exists and E-0090's ADRs produced section cites it by id
      status: met
    - id: AC-2
      title: E-0090's Open questions table routes every row to that ADR
      status: met
---
## Goal

Settle, in one ADR, what closing an epic is: who may declare it on each terminal edge, and how an operator-driven scope end coexists with the automatic one.

## Context

The kernel gates one sovereign transition, epic `proposed → active`, and neither edge that closes an epic. The gate is promote-only: `requireHumanActorForSovereignAct` has a single call site, so `aiwf cancel` never consults the closed set. Separately, a scope's only exit is the terminal promote or cancel of its own entity, which stamps `aiwf-scope-ends` as a side effect.

Three questions have to be answered before either code milestone can be built, and answering them apart would fork one decision across two records.

## Acceptance criteria

### AC-1 — An accepted ADR exists and E-0090's ADRs produced section cites it by id

The ADR resolves through the loader at status `accepted`, and the epic's `## ADRs produced` section names that id. The check is a relationship between two artifacts: it fails if the ADR is missing, is not accepted, or the epic cites a different id.

It deliberately does not assert that the ADR answers the questions *well*. That is content correctness over prose, which this repo holds at review; asserting it with a phrase match would pin a reading that rewording breaks.

### AC-2 — E-0090's Open questions table routes every row to that ADR

Every row of the epic's `## Open questions` table names the AC-1 ADR as its resolution path. Fails if a row still points elsewhere, or if a question was added after the ADR landed and left unrouted.

## Constraints

- The ADR records the choice, not the schedule for acting on it. No gate language in the body.
- The end-mode targeting answer must be consistent with the provenance model's existing rule that a verb under multiple active scopes picks the most-recently-opened.

## Design notes

The three questions carried from the epic:

1. Does the end mode target the most-recently-opened active scope, mirroring `--pause` / `--resume`, or every active scope on the entity, mirroring today's auto-end? G-0460 establishes that multiple simultaneously-active scopes are legal and intended, so the two existing answers genuinely disagree.
2. Is cancelling an epic a sovereign act? The epic's scope already commits to yes; the ADR records that and its reasoning.
3. What undoes an end? `ended` is terminal in the scope FSM and re-authorizing opens a fresh scope rather than reviving the old one — the ADR states whether that is the whole answer, per the kernel's what-undoes-this rule for a new verb surface.

ADR-0040 constrains the shape of any answer to question 2: prevention belongs at the verb route, so a widened closed set arrives with its call site rather than before it.

## Out of scope

- Any production-code change. The deliverable is a decision record; the only code that lands here is the policy tests pinning the two ACs.
- Whether the automatic scope-end survives. The epic already decided it does.

## Work log

### AC-1 — Produced ADR resolves and is accepted

ADR-0047 written and ratified; E-0090's `## ADRs produced` cites it. Pinned by a
relationship check that reads the id out of the epic rather than expecting a
literal · commit e457200c8 · tests 2/2, each watched failing on a nonexistent
ADR and on a real-but-unratified one before being accepted as evidence.

### AC-2 — Open questions route to that ADR

All three rows routed to ADR-0047, each carrying the answer in one clause. Same
commit; the check derives the expected id from `## ADRs produced`, so the two ACs
cannot drift apart · commit e457200c8 · tests 2/2, watched failing on a row
routed elsewhere and on an added row left unrouted.

## Decisions made during implementation

Four questions were settled, three from the epic's table and one found while
reading the code:

- Every edge into a terminal epic status is sovereign, `proposed → cancelled`
  included, because `cancelled` is terminal whatever state it is reached from.
- An end names its scope by authorize-commit SHA and defaults to the entity's
  sole candidate; more than one candidate and no name refuses, none at all
  refuses.
- Nothing undoes an end. The inverse is a fresh grant.
- Ending covers paused scopes as well as active ones. The automatic end's
  predicate collects only `active` today, so a paused scope survives its
  entity's closure permanently; ADR-0047 rules that a defect and M-0325 carries
  the one-predicate fix.

## Validation

Run on the milestone branch at the readiness pass, in the devcontainer (Linux,
`go build ./...` unwrapped):

| Gate | Command | Result |
|---|---|---|
| Tests + lint + vet | `make check-fast` | exit 0, zero `FAIL` |
| Full linter set | `make lint` | `0 issues.` |
| Diff-scoped coverage | `AIWF_COVERAGE_BASE=main make coverage-gate` | `ok internal/policies` |
| Kernel check | `aiwf check` | 0 errors, 1 warning |

The one warning is `provenance-untrailered-scope-undefined`: a freshly cut branch
has no upstream, so the provenance audit has no range to walk. It clears on the
first push and is not a finding about this work.

Both AC tests were watched failing before being accepted as evidence — a
nonexistent ADR id, a real but unratified one, a question row routed to another
ADR, and a row added with no route. Each produced a distinct message naming the
row or id at fault.

## Deferrals

None. The one defect found while reading the code — the automatic scope-end
collecting only `active` scopes, so a paused scope survives its entity's closure
— was not punted: ADR-0047 rules on it and M-0325 carries the fix as AC-4.

## Reviewer notes

**Recurring obligation.** This milestone adds two standing checks, and they are a
real ongoing cost rather than a one-off: every future edit to E-0090's body must
keep `## ADRs produced` citing an accepted ADR and every `## Open questions` row
routed to it. The obligation is narrow — it binds one entity, not a class of
future change — and nothing retires it, because the loader resolves archived
entities too, so the tests keep passing after the epic is swept rather than
needing deletion. Owner is E-0090.

**Deletions.** Nothing was retired. Nothing qualified: the milestone's output is a
decision record plus its evidence, and it touched no existing test.

**Same-outcome clusters.** Two tests, so no cluster of three or more to judge.
They fail for different reasons — one on the epic-to-ADR citation, one on the
question-table routing — and share only their fixture loader.

**A hole found in the ADR while writing the specs against it.** The end-mode rule
covered one candidate and many, but not zero. It now states that a bare `--end`
with no candidate refuses, matching what `--pause` already does when no scope
qualifies, while `--scope` naming an already-ended scope converges.

**Not asserted, deliberately.** Neither AC checks that ADR-0047 answers its three
questions *well*. That is content correctness over prose, which this repo holds at
review; a phrase assertion would pin one reading that any rewording breaks.

## Dependencies

- None. This is the epic's first milestone; both others depend on it.

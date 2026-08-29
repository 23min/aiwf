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
      status: open
      tdd_phase: green
    - id: AC-2
      title: Cancelling an epic with a non-human actor is refused at the verb, not the audit
      status: open
    - id: AC-3
      title: Every commit the widened audit fires on is ratified and check reports no error
      status: open
    - id: AC-4
      title: The static audit catches a scripted aiwf cancel of a sovereign edge
      status: open
    - id: AC-5
      title: The audit catalogue names every transition in the sovereign closed set
      status: open
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

### AC-4 — The static audit catches a scripted aiwf cancel of a sovereign edge

The audit keys its patterns on `(prefix, To)`, so it matches only the `aiwf promote <id> <to>` spelling. For an entry a human would reach with `aiwf cancel`, it emits a pattern for that spelling too, and fires on a line carrying it without `--force`. The qualifying entries are those where `entity.CancelTarget(s.Kind, s.From) == s.To`, so the pattern set stays derived from the closed set rather than enumerating spellings by hand.

Fails if a widened set adds a cancel-reachable entry whose spelling the audit cannot see.

### AC-5 — The audit catalogue names every transition in the sovereign closed set

Every entry in `entity.SovereignActShapes()` is named by the sovereign-acts section of `docs/design/legal-workflows-audit.md`, with the expectation derived from the closed set rather than written as a literal — so widening the set without touching the catalogue turns the check red and names the missing transition.

It pins that the transitions are *named*, not that what the rows say about them is true. R-RULE-001's Note is false in a way this check does not reach; content correctness in the catalogue stays held at review.

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

## Out of scope

- Any change to the automatic scope-end at terminal promote.
- The identity substrate. The gate keys on a self-declared actor: an invocation that omits `--actor` inherits the human identity from `git config` and passes through. That property is shared with the shipped activation gate and is not addressed here.

## Dependencies

- M-0323 — produced ADR-0047, which rules the cancel edges sovereign and requires the call site to land with them.

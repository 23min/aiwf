---
id: M-0324
title: Refuse a non-human actor on every terminal epic edge
status: draft
parent: E-0090
depends_on:
    - M-0323
tdd: required
acs:
    - id: AC-1
      title: Promoting an epic to done with a non-human actor is refused before any write
      status: open
    - id: AC-2
      title: Cancelling an epic with a non-human actor is refused at the verb, not the audit
      status: open
    - id: AC-3
      title: Every commit the widened audit fires on is ratified and check reports no error
      status: open
    - id: AC-4
      title: The static audit catches a scripted aiwf cancel of a sovereign edge
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

## Constraints

- Sovereign-act shape is a property over legal transitions (D-0008); every entry added is FSM-legal and `TestSovereignActShapes_AllFSMLegal` stays green.
- A closed-set entry and the call site that enforces it land together (ADR-0040). Neither edge is added ahead of its verb-time refusal.
- `--force` stays human-only; the coherence guard at `verb.Apply` is not modified.
- The ratification is a sovereign act, so it is performed by a human and cannot be delegated.

## Design notes

The refusal message names only the human-run path. Offering `--force` there would be wrong every time it appeared: the message is reachable only for a non-human actor, and `verb.Apply` refuses that actor's force trailer anyway.

Two other consumers read the same closed set and widen automatically — the history audit in `internal/check/fsm_history_consistent.go`, and the static audit in `internal/policies/aiwf_promote_epic_active_audit.go` that builds one regex per entry. Neither needs an edit to pick up the new entries.

The static audit does need a decision, though, because widening the set exposes an asymmetry in what it scans. Its pattern is `aiwf\s+promote\s+<prefix>\S+\s+<to>`, so after the widening it catches `aiwf promote E-NNNN cancelled` in automation-shaped source and misses `aiwf cancel E-NNNN` — the natural spelling, and the very route this milestone adds a call site for. Either extend the pattern to the cancel form or record the gap in this milestone's Out of scope; silently inheriting it is the one option to refuse.

Three rows of `docs/design/legal-workflows-audit.md` scope sovereignty to epic activation and go stale the moment three entries join the set: R-AUDIT-0050 (line 139) describes the static audit as scanning for `aiwf promote E-<id> active`; R-RULE-001 (line 543) notes `proposed → active` is the sovereign-act edge; R-RULE-078 (line 640) is titled "Epic activate". Only R-RULE-078 is pinned at all, in `m0293_force_enforcement_surfaces_test.go`, and that pin asserts its phrasing rather than its coverage. R-AUDIT-0050 and R-RULE-001 carry no pin, so nothing turns red when any of the three stops being true.

The ratification burden falls on the `done` edge alone: every epic cancel in this repo's history was run by a human actor, so the audit's non-human predicate excludes them all.

## Out of scope

- Any change to the automatic scope-end at terminal promote.
- The identity substrate. The gate keys on a self-declared actor: an invocation that omits `--actor` inherits the human identity from `git config` and passes through. That property is shared with the shipped activation gate and is not addressed here.

## Dependencies

- M-0323 — produced ADR-0047, which rules the cancel edges sovereign and requires the call site to land with them.

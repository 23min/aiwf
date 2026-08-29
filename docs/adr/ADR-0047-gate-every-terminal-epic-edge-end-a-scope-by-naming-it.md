---
id: ADR-0047
title: Gate every terminal epic edge; end a scope by naming it
status: proposed
---
## Context

Two surfaces meet here, because both answer the same question: who may declare
that authorized work is over.

The kernel gates one sovereign transition — epic `proposed → active`, the single
entry in `sovereignActShapes`. No edge *into* a terminal epic status is gated.
Measured in a fixture repo against an `active` epic under a live delegated scope,
`aiwf promote <epic> done --actor ai/claude --principal human/fixture` exits 0,
flips the status, writes no force trailer, and leaves a tree `aiwf check` reports
clean. Commit `c030cb926` is such an act, already in this repo's history.

The epic FSM offers three edges into a terminal status: `proposed → cancelled`,
`active → done`, and `active → cancelled`. All three are irreversible. `done` and
`cancelled` have no outgoing edges, and the kernel answers an unwanted terminal
status with a new entity rather than a transition back.

The gate is also promote-only. `requireHumanActorForSovereignAct` has a single
call site, in `internal/verb/promote.go`, so `aiwf cancel` never consults the
closed set. The history audit that backstops the gate does not share that
limitation: it walks the file diff, comparing an entity's `status:` field at a
commit against its parent, so it observes a cancel exactly as it observes a
promote.

Separately, an authorization scope has one exit — the terminal promote or cancel
of its own entity, which stamps `aiwf-scope-ends:` as a side effect of the status
change. `AuthorizeMode` is a closed three-value set (open, pause, resume) with no
end, so withdrawing a delegation requires closing the work. G-0022 reserved a
trailer slot for a revoke verb that was never built.

The two mechanisms that already act on a scope address their targets differently.
`aiwf-scope-ends:` names a scope by its authorize-commit SHA and may repeat on one
commit to end several. A pause or resume names nothing: the commit records only
`aiwf-scope: paused`, and `ReplayScopes` re-derives the target as the
most-recently-opened scope in the matching state. Pause can afford that, because a
scope paused in error is restored by a resume.

## Decision

### Every edge into a terminal epic status is a sovereign act

`proposed → cancelled`, `active → done`, and `active → cancelled` all require a
`human/` actor, joining `proposed → active` in `sovereignActShapes`.

Sovereignty tracks irreversibility, not effort. Cancelling a *proposed* epic
discards a plan that never became work, which invites treating it as the lesser
act — but `cancelled` is terminal whatever state it is reached from, so that
cancel is exactly as unrecoverable as one from `active`. Gating all three gives
the closed set a single reading: only a human puts an epic into a terminal
status. It also makes the existing activation gate coherent rather than merely
stricter, since an agent already cannot activate a proposed epic and now cannot
dispose of one either.

Milestones share the same terminal shape and are deliberately not included. An
epic is the unit a scope is opened on and whose closure ends delegation; a
milestone is work inside a scope someone already holds. Extending sovereignty
there is a separate decision needing its own authorizing record.

### The gate widens only alongside the call site that enforces it

`aiwf cancel` consults `requireHumanActorForSovereignAct` before writing. A
closed-set entry without that call site would leave the verb silent while the
history audit fired on the landed commit — refusal after the act, which is the
record ADR-0040 exists to prevent.

### An operator ends a scope by naming it, and `--end` defaults to the only candidate

`aiwf authorize <id> --end` gains `--scope <auth-sha>`, a peer of `--to`,
`--pause` and `--resume`. It ends the scope named by `--scope`; with no `--scope`
it ends the entity's sole non-ended scope; with more than one candidate and no
`--scope` it refuses, listing them.

Ending is terminal, so the target must be unambiguous. Copying pause's
convention — pick the most-recently-opened and record nothing — would convert a
re-derivation error into an unrecoverable one, and pause's safety net does not
exist here. Ending every candidate instead would borrow the automatic end's
justification without its premise: that end covers everything because the entity
reached a terminal status and no delegation on it can continue, which is a
complete argument at the entity level. An operator ending a delegation on a
living entity makes a narrower claim, and "all" coincides with it only while
there is exactly one candidate.

Defaulting to the sole candidate is not a guess: with one candidate it is the
only resolution available. The default therefore costs nothing in precision and
removes the lookup from the common path.

The end commit carries the same `aiwf-scope-ends: <auth-sha>` the automatic end
writes, so the scope replay is unchanged. An operator end stays distinguishable
in history because it rides an `aiwf-verb: authorize` commit rather than a
`promote` or `cancel`; no additional trailer records what the verb already does.

### Ending covers every non-ended scope, paused ones included

Both the operator end and the automatic end target scopes in `active` *or*
`paused` state. `paused → ended` is legal in the scope FSM, and a delegation left
paused on a closed entity is stranded rather than finished — its own FSM permits
the exit that nothing fires.

### Nothing undoes an end

`ended` is terminal. The inverse of an end is a fresh grant: `aiwf authorize <id>
--to <agent>` opens a new scope with a new authorize-commit SHA, and the ended
scope's SHA stays valid forever as the `aiwf-authorized-by:` reference on commits
made under it. Re-authorizing therefore restores the delegation without
resurrecting the record of the one that ended.

## Consequences

Widening `sovereignActShapes` widens two consumers automatically, because both
derive from the closed set: the history audit in
`internal/check/fsm_history_consistent.go` and the static audit in
`internal/policies/` that builds one regex per entry.

That widening reaches commits already in the log. On the `done` edge, the audit
fires on any epic closed by a non-human actor without a force trailer; `c030cb926`
is the one such commit at the time of writing, and it is ratified by a human with
a written reason in the milestone that widens the set. The two cancel edges reach
nothing: every epic cancel in this repo's history was run by a human actor, so the
audit's predicate excludes them all. The count of qualifying commits is a function
of when the set widens, not a fixed number — deferring the gate can only raise it.

`--force` remains human-only and the coherence guard at `verb.Apply` is unchanged.
A sovereign-act refusal names the human-run path only: offering `--force` there
would be wrong every time it appeared, since the message is reachable only for a
non-human actor whose force trailer `verb.Apply` refuses anyway.

The automatic end's predicate changes from active-only to non-ended, which is a
correctness fix rather than a behaviour change anyone relies on: no scope has ever
been paused in this repo, so no tree carries the stranded state the old predicate
produced.

## Validation

The `--scope` disambiguation exists because more than one non-ended scope per
entity is reachable. G-0460 asks whether that state should be legal at all. If it
resolves toward one live scope per entity and the kernel refuses to create a
second, the refusal branch becomes unreachable and this decision's targeting rule
should be revisited — the default would then be the whole behaviour, and `--scope`
a guard for a state nothing can produce.

## References

- Related ADRs: `ADR-0040` (prevention at the verb route, ratification at the
  history route), `ADR-0038` (the two-seam placement reasoning this follows).
- aiwf decisions: `D-0008` (sovereign-act shape is a property over legal
  transitions, never below them).
- Linked epics and milestones: `E-0090`, `M-0323`, `M-0324`, `M-0325`.
- Gaps: `G-0646` (both closing edges ungated, gate reaches only promote), `G-0022`
  (the reserved revoke trailer slot), `G-0460` (repeat authorize leaves two active
  scopes), `G-0111` (scope end welded to the terminal promote).
- `docs/design/provenance-model.md` §"Multiple parallel scopes" — a human may hold
  several active scopes at once, and the kernel resolves a verb against the
  most-recently-opened.

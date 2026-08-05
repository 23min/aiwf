---
id: M-0292
title: Give provenance-force-non-human a ratification path
status: in_progress
parent: E-0079
tdd: required
acs:
    - id: AC-1
      title: Acknowledging a forced non-human commit clears its finding
      status: open
      tdd_phase: done
    - id: AC-2
      title: The acknowledged commit is unchanged and its reason is readable
      status: open
      tdd_phase: green
---

## Goal

Let a human clear the finding a forced non-human act leaves behind — through a
verb, with the reason recorded — instead of rewriting history.

## Context

The rule walks git history and fires at error severity, so the commit blocks the
push. `aiwf acknowledge illegal` does not clear it: the rule is absent from the
consumer roster the acknowledgement mechanism reads. What remains is re-authoring
the commit or amending a human actor into it — history rewrites, in a repo whose
tooling is policed against them.

This is needed whether or not M-0291 lands. That milestone closes the verb route,
but the rule fires on commits no verb produced, so the history route keeps
producing findings that nothing can clear.

## Acceptance criteria

### AC-1 — Acknowledging a forced non-human commit clears its finding

After the acknowledgement, the finding for that commit is gone on the next check
run.

An unacknowledged sibling in the same tree still reports. Without that half, a
passing test cannot distinguish an acknowledgement keyed on the commit from the
rule having been switched off — which is the failure mode worth guarding, since
the epic forbids clearing the finding by weakening it.

The roster joined must be the one that drives enforcement, not the one that
enumerates the rules for readers. Two rosters exist and the chokepoint checks
them for different properties, so joining the wrong one passes the policy while
changing nothing an operator can observe.

Evidence: a real-repo test acknowledging one forced commit with a second left
unacknowledged, asserting exactly one finding survives.

### AC-2 — The acknowledged commit is unchanged and its reason is readable

Ratification records; it does not rewrite. The acknowledged commit is
byte-identical afterwards — same tree, same parents, same trailers — and the
reason is readable from the acknowledging commit itself, matching how the
existing acknowledge verbs already behave.

Evidence: the acknowledged SHA compared before and after, plus the reason read
back out of the acknowledging commit.

## Constraints

- The ratification path is human-only and carries a written reason.
- `provenance-force-non-human` stays at error severity. This milestone adds a way
  to clear the finding, never a way to ignore it.

## Design notes

- **An acknowledgement is keyed on the commit, not on the rule.** It therefore
  covers every rule consuming the roster for that SHA. Accepted as-is: an
  acknowledgement is a judgment about a commit, which is what the existing
  consumers already assume. Revisit only if someone needs to accept one rule
  while another still blocks — recorded as a non-blocking open question on the
  epic.

## Surfaces touched

- `internal/check/acks.go` — the consumer roster.

## Out of scope

- The verb-route wiring (M-0291).

## Dependencies

- None. Independent of M-0291 and can land in either order.


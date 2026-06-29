---
id: M-0213
title: Opt-in best-effort fetch before id allocation
status: draft
parent: E-0052
tdd: required
acs:
    - id: AC-1
      title: aiwf add --fetch refreshes the trunk ref before allocating
      status: open
      tdd_phase: red
    - id: AC-2
      title: The fetch is best-effort and never blocks the add
      status: open
      tdd_phase: red
---
## Goal

Add an opt-in, best-effort refresh of the trunk-tracking ref immediately before
allocation, so `max` is computed against the freshest published trunk. A session
that has not fetched recently allocates against a stale `refs/remotes/origin/...`
view and can hand back an id that already landed upstream; an opt-in fetch
narrows that window (class 2 of G-0272's taxonomy).

Best-effort is load-bearing: a fetch failure (offline, no remote, network error)
must degrade to current local-only allocation with a warning — never block or
fail the add. The fetch narrows the window; it does not close it (another machine
can publish between the fetch and the commit — that residual is G-0274's to
cure). Surfaced as a `--fetch` flag on `aiwf add`; a `doctor` staleness nudge is
a possible complement, deferred.

Source: G-0273. Parent epic E-0052.

### AC-1 — aiwf add --fetch refreshes the trunk ref before allocating

`aiwf add <kind> --fetch` refreshes the configured trunk ref (only that ref, not
a full `fetch --all`) before computing `max`, so an id that landed on trunk since
the last local fetch is seen and skipped.

Evidence: a test with a local clone whose trunk ref is advanced out-of-band — the
`--fetch` allocation reflects the upstream id; the same allocation without
`--fetch` does not.

### AC-2 — The fetch is best-effort and never blocks the add

The fetch is best-effort and never blocks the add: a failure (no remote, an
unreachable origin, a network error) degrades to local-only allocation with a
warning and a success exit, identical to today's behavior.

Evidence: a no-remote repo where `aiwf add --fetch` succeeds, emits a warning,
and allocates against the local view.


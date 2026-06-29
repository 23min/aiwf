---
id: M-0213
title: Opt-in best-effort fetch before id allocation
status: draft
parent: E-0052
tdd: required
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

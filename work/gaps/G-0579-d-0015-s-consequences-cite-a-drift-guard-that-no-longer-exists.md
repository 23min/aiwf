---
id: G-0579
title: D-0015's consequences cite a drift guard that no longer exists
status: open
---
## What's missing

D-0015 is `accepted`, and its Consequences state that the embedded skill bodies are
not rewritten to point at the materialized template path because they are "a
drift-checked verbatim snapshot of upstream (M-0148's
`TestRituals_VendoredMatchesUpstream`), so editing them would fail the drift guard."

The named test no longer exists — a repo-wide search for the symbol returns nothing.
The reason it gave has lapsed twice over: ADR-0016 retired the upstream channel and
made the embedded snapshot canonical, and G-0345 rewrote those skill bodies to cite
the materialized path.

## Why it matters

An accepted decision reads as current truth. A reader deciding whether a shipped
skill body may cite the templates directory is told it may not, and pointed at a
guard to verify that against which does not exist.

The Decision section itself is still correct — the four templates do materialize
where it says. Only the Consequences have gone stale. Whether they are corrected in
place or the decision is superseded is the open question, and it is a real one: an
accepted decision's Consequences are part of the record, not a scratch field.

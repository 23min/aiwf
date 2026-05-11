---
id: G-0114
title: 'HTML render gap surface: status and archive state not glanceable from sidebar'
status: open
---
## What's missing

The rendered HTML site's gap surface doesn't make two distinctions visible at the level a reader naturally scans from. The page-level sidebar — the navigation that runs down every page and surfaces epics and milestones — does not include gaps; a reader looking for "what's currently broken or missing?" has to find the small "Browse by kind" block at the bottom of `index.html`, then pick between `gaps.html` (non-archived) and `gaps-all.html` (everything) via a tiny `all` sub-link. From the sidebar's perspective the active-vs-archived distinction is invisible: a reader can't tell at a glance which subset they're looking at, and the second view lives one click away with no in-sidebar cue that two views even exist. Within `gaps.html` itself the per-row status badges (`open`, `addressed`) are present in the markup but don't function as a glanceable organizer — open and addressed rows sit equally-weighted in a flat table with small text-badges, so a reader skimming for "what's actually in flight right now" has to scan the full list rather than pick up status from layout or color.

## Why it matters

Gaps are one of the project's primary current-state surfaces — the place a reader goes to answer "what's broken or missing today?" If using the surface for that question costs more attention than the answer warrants, the reader falls back to grep or tunes it out, and the rendered site loses one of its most useful functions for its main consumer (the maintainer and anyone reviewing project health). The status-distinction half (open vs. addressed at a glance) is the high-value signal: a reader skimming wants the in-flight subset to pop, not to scan a flat list of equally-weighted rows. The active-vs-archived half compounds this — landing on `gaps-all.html` by accident produces a 112-row page that looks like the same surface but isn't, and nothing in the sidebar or page chrome distinguishes the two views or makes the filtered default's existence obvious. Both halves land on the same workflow loss: a reader trying to use gaps for current-state synthesis gets less signal than the underlying data supports.

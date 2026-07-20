---
id: D-0045
title: Duplicate the empty-repo git guard in entityview instead of importing cliutil
status: proposed
relates_to:
    - M-0272
---
# D-0045 — Duplicate the empty-repo git guard in entityview instead of importing cliutil

> **Date:** 2026-07-20 · **Decided by:** human/peter (via ai/claude)

## Question

`entityview.ReadHistoryChain` (moved from `internal/cli/history`) needs the
same empty-repo guard `history.go` used before moving: convert `git log`'s
"your current branch X does not have any commits yet" error into a clean
`(nil, nil)`. That guard already exists as `cliutil.HasCommits` — a
four-line `git rev-parse --verify HEAD` wrapper. Should `entityview` import
`internal/cli/cliutil` to reuse it, or carry its own copy?

## Decision

`entityview` carries a small private `hasCommits` duplicate of
`cliutil.HasCommits` rather than importing `cliutil`.

## Reasoning

`cliutil` is a single Go package: importing any one function from it pulls
in the whole package's dependency closure, and `cliutil` genuinely imports
`spf13/cobra` (via `completion.go`, `outputformat.go`) for its
flag-completion helpers. M-0272/AC-1's whole point is that `entityview` is
free of `internal/cli/*` — reusable without dragging Cobra along — so an
import of `cliutil`, even for one four-line helper, would silently reopen
exactly the coupling the milestone closes. The alternative of moving
`HasCommits` itself into `entityview` and having `cliutil.HasCommits`
delegate to it was considered and rejected as disproportionate to this
milestone's declared "mechanical only: import-path changes... no API
redesign" scope — `cliutil.HasCommits` has seven call sites across
`status`, `authorize`, `check`, and `history`, none of which this milestone
otherwise touches; relocating its canonical home would ripple beyond the
entityview extraction for no behavioral gain. A four-line, single-purpose
git-plumbing check duplicated once is cheaper than either of those options.

## Consequences

If `cliutil.HasCommits`'s behavior ever needs to change, `entityview`'s
private copy needs the same edit — a comment on `entityview.hasCommits`
points back to this decision so a future editor doesn't miss the pairing.

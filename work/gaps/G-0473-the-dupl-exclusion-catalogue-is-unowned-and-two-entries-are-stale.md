---
id: G-0473
title: The dupl exclusion catalogue is unowned and two entries are stale
status: open
---
## What's missing

`.golangci.yml` grandfathers eight production files from `dupl`, under a comment
that calls each one real duplication and says it is "tracked for follow-up triage
rather than fixed as part of this patch." The gap doing that tracking, G-0423, is
`addressed` and archived. The deferral survives; its owner does not.

Two of the eight entries are also stale. A whole-tree `dupl` run at threshold 100
with the exclusions lifted reports no clone in `internal/check/acs.go` or
`internal/cli/check/check.go`. Whatever duplication earned them a place on the
list is gone.

The remaining six are live: `internal/aiwfyaml/aiwfyaml.go`,
`internal/aiwfyaml/hooks.go`, `internal/cli/contract/recipes.go`,
`internal/cli/contract/unbind.go`, `internal/config/config.go` and
`internal/initrepo/initrepo.go`.

## Why it matters

A stale entry is not tidy-up. The exclusions are file-scoped, and the config says
why in its own comment — so "a future clone elsewhere in the same file would also
go unflagged until revisited." Each stale entry is therefore an open blind spot in
a tripwire, sized to a whole file, protecting nothing.

The unowned half matters differently. The comment tells a reader that the debt is
tracked, and a reader who checks finds a closed gap. That is worse than an
untracked exclusion honestly labelled, because it spends the reader's trust: the
next person to wonder whether these clones are known will conclude they are, and
stop looking.

The exclusion list is also the only inventory of acknowledged duplication in the
repo. Read as a catalogue rather than as build configuration, it is the answer to
"what did we already decide to defer" — which is exactly what it stops being once
its entries no longer correspond to clones and its owner no longer exists.

## Options

1. **Drop the two stale entries, and reopen tracking for the six that remain** —
   either by reopening G-0423's concern under a new gap or by pointing the config
   comment at the gap that supersedes it. Smallest change, restores both the
   tripwire's coverage and the comment's honesty.
2. **Clear the list by collapsing the clones**, so the exclusions can be deleted
   rather than re-owned. Correct end state, but it is a larger body of work and
   the exclusions stay wrong until it lands.
3. **Narrow the exclusions from file scope to line ranges** so a new clone
   elsewhere in the same file still fires. Removes the blind spot without
   resolving the duplication, and costs a re-anchoring every time those files
   move.

Option 1 is the lean and should land regardless — it is small, and options 2 and 3
both leave the comment's false claim standing unless it is fixed separately. Option 2
is the right destination and is tracked separately as the clone-collapse work; this
gap should not wait on it.

## Scope

Surfaced by a `wf-structural-sweep` pass after E-0073 wrapped, reading the
exclusion list as an inventory rather than as pass/fail.

Measurement note for whoever picks this up: golangci-lint truncates at
`max-issues-per-linter` (default 50) and `max-same-issues` (default 3). A run
without `--max-issues-per-linter 0 --max-same-issues 0` reports a fraction of the
clones and makes more entries look stale than actually are.

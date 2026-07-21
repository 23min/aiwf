---
id: G-0429
title: Collapse duplicated history/scope read tail in show.go's view builders
status: addressed
discovered_in: M-0269
addressed_by_commit:
    - ff7d9dbd
---
## What's missing

`BuildShowView` and `BuildCompositeShowView` (`internal/cli/show/show.go`)
each carry an identical ~10-line tail: read history, propagate an error,
read scopes, propagate an error (with the same `//coverage:ignore`
rationale duplicated verbatim), assign both onto the view, then run
`check.Run` and `filterFindingsByID`. The duplication predates M-0269 (the
old `if err == nil` swallow was already copy-pasted between the two
functions); M-0269/AC-2's fail-loud fix faithfully mirrored the existing
shape into both call sites rather than introducing new duplication, but
the shared tail is now a clear candidate for a small helper — e.g.
`finishShowView(ctx, root, id, historyLimit, t, loadErrs, view, parent)`
— that both functions call.

## Why it matters

Two independent copies of the same error-propagation logic (including
the same explanatory `//coverage:ignore` comment) is exactly the
per-verb-pair duplication class this epic's M-0270 milestone targets
("mechanical housekeeping: the shared-seam collapses"). A future change
to the history/scope read shape (or its coverage-ignore rationale) has
to remember to land in both places.

---
id: G-0336
title: IDGrepAlternation docstring example uses below-floor milestone id
status: addressed
addressed_by_commit:
    - 7eebec78
---
## Problem

The docstring on `entity.IDGrepAlternation`
(`internal/entity/canonicalize.go:86-88`) gives a worked example the code
cannot produce:

```
// Concretely, an input of `E-22` returns `(E-0*22)` (any width that
// equals 22 numerically); `E-0022` returns the same. `M-22/AC-1`
// returns `(M-0*22)/AC-1`.
```

The `E-22` / `E-0022` examples are correct. The `M-22/AC-1` example is
wrong: the code returns `M-22/AC-1` **verbatim**, not `(M-0*22)/AC-1`.

## Why

`M-22` is below the milestone id floor. The composite grammar is
`compositeIDPattern = ^(M-\d{3,})/(AC-\d+)$` (`internal/entity/entity.go:227`)
— the milestone parent requires **≥3 digits**. `M-22` has two, so
`ParseCompositeID("M-22/AC-1")` returns `ok=false`, the composite branch in
`IDGrepAlternation` is never taken, and the input falls through to the
verbatim `regexp.QuoteMeta(id)` return. `M-22` is not a valid milestone id in
the first place — the narrowest legacy milestone is `M-001`. The docstring
reused the `E-22` numeral (valid for epics, which allow `\d{2,}`) for a
milestone, where two digits is below-floor.

Empirically:

```
IDGrepAlternation("M-22/AC-1")   = "M-22/AC-1"        (verbatim)
IDGrepAlternation("M-221/AC-1")  = "(M-0*221)/AC-1"
IDGrepAlternation("M-0221/AC-1") = "(M-0*221)/AC-1"
ParseCompositeID("M-22/AC-1")    -> ok=false
```

## Why this matters

The test suite already pins the correct opposite behavior, so the docstring
contradicts the contract its own package tests enforce:

- `internal/entity/canonicalize_test.go` —
  `{"composite-below-floor-passthrough", "M-22/AC-1", "M-22/AC-1"}`.
- `internal/entity/canonicalize_test.go` — the `IDGrepAlternation` edge-case
  test special-cases `M-22/AC-1` with the comment "compositeIDPattern
  requires `M-\d{3,}` so `M-22` is still [passthrough]".

Per the kernel principle "kernel functionality must be AI-discoverable"
through docstrings, a worked example the code cannot produce mismodels the
below-floor behavior for any reader (human or AI) who trusts it. No runtime
behavior is wrong; this is a documentation defect only.

## Fix

Replace the `M-22/AC-1` example with a valid milestone composite that
exercises the same width-tolerant composite path — e.g. `M-221/AC-1` returns
`(M-0*221)/AC-1`, or `M-007/AC-1` returns `(M-0*7)/AC-1` — optionally noting
that a below-floor parent (`M-22/AC-1`) passes through unchanged, matching
the pinned test.

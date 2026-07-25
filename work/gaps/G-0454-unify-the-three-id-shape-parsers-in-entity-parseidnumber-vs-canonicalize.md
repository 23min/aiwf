---
id: G-0454
title: Unify the three id-shape parsers in entity (parseIDNumber vs canonicalize)
status: open
---
## What's missing

Three independent "strip the kind prefix, `strconv.Atoi` the numeric tail" id parsers live in `internal/entity`: `parseIDNumber` (`allocate.go`) and two spots inside `Canonicalize` / `IDGrepAlternation` (`canonicalize.go`).

## Why it matters

The same id-shape parse is written three times — convergent duplication a future id-grammar change must touch in three places. Split out of the G-0447 mechanical sweep because the copies are **not** identical: `parseIDNumber` takes a *known* kind and strips its prefix, while `canonicalize.go` *discovers* the kind by iterating `idPrefix` and validates the tail against `idPatterns` before reformatting. A shared helper must reconcile the assumes-kind vs discovers-kind asymmetry (likely a `splitID(id) (kind, num, ok)` primitive that the assumes-kind caller wraps), not a mechanical swap.

## Resolution shape

Extract one id-shape primitive that discovers-and-validates, then express `parseIDNumber` (known kind) as a thin specialization. Pin the primitive's behavior once (valid ids across every kind, narrow-width legacy ids, non-id strings that happen to start with a kind prefix) before deleting the copies against it.

## Where to fix

- `internal/entity/allocate.go` — `parseIDNumber`.
- `internal/entity/canonicalize.go` — the prefix-strip + `Atoi` inside `Canonicalize` and `IDGrepAlternation`.

## Related

- G-0447 — the convergent-duplication cleanup this was split from (seam 5b).

---
id: G-0447
title: 'Convergent-duplication tax: shared scaffolds unextracted or bypassed'
status: addressed
addressed_by_commit:
    - ccf877b0
---
## What's missing

A shared-scaffold seam is missing (or, where it exists, bypassed) across the core packages: the same *job* is retyped as textually-divergent code many times over — the convergence-aware duplication `dupl`'s exact-clone matcher cannot see. This gap has been worked in independently-shippable seams; what remains is the largest one, in `internal/cli`.

**Remaining scope — the `internal/cli` diagnostic/prelude scaffolds:**

- **`cliutil.BeginVerbDiag`** — an ~11-line diagnostic-logging wiring block (`ResolveLogger` → `defer closeDiagLog` → run-id/`WithVerb` setup → `defer EmitVerbOutcome`) is copy-pasted into ~30 verbs; it also forces `log/slog` + `os` + `logger` imports into every verb solely to host the block. Missing helper: `cliutil.BeginVerbDiag(...)` returning a `finish` closure that captures the named `code`/`sha` returns. Highest-LOC item in the whole gap.
- **The `ResolveRoot` → `ResolveActor` prelude** (with identical usage-error arm) is duplicated across ~23 verbs.

**Both remaining items land in `internal/cli/cliutil`, which G-0227 wants to split into `cliidentity`/`clioutput`/`cligitstate`/`cliflagsupport`. Sequencing with G-0227 must be decided before this seam is cut** (extract into today's `cliutil` and let G-0227's gofmt-aware rewrites relocate the helpers, or split cliutil first so they land in their final home).

## Progress — landed seams

Six sub-seams have shipped as independent `wf-patch`es, each behavior-preserving and independently reviewed:

- **Trailer-triple → `standardTrailers`** (verb) — merge `5ae9a8ea`.
- **Read-verb JSON envelope → `OKEnvelope`/`FindingsEnvelope`/`ErrorEnvelope`** (cli) — merge `9d2d496b`. (This was the second of the three original `internal/cli` items; the envelope duplication is fully resolved.)
- **Single-entity write tail → `planEntityWrite`** (verb) — merge `13a2e575`.
- **Closed-set membership scans → `slices.Contains`** (entity) — merge `bfb475f4`.
- **Milestone-rule preamble → `eachActiveMilestone`** + merged the two terminal-incomplete-AC rules (check) — merge `da60059f`.
- **Area FS-preamble → `areaFS`** across `AreaDeadGlob`/`AreaOverlap` (check) — merge `4c8f4c16`.

## Deferred — split into their own gaps

Three sub-seams were split out because each carries a decision or risk rather than being a mechanical, behavior-preserving sweep:

- **G-0453** — unify the `shortHash`/`short` SHA-abbreviation helpers (a displayed-width decision, not behavior-preserving).
- **G-0454** — unify the three entity id-shape parsers (the assumes-kind vs discovers-kind asymmetry).
- **G-0455** — consolidate the `body.go` markdown heading-walkers (highest risk; evaluate-first, possibly won't-do).

## Related

- `wf-codebase-health` — the ritual that surfaced this (A2 low-coupling, C1 single-source-of-truth).
- G-0227 — the layering/cohesion refactor that splits `cliutil`; the remaining seam must be sequenced against it.
- G-0423 — enabled `dupl`; this gap is the textually-divergent duplication class `dupl` structurally cannot catch.
- G-0453 / G-0454 / G-0455 — the deferred sub-seams split from this gap.

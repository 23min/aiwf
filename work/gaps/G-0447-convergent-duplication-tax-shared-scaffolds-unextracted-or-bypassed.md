---
id: G-0447
title: 'Convergent-duplication tax: shared scaffolds unextracted or bypassed'
status: open
---
## What's missing

A shared-scaffold seam is missing (or, where it exists, bypassed) across the four core packages. The same *job* is retyped as textually-divergent code many times over — the class of duplication `dupl`'s exact-clone matcher cannot see, surfaced instead by a convergence-aware code-health pass. Concentrations:

**`internal/cli` — the largest surface:**
- An ~11-line diagnostic-logging wiring block (`ResolveLogger` → `defer closeDiagLog` → run-id/`WithVerb` setup → `defer EmitVerbOutcome`) is copy-pasted into ~22 verbs — e.g. `promote/promote.go:164`, `cancel/cancel.go:104`, `move/move.go:87`, `add/add.go:187`, `archive/archive.go:144`, `authorize/authorize.go:228`, `setpriority/setpriority.go:127`. ~240 LOC of pure convergence; it also forces `log/slog` + `os` + `logger` imports into every verb solely to host the block. Missing helper: `cliutil.BeginVerbDiag(...)` returning a `finish` closure that captures the named `code`/`sha` returns.
- The read-verb JSON envelope (`render.Envelope{Tool:"aiwf", Version:version.Current().Version, ...}`) is hand-assembled at 11+ sites — `schema/schema.go:86`, `check/check.go:337`, `history/history.go:163`, `show/show.go:149`, `list/list.go:229` (+ status/template/worktree/contract). Mutating verbs already funnel through `outputformat.go`'s `emitSuccess`/`emitFindings`; the read side never got the equivalent constructor.
- The `ResolveRoot` → `ResolveActor` prelude (with identical usage-error arm) is duplicated across ~23 verbs.

**`internal/verb`:**
- The single-field frontmatter writer is reimplemented rather than shared: `setpriority.go:56` ↔ `setarea.go:58` (SetPriority's own doc comment calls itself "the write-surface sibling of SetArea"), and `milestone_depends_on.go:74` ↔ `milestone_tdd.go:70`. The `readBody → Serialize → projectReplace → projectionFindings → single-OpWrite plan → Metadata` tail recurs verbatim across ~8 verbs. Missing helper: `singleEntityFieldWrite(...)`.
- The trailer triple `{verb, Canonicalize(entity), actor}` already has a shared constructor (`standardTrailers`, `ac.go:409`) but is reimplemented inline at `setpriority.go:110`, `setarea.go:143`, `milestone_tdd.go:92`, `milestone_depends_on.go:100`, and duplicated as a second named copy `editBodyTrailers` (`editbody.go:179`) — one audit-trail fact with ≥3 sources of truth.

**`internal/check`:**
- The milestone `kind + IsArchivedPath + status` filter preamble is retyped ~6× (`acs.go:299/426/471/527`, `milestone_tdd_undeclared.go:40`); `milestoneDoneIncompleteACs` (`acs.go:299`) and `milestoneCancelledIncompleteACs` (`acs.go:471`) are near-byte-identical, differing only in the status compared and the finding code. Missing: an `eachActiveMilestone(t, status)` iterator.
- The area FS-preamble (`t.Root==""` guard → `os.DirFS` → iterate areas×globs through `areamatch`) is copied across `area_dead_glob.go:43`, `area_overlap.go:35`, `area_coverage.go:80`.
- Two gratuitously-different SHA-abbreviation helpers: `shortHash` (8 chars, `fsm_history_consistent.go:430`) and `short` (7 chars, `provenance.go:885`).

**`internal/entity`:**
- Five copies of the closed-set membership scan (`IsAllowedStatus`/`IsAllowedACStatus`/`IsAllowedTDDPhase`/`IsAllowedTDDPolicy`/`IsAllowedPriorityLevel`, `entity.go:119/144/168/194/228`) plus the same shape in `IsValidAreaValue` and two transition scans — all replaceable by `slices.Contains` (already the repo idiom in `verb/promote.go`).
- Three independent "strip kind prefix, `strconv.Atoi` the tail" id parsers (`allocate.go:88`, and inside `Canonicalize`/`IDGrepAlternation` in `canonicalize.go`).
- Three-to-four near-identical markdown heading-walk state machines (`body.go:29/91/211`, plus `SectionLineBounds` at `:155`).

## Why it matters

Two costs. **Drift:** a change to any of these contracts — the diagnostic-log event shape, the output envelope's `Tool`/`Version` pair, the trailer triple, the milestone-filter guard — must be made in N places, and nothing mechanically ties the copies together. The trailer-triple case is the sharpest: an audit-trail schema change today requires editing ≥3 sites, and a missed one silently emits a divergent trail. **Signal:** ~several-hundred LOC of scaffolding obscures the ~lines of actual per-verb logic, which is the legibility tax `wf-codebase-health` A2/C1 exist to prevent.

The notable sub-pattern is *bypass*, not just *absence*: `standardTrailers` and the mutating-verb envelope builder already exist and are the correct seam — several call sites simply don't route through them. Those are the cheapest wins (route, don't design).

## Resolution shape

Independently-shippable per seam; no single big-bang refactor. Rough leverage order:
1. `cliutil.BeginVerbDiag` — collapses ~240 LOC and the `slog`/`os`/`logger` import churn across ~22 verbs; single point to evolve the run-id/outcome contract.
2. A read-verb envelope constructor mirroring `outputformat.go` — removes the 11+ hand-built `render.Envelope{...}`.
3. `singleEntityFieldWrite` in `verb`, and route `standardTrailers` everywhere (fold `editBodyTrailers` in).
4. `eachActiveMilestone` iterator + shared area FS-preamble in `check`; merge the two near-identical AC rules.
5. `slices.Contains` sweep + one id-shape parser in `entity`.

Each extraction carries the repo's branch-coverage obligation for the new shared unit; pin the shared helper's behavior once, delete the copies against it.

## Where to fix

`internal/cli/cliutil/` (new `BeginVerbDiag`, read-envelope constructor), `internal/verb/` (`singleEntityFieldWrite`, `standardTrailers` routing), `internal/check/` (`eachActiveMilestone`, area-preamble helper), `internal/entity/` (`slices.Contains`, unified id parser, shared section-walker).

## Related

- `wf-codebase-health` — the ritual that surfaced this (A2 low-coupling, C1 single-source-of-truth).
- G-0423 — enabled `dupl`; this gap is the textually-divergent duplication class `dupl` structurally cannot catch.

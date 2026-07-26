---
id: G-0227
title: Relocate LoadEntityScopes to internal/scope + Options-struct adapters
status: open
priority: medium
---
## What's missing

One genuine layering fix, plus one contained readability cleanup. The regression guard this gap was named around — `internal/policies/layering_direction.go`, the AST policy that pins the canonical import direction — has already landed, so the load-bearing piece is banked; what remains is targeted cleanup, not a refactor.

1. **Relocate `internal/cli/cliutil/scopes.go`'s `LoadEntityScopes` to `internal/scope/`.** This is a real upward-import inversion, not mere drift: two *lower-layer* packages — `internal/entityview/scopeguard.go` and `internal/cellcoverage/authorized_scope.go` — call it, reaching up into the cli-support layer for a domain helper that walks entity history. It's exactly why the layering policy carries `internal/cellcoverage` as its one allowlisted exception. Move the function down into `internal/scope/` (the package already exists; 7 non-test callers), then **delete the `internal/cellcoverage` allowlist entry** in `layering_direction.go` — turning a documented exception into a clean rule. That deletion is the mechanical evidence the relocation worked.
2. **Adopt an `Options`-struct at the `cli/<verb>/Run(...)` boundary** for the verbs with long positional signatures — `list.Run` takes 11 positional params, `authorize.Run` 10, `cancel.Run` 8 — mirroring what `internal/verb/` already does with `PromoteOptions` / `AddOptions`. A transpose-two-strings hazard today; one struct per verb, ~one caller each. Independent per-verb polish, do the ones that are in the way.

## Out of scope

**No cliutil package split.** The original scope proposed cleaving `internal/cli/cliutil/` into four sub-packages (`cliidentity` / `clioutput` / `cligitstate` / `cliflagsupport`). The premise doesn't hold: the package doc names ~nine responsibilities, not a clean four-way seam, and the 26 files are already single-responsibility per file (`actor.go`, `lock.go`, `completion.go`, …). The "grab-bag" smell is aesthetic. Against it: 244 files import cliutil, so a split rewrites all of them and leaves each caller importing several sub-packages instead of one — worse ergonomics for no coupling reduction (they're all the same layer). With the layering policy already guarding direction, there's no chokepoint pressure forcing this, and it's refactor-for-its-own-sake against Strong-verdict code.

**No `resolver.go` file-split.** Splitting `internal/cli/render/resolver.go` (~785 lines, 24 methods on one `Resolver`) into per-page sub-files is pure file-organization — the type stays one type. Cosmetic; not worth the churn.

## Why it matters

The A1–A3 verdicts were all Strong; the concrete residue worth acting on is the `scopes.go` upward import, because it's the one item that measurably improves the layering the gap is named after and lets the policy shed its lone exception. The positional-param structs are a modest safety win. The two dropped items are aesthetic reorganizations the landed layering policy already makes unnecessary to force.

## Source

`docs/archive/pocv3/health-scorecard-2026-06-04.md` §A1 (move: relocate the drifted helper), §A3 (move: Options-struct at the adapter boundary).

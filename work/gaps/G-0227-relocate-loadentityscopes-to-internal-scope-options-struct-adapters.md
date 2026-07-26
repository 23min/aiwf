---
id: G-0227
title: Relocate LoadEntityScopes to internal/scope + Options-struct adapters
status: open
priority: medium
---
## What's missing

One contained readability cleanup. The regression guard this gap was named around — `internal/policies/layering_direction.go`, the AST policy that pins the canonical import direction — has already landed, so the load-bearing piece is banked and the layering is clean today. What remains is targeted polish, not a refactor.

**Adopt an `Options`-struct at the `cli/<verb>/Run(...)` boundary** for the verbs with long positional signatures — `list.Run` takes 11 positional params, `authorize.Run` 10, `cancel.Run` 8 — mirroring what `internal/verb/` already does with `PromoteOptions` / `AddOptions`. A transpose-two-strings hazard today; one struct per verb, ~one caller each. Independent per-verb polish, do the ones that are in the way.

## Out of scope

**No `LoadEntityScopes` relocation.** The original scope proposed moving `internal/cli/cliutil/scopes.go`'s `LoadEntityScopes` down into `internal/scope/` and deleting the `internal/cellcoverage` allowlist entry in `layering_direction.go` as the mechanical evidence. Neither half survives contact with the tree:

- **No live upward import to fix.** `internal/entityview` — named as the second below-tier caller — references `LoadEntityScopes` only in comments; it does not import `cliutil`. The sole real below-tier caller is `internal/cellcoverage`, which is test-only and already allowlisted; every other caller (`cli/show`, `cli/authorize`, `cliutil/provenance`) is same-tier and legal. The layering policy is green as-is.
- **The allowlist entry can't be shed.** `internal/cellcoverage` is exempt because it imports `internal/verb` (tier 2) for fixture construction — wholly independent of `LoadEntityScopes`. Relocating the helper leaves that import in place, so the entry must stay; deleting it would only make the policy report `cellcoverage` as untiered. The promised mechanical evidence does not exist.
- **The move isn't cohesion-cheap.** `LoadEntityScopes` is entangled with a cluster in `scopes.go` (`ReplayScopes`, `OpenersFrom`, `CommitTrailers`, `readEntityScopeCommits`) plus `HasCommits` from `cliutil/gitstate.go`, all consumed by `render`'s single pass and `show` as `cliutil.<fn>`. Moving only `LoadEntityScopes` introduces a *new* `scope → cliutil` upward edge; moving the whole cluster and resolving the `HasCommits` dependency is a multi-package refactor for a cohesion-only gain, with no mechanical evidence to show for it. If ever worth doing, it is milestone-shaped, not part of this gap.

**No cliutil package split.** The original scope also proposed cleaving `internal/cli/cliutil/` into four sub-packages (`cliidentity` / `clioutput` / `cligitstate` / `cliflagsupport`). The premise doesn't hold: the package doc names ~nine responsibilities, not a clean four-way seam, and the 26 files are already single-responsibility per file (`actor.go`, `lock.go`, `completion.go`, …). The "grab-bag" smell is aesthetic. Against it: 244 files import cliutil, so a split rewrites all of them and leaves each caller importing several sub-packages instead of one — worse ergonomics for no coupling reduction (they're all the same layer). With the layering policy already guarding direction, there's no chokepoint pressure forcing this, and it's refactor-for-its-own-sake against Strong-verdict code.

**No `resolver.go` file-split.** Splitting `internal/cli/render/resolver.go` (~785 lines, 24 methods on one `Resolver`) into per-page sub-files is pure file-organization — the type stays one type. Cosmetic; not worth the churn.

## Why it matters

The A1–A3 verdicts were all Strong; with the layering policy already guarding import direction, the only concrete residue worth acting on is the positional-param structs — a modest safety win at the CLI adapter boundary. The relocation and the two file-organization items are cohesion/aesthetic reorganizations the landed policy makes unnecessary to force.

## Source

`docs/archive/pocv3/health-scorecard-2026-06-04.md` §A1 (move: relocate the drifted helper), §A3 (move: Options-struct at the adapter boundary).

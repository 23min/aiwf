---
id: G-0227
title: 'Layering & cohesion refactor: cliutil split + Options-struct adoption + policy'
status: open
---
## What's missing

A consolidated refactor of the Cobra-adapter ring that (a) splits the `internal/cli/cliutil/` grab-bag along its own package-doc fault lines, (b) extends the `internal/verb/` `Options`-struct pattern outward one layer to the `cli/<verb>/Run(...)` adapters that still carry 8–10 positional flag parameters, (c) relocates the one domain-shaped helper that drifted upward, and (d) lands a kernel policy test that pins the layering direction mechanically so the next sideways/upward import gets caught at CI time rather than at the next audit.

Specific work, file-level:

1. **Split `internal/cli/cliutil/`** into four focused packages along the seams its own package-doc already names: `internal/cli/cliidentity/` (actor/principal resolution), `internal/cli/clioutput/` (formatter, envelope helpers), `internal/cli/cligitstate/` (lock, repo-state helpers), `internal/cli/cliflagsupport/` (completion, annotation helpers). Importers update via gofmt-aware rewrites.
2. **Relocate `internal/cli/cliutil/scopes.go`'s `LoadEntityScopes`** to `internal/scope/history.go` — it's domain-shaped (walks entity history) and currently sits one layer above the domain it operates on.
3. **Adopt `Options`-struct at `cli/<verb>/Run(...)` boundary** for the four verbs the scorecard named (`list`, `cancel`, `authorize`, `milestone`) — mirrors what `internal/verb/` already does with `PromoteOptions` / `ContractBindOptions` / `AddOptions`. One level deep; not a wholesale rewrite of every cmd.
4. **Consider splitting `internal/cli/render/resolver.go`** (785 lines, 24 methods on one `Resolver` type) into per-page sub-files (`resolver_epic.go`, `resolver_milestone.go`, `resolver_entity.go`, `resolver_index.go`). One file per page-shape; the type stays one type.
5. **Add `internal/policies/layering_direction.go`** — an AST-level policy test that asserts the canonical import direction: `cmd → cli → verb → check|render|htmlrender|initrepo → tree|scope|trunk|contractcheck → entity|gitops|aiwfyaml|config → codes|pathutil`. Any upward or sideways import outside a documented allowlist (`internal/cellcoverage` is the existing exception) fails CI. The `internal/cellcoverage` exception is allowlisted by name + rationale.

## Why it matters

The A1 / A2 / A3 verdicts were all Strong but the adversarial passes named the cliutil grab-bag, the positional-param shape, and the `scopes.go` latent inversion as concentrated mid-layer smell. Today nothing in CI tells the next contributor "you broke layering" — the property exists by reviewer vigilance and prior-discipline momentum. The layering policy test is the load-bearing piece: cleanup without a chokepoint reverts.

## Source

`docs/pocv3/health-scorecard-2026-06-04.md` §A1 (recommended moves 1–3), §A2 (move 2: codify layering doctrine), §A3 (moves 1–2).

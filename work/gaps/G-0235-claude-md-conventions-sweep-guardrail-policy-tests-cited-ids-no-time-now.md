---
id: G-0235
title: CLAUDE.md conventions sweep + guardrail policy tests (cited-ids, no-time-now)
status: open
---
## What's missing

Two parallel groups: (a) conventions practiced in code that haven't been written down in CLAUDE.md yet, and (b) the small set of remaining policy tests that don't fit naturally under one of the other cluster gaps.

### CLAUDE.md additions

- **The `raw*` decoder shim pattern** (`internal/recipe/recipe.go:137`, `internal/aiwfyaml/aiwfyaml.go:275`) — formalize as a documented convention. Today reviewers re-derive it per site.
- **The recurring "shares memory with the package-level constant" phrase** — lift into a documented convention under §"Type design" or §"Naming."
- **Altitude taxonomy explicit in §Testing.** Document unit / module-boundary / binary subprocess / browser e2e with one sentence + one canonical example per altitude. CLAUDE.md mentions seam-testing inline; the altitude taxonomy isn't explicit.
- **`D-NNNN` vs ADR axis explicit.** Today the distinction is conventional (ADR = architectural, long-lived; `D-NNNN` = project-scoped). Document the predicate so future authors know which to reach for.
- **OpDelete-absence invariant** — document explicitly in `internal/verb/verb.go`'s `Plan` / `FileOp` doc comment that the absence is the design choice, not an oversight.
- **Catalogue "derived artifacts"** (STATUS.md, ROADMAP.md, rendered HTML site, embedded ritual snapshot) somewhere AI-discoverable — currently a reader has to reason from architecture to identify which files are derived.

### Guardrail policy tests

- **`internal/policies/cited_entity_ids_resolve.go`** — every `ADR-NNNN` / `D-NNNN` / `G-NNNN` / `M-NNNN` / `E-NNNN` cited in a Go-source comment must resolve via the loader. Today `TestM0123_AC6_RuleDecisionSourcesResolve` does this for legal-workflow rules but not for arbitrary citations.
- **`internal/policies/no_time_now_in_core.go`** — mirrors `no_timestamp_manipulation.go` but for `time.Now()` / `time.Since()` / `time.Until()` calls in `internal/verb`, `internal/entity`, `internal/check`, `internal/gitops`, `internal/htmlrender`. Forces clock injection at the edge.
- **`internal/policies/validate_check_is_never_writes.go`** — extends `PolicyVerbsValidateThenWrite` from the verb naming family to the `Validate*` / `Is*` / `Check*` / `Has*` families across `internal/*`. Asserts these never call `os.WriteFile` / `os.Create` / `os.Remove*` / `gitops` write primitives. Catches "the helper named `IsValid` quietly writes a cache file" before it ships.
- **`internal/policies/cache_invalidation_documented.go`** — for any cache-mutation API in the kernel, asserts an adjacent documented invalidation rule (comment block, doc reference). Niche but C1-relevant; deferrable.

Plus the **clock-injection refactor** that the no-time-now policy implies for the one current violation: `internal/cli/status/status.go:326`'s `BuildStatus` stamps `Date: time.Now().UTC()`. Inject the clock (parameter or `Clock interface`) so the policy can land without exempting BuildStatus.

## Why it matters

Documentation drift (code knows the convention, CLAUDE.md doesn't say it) is the single failure mode CLAUDE.md was built to prevent — the doc itself says *"kernel functionality must be AI-discoverable."* Each missing convention is one place a new reader (human or LLM) re-derives a discipline that already exists.

The four policy tests are the leftover P1–P8 set after the others fold into their natural-parent cluster gaps (layering → G-α; envelope structural → G-ζ; NoOp invariant → G-δ; DOM-structural → G-η). These four don't fit any other cluster.

## Source

`docs/pocv3/health-scorecard-2026-06-04.md` §B2 (move 2), §B3 (move 2), §D4 (move 1), §F3 (moves 1–2), §G2 (move 2), §C1 (moves 2–3), §F1 (move 2), §G1 (moves 1–2).

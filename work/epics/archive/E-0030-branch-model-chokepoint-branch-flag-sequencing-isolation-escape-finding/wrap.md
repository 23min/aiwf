# Epic wrap — E-0030

**Date:** 2026-06-05
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0030-branch-model-chokepoint-branch-flag-sequencing-isolation-escape-finding
**Merge commit:** *filled at step 5 — see wrap-bundle commit chain*

## Milestones delivered

- M-0102 — aiwf authorize --branch flag + scope-branch trailer coupling (merged `d7ebe082`)
- M-0103 — AI-side preflight: aiwf authorize refuses without ritual branch context (merged `3c131beb`)
- M-0104 — aiwfx-start-epic sequencing fix (closes G-0116) (merged `e673d8d1`)
- M-0105 — aiwfx-start-milestone sequencing alignment (merged `7198feee`)
- M-0106 — Kernel finding isolation-escape (closes G-0099) (merged `afc19709`)
- M-0158 — Layer-4 branch-choreography spec cells + drift-policy extension (merged `fd54acf5`)
- M-0159 — Real-world hardening of branch-model chokepoint (merged `fb9c716f`)
- M-0160 — Operational pain: reallocate stress, trunk-collision regress, apply rollback (merged `13b89bcf`)
- M-0161 — Imagination-driven hardening: shallow, force-push, rename, detached, trunk (merged `1ea44526`)
- M-0162 — Layer-4 spec-catalog refactor: bijection + Pin registry (merged `5b2f1d0b`)

## Summary

E-0030 closed the branch-model chokepoint: ritualized work on `epic/E-*` and `milestone/M-*` branches with author iteration on `main` (ADR-0010), enforced via the `aiwf authorize --branch` coupling between scope and branch, the kernel's `isolation-escape` finding for AI-actor commits that leave their scope's bound branch, and a layer-4 branch-choreography spec catalog whose 129 cells are mechanically pinned to their tests by a bijection meta-test. The epic delivered both the conceptual model (ADR-0010 — the load-bearing decision the rest of the framework now rests on) and the mechanical enforcement: 6 new check-time kernel findings + 2 new verb-time typed errors, plus the static + runtime bijection chokepoint architecture. No new top-level verbs were introduced; `aiwf authorize` gained a `--branch` flag + `aiwf-branch` / `aiwf-branch-sha` trailers, and `aiwf acknowledge-illegal` (pre-existing from M-0136/E-0033) gained two additional silencing targets via the shared `aiwf-force-for: <sha>` trailer mechanism. Scope shifted twice mid-flight: a swap of the AC-2/AC-3 ordering in M-0162 to land Pin-registry-before-cell-expansion (B1 fix at reviewer pass), and a deferral of M-0161/AC-9's catalog refactor scope to M-0162 via D-0022.

## ADRs ratified

- ADR-0010 — Branch model: ritualized work on branches, author iteration on main
- ADR-0011 — Legal-workflow spec methodology (the catalog approach M-0158 then realized)
- ADR-0012 — Typed coded-error pattern for legality-pertinent verb refusals
- ADR-0013 — Represent a global precondition; classify out-of-scope as legality

## Decisions captured

- D-0018 — `branch-not-found` subsumed by `rung-pair-illegal`; catalog cleanup deferred to AC-9 (M-0102 era)
- D-0019 — Oracle partial-coverage: fail-shut on correctness, fail-open on coverage (M-0161/AC-3)
- D-0020 — M-0161/AC-5 cell-5 orphan-acknowledgment deferred to verb extension (G-0226)
- D-0021 — M-0161/AC-7 doctor JSON envelope deferred; substring match ships now (G-0070)
- D-0022 — M-0161/AC-9 deferred to follow-up milestone; M-0161 wraps 8/9
- D-0023 — M-0162/AC-3 cell expansion deferred for reallocate_scenarios_test.go
- D-0024 — M-0162/AC-4 bijection split architecture: static AST + runtime post-hook

## Follow-ups carried forward

E-0030 discharged everything in its scope. The following gaps surfaced during the epic but were either explicit out-of-scope (future epic) or operator-discipline residues that don't block any current chokepoint:

- G-0211 — Combinatorial verb-composition scenarios untested at branch-choreography E2E (next E2E hardening pass)
- G-0212 — Data-loss audit for verb composition across kernel surface (future epic)
- G-0213 — Cellcoverage fixture writes fictional `aiwf-branch` values (latent landmine)
- G-0215 — Kernel-wide audit: production nil-arg passes need structural chokepoint
- G-0218 — Operator-typed commit messages bypass `aiwf-verb` registry at composition
- G-0224 — `aiwfx-start-epic` / `aiwfx-start-milestone` SKILL.md cites retired `branch-not-found` code (doc-debt; was always going to be paired with D-0018's catalog cleanup)
- G-0225 — Legacy scopes lack `aiwf-branch-sha` trailer; rename triggers false positive (cleanly named carve-out from M-0161/AC-6)
- G-0226 — `aiwf acknowledge-illegal` hard-requires SHA reachable from HEAD (paired with D-0020's deferral)

Gaps inherited from prior epics that remain open across the kernel surface (not E-0030 responsibility): G-0200, G-0201, G-0202, G-0203, G-0204, G-0205, G-0206, G-0207, G-0209, G-0216, G-0217, G-0220, G-0222. These were either *originally surfaced* by E-0030 work but explicitly out of the epic's enforcement scope, or *partially addressed* by E-0030 milestones with residual carve-outs.

## Handoff

**Ready for the next epic:**

- Branch-choreography catalog is mechanically bijection-enforced at CI time. New ACs adding rules to the kernel that produce `ClassBranchChoreography` codes must register a cell in `internal/workflows/spec/branch/rules.go` and a Pin call site in their test, or AC-4 invariant 1/2 fires.
- `branchtest.Pin(cellID, testName)` registry under `//go:build testpins` is the single Pin chokepoint for any future cell-expansion. The bijection check + the allowlist verification test mechanically catch rename/delete/scanner-coverage regressions.
- ADR-0010 is the foundation for any future branch-discipline work. ADR-0011's methodology (cell catalog as the spec-table form) is now load-bearing across the kernel.
- The 6 new check-time findings (`isolation-escape`, `isolation-escape-oracle-failure`, `isolation-escape-shallow-clone`, `isolation-escape-orphaned-ai-commit`, `id-rename-untrailered`, `promote-on-wrong-branch`) + 2 new verb-time typed errors (`branch-context-required`, `rung-pair-illegal`) are all production-ready under typed `ClassBranchChoreography` discipline. The `aiwf authorize --branch` flag and the extended `aiwf acknowledge-illegal` silencing reach are wired through the same surface.

**Deliberately left open:**

- Combinatorial E2E expansion (G-0211, G-0212) is its own scope and will be larger than E-0030 was. Recommended as its own epic when capacity exists.
- Operator-discipline gaps (G-0218, G-0224 doc-debt, G-0225 legacy-scopes carve-out, G-0226 verb-reach extension) are bounded and low-risk; pick them up opportunistically.
- The named-cell allowlist's "primary test lives in X" claims are mechanically existence-checked but not semantically verified — fundamentally non-structural per the R1-T4 honesty correction at M-0162 wrap.

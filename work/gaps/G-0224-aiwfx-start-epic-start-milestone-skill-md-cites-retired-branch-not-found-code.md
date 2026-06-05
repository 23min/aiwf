---
id: G-0224
title: aiwfx-start-epic/start-milestone SKILL.md cites retired branch-not-found code
status: open
prior_ids:
    - G-0222
discovered_in: M-0161
---
## What's stale

The embedded ritual skills `aiwfx-start-epic` and `aiwfx-start-milestone` reference the verb error code `branch-not-found` in their SKILL.md bodies as a refusal code an operator might encounter:

- [`internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md`](../../internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md) line ~95
- [`internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md`](../../internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md) line ~81

After M-0161/AC-2 (G-0201) landed at commit `51462d52`, the verb-layer authorize carve-out no longer constructs `PreflightBranchNotFoundError`. The single rung-pair check (`PreflightRungPairError`, code `rung-pair-illegal`) subsumes the prior `branch-not-found` semantics for the AI-target + explicit-`--branch` path. Operators following the skills will see `rung-pair-illegal` in those scenarios, not `branch-not-found`.

The skill text is now imprecise — not technically broken (the skills still work), but the named refusal code doesn't match what the kernel emits today. Per CLAUDE.md "Kernel functionality must be AI-discoverable," skill content should name the codes operators will actually see.

## Why parked

This is reviewer-pass nit-level documentation drift. The skill content was correct at write time; AC-2 shifted the boundary. Cleanest sweep point is alongside the broader spec-table cleanup that M-0161/AC-9 (G-0210) covers — see [D-0018](D-0018-branch-not-found-subsumed-by-rung-pair-illegal-catalog-cleanup-defers-to-ac-9.md) for the architectural decision.

Per the M-0161/AC-2 reviewer pass (subagent, 2026-06-04), this nit was the only one of three reviewer-flagged nits worth formal tracking — the other two (test-helper local-map tautology, fixture comment refinement) are vanishingly small and stay in the reviewer transcript record without kernel-side mechanical tracking.

## Fix shape

Replace the `branch-not-found` mentions in both SKILL.md files with `rung-pair-illegal` (and / or both codes, with a note that `branch-not-found` is dead at the emission site). Either:

1. **Minimum**: substitute `branch-not-found` → `rung-pair-illegal` in the two SKILL.md bodies.
2. **Better**: rewrite the surrounding sentence to name the rung-pair semantic so the operator understands WHY the code names a rung-pair issue (e.g., "if the carve-out's rung-pair predicate refuses, you'll see `rung-pair-illegal` naming both branches' rungs").

Both versions land in the same commit as the M-0161/AC-9 SKILL.md-touching work (if AC-9 ends up editing the skills), or as a small `wf-patch` if AC-9 keeps to the spec-table only.

## Out of scope

- The dead `PreflightBranchNotFoundError` type and `CodePreflightBranchNotFound` constant in [`internal/verb/authorize.go`](../../internal/verb/authorize.go) — D-0018 covers retention rationale.
- The stale spec-table cells (`GlobalRules()` `branch-not-found` rule, `branch-cell-2` `ExpectedErrorCode`, `internal/policies/m0158_ac2_corner_cells_test.go` keyword mapping) — D-0018 + M-0161/AC-9 catalog refactor cover those.

## Discovered in

M-0161 — flagged by the M-0161/AC-2 reviewer subagent pass at the AC-2 wrap (2026-06-04), specifically nit-1 in the reviewer findings.

## Closing this gap

When the skill bodies are updated and verified against the live verb code paths:

1. The two SKILL.md mentions of `branch-not-found` are either replaced with `rung-pair-illegal` or rewritten with the broader rung-pair framing.
2. A structural test under `internal/policies/` (per CLAUDE.md "Ritual content authoring" + G-0220's discipline note) pins the new wording in the relevant skill body section.
3. Promote G-0224 to `addressed --by M-NNNN` referencing the cleanup commit / milestone.

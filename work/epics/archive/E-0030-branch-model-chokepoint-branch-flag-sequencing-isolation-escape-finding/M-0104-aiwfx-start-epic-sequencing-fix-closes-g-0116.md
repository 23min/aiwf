---
id: M-0104
title: aiwfx-start-epic sequencing fix (closes G-0116)
status: done
parent: E-0030
depends_on:
    - M-0102
    - M-0103
tdd: required
acs:
    - id: AC-1
      title: Embedded snapshot reflects new step ordering
      status: met
      tdd_phase: done
    - id: AC-2
      title: Stale G-0059 paragraph removed; replacement names ADR-0010
      status: met
      tdd_phase: done
    - id: AC-3
      title: Workflow headings structurally appear in new order
      status: met
      tdd_phase: done
    - id: AC-4
      title: Preflight accepts --branch <future> from main (future-branch refinement)
      status: met
      tdd_phase: done
    - id: AC-5
      title: Skill body names --force --reason override at appropriate step
      status: met
      tdd_phase: done
---
## Goal

Reorder `aiwfx-start-epic` so the sovereign promote (`aiwf promote E-NNNN active`) and authorize (`aiwf authorize E-NNNN --to ai/<id> --branch epic/E-NNNN-<slug>`) commits fire on `main` *before* the worktree/branch is cut. Retire the stale "G-0059 frames the open question of which branch-model convention aiwf should bless" paragraph at step 6 — ADR-0010 is the answer. Closes [G-0116](../../gaps/G-0116-aiwfx-start-epic-creates-worktree-before-promote-authorize-on-trunk-based-repos.md).

## Context

G-0116 documented the sequencing inversion in today's `aiwfx-start-epic`: step 5 (worktree placement) precedes step 8 (sovereign promote) and step 9 (optional authorize). With M-0103's preflight active, the existing ordering would *fail* — the worktree-first cut hits the preflight before any ritual branch context exists.

This milestone fixes the ordering so the ritual works with the chokepoint. It also implements [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s sequencing rule for opening an epic: state-announcement commits on main, *then* branch cut, *then* implementation work on the branch.

Ritual content edits land at the canonical authoring location per [ADR-0014](../../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) and [ADR-0016](../../../docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md):

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md` — the embedded snapshot that `aiwf init` / `aiwf update` materializes into the consumer repo's `.claude/skills/`. Per [G-0182](../../gaps/archive/G-0182-consolidate-testdata-ritual-fixtures-onto-the-embedded-snapshot-dedupe.md), this is also the path the per-AC content-assertion tests read (via `aiwfxStartEpicFixturePath` in `internal/policies/aiwfx_start_epic_test.go`).

One edit in one commit. The upstream `ai-workflow-rituals` repo was archived under ADR-0016 — no cross-repo coordination, no `make sync-rituals`, no `rituals.lock` to refresh.

## Pre-decided design

- **New step ordering** (the reorder):
  1. Preflight (existing steps 1–4).
  2. Delegation prompt (Q&A — promoted earlier so the operator's choice is known *before* the sovereign acts).
  3. Sovereign promote on `main` (or parent branch): `aiwf promote E-NNNN active`.
  4. *(if delegating)* Sovereign authorize on `main`: `aiwf authorize E-NNNN --to ai/<id> --branch epic/E-NNNN-<slug>`. The branch flag is required by M-0103's preflight; the named branch does not yet exist at this point — it is allowed because preflight's "branch exists" check is suppressed when both (a) the current checkout is `main` AND (b) the `--branch` value parses as a ritual shape per `internal/branchparse/` (`epic/`/`milestone/`/`patch/` + a canonical id). Implementation note: M-0103's preflight is refined here — see the carve-out at `internal/verb/authorize.go`. Ritual-shape (not arbitrary "valid ref name") is the stricter test the implementation adopted; without it the gate would become a no-op for any string under `--branch` from `main`. The cell for this is added to the consolidation milestone (M-0158); the carve-out's guard against the looser reading is `TestAuthorize_Open_AITarget_MainPlusNonRitualMissingBranch_Refuses`.
  5. Worktree placement + branch creation (Q&A). The operator picks an in-repo or sibling worktree, the branch is cut against the existing authorize trailer's binding.
  6. Hand-off.
- **Retire the G-0059 paragraph at the original step 6** — replace with a one-line *"per ADR-0010 §"Decision", the operator stays on `main` for the sovereign acts (steps 3–4) and the epic branch is cut afterwards at step 5."*.
- **Step 4's authorize-with-future-branch refinement** is a small extension to M-0103's preflight, not a separate milestone. The cell that proves it is "AI-actor authorize with `--branch epic/E-NNNN-<slug>` on a branch that doesn't yet exist, AND operator is on `main`" → preflight accepts (rationale: this is the ritual's well-formed pattern; cutting the branch is step 5's deliverable). Without this refinement, the ritual could not work — the authorize step would fail M-0103's "branch exists" check.

## Out of scope

- `aiwfx-start-milestone` (M-0105 — symmetric fix one level down).
- Kernel finding for post-hoc detection (M-0106).
- Spec-cell consolidation (the consolidation milestone).
- Other ritual surfaces beyond `aiwfx-start-epic`.

## Dependencies

- **M-0102** — the `--branch` flag and `internal/branchparse/` the new ordering invokes.
- **M-0103** — the preflight that makes the ordering necessary. M-0103's "branch exists OR --branch parses as a ritual shape AND current checkout is on `main`" refinement is *part of* this milestone's deliverable, not a separate prerequisite (the refinement is small enough to land alongside the ritual edit; cell coverage lives in the consolidation milestone).

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0104` time. AC seed set:
1. The embedded snapshot at `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md` reflects the new step ordering: preflight → delegation prompt → sovereign promote → sovereign authorize (if delegating) → worktree placement → hand-off.
2. The stale "G-0059 frames the open question" paragraph at the original step 6 is removed; the replacement names ADR-0010 explicitly.
3. The skill's `## Workflow` section's headings, parsed structurally (per CLAUDE.md §"Substring assertions are not structural assertions"), appear in the order specified above. A flat substring match is not sufficient — the assertion is structural.
4. M-0103's preflight accepts `aiwf authorize E-NNNN --to ai/<id> --branch epic/E-NNNN-<slug>` from a checkout on `main` even when the named branch doesn't yet exist, provided `--branch` parses as a ritual shape per `internal/branchparse/`. This is the "future branch" refinement; the cell is registered in the consolidation milestone. (The implementation tightened "valid ref name" from the seed wording to "ritual shape" to keep the gate from becoming a no-op for any string from main; rationale and guard test recorded in pre-decided-design point 4 above.)
5. The skill's "Workflow" prose names the override path (`--force --reason "..."`) at the appropriate step so an operator reading the skill body sees it.
-->

### AC-1 — Embedded snapshot reflects new step ordering

The embedded snapshot at [`internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md`](../../../internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md) carries 9 numbered workflow steps (down from 10 — the old worktree-placement step 5 and branch-shape step 6 merge into the new step 8). The preflight items (steps 1–4) are unchanged in content; steps 5–9 implement ADR-0010's sequencing: delegation prompt → sovereign promote on main → sovereign authorize on main (if delegating) → worktree placement and branch creation → hand-off.

**Pinned by:** [`TestAiwfxStartEpic_AC1_FixtureAndWorkflow`](../../../internal/policies/aiwfx_start_epic_test.go) — asserts the SKILL.md exists at the canonical authoring path, frontmatter `name:` and `description:` are valid, and exactly the integers 1..9 appear as `### N.` subheadings under `## Workflow` with no gaps and no extras. The test fails if any step is missing, duplicated, or if the step count drifts from 9. Sabotage-verified by manually inserting a renumber regression — the test fires.

### AC-2 — Stale G-0059 paragraph removed; replacement names ADR-0010

The pre-M-0104 SKILL.md at step 6 carried the prose *"G-0059 frames the open question of which branch-model convention aiwf should bless… Until G-0059 resolves, the skill surfaces the choice rather than presuming."* That deferral retired because ADR-0010 is the answer: branch shape is `epic/E-NN-<slug>`, settled.

The new SKILL.md contains zero "G-0059" mentions and references "ADR-0010" multiple times under `## Workflow` and adjacent sections, surfacing the convention explicitly so a reader of the workflow lands on the settled answer rather than the open question.

**Pinned by:** [`TestAiwfxStartEpic_M0104_AC2_G0059Removed_ADR0010Referenced`](../../../internal/policies/aiwfx_start_epic_test.go) — two-sided assertion. The G-0059 absence is checked over the full fixture body (substring unambiguous; the only legitimate reason it would re-appear is the regression this test catches). The ADR-0010 presence is asserted under `## Workflow` to scope the claim to the orchestration prose. Sabotage-verified — re-introducing "G-0059" anywhere in the body fails the test.

### AC-3 — Workflow headings structurally appear in new order

The `## Workflow` section's `### N.` headings, parsed structurally, appear in the sequence: preflight → drafted-milestone → `aiwf check` → tests/build → delegation → sovereign promotion → sovereign authorize → worktree → hand-off. The ordering is what the AC pins; the exact heading text is allowed to evolve so long as the conceptual sequence holds.

This is the load-bearing assertion of the milestone — without the structural ordering, a future edit could quietly swap two steps (e.g., move "worktree placement" before "sovereign promote", regressing the M-0103-driven sequencing invariant) and a flat substring search would not catch it. Per CLAUDE.md §"Substring assertions are not structural assertions", the assertion is heading-content driven and order-aware.

**Pinned by:** [`TestAiwfxStartEpic_M0104_AC3_WorkflowHeadingsInNewOrder`](../../../internal/policies/aiwfx_start_epic_test.go) — extracts the ordered list of `### N. <heading>` headings under `## Workflow`, then iterates an expected token sequence (`preflight`, `drafted-milestone`, `aiwf check`, `tests/build`, `delegation`, `sovereign promot`, `sovereign authoriz`, `worktree`, `hand-off`) asserting each token appears in the i-th heading. A reorder regression fails the test on the misplaced step's token mismatch. Sabotage-verified — swapping steps 5 and 6 headings fires the test with a precise pointer at the misordered slot.

### AC-4 — Preflight accepts --branch <future> from main (future-branch refinement)

M-0103's AI-target preflight is refined: when [`opts.CurrentBranch == "main"`](../../../internal/verb/authorize.go) AND `opts.Branch` parses as a ritual shape per [`internal/branchparse/`](../../../internal/branchparse/branchparse.go) (`epic/E-NNNN-...`, `milestone/M-NNNN-...`, `patch/g-NNNN-...`), the `BranchExists=false` refusal is suppressed. This is the well-formed step-7 pattern of the new `aiwfx-start-epic`: from main, name the future epic branch (cut at step 8) in the authorize trailer.

The implementation tightened the spec's literal *"valid ref name"* / *"git check-ref-format"* (pre-decided-design line 54) to *"ritual shape per `branchparse`"* — without the tightening, the gate becomes a no-op for any string under `--branch` from main. Spec body amended in the same milestone to align with the implementation (reviewer feedback Cycle 1 finding #1). The hardcoded `"main"` literal is parked as [G-0200](../../gaps/G-0200-preflight-main-only-carve-out-generalize-to-trunk-name-from-aiwf-yaml.md) (generalize to `aiwf.yaml.allocate.trunk` short name) — out of scope for M-0104 per the spec's literal "main" wording.

**Pinned by:**
- [`TestAuthorize_Open_AITarget_MainPlusRitualFutureBranch_Accepts`](../../../internal/verb/authorize_test.go) — verb-layer acceptance: main + ritual --branch + `BranchExists=false` accepts; trailer stamps the future ref.
- [`TestAuthorize_Open_AITarget_MainPlusNonRitualMissingBranch_Refuses`](../../../internal/verb/authorize_test.go) — carve-out guard: main + non-ritual --branch + `BranchExists=false` still refuses (without this guard the carve-out would be a gate-bypass).
- [`TestRunAuthorize_AITarget_MainPlusRitualFutureBranch_Accepts`](../../../internal/cli/integration/authorize_cmd_test.go) — CLI seam: `git init -b main` deterministically pins `CurrentBranch`; binary end-to-end proves the gather (`git symbolic-ref`, `git show-ref`) flows through; HEAD stays on main post-commit.
- Narrowed [`TestAuthorize_Open_AITarget_BranchMissing_Refuses`](../../../internal/verb/authorize_test.go) and [`TestRunAuthorize_AITarget_BranchMissing_Refuses`](../../../internal/cli/integration/authorize_cmd_test.go) — M-0103/AC-2 case kept faithful by pinning to a ritual non-main current branch where the AC-4 carve-out does not apply.

Sabotage-verified in both directions: removing `== "main"` fails the narrowed M-0103/AC-2 test; removing the ritual-shape constraint fails the carve-out guard.

### AC-5 — Skill body names --force --reason override at appropriate step

Both sovereign acts in the new workflow name the `--force --reason` override path: the sovereign-promotion step (step 6) inherits the existing M-0095 override hint; the new sovereign-authorize step (step 7) names it for the same reason — M-0103's preflight refuses on non-main checkouts, and the override is the same sovereign-act-shape escape valve, gated by the existing trailer-coherence rule that `--force` requires a `human/` actor.

The authorize step also names the M-0104/AC-4 carve-out's two preconditions (main + ritual `--branch`) so a reader who hits step 7 cold understands why the verb does not refuse despite the future-branch shape — the discoverability rationale aligns with CLAUDE.md §"kernel functionality must be AI-discoverable".

**Pinned by:**
- [`TestAiwfxStartEpic_AC3_SovereignPromotionStep`](../../../internal/policies/aiwfx_start_epic_test.go) (M-0096 carryover, still valid) — sovereign-promotion subsection names the verb, the human-only rule substance, and `--force --reason`.
- [`TestAiwfxStartEpic_M0104_AC5_SovereignAuthorizeStepNamesOverride`](../../../internal/policies/aiwfx_start_epic_test.go) — sovereign-authorize subsection names `aiwf authorize`, `--force --reason`, `--branch`, and `main`. Sabotage-verified — removing the override from the authorize step fires the test.

Reviewer Cycle 1 finding #6 (discoverability) folded into Cycle 2 alongside this AC: extended the [`--branch` flag help text](../../../internal/cli/authorize/authorize.go) and the [`PreflightBranchNotFoundError` error message](../../../internal/verb/authorize.go) to name the from-main future-branch carve-out. Operators hitting the gate cold from the CLI now see the discovery path without having to read the SKILL.md.

## Work log

### AC-4 — Preflight accepts --branch <future> from main

Implementation landed at commit `c1e26d5f` (originally `0bbaebee`; SHA changed when `aiwf-verb: feat` was filter-branch-stripped post-self-review). One-cycle TDD: red → green → done. Two new verb-layer tests + one CLI seam test + narrowed both existing M-0103/AC-2 tests so the AC-2 case pins refusal outside the AC-4 carve-out. Reviewer subagent dispatched mid-cycle; verdict approved with three follow-ups (spec amend, discoverability, G-0200 gap). Cycle 1 sabotage probes: `"main"` → `"Main"` (caught), drop ritual-shape (caught), `!opts.Force` → `opts.Force` (caught by 5 existing tests), CLI never sets `CurrentBranch` (caught), CLI hardwires `BranchExists=true` (caught).

### AC-1 + AC-2 + AC-3 + AC-5 — SKILL.md restructure

Implementation landed at commit `edadc6ec` (originally `fdf9b74d`; same filter-branch reason). One-cycle. SKILL.md restructured from 10 to 9 workflow steps; G-0059 deferral retired; ADR-0010 referenced in Principles, Workflow, and Constraints sections; `--force --reason` named at both sovereign acts. Drift-prevention tests updated: `TestAiwfxStartEpic_AC1_FixtureAndWorkflow` (10 → 9 steps), `TestAiwfxStartEpic_AC4_BranchPromptDefersToG0059` deleted (the section it pinned no longer exists), `findBranchPromptSection` + branch-coverage test deleted. New tests: `TestAiwfxStartEpic_M0104_AC2_G0059Removed_ADR0010Referenced`, `TestAiwfxStartEpic_M0104_AC3_WorkflowHeadingsInNewOrder`, `TestAiwfxStartEpic_M0104_AC5_SovereignAuthorizeStepNamesOverride`, plus `findSovereignAuthorizeSection` helper with its own branch-coverage test. Cycle 2 reviewer must-addresses folded in: spec body amended via `aiwf edit-body M-0104` (commit `fd5ca26e`); `--branch` flag help + `PreflightBranchNotFoundError` message extended for discoverability. Cycle 2 sabotage probes: swap step 5/6 headings (caught), re-introduce "G-0059" (caught), remove `--force --reason` from authorize step (caught).

## Decisions made during implementation

- **Carve-out condition shape: ritual-shape, not "valid ref name".** The pre-decided-design at line 54 named `git check-ref-format` as the validator; the implementation uses `branchparse.ParseEntityFromBranch` (stricter — must parse as a ritual shape). Rationale: a `check-ref-format` test would accept `--branch any-string`, making the M-0103 preflight a no-op for any operator on main. The ritual-shape test keeps the chokepoint useful. Spec body amended in the same milestone to align with the implementation; rationale recorded inline so a future re-litigation finds the prior reasoning.
- **`"main"` hardcoded literal, not derived from `aiwf.yaml.allocate.trunk`.** The spec says "main" literally; trunk-name configurability is internally consistent with the rest of `internal/branchparse/`'s hardcoded prefixes. Generalization parked as [G-0200](../../gaps/G-0200-preflight-main-only-carve-out-generalize-to-trunk-name-from-aiwf-yaml.md). Per YAGNI, deferred until a real consumer hits the friction; the implicit-ritual-current path and `--force --reason` escape valve cover the workaround.
- **Two M-0103 tests narrowed (not deleted) post-AC-4.** The verb-level `TestAuthorize_Open_AITarget_BranchMissing_Refuses` and CLI-level `TestRunAuthorize_AITarget_BranchMissing_Refuses` originally used `CurrentBranch="main"` — after AC-4 that scenario now accepts. Both tests pinned to a ritual non-main current (verb) / explicit `git checkout -b epic/E-0001-engine` (CLI) so the AC-2 refusal case is preserved outside the AC-4 carve-out's scope. The narrowing keeps M-0103/AC-2's spirit intact without false-positives from the carve-out.
- **9-step workflow, not 10.** The spec's "new step ordering" is 6 logical phases but step 1 ("Preflight") unfolds into 4 numbered sub-steps (read epic spec, drafted-milestone, `aiwf check`, tests/build). Treating the preflight as 4 numbered steps (not 1 collapsed step) preserves the existing structural assertions and reader's navigation; the merged worktree+branch-creation step drops total by 1.

## Validation

- `go test -race -parallel 8 ./...` — green across all 35 packages.
- `go build -o /tmp/aiwf-final ./cmd/aiwf` — green.
- `aiwf check` — 0 errors, 5 warnings (all `entity-body-empty` on the AC body sections pre-wrap; this commit fills them).
- Sabotage probes (Cycle 1 + Cycle 2 combined): 8 single-line regressions, each caught by at least one test.
- `wf-doc-lint` scoped to the changeset: clean.
- Trailer hygiene: `aiwf-verb: feat` lines stripped from Cycle 1 + Cycle 2 implementation commits via `git filter-branch --msg-filter` post-self-review (the warnings were caught at self-review, fixed before wrap, re-verified clean).

## Deferrals

- [G-0200](../../gaps/G-0200-preflight-main-only-carve-out-generalize-to-trunk-name-from-aiwf-yaml.md) — generalize the hardcoded `"main"` literal in the AC-4 carve-out to `aiwf.yaml.allocate.trunk` short name. Real layering smell (verb-layer gate carries a hardcoded value the allocator layer already configures), but out of M-0104 scope per the spec's literal "main" wording. Filed during Cycle 1.

## Reviewer notes

- Reviewer subagent dispatched mid-Cycle 1 (post-tests, pre-commit). Verdict: **approve**. Three findings folded into Cycle 2: (a) spec-body amendment AC-4 wording → "ritual shape" (committed as `fd5ca26e`), (b) `--branch` flag help text + `PreflightBranchNotFoundError` discoverability extension (Cycle 2 commit `edadc6ec`), (c) G-0200 gap filed pre-Cycle 2.
- All sabotage probes (8 across both cycles) caught by at least one test. The carve-out's narrow conjunction (main AND ritual) is independently guarded — both halves can be broken and the test set fires.
- Branch-coverage hard rule satisfied: the 4 reachable arms of the explicit-branch path in `authorizeOpen` are each exercised by at least one test.
- The implementation is intentionally narrow — no support for configurable trunk-branch name, no support for arbitrary "valid ref name" carve-outs. Both are documented as deliberate YAGNI/intent-tightening decisions in the Decisions section above.
- [G-0199](../../gaps/G-0199-finding-hints-must-name-the-exact-remediation-command.md) was filed earlier in the session before M-0104 work, unrelated to this milestone's scope; it appears in the diff range but is not a deferral of M-0104 — it stands as its own kernel-discoverability concern.
- One pre-existing test flake observed during self-review (`TestReallocate_RewritesProseAcrossMultipleEntities`); confirmed flake by re-run alone (passes deterministically). Not from M-0104 changes — same `internal/verb` package, different test, different domain (reallocate, not authorize).


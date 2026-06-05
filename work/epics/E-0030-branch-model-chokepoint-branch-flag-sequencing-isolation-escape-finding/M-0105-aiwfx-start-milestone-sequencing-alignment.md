---
id: M-0105
title: aiwfx-start-milestone sequencing alignment
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
      title: Skill asserts tightened parent-epic-branch precondition
      status: met
      tdd_phase: done
    - id: AC-3
      title: Silent fallthrough to checkout -b epic/<slug> if missing removed
      status: met
      tdd_phase: done
    - id: AC-4
      title: Workflow headings structurally appear in new order
      status: met
      tdd_phase: done
    - id: AC-5
      title: Skill body names --force --reason override at appropriate step
      status: met
      tdd_phase: done
    - id: AC-6
      title: Milestone scope aiwf-branch trailer records milestone branch
      status: met
      tdd_phase: done
---
## Goal

Align `aiwfx-start-milestone`'s step order with M-0104's epic-side fix: `aiwf promote M-NNNN draft → in_progress` lands on the parent epic branch (which already exists at this point from `aiwfx-start-epic`), then — *if* the work is being delegated — `aiwf authorize M-NNNN --to ai/<id> --branch milestone/M-NNNN-<slug>` lands on the same parent epic branch, then the milestone work branch is cut off the parent. Tighten the "must be on parent epic branch" precondition so silent fallthrough to `git checkout -b epic/E-NNNN-<slug> if missing` is removed — missing parent epic branch is a hard precondition failure pointing the operator at `aiwfx-start-epic`.

## Context

M-0104 establishes the pattern for `aiwfx-start-epic`; this milestone applies the same shape one level down at the milestone-start ritual. [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s symmetric rule for milestones: the promote-to-in_progress is a state-announcement that belongs on the parent epic branch (which already exists at this point), not on the milestone work branch (which hasn't been cut yet).

Today's embedded `aiwfx-start-milestone` step 2 *does* promote before branch setup — which matches ADR-0010's order — but step 3 contains a silent fallthrough (`git checkout -b epic/E-NNNN-<slug> origin/main # if missing`) that masks the precondition failure case. This milestone removes the fallthrough and adds the explicit "epic branch must exist; if it doesn't, run `aiwfx-start-epic` first" check.

Ritual content edits land at the canonical authoring location (per [ADR-0014](../../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) and [ADR-0016](../../../docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md)):

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md`

One edit in one commit. The upstream `ai-workflow-rituals` repo was archived under ADR-0016 — no cross-repo coordination.

## Pre-decided design

- **New step ordering** (matches the symmetric pattern from M-0104):
  1. Preflight (existing — includes "parent epic branch must exist and be currently checked out"; this is the tightened precondition).
  2. Delegation prompt (Q&A — promoted earlier so the operator's choice is known before the sovereign acts).
  3. `aiwf promote M-NNNN in_progress` on the parent epic branch (existing step 2's content, now explicitly named as "lands on parent epic branch").
  4. *(if delegating)* `aiwf authorize M-NNNN --to ai/<id> --branch milestone/M-NNNN-<slug>` on the parent epic branch. Same "future branch" refinement from M-0104's preflight applies: the named milestone branch doesn't yet exist; M-0103's preflight accepts it because the operator is on a recognized ritual shape (the parent epic branch) and `--branch` parses as a valid ref.
  5. Cut the milestone branch off the parent epic branch (`git checkout -b milestone/M-NNNN-<slug>`).
  6. Hand off to `wf-tdd-cycle` for each AC (existing).
- **Tightened precondition (step 1):** the current checkout must be the parent epic branch identified by `aiwf show M-NNNN`'s parent field. If the parent epic branch doesn't exist locally, the ritual stops and points at `aiwfx-start-epic E-NNNN`. The silent `git checkout -b epic/E-NNNN-<slug> origin/main # if missing` fallthrough is removed.
- **Scope inheritance:** the milestone's `aiwf authorize` opens a **new** scope independent of the epic's (the current kernel semantics — one scope per entity, no cross-entity coordination). The milestone scope's `aiwf-branch:` records the milestone branch; the epic scope's `aiwf-branch:` records the epic branch. M-0106's finding rule walks back to the nearest active scope on the entity in question (the milestone for milestone-entity commits, the epic for epic-entity commits) — no special cross-scope logic. (Conceptual framing per [ADR-0009](../../../docs/adr/ADR-0009-orchestration-substrate-vs-driver-split.md) substrate/driver split, but no mechanical dependency on ADR-0009 ratifying.)
- **Override path naming** in skill body: same shape as M-0104 — the skill body names `--force --reason "..."` at the relevant step so operators see it.

## Out of scope

- `aiwfx-start-epic` (M-0104, sibling).
- Kernel finding (M-0106).
- AC-level branch behavior — ACs ride on the milestone branch alongside test/code commits per ADR-0010; no separate AC-branch convention is in scope here.
- Spec-cell consolidation (the consolidation milestone).

## Dependencies

- **M-0102** — `--branch` flag and `internal/branchparse/` helpers.
- **M-0103** — preflight refuses dispatch without ritual branch context; the "future branch" refinement (added under M-0104) makes the `--branch milestone/M-NNNN-<slug>` call on a yet-uncut branch acceptable.

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0105` time. AC seed set:
1. The embedded snapshot at `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md` reflects the new step ordering: tightened preflight → delegation prompt → promote on parent epic branch → authorize on parent epic branch (if delegating) → cut milestone branch → wf-tdd-cycle hand-off.
2. The tightened precondition is explicit: skill body asserts "parent epic branch must exist and be the current checkout; if missing, run `aiwfx-start-epic E-NNNN` first."
3. The silent `git checkout -b epic/E-NNNN-<slug> origin/main # if missing` fallthrough is removed from the skill body.
4. The skill's `## Workflow` section, parsed structurally, presents the steps in the order specified above.
5. The skill body names the override path (`--force --reason "..."`) at the relevant step.
6. The milestone scope's `aiwf-branch:` trailer records the milestone branch (verified via an end-to-end fixture: run the ritual against a fixture epic, inspect the resulting authorize commit's trailer).
-->

### AC-1 — Embedded snapshot reflects new step ordering

The embedded snapshot at [`internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md`](../../../internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md) carries 8 numbered workflow steps (up from 6): the old 6-step shape (preflight → promote → branch setup → implementation → self-review → hand off) becomes the new 8-step shape (preflight tightened → delegation prompt → sovereign promote on parent → sovereign authorize on parent → cut milestone branch → implementation → self-review → hand off). The growth reflects ADR-0010's sequencing — the state-announcement commits land on the parent epic branch BEFORE the milestone branch is cut, mirroring M-0104's pattern one rung down.

**Pinned by:** [`TestAiwfxStartMilestone_M0105_AC1_FixtureAndWorkflow`](../../../internal/policies/aiwfx_start_milestone_test.go) — asserts the SKILL.md exists at the canonical authoring path, frontmatter `name:` and `description:` are valid, and exactly the integers 1..8 appear as `### N.` subheadings under `## Workflow` with no gaps and no extras. Adding a 9th step or dropping one fires the test on count mismatch.

### AC-2 — Skill asserts tightened parent-epic-branch precondition

The new step 1 (Preflight) explicitly names the precondition: *"the current checkout must be the parent epic branch identified by `aiwf show M-NNNN`'s parent field"*, with `epic/E-NNNN-<slug>` named as the canonical shape and `aiwfx-start-epic E-NNNN` named as the escape hatch when the parent branch is missing. The skill also handles the "parent exists but not checked out" case by directing the operator to `git checkout` before continuing.

This replaces the prior implicit assumption (the old skill body did not state any precondition about the parent branch and silently fell through to creating it).

**Pinned by:** [`TestAiwfxStartMilestone_M0105_AC2_PreflightAssertsParentEpicBranchPrecondition`](../../../internal/policies/aiwfx_start_milestone_test.go) — heading-scoped to step 1, asserts 4 load-bearing markers: `epic/E-NNNN` (branch identifier), `must exist` (existence requirement), `current checkout` (active-checkout requirement), `aiwfx-start-epic` (escape-hatch pointer). All 4 must be present; partial regression (e.g. keeping "must exist" but dropping "current checkout") fires the test on the dropped marker.

### AC-3 — Silent fallthrough to checkout -b epic/<slug> if missing removed

The old skill body's step 3 (Branch setup) included a silent fallthrough:

```bash
git checkout -b epic/E-NNNN-<slug> origin/main      # if missing
```

This masked the missing-parent-branch precondition failure case — an operator whose parent epic was not activated would land on a freshly-cut branch with no `aiwf promote E-NNNN active` commit on it. ADR-0010 and the AC-2 tightened precondition together replace that improvisation with an explicit "stop and run `aiwfx-start-epic`" handoff.

The new SKILL.md retains the old fallthrough text only in the Anti-patterns section, as a documented "don't do this" — the AC-3 test scopes its check to `## Workflow` so the documentation-of-anti-pattern usage doesn't false-positive.

**Pinned by:** [`TestAiwfxStartMilestone_M0105_AC3_NoSilentFallthroughToParentCheckout`](../../../internal/policies/aiwfx_start_milestone_test.go) — scoped to `## Workflow`, forbids three markers: `# if missing` (the exact stale comment), `origin/main` (the literal branch ref in the old git command), and — per Cycle 2 reviewer feedback — the structural shape `git checkout -b epic/` that catches rephrased regressions (no comment, different verb tense, etc.). Sabotage-verified: inserting any of the three fires the test.

### AC-4 — Workflow headings structurally appear in new order

The `## Workflow` section's `### N.` headings, parsed structurally, appear in the sequence: preflight → delegation prompt → sovereign promote → sovereign authorize → cut milestone branch → implementation → self-review → hand off. Heading-content driven per CLAUDE.md §"Substring assertions are not structural assertions"; the assertion is order-aware and token-based so wording polish doesn't churn the test.

The load-bearing reorder relative to the old skill body is: (a) the delegation prompt moves to step 2 (was buried in the implementation phase); (b) the sovereign acts (promote at 3, authorize at 4) explicitly run on the parent epic branch BEFORE the milestone-branch cut at step 5; (c) the milestone-branch cut becomes its own named step rather than half-buried in old step 3's branch setup.

**Pinned by:** [`TestAiwfxStartMilestone_M0105_AC4_WorkflowHeadingsInNewOrder`](../../../internal/policies/aiwfx_start_milestone_test.go) — extracts the ordered list of headings, asserts each contains an expected lowercase token (`preflight`, `delegation`, `sovereign promot`, `sovereign authoriz`, `cut`, `implementation`, `self-review`, `hand off`) at its expected index. Sabotage-verified — swapping any two steps fires the test on both the misplaced indices.

### AC-5 — Skill body names --force --reason override at appropriate step

Both sovereign acts (promote at step 3, authorize at step 4) name `--force --reason` as the override path — symmetric to M-0104's pattern for `aiwfx-start-epic`. The authorize step additionally names the M-0105/AC-6 carve-out's preconditions inline (current on parent epic branch identified as `epic/E-NNNN-<slug>`, `--branch milestone/M-NNNN-<slug>` future-binding) so an operator reading step 4 cold understands why the verb does not refuse despite the future-branch shape.

**Pinned by:** [`TestAiwfxStartMilestone_M0105_AC5_SovereignActsNameOverride`](../../../internal/policies/aiwfx_start_milestone_test.go) — two-sided assertion. The promote section (step 3) must contain `aiwf promote` and `--force --reason`. The authorize section (step 4) must contain `aiwf authorize`, `--force --reason`, `--branch`, `milestone/M-NNNN` (the future-binding shape), and `epic/E-NNNN` (the current-context shape). Sabotage-verified — removing `--force --reason` from the authorize step fires the test.

### AC-6 — Milestone scope aiwf-branch trailer records milestone branch

M-0103's AI-target preflight had a main-only future-branch carve-out (M-0104/AC-4): from `main` + ritual `--branch` shape + `BranchExists=false` → accept. AC-6 extends this so the milestone-level invocation pattern (`aiwf authorize M-NNNN --to ai/<id> --branch milestone/M-NNNN-<slug>` from the parent epic branch, with the milestone branch not yet existing) is also accepted. The extended condition: `(CurrentBranch == "main" || ritual(CurrentBranch)) && ritual(--branch)`.

The resulting commit's `aiwf-branch:` trailer carries the future milestone ref, exactly as the spec language requires ("milestone scope's aiwf-branch trailer records milestone branch"). The trailer is a forward-binding; step 5 of the ritual closes it by cutting the named branch.

The extension uses a flat union (main-or-ritual current × ritual --branch) — no hierarchical parent/child check between the two shapes. Cross-rung mismatches (e.g. `epic/E-0001-foo` current + `epic/E-0002-bar` --branch) syntactically accept; deliberate YAGNI parking, filed as [G-0201](../../gaps/G-0201-authorize-preflight-carve-out-accepts-cross-rung-ritual-mismatches.md).

**Pinned by:**
- [`TestAuthorize_Open_AITarget_RitualCurrentPlusRitualFutureBranch_Accepts`](../../../internal/verb/authorize_test.go) — verb-layer acceptance: ritual current (`epic/E-0001-engine`) + ritual future `--branch` (`milestone/M-0001-cache`) + `BranchExists=false` → accepts; trailer stamps the future milestone ref.
- [`TestAuthorize_Open_AITarget_NonRitualNonMainCurrent_BranchMissing_Refuses`](../../../internal/verb/authorize_test.go) — carve-out lower-bound guard: non-ritual non-main current + missing `--branch` still refuses. Without this guard the extended carve-out would be a gate-bypass.
- [`TestRunAuthorize_AITarget_RitualCurrentPlusMilestoneFutureBranch_Accepts`](../../../internal/cli/integration/authorize_cmd_test.go) — CLI seam end-to-end: drives the real binary against a milestone fixture, on the parent epic branch, asserts the resulting commit's trailer is `aiwf-branch: milestone/M-0001-cache` and HEAD stays on the parent epic branch (carve-out doesn't move the operator).
- Re-narrowed [`TestAuthorize_Open_AITarget_BranchMissing_Refuses`](../../../internal/verb/authorize_test.go) and [`TestRunAuthorize_AITarget_BranchMissing_Refuses`](../../../internal/cli/integration/authorize_cmd_test.go) — M-0103/AC-2 case kept faithful by pinning to a non-main non-ritual `feature/...` current branch where neither M-0104 nor M-0105 carve-out applies.

Sabotage-verified in all directions: dropping the ritual-current arm fails the AC-6 acceptance test; dropping the ritual-future requirement fails the M-0104 guard; setting `futureBindingAccepted = false` fails both M-0104 and M-0105 acceptance tests; typo-swapping `branchparse.ParseEntityFromBranch(opts.CurrentBranch)` to use `branchExplicit` fails the M-0103/AC-2 narrowed tests and the M-0105 lower-bound guard.

## Work log

### AC-6 — Extend preflight carve-out to ritual current

Implementation landed at commit `782fae8c`. One-cycle TDD: red → green → done. Two new verb-layer tests (acceptance + lower-bound guard) + one CLI seam test + re-narrowed both M-0103/AC-2 tests to use non-ritual non-main current. Reviewer subagent dispatched mid-cycle; verdict approved with one nice-to-have (cross-rung looseness — filed as G-0201) and one note (spec-cell elaboration deferred to M-0158). Cycle 1 sabotage probes: drop ritual-current arm, typo `"main"` → `"Main"`, swap-typo of `branchparse` argument, drop entire carve-out, drop trailer emission — all caught.

### AC-1 + AC-2 + AC-3 + AC-4 + AC-5 — SKILL.md restructure

Implementation landed at commit `4a6047ef`. One-cycle. SKILL.md restructured from 6 to 8 workflow steps; preflight tightened (parent epic branch precondition); silent fallthrough removed; sovereign acts cite the M-0105/AC-6 carve-out and `--force --reason` override. New drift-prevention test file `internal/policies/aiwfx_start_milestone_test.go` mirroring `aiwfx_start_epic_test.go` shape — 5 AC tests + 2 helper branch-coverage tests + 2 helper functions. Reviewer subagent dispatched pre-commit; verdict approved with one nice-to-have (AC-3 marker class — added `git checkout -b epic/` per feedback) and one clarification (M-0106 forward-reference — qualified wording so it's robust if E-0030 ships in stages). Cycle 2 sabotage probes: swap step 5/6 headings, re-introduce `# if missing` and `origin/main` inside workflow, insert `git checkout -b epic/` (post-reviewer marker), delete preflight precondition block, remove `--force --reason` from authorize step — all caught.

## Decisions made during implementation

- **Carve-out condition: main-or-ritual current × ritual --branch (flat union, no hierarchy).** The looser check covers every legitimate ritual invocation (`aiwfx-start-epic` step 7 with main + epic; `aiwfx-start-milestone` step 4 with epic + milestone) and refuses the loudest mistakes (non-ritual non-main current + missing `--branch`). A hierarchical check (current must be the parent rung of `--branch`) would be more code for a narrower window. Cross-rung mismatches (e.g. `epic/E-0001-foo` + `epic/E-0002-bar`, `milestone/...` + `epic/...`) syntactically accept; tracked as [G-0201](../../gaps/G-0201-authorize-preflight-carve-out-accepts-cross-rung-ritual-mismatches.md). Documented inline at the carve-out site so a future reader does not re-litigate the trade-off.
- **8-step shape, not 6.** The new sequencing introduces 2 logical phases (delegation prompt, sovereign authorize) that were absent from the old skill, and splits the old "branch setup" step into a preflight precondition + a named step-5 cut. Net +2 steps. The alternative — folding delegation+authorize into a single conditional block under one heading — was rejected because the structural drift test (AC-4 ordering) reads heading shape, and a single conditional block doesn't surface the two-act sequencing the spec calls for.
- **Re-narrow M-0103/AC-2 tests (verb + CLI) to non-ritual non-main current.** The previous M-0104 narrowing used `epic/E-0001-engine` (ritual non-main). After AC-6's extension that scenario now ACCEPTS instead of refusing — breaking the original AC-2 spirit. Re-narrowing to `feature/test-fixture` / `feature/scratch` pins the missing-branch refusal outside BOTH carve-outs, preserving AC-2's intent.
- **AC-3 forbidden-pattern set widened to include `git checkout -b epic/`.** Per Cycle 2 reviewer feedback: the original 2-marker set (`# if missing`, `origin/main`) only catches the verbatim old fallthrough. A rephrased regression (e.g. no comment, different ref source) would slip through. Adding the structural marker `git checkout -b epic/` catches the regression class — the skill body must NEVER prescribe creating the parent epic branch under `## Workflow`, since that's `aiwfx-start-epic`'s job per AC-2's tightened precondition.
- **M-0106 forward-reference qualified.** The Anti-patterns bullet at SKILL.md:157 originally said *"M-0106 finding catches the same shape post-hoc"* in present tense. M-0106 is `draft` and ships later in E-0030. The wording is qualified to *"once M-0106 ships ... the same shape is also caught post-hoc"* so the skill body is accurate even if E-0030 ships in stages.

## Validation

- `go test -race -parallel 8 ./...` — green across all packages.
- `go build -o /tmp/aiwf-m0105-final ./cmd/aiwf` — green.
- `aiwf check` — 0 errors, 6 warnings (all `entity-body-empty` on AC body sections pre-wrap; this commit fills them).
- Sabotage probes (Cycle 1 + Cycle 2 combined): 9 single-line regressions, each caught by at least one test.
- `wf-doc-lint` scoped to the changeset: clean.
- Trailer hygiene: no `aiwf-verb: feat` on implementation commits (lesson from M-0104 self-review applied prospectively this milestone).

## Deferrals

- [G-0201](../../gaps/G-0201-authorize-preflight-carve-out-accepts-cross-rung-ritual-mismatches.md) — cross-rung looseness in the extended carve-out. Filed during Cycle 1 reviewer feedback. Tighten to hierarchical (current parent-rung of `--branch`) if cross-rung typos become a real incident class. Out of scope per the spec's pre-decided design (the M-0105/AC-6 spec did not call for hierarchy).

## Reviewer notes

- Two reviewer subagent passes during this milestone: Cycle 1 (post-tests, pre-commit) and Cycle 2 (post-tests, pre-commit). Both verdicts: approve. Total 9 sabotage probes verified across cycles.
- Spec-cell elaboration for both M-0104 and M-0105 carve-outs remains scaffold-quality in `internal/workflows/spec/rules.go`. The M-0123/AC-5 drift policy operates at code-ID level (not predicate fidelity) so the existing scaffolding holds. Predicate elaboration deferred to **M-0158 (layer-4 branch choreography spec cells + drift policy extension)** — same epic, scheduled after M-0106.
- Branch-coverage hard rule satisfied across all reachable arms of the extended carve-out in `internal/verb/authorize.go` and the two new helper functions in `internal/policies/aiwfx_start_milestone_test.go`.
- The implementation is intentionally narrow: flat union of carve-out conditions (no hierarchy), 8 workflow steps (no merger of delegation+authorize), markers-based AC tests scoped to specific sections (not body-wide). Each narrowing is a deliberate YAGNI/intent-tightening decision documented in the Decisions section above.
- No new typed Coded errors introduced; the existing `PreflightBranchNotFoundError` covers the extended carve-out's refusal path. The CLI `--branch` help text and `PreflightBranchNotFoundError` message both extended to name both M-0104/AC-4 and M-0105/AC-6 carve-outs for discoverability.


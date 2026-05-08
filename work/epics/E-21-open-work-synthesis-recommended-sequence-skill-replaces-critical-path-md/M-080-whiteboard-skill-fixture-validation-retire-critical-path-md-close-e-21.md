---
id: M-080
title: Whiteboard skill fixture validation; retire critical-path.md; close E-21
status: draft
parent: E-21
tdd: required
depends_on: [M-079]
acs:
    - id: AC-1
      title: Fixture-validation test runs the skill against the current planning tree
      status: open
      tdd_phase: red
    - id: AC-2
      title: Output structurally agrees with critical-path.md (tier set, sequence, fork)
      status: open
      tdd_phase: red
    - id: AC-3
      title: Pending-decisions section enumerates at least the decisions in critical-path.md
      status: open
      tdd_phase: red
    - id: AC-4
      title: Three natural-language test prompts route to the skill via description-match
      status: open
      tdd_phase: red
    - id: AC-5
      title: work/epics/critical-path.md deleted in this milestone's wrap commit
      status: open
      tdd_phase: red
    - id: AC-6
      title: aiwf check shows no unexpected-tree-file warning for critical-path.md
      status: open
      tdd_phase: red
    - id: AC-7
      title: E-21 promoted to done; epic wrap commit cites the closure
      status: open
      tdd_phase: red
---

# M-080 — Whiteboard skill fixture validation; retire critical-path.md; close E-21

## Goal

Validate the `aiwfx-whiteboard` skill (M-079) against the existing `critical-path.md` as a fixture, retire the holding doc in the wrap commit, and close E-21. The skill's output, run on the current planning tree, should structurally agree with `critical-path.md` (same tier set, same recommended sequence, same first-decision options) — that's the proof the synthesis pattern survived graduation from one-off conversation into reproducible skill body. After this milestone, `critical-path.md` is deleted, the standing `unexpected-tree-file` warning it generated is gone, and E-21 is `done`.

## Context

`work/epics/critical-path.md` was authored on 2026-05-08 during E-20 planning as a temporary holding pattern for direction synthesis the operator would otherwise have lost when conversation context scrolled. Its scope is specifically a snapshot — not a maintained doc. E-21 graduates the synthesis pattern into the `aiwfx-whiteboard` skill (M-079); this milestone closes the loop by demonstrating the skill produces equivalent output and removing the snapshot.

The fixture-validation in this milestone is a *one-shot graduation check*, not a CI-running regression. The actual planning tree drifts continuously — a CI test asserting "skill output equals critical-path.md" would go stale within hours of merge. Instead, AC-1/2/3 are validated once at this milestone's wrap, with the validation paste captured in the milestone body. AC-5/6/7 carry the programmatic surface (file absence, warning class, epic-status assertion) suited to `tdd: required`.

Per the kernel rule *"render output must be human-verified before the iteration closes"* (CLAUDE.md), this milestone's validation explicitly includes opening Claude Code, invoking the skill, and reading the output against `critical-path.md` side-by-side. Test suites pin code correctness; only a manual look pins feature correctness for renderable output.

## Acceptance criteria

### AC-1 — Fixture-validation test runs the skill against the current planning tree

Operator opens a Claude Code session against this repo, invokes the `aiwfx-whiteboard` skill via natural-language query (one of the AC-4 prompts), captures the skill's output as a transcript paste under this milestone's *Validation* section. The capture is the fixture this milestone preserves in the entity body before `critical-path.md` is deleted.

### AC-2 — Output structurally agrees with critical-path.md (tier set, sequence, fork)

Validation paste demonstrates structural agreement: same tier set (Tier 1–5 with same axes — leverage, foundational, ritual, debris, defer), same item placements at Tier 1 (G-071, G-072 minimally; others permissible), same recommended-sequence section with similar pre-E-20 / E-20 / post-E-20 / parallel ordering, same first-decision fork with three options A/B/C carrying the bundling-into-M-072 question. Diff is permissible (LLM judgement varies); structural agreement on sections and tier set is required. The validation paste explicitly notes any structural divergences with one-line rationale.

### AC-3 — Pending-decisions section enumerates at least the decisions in critical-path.md

Validation paste's *Pending decisions* section enumerates at least: (1) Tier 1 bundling fork (A/B/C), (2) ratification of ADR-0001/0003/0004, (3) ordering of ADR implementation epics, (4) audit of G-058 status, (5) graduation question for `critical-path.md` itself (which this very milestone closes — note in the paste that this decision is being resolved by M-080's wrap). Additional pending decisions surfaced by the skill are welcome; the floor is the five from critical-path.md.

### AC-4 — Three natural-language test prompts route to the skill via description-match

Operator runs three distinct natural-language queries against a Claude Code session and confirms each routes to `aiwfx-whiteboard` (not to `aiwf-status`, `aiwfx-plan-epic`, or other adjacent skills). The three prompts are: *"what should I work on next?"*, *"give me the landscape"*, *"draw the whiteboard"*. Capture the routing-confirmation in the milestone's *Validation* section. If a prompt routes to the wrong skill, AC-4 is not met until the description in `aiwfx-whiteboard`'s frontmatter is amended (small backstop scope on M-079) or the misrouting is documented as a known limitation.

### AC-5 — work/epics/critical-path.md deleted in this milestone's wrap commit

`git rm work/epics/critical-path.md` is staged into the milestone's wrap commit. Test surface: a Go test (e.g., under `internal/policies/` or as a one-shot check in this milestone's validation block) asserts the file does not exist on disk. Red phase: the test fails because the file is still present pre-wrap. Green phase: the test passes after deletion. The deletion is part of the same atomic commit that promotes the milestone — not a separate uncommitted change.

### AC-6 — aiwf check shows no unexpected-tree-file warning for critical-path.md

After deletion, `aiwf check` produces zero warnings of class `unexpected-tree-file` for the path `work/epics/critical-path.md`. Test surface: a Go test invokes `check.Run` against a fixture tree containing the deletion and asserts no `unexpected-tree-file` finding cites the path. Red phase: with the file present, the check warns. Green phase: with the file absent, the check is silent on that path. This AC also implicitly verifies the skill itself does not regenerate a `critical-path.md`-shaped artefact (per M-079's "no persisted artefact" constraint).

### AC-7 — E-21 promoted to done; epic wrap commit cites the closure

`aiwf promote E-21 done --reason "M-078/M-079/M-080 wrapped; aiwfx-whiteboard skill ships and replaces critical-path.md"`. The promotion produces one commit with `aiwf-verb: promote`, `aiwf-entity: E-21`. The commit body cites the three milestones and notes critical-path.md retirement. Test surface: a Go test asserts `aiwf show E-21` returns status `done` after this milestone's wrap; alternatively, the per-AC validation reads `aiwf show E-21 --format=json` and asserts `.status == "done"`.

## Constraints

- **One-shot fixture validation, not regression test.** AC-1/2/3 are captured by the milestone's *Validation* paste and are not converted into a permanent CI test. The planning tree drifts; a fixture-pinned regression test would go stale within hours.
- **Manual route-check is acceptable for AC-4.** Claude Code's plugin tooling does not currently expose programmatic description-match testing in this repo's CI. The route-check is operator-run with the result captured in the validation paste. If a programmatic harness ships later, this milestone's AC-4 does not retroactively require migration.
- **Critical-path.md deletion is atomic with the wrap.** Do not delete the file in a setup commit and wrap separately — the deletion is part of the milestone's promotion commit so that history shows "milestone wrapped → file gone" as one event with the right `aiwf-verb` / `aiwf-entity` trailers.
- **No re-introduction of persisted synthesis artefacts.** This milestone does not file `whiteboard.md`, `landscape.md`, or any other on-disk synthesis output. The skill is on-demand by contract; M-080 enforces that contract by removing the only existing exception.
- **E-21 promotion only after M-078, M-079, M-080 are all `done`.** Standard epic-wrap discipline. `aiwfx-wrap-epic` is the skill that drives this; M-080's AC-7 is the entity-level expression of its closing act.
- **Render-output human-verified per CLAUDE.md.** AC-1/AC-4 explicitly require a human Claude Code session to confirm the skill renders correctly and routes correctly. Tests pin code; humans pin features.

## Design notes

- Validation flow at start-milestone (refine in TDD red phase):
  1. Operator opens a Claude Code session in this repo.
  2. Operator queries the skill via *"what should I work on next?"* — confirms route to `aiwfx-whiteboard` (AC-4 #1).
  3. Operator captures the skill's full output into the milestone body's *Validation* section (AC-1).
  4. Operator pastes `critical-path.md`'s tier table beside the skill's tier table in the validation; one-line diff per row identifies structural agreement or noted divergences (AC-2).
  5. Operator pastes critical-path.md's *Pending decisions* list and the skill's pending list side-by-side; AC-3 verified.
  6. Operator runs the other two route prompts (*"give me the landscape"*, *"draw the whiteboard"*); AC-4 #2 and #3 confirmed.
  7. Test code (Go) for AC-5/6 written first as red — file present, warning fires.
  8. `git rm work/epics/critical-path.md` — green for AC-5; warning gone for AC-6.
  9. `aiwf promote E-21 done --reason "..."` — green for AC-7.
- Test code outline for AC-5/6 (refine at red phase):
  ```go
  func TestCriticalPathRetired(t *testing.T) {
      _, err := os.Stat("work/epics/critical-path.md")
      if !os.IsNotExist(err) {
          t.Fatalf("critical-path.md should be retired in M-080 wrap; still present")
      }
      // run aiwf check against the live tree, assert no unexpected-tree-file
      // finding cites this path
      ...
  }
  ```
  Lives under `internal/policies/` or an integration-test package if a fitting one exists. Follows the precedent of `internal/policies/policies_test.go`.
- Validation paste format (refine at validation): two side-by-side tables (critical-path.md vs skill output) for the tier landscape; literal copy of the recommended-sequence prose with [agreement/divergence] annotations; literal copy of the first-decision fork with the three options; literal copy of the pending-decisions list. The validation paste lands in this entity's *Validation* section, not in a separate file.
- Epic-wrap discipline: `aiwfx-wrap-epic` is the skill that runs at E-21 close. M-080's AC-7 is the granular expression of its work; the wrap skill orchestrates promotion, doc-lint scope check, and harvested-ADR candidates (M-078's ADR is one such candidate, status decision happens at wrap).

## Surfaces touched

- `work/epics/critical-path.md` — DELETED (AC-5)
- `internal/policies/critical_path_retired_test.go` (or equivalent path; new — small file for AC-5/6 tests)
- This milestone's body — *Validation* section gets the fixture paste (AC-1/2/3/4)
- `work/epics/E-21-*/epic.md` — Milestones list updated to reflect final state; status promoted (AC-7)
- No new code in `cmd/aiwf/` or `internal/skills/` (skill ships in M-079)

## Out of scope

- Migrating the fixture-validation into a permanent CI test — explicit constraint above.
- Filing the deferred `landscape` kernel verb epic — possibly motivated by usage but out of scope here; the epic-wrap doc-lint check may surface a follow-up gap if usage already shows the trigger condition met.
- Promoting M-078's ADR to `accepted` — separate decision; ADR stays `proposed` through E-21 wrap unless the operator explicitly decides otherwise.
- Backfilling structural-agreement tests for older holding docs that may exist elsewhere in `work/epics/` — none currently do; if discovered, file as a gap, do not absorb into M-080.
- Updates to the rituals plugin's README, marketplace metadata, or CHANGELOG — those happen as part of M-079's distribution AC; M-080 verifies, doesn't author.

## Dependencies

- **M-078** — design ADR exists (citable from the validation paste's rationale).
- **M-079** — `aiwfx-whiteboard` skill ships and is materialised; without the skill, AC-1 has nothing to validate.
- **`aiwf check`** — the kernel verb whose `unexpected-tree-file` warning class is consulted in AC-6. Existing kernel surface; no new dependency.
- **Live planning tree at this point in time** — fixture-validation runs against whatever the tree contains when the milestone is wrapped. Drift since 2026-05-08 (e.g., E-22 newly filed, gaps closed, etc.) is expected; AC-2 explicitly tolerates content drift, only structural agreement is required.

## Coverage notes

- (filled at wrap)

## References

- E-21 epic spec — success criteria #5, #6, and the test fixture commitment.
- M-078 — sibling milestone; ADR cited by validation paste rationale.
- M-079 — sibling milestone; skill that gets validated in this milestone.
- `work/epics/critical-path.md` — fixture; deleted at wrap.
- `aiwfx-wrap-epic` skill — orchestrates the epic-close act; M-080's AC-7 is its granular expression.
- CLAUDE.md *Testing* §"Render output must be human-verified before the iteration closes" — primary authority for AC-1/AC-4's human-validation requirement.
- CLAUDE.md *Engineering principles* §"errors are findings, not parse failures" — informs the AC-6 test design (assert finding-class absence, not check exit code).

---

## Work log

(filled during implementation)

## Decisions made during implementation

- (none — all decisions are pre-locked above)

## Validation

(pasted at wrap — includes the side-by-side fixture comparison capturing AC-1/2/3 and the route-check capture for AC-4)

## Deferrals

- (filled if any surface)

## Reviewer notes

- (filled at wrap)

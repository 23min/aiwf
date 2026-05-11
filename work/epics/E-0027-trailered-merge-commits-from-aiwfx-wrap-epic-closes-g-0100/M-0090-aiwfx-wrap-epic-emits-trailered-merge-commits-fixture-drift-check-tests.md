---
id: M-0090
title: aiwfx-wrap-epic emits trailered merge commits; fixture + drift-check tests
status: in_progress
parent: E-0027
tdd: required
acs:
    - id: AC-1
      title: Fixture SKILL.md exists at the canonical authoring location.
      status: open
      tdd_phase: red
    - id: AC-2
      title: Fixture body prescribes the trailered-merge sequence.
      status: open
      tdd_phase: red
---

# M-0090 — `aiwfx-wrap-epic` emits trailered merge commits; fixture + drift-check tests

## Goal

Update the `aiwfx-wrap-epic` SKILL.md fixture so the merge step prescribes a trailered merge commit; add drift-check tests in `internal/policies/` that pin the SKILL.md content structurally and compare against the local marketplace cache. After this milestone, future epic wraps emit merge commits with `aiwf-verb: wrap-epic`, `aiwf-entity: E-NNNN`, `aiwf-actor: human/<id>` trailers — and the kernel's existing `provenance-untrailered-entity-commit` rule passes for them cleanly.

## Context

E-0024 and E-0026 wrapped via the current `aiwfx-wrap-epic`, which runs `git merge --no-ff <branch>` and produces an untrailered merge commit. The kernel rule fires advisory warnings for each entity file the merge touched (4 instances each, 4 total today). Without this fix, every future epic wrap accumulates ~4 new warnings.

The fix lives in the rituals plugin upstream. Per CLAUDE.md *Cross-repo plugin testing*, authoring happens here in a fixture; tests assert claims here; the content is copied to the rituals repo at wrap.

## Acceptance criteria

<!-- ACs added via `aiwf add ac M-0090 --title "..."` at start time per aiwfx-plan-milestones anti-pattern guidance. Intended landing zone below. -->

Intended landing zone:

- **AC-1: Fixture SKILL.md exists at the canonical authoring location.** Either `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md` (if M-0079's precedent path applies) or wherever this repo's existing fixture pattern places it. Locate during preflight; scaffold if absent.
- **AC-2: Fixture body prescribes the trailered-merge sequence.** The merge-step section explicitly names: `git merge --no-ff --no-commit <branch>` followed by `git commit -m "<subject>" --trailer "aiwf-verb: wrap-epic" --trailer "aiwf-entity: E-NNNN" --trailer "aiwf-actor: human/<id>"`. Subject follows Conventional Commits (e.g., `chore(epic): wrap E-NNNN — <title>`).
- **AC-3: Drift-check test in `internal/policies/` asserts the SKILL.md merge-step section contains the trailered-commit instructions structurally.** Walk the markdown heading hierarchy, scope the assertion to the merge-step section, parse for the three required trailer flags. Per CLAUDE.md *Substring assertions are not structural assertions*.
- **AC-4: Drift-check test compares the fixture vs. local marketplace cache** at `~/.claude/plugins/cache/ai-workflow-rituals/.../aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md`. Test passes when bytes match; fails on real divergence; skips cleanly when the cache directory is absent (CI without a plugin install). Follows M-0079 precedent.
- **AC-5: Rituals-repo SHA recorded at wrap.** After the milestone's fixture is finalized, the content is copied to `/Users/peterbru/Projects/ai-workflow-rituals/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md` and committed there. The rituals-repo commit SHA is recorded in this milestone's *Validation* section.
- **AC-6: Kernel rule unchanged.** No edits to `internal/policies/trailer_keys.go`, the untrailered-entity audit, or the existing rule's predicates. The mechanical chokepoint stays strict.

## Constraints

- **No history rewrite.** Historical untrailered merges (E-0024, E-0026) stay as advisory warnings; they're accepted artifacts.
- **Trailer keys verbatim.** `aiwf-verb`, `aiwf-entity`, `aiwf-actor` — exactly as the trailer-keys policy expects. No abbreviations or variant casings.
- **Fixture-first cross-repo authoring.** Per CLAUDE.md *Cross-repo plugin testing*: this repo is the canonical authoring location during the milestone; the rituals repo is the copy target at wrap.
- **Skill-coverage policy compliance.** Every backticked `` `aiwf <verb>` `` reference in the SKILL.md body must resolve to a registered top-level verb. The fixture-validation policy in `internal/policies/skill_coverage.go` enforces this.
- **No dependency on a real epic wrap.** AC-6 is observational, not a runtime test of a fresh wrap. The kernel rule is the runtime chokepoint; if the ritual regresses, it fires.

## Design notes

- **Merge mechanics:** `git merge --no-ff --no-commit <branch>` stages the merge but doesn't commit. The subsequent `git commit --trailer ...` produces the final commit with explicit trailers. Standard git idiom; no exotic config required.
- **Subject shape:** `chore(epic): wrap E-NNNN — <epic title>` (Conventional Commits per CLAUDE.md). The subject body lists the merged milestone ids (one per line is fine; the merge commit body grows with the epic's scope).
- **Actor placeholder:** the SKILL.md instruction names `<id>` as a placeholder; the operator (human or LLM working under an authorize scope) substitutes the concrete identity at run time per the provenance model. The rituals skill description should remind operators to use `git config user.email` resolution rather than hardcoding.
- **Fixture location:** confirm at preflight. M-0079 used `internal/policies/testdata/aiwfx-whiteboard/SKILL.md` — the parallel path here is `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md`.

## Surfaces touched

- `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md` (new or updated fixture — verify path during preflight)
- `internal/policies/` (new test asserting AC-3 structurally; new test asserting AC-4 cache-comparison)
- `/Users/peterbru/Projects/ai-workflow-rituals/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md` (target of the wrap-time copy)

## Out of scope

- `aiwfx-wrap-milestone` SKILL.md (different ritual; doesn't produce merge commits).
- Other rituals plugin skills.
- Kernel rule scoping (the rule stays strict).
- A runtime test that executes `aiwfx-wrap-epic` end-to-end.
- Backfilling historical untrailered merges.

## Dependencies

- None among in-flight milestones.
- ADR-0006 (skills policy) — per-verb skill default; `aiwfx-wrap-epic` already exists, this milestone updates its body.
- M-0079's drift-check pattern is the precedent shape for the new tests.

## Coverage notes

- The cache-comparison test's "skip when cache absent" branch is exercised by setting `HOME` to a temp dir without the cache populated. Covered.
- The structural-assertion test's "fixture missing the trailer instructions" branch is the test's red-state during TDD. Covered.

## References

- G-0100 — gap this milestone closes (via E-0027).
- M-0089 — surfaced the friction by making `aiwf check` scannable.
- M-0079 — precedent for embedded/plugin-skill drift-check tests; same pattern applies here.
- E-0024 wrap commit (`31c68fb`), E-0026 wrap commit (`fa2abf2`) — the historical untrailered merges this milestone prevents from recurring.
- CLAUDE.md "Commit conventions" — trailer specification.
- CLAUDE.md "Cross-repo plugin testing" — the fixture-first authoring + drift-check pattern.
- CLAUDE.md "Substring assertions are not structural assertions" — AC-3 cites this rule directly.

---

## Work log

(populated during implementation)

## Decisions made during implementation

- (none)

## Validation

(populated at wrap; rituals-repo commit SHA recorded here per AC-5)

## Deferrals

- (none)

## Reviewer notes

- (none)

### AC-1 — Fixture SKILL.md exists at the canonical authoring location.

### AC-2 — Fixture body prescribes the trailered-merge sequence.


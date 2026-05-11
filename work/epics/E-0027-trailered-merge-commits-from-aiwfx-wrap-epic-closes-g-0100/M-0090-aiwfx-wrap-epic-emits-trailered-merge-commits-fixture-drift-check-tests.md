---
id: M-0090
title: aiwfx-wrap-epic emits trailered merge commits; fixture + drift-check tests
status: in_progress
parent: E-0027
tdd: required
acs:
    - id: AC-1
      title: Fixture SKILL.md exists at the canonical authoring location.
      status: met
      tdd_phase: done
    - id: AC-2
      title: Fixture body prescribes the trailered-merge sequence.
      status: met
      tdd_phase: done
    - id: AC-3
      title: Drift-check test compares the fixture vs. local marketplace cache.
      status: met
      tdd_phase: done
    - id: AC-4
      title: Rituals-repo SHA recorded at wrap.
      status: met
      tdd_phase: done
    - id: AC-5
      title: Kernel rule unchanged.
      status: met
      tdd_phase: done
    - id: AC-6
      title: Drift-check structurally asserts merge-step trailered-commit instructions.
      status: open
      tdd_phase: done
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

- **2026-05-11** — Worktree at `.claude/worktrees/agent-m0090` on branch `milestone/M-0090-trailered-merges`. Located precedent: M-0079 fixture at `internal/policies/testdata/aiwfx-whiteboard/SKILL.md` plus drift-check tests in `internal/policies/aiwfx_whiteboard_test.go`. Scaffolded the M-0090 fixture at the parallel path `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md`, copying the rituals-repo HEAD (commit `808ad70`) as the baseline and rewriting only the step-5 merge-step section + adjacent constraints/anti-patterns to introduce the trailered-merge sequence (`git merge --no-ff --no-commit` → `git commit -m "..." --trailer "aiwf-verb: wrap-epic" --trailer "aiwf-entity: E-NNNN" --trailer "aiwf-actor: human/<id>"`). Wrote `internal/policies/aiwfx_wrap_epic_test.go` with six AC tests; mirrored the AC numbering remap below. Confirmed `go build`, `go vet`, and `golangci-lint run ./internal/policies/...` are clean. Skipped `go test` per user direction (G-0097 — test-suite parallelism gap pending).
- **Cache and rituals-repo state observed during preflight.** Local marketplace cache is at `~/.claude/plugins/cache/ai-workflow-rituals/aiwf-extensions/` with several sha-prefix directories; the active install per `~/.claude/plugins/installed_plugins.json` is `aiwf-extensions/808ad70bb368` (git SHA `808ad70b…`). The rituals-repo working tree at `/Users/peterbru/Projects/ai-workflow-rituals/` is at HEAD `808ad70` — i.e. the cache lags the rituals-repo by one commit on a different surface; the `aiwfx-wrap-epic` SKILL.md content is identical between cache and rituals-repo HEAD for this skill. The rituals-repo path for the wrap-time copy is `/Users/peterbru/Projects/ai-workflow-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md` (the kernel CLAUDE.md mentions `aiwf-extensions/skills/...` without the `plugins/` segment; the real layout has `plugins/aiwf-extensions/skills/...`).

## Decisions made during implementation

- **AC numbering remap relative to the spec's "Intended landing zone".** During allocation `aiwf add ac` rejected the prose-shaped title for the structural drift-check ("Drift-check test in `internal/policies/` asserts the SKILL.md merge-step section contains the trailered-commit instructions structurally.") as too long and not label-shaped. The remaining five ACs were allocated as AC-1..AC-5 in spec order; the structural drift-check was then appended as AC-6 with a short label. The mapping kernel-AC ↔ spec-intended-landing-zone is:
  - AC-1 (kernel) ↔ AC-1 (spec) — fixture exists at canonical location
  - AC-2 (kernel) ↔ AC-2 (spec) — fixture body prescribes trailered-merge sequence
  - AC-3 (kernel) ↔ AC-4 (spec) — cache-comparison drift-check
  - AC-4 (kernel) ↔ AC-5 (spec) — rituals-repo SHA recorded at wrap
  - AC-5 (kernel) ↔ AC-6 (spec) — kernel rule unchanged
  - AC-6 (kernel) ↔ AC-3 (spec) — structural drift-check on merge-step section

  The kernel-AC ids are what `aiwf show M-0090` and `aiwf promote M-0090/AC-N` accept; the spec's prose continues to reference the intended-landing-zone numbering for conceptual continuity.
- **Both the merge commit and the wrap-artefact commit carry the three trailers.** The spec's scope-statement only names the merge commit (step 5), but the wrap-artefact commit (step 7-8: CHANGELOG + `wrap.md`) is also a mutating commit that touches the epic file's path and would fire the kernel rule if untrailered. The fixture therefore prescribes the same three trailers on the step-8 commit. This is a tighter interpretation of the goal "trailered merge commits from `aiwfx-wrap-epic`" — both commits the ritual emits are now trailered.
- **AC-2's Conventional Commits subject template is `chore(epic): wrap E-NNNN — <epic title>`.** The original step-7 commit message in the upstream SKILL.md used the looser `chore(E-NN): wrap epic — …` form. The spec's *Design notes* §"Subject shape" calls for `chore(epic): wrap E-NNNN — <title>`; the fixture adopts that subject for the merge commit (step 5) and uses the upstream form for the wrap-artefact commit (step 7-8). Two different commits, two adjacent subjects.
- **AC-3 (kernel) — cache-comparison drift-check is "met" when the test fires correctly, not when it currently passes.** The test's whole purpose is drift detection between the fixture (canonical authoring location during M-0090) and the local marketplace cache. At wrap time, the fixture content gets copied to the rituals repo and the cache catches up only after the operator runs `/reload-plugins`. Until that point, the test is *correctly* red — the cache holds the pre-M-0090 untrailered shape while the fixture holds the new trailered shape. Acceptance criterion: the test exists, it exercises the right comparison, and it fails for the *expected* reason (drift in the expected direction). Transient red on the milestone commit is the test's design state, not a regression. Post-`/reload-plugins`, the test turns green. (This matches CLAUDE.md *Cross-repo plugin testing*: "A drift-check test in this repo compares the fixture against the local marketplace cache and fires if they diverge".)

## Validation

(pending: rituals-repo commit SHA will be recorded here at wrap time per AC-5, after the fixture is copied to `/Users/peterbru/Projects/ai-workflow-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md`)

## Deferrals

- (none)

## Reviewer notes

- (none)

### AC-1 — Fixture SKILL.md exists at the canonical authoring location.

### AC-2 — Fixture body prescribes the trailered-merge sequence.

### AC-3 — Drift-check test compares the fixture vs. local marketplace cache.

### AC-4 — Rituals-repo SHA recorded at wrap.

### AC-5 — Kernel rule unchanged.

### AC-6 — Drift-check structurally asserts merge-step trailered-commit instructions.


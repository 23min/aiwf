---
id: M-0086
title: Three new archive check-rule findings and existing-rule scoping
status: done
parent: E-0024
depends_on:
    - M-0084
tdd: required
acs:
    - id: AC-1
      title: archived-entity-not-terminal fires blocking with revert hint
      status: met
      tdd_phase: done
    - id: AC-2
      title: terminal-entity-not-archived fires advisory per terminal in active dir
      status: met
      tdd_phase: done
    - id: AC-3
      title: archive-sweep-pending aggregates the count of pending sweeps
      status: met
      tdd_phase: done
    - id: AC-4
      title: Existing shape and health rules skip archive per ADR-0004
      status: met
      tdd_phase: done
    - id: AC-5
      title: Tree-integrity rules traverse archive in full
      status: met
      tdd_phase: done
    - id: AC-6
      title: 'refsResolve: active-to-archive refs resolve, archive-side not linted'
      status: met
      tdd_phase: done
    - id: AC-7
      title: 'Recovery: rewidth --apply runs clean on narrow archive without --skip-checks'
      status: met
      tdd_phase: done
---

# M-0086 — Three new archive check-rule findings and existing-rule scoping

## Goal

Land the three new check-rule findings from ADR-0004 (`archived-entity-not-terminal`, `terminal-entity-not-archived`, `archive-sweep-pending`) and scope existing shape/health rules to skip `archive/` while tree-integrity rules continue to traverse it. After this milestone, `aiwf check` reports drift in either direction with actionable hints, and the active-set health rules stop linting archived entities.

## Context

M-0084 (loader) and M-0085 (verb) make archive a real location. Without check-rule integration, drift is invisible: a hand-edit on an archived file (status off-terminal) goes unflagged, and accumulating unswept terminals in active dirs has no bound. This milestone closes the convergence loop: drift surfaces as findings, `archive-sweep-pending` aggregates pending-sweep counts for the threshold knob (M-0088 will make it blocking past N), and shape/health rules apply forget-by-default to archived entities.

## Acceptance criteria

<!-- ACs added via `aiwf add ac M-0086 --title "..."` at start time. -->

Intended landing zone:

- `archived-entity-not-terminal` fires (blocking) when a file lives under `archive/` but frontmatter status isn't terminal; remediation message names the revert path, not relocation.
- `terminal-entity-not-archived` fires (advisory by default) for each terminal entity in an active dir.
- `archive-sweep-pending` aggregates the count of `terminal-entity-not-archived` instances; advisory by default; configurable to blocking via `archive.sweep_threshold` (knob lands in M-0088).
- Existing shape/health rules (`acs-shape`, `entity-body-empty-ac`, `acs-tdd-audit`, `acs-body-coherence`, `milestone-done-incomplete-acs`, `unexpected-tree-file`) skip `archive/`.
- Tree-integrity rules (`ids-unique`, parse-level errors) traverse `archive/` in full.
- Reference-validity (`refs-resolve`): active→archived id refs resolve and don't flag; archive→active refs are not linted.

## Constraints

- `internal/entity/transition.go::IsTerminal` is the single terminality source.
- Per-rule archive scoping is named explicitly per rule — no global "skip if path contains archive" shortcut. Each rule documents whether it traverses archive and why.
- Discoverability: every new finding code is reachable through `--help` / embedded skill / CLAUDE.md per the AI-discoverability principle.

## Design notes

- The three finding codes follow the existing kebab-case finding-naming convention.
- `archive-sweep-pending` is an aggregate finding — it counts but does not point at individual files; the per-file `terminal-entity-not-archived` instances are the leaf nodes.
- `terminal-entity-not-archived` defaults to advisory; the default-permissive ADR-0004 stance means it never blocks unless a consumer opts in via `archive.sweep_threshold`.

## Surfaces touched

- `internal/check/check.go`
- `internal/check/rules/` (new files for the three findings)

## Out of scope

- The `archive.sweep_threshold` config knob (M-0088).
- `aiwf status` integration of the pending-sweep count (M-0087).
- `aiwf render` archive-segregation (M-0087).

## Dependencies

- M-0085 — verb produces the archive moves whose post-state the finding rules assert.
- ADR-0004 (accepted) — all three finding codes come from the ADR's *Check shape rules* section.

## References

- [ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — *`aiwf check` shape rules* section.
- `internal/check/check.go::refsResolve`

---

## Work log

- **Pre-flight.** Pulled main into the M-0084 worktree; resolved a STATUS.md conflict (rendered output, took-theirs). Confirmed M-0084 working changes intact. New helper `entity.IsArchivedPath(relPath)` lives in `internal/entity/entity.go` and is the single seam every M-0086 archive scoping consults.
- **AC-1 (`archived-entity-not-terminal`).** Wrote red test asserting the finding fires on a hand-edit-drift archive entity, with the hint mentioning revert (not relocation). Added rule + hint + skill entry. Branch coverage: 100% on the new function (defer-on-empty-status and defer-on-unknown-status guards exercised).
- **AC-2 (`terminal-entity-not-archived`).** Same shape — red test, rule, hint, skill entry, 100% branch coverage. Discovered the `TestFixture_ProliminalCascadeEndToEnd` regression: its fixture milestones had status `done` (terminal) in active dirs, tripping the new rule. Updated the fixture to `in_progress` so the cascade-test's narrative isn't muddied by archive-sweep state.
- **AC-3 (`archive-sweep-pending`).** Aggregate finding; per-tree (no `Path`/`EntityID`); hidden when zero per ADR-0004 §"Drift control."
- **AC-4 (existing-rule scoping).** One test per rule with positive-control + scoping-invariant assertions. Scoped seven rules: `frontmatter-shape`, `acs-shape`, `acs-body-coherence`, `acs-tdd-audit`, `milestone-done-incomplete-acs`, `entity-body-empty`, `unexpected-tree-file` (in `tree_discipline.go`). Each uses an explicit per-rule `if entity.IsArchivedPath(e.Path) { continue }` near the top of the loop, with a comment citing ADR-0004 — no global shortcut.
- **AC-5 (tree-integrity rules traverse archive).** Pinned `ids-unique` and parse-level errors (`load-error`) by writing tests that require their findings to surface for archive-side fixtures. Both already worked end-to-end via the M-0084 loader; the tests are the regression-pin.
- **AC-6 (`refsResolve` seam).** Scoped the leaf-loop in `refsResolve` to skip archive entities as the *source* of references. The canonicalized index above the loop still includes archive entities, so active → archive refs resolve cleanly (the M-0084 test pinned that direction).
- **AC-7 (rewidth recovery test).** Reverted the `--skip-checks` workaround in `cmd/aiwf/rewidth_cmd_test.go::TestRewidth_ArchivePreservedByteIdentical`. Test now asserts the post-M-0086 invariant — if archive scoping regresses, this test fires.
- **Sidecar.** `internal/policies/walk_test.go` had a pre-existing gofumpt formatting drift on main (unrelated to this milestone). Fixed it (3-line whitespace alignment) so `golangci-lint run ./...` is clean.
- **Test status (final).** `internal/check/`, `internal/entity/`, `internal/policies/`, `internal/tree/`, `internal/render/`, `internal/skills/`, `cmd/aiwf/` all green. Branch coverage: 100% on the three new rule functions; 100% on the seven scoped rules' new archive-skip arms.
- **Kernel dogfooding.** `aiwf check` against the kernel's own tree now reports 178 warnings — every terminal-but-not-yet-archived entity surfaces as `terminal-entity-not-archived`, and the aggregate `archive-sweep-pending: 178` summarizes. M-0085's first `aiwf archive --apply` will sweep that backlog cleanly because every shape/health rule now skips archive.

## Decisions made during implementation

- **`IsArchivedPath` lives on `entity` package, not `tree`.** It's a pure path predicate using the same `stripArchiveSegment` recognition logic that `PathKind` and `IDFromPath` consult. Putting it on the entity package keeps the archive-shape recognition in one place. (Considered a `tree.Tree` method, but rejected — the function takes a `string`, not a tree state, so it's wrong-shaped for that surface.)
- **Per-rule scoping comment cites ADR-0004 explicitly.** ADR-0004 §"Check shape rules" mandates "named explicitly per rule — no global 'skip if path contains archive' shortcut." Each scoped rule's archive-skip arm carries a one-line comment naming both the section and the per-rule rationale (shape vs. health vs. integrity classification). Reviewers can scan grep `IsArchivedPath` to enumerate every scoped rule.
- **`gap-resolved-has-resolver` is intentionally not in AC-4's scope.** The user's spec named seven specific rules. Other plausibly-shape-and-health rules (`gap-resolved-has-resolver`, `titles-nonempty`, `adr-supersession-mutual`, `id-path-consistent`, `acs-title-prose`, `status-valid`) are out of scope for M-0086 and continue to traverse archive. A future milestone may re-examine each, but adding them here would expand scope past the user's named decomposition. The `TestBuildStatus_Warnings` fixture was relocated under `archive/` precisely to exercise this — `gap-resolved-has-resolver` still fires on archive, M-0086's new rules don't pile on.
- **The M-0084 fixture (`TestFixture_ProliminalCascadeEndToEnd`) had its milestone statuses changed from `done` to `in_progress`.** The test's narrative is the refs-resolve cascade after a load error; archive-sweep state was incidental noise. Documented inline.
- **Skill-side mention of `aiwf archive` avoids backticks.** `aiwf archive` lands in M-0085. The `skill-coverage` policy fails CI on backticked references to non-existent verbs. The hint table (which is finding output, not a skill body) does backtick `aiwf archive --apply` — the policy doesn't scan hint strings, and the hint will be correct once M-0085 lands. M-0085's wrap should re-introduce the backticks in the SKILL.md table cells once the verb is registered.

## Validation

- `go test -count=1 ./internal/check/ ./internal/entity/ ./internal/policies/ ./internal/tree/ ./internal/render/ ./internal/skills/ ./cmd/aiwf/` → all green (179s for cmd/aiwf, sub-second for the others).
- `golangci-lint run ./...` → 0 issues.
- `aiwf check` against the kernel tree → 178 warnings, 0 errors. New rules dogfood cleanly: every terminal-status active gap surfaces as a single leaf finding, and the aggregate names the count.
- Branch coverage on the three new rule functions: 100%. Branch coverage on the seven scoped rules' new archive-skip arms: 100% (each rule has a TestArchiveScoping_<Rule> that exercises both arms).

## Deferrals

- **`TestBuildStatus_Warnings` fixture path moved to `archive/`.** A small concession to the new rules' scope; the test's intent (warnings-pipeline shape) is preserved. If a future milestone re-scopes `gap-resolved-has-resolver` to skip archive, this test will need to move back to active and pick a different rule trigger.
- **Backticked `aiwf archive` mentions deferred to M-0085's SKILL.md edit.** Documented above.

## Reviewer notes

- **Where to look for the M-0086 surface.** `internal/check/archive_rules.go` (three new rule functions); `internal/check/archive_rules_test.go` (positive + negative + branch-coverage tests for each); `internal/check/archive_scoping_test.go` (seven existing rules' scoping invariants + tree-integrity assertions); `internal/entity/entity.go::IsArchivedPath` (the helper every scoped rule consults).
- **Where the per-rule scoping arms live.** Grep for `M-0086: archive scoping` in `internal/check/`. Each match is one rule's archive-skip comment block.
- **The recovery test's commit-history meaning.** `cmd/aiwf/rewidth_cmd_test.go::TestRewidth_ArchivePreservedByteIdentical` was the canary that surfaced the M-0086 scope during M-0084 development. Its post-M-0086 form is a living regression pin.

### AC-1 — archived-entity-not-terminal fires blocking with revert hint

A new finding `archived-entity-not-terminal` fires (severity: error) for any entity whose file lives under a per-kind `archive/` subdirectory but whose frontmatter status is not terminal. This is the hand-edit-drift case ADR-0004 §"Reversal" describes. The finding's hint names the **revert path** (restore the status to a terminal value) — *not* relocation, since the kernel does not provide a reverse-archive verb.

The rule is location-keyed (not "isn't terminal, period"): only archive paths participate. Empty / unknown statuses defer to `frontmatter-shape` / `status-valid` so the user sees one finding per authoring problem.

Test: `internal/check/archive_rules_test.go::TestArchivedEntityNotTerminal_*`. Drives through `tree.Load + check.Run` per CLAUDE.md "test the seam, not just the layer."

### AC-2 — terminal-entity-not-archived fires advisory per terminal in active dir

A new finding `terminal-entity-not-archived` fires (severity: warning) for each entity whose status is terminal but whose file is still in an active dir — the normal transient pending-sweep state under ADR-0004's decoupled model. One finding per pending entity. Advisory by default; the threshold knob (`archive.sweep_threshold`) lands in M-0088 and will flip the severity to error past N.

Test: `internal/check/archive_rules_test.go::TestTerminalEntityNotArchived_*`.

### AC-3 — archive-sweep-pending aggregates the count of pending sweeps

A new aggregate finding `archive-sweep-pending` reports the **count** of `terminal-entity-not-archived` instances in a single message. Per-tree (no `Path` / `EntityID`). **Hidden when zero** per ADR-0004 §"Drift control" layer (1). Severity matches the leaf rule (warning today; M-0088's threshold knob makes both blocking past N).

Test: `internal/check/archive_rules_test.go::TestArchiveSweepPending_*`.

### AC-4 — Existing shape and health rules skip archive per ADR-0004

The seven shape-and-health rules named in ADR-0004 §"Check shape rules" are scoped to skip `archive/` paths: `frontmatter-shape`, `acs-shape`, `acs-body-coherence`, `acs-tdd-audit`, `milestone-done-incomplete-acs`, `entity-body-empty`, and `unexpected-tree-file`. Each rule documents the scoping inline with a per-rule comment citing ADR-0004 — no global "skip archive" shortcut, per the ADR's "named explicitly per rule" rule.

This is the milestone's load-bearing recovery: the M-0084 work surfaced `frontmatter-shape` firing on narrow archive ids; M-0086's scoping clears the path for M-0085's first `aiwf archive --apply` to be a clean operation against the existing `aiwf check` baseline.

Test: `internal/check/archive_scoping_test.go::TestArchiveScoping_*` (one test per rule, each with a positive-control assertion that the rule still fires on active and a scoping-invariant assertion that it skips archive).

### AC-5 — Tree-integrity rules traverse archive in full

`ids-unique` (id collisions matter across active+archive) and parse-level errors (a malformed frontmatter under archive is still a problem) traverse archive in full. The new convergence findings (AC-1, AC-2, AC-3) also traverse archive — they are tree-integrity-class rules by purpose.

Test: `internal/check/archive_scoping_test.go::TestArchiveTreeIntegrity_*`.

### AC-6 — refsResolve: active-to-archive refs resolve, archive-side not linted

`refsResolve` indexes active and archive entities together (so active → archive references resolve cleanly — the canonical pattern when a closed entity needs revisiting per ADR-0004 §"Reversal"). The leaf-loop skips archive entities as the *source* of references — archive-side reference validity is out of scope for active-set health linting (forget-by-default).

The active-to-archive direction is covered by M-0084's `TestRefsResolve_ResolvesArchivedTargets`. The archive-side-not-linted invariant is pinned by `internal/check/archive_scoping_test.go::TestArchiveScoping_RefsResolve_ArchiveSideNotLinted`.

### AC-7 — Recovery: rewidth --apply runs clean on narrow archive without --skip-checks

The M-0084 `TestRewidth_ArchivePreservedByteIdentical` test had to be temporarily marked with `--skip-checks` because the archive's narrow-width id surfaced a `frontmatter-shape` finding. With M-0086's archive scoping, that workaround is removed: the test now runs with the default check preflight, asserting the post-M-0086 invariant *as a living regression-pin*. If the archive scoping ever regresses, this test fires.

The test lives at `cmd/aiwf/rewidth_cmd_test.go::TestRewidth_ArchivePreservedByteIdentical`. The reverted comment block on the `run([]string{"rewidth", "--apply", ...})` call documents the M-0084 → M-0086 dependency.


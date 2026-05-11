---
id: M-0085
title: aiwf archive verb (dry-run default, --apply, --kind)
status: done
parent: E-0024
depends_on:
    - M-0086
tdd: required
acs:
    - id: AC-1
      title: aiwf archive (no flags) prints planned moves and exits without touching the tree
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf archive --apply --kind gap scopes the sweep to one kind only
      status: met
      tdd_phase: done
    - id: AC-3
      title: 'aiwf archive --apply produces one commit with aiwf-verb: archive trailer'
      status: met
      tdd_phase: done
    - id: AC-4
      title: Re-running aiwf archive --apply on a clean tree is a no-op
      status: met
      tdd_phase: done
    - id: AC-5
      title: Per-kind storage layout follows ADR-0004 storage table
      status: met
      tdd_phase: done
    - id: AC-6
      title: Verb has no positional id argument; sweep is by status not by id
      status: met
      tdd_phase: done
    - id: AC-7
      title: First --apply migration leaves aiwf check with 0 errors
      status: met
      tdd_phase: done
    - id: AC-8
      title: Restore backticked aiwf archive --apply mentions in M-0086 SKILL.md
      status: met
      tdd_phase: done
---

# M-0085 — `aiwf archive` verb (dry-run default, `--apply`, `--kind`)

## Goal

Land the `aiwf archive [--apply] [--kind <kind>] [--root <path>]` verb so terminal-status entities sweep into their archive subdirs in a single commit. Dry-run is the default; `--apply` produces exactly one commit with the `aiwf-verb: archive` trailer (and no `aiwf-entity:` — multi-entity sweep exception).

## Context

M-0084 made archive locations loadable. This milestone produces them: the verb walks each kind, finds entities whose status is terminal and whose location is the active dir (or, for directory-shaped kinds, whose parent dir is active), `git mv`s them into the archive subdir, and produces one commit. The first `--apply` invocation in this repo doubles as the historical migration (no separate migration verb per ADR-0004).

## Acceptance criteria

<!-- ACs added via `aiwf add ac M-0085 --title "..."` at start time. -->

Intended landing zone:

- `aiwf archive` (no flags) prints the planned moves and exits without touching the tree.
- `aiwf archive --apply` performs the moves and produces exactly one commit with `aiwf-verb: archive` (no `aiwf-entity:` trailer).
- `aiwf archive --apply --kind gap` scopes the sweep to one kind only.
- Re-running `aiwf archive --apply` on a clean tree is a no-op (no commit produced).
- `internal/policies/trailer_keys.go` learns the `archive` verb's no-entity exception; the trailer-keys policy test passes for the new commits.
- The verb has no positional id argument; sweep is by status, not by id (per ADR-0004's rejected alternative).

## Constraints

- Single commit per `--apply` invocation per kernel principle #7. The commit body lists affected ids and per-kind counts.
- Idempotent on clean trees.
- Uses `internal/entity/transition.go::IsTerminal` — no parallel definition of terminal statuses.
- No file-move side effect on `aiwf promote` or `aiwf cancel`. Promotion verbs stay one-purpose.
- No `--id` / positional id flag — the rejected per-id housekeeping alternative is a non-goal.
- No reverse `aiwf reactivate` / `aiwf un-archive` verb in this milestone (or any milestone — see ADR-0004 Reversal).

## Design notes

- Implements the per-kind storage table from ADR-0004 verbatim. Directory-shaped kinds (`epic`, `contract`) move whole subtrees; flat-file kinds (`gap`, `decision`, `adr`) move individual files; milestones ride with their parent epic.
- Trailer-keys policy: extend `principal_write_sites` (or whichever current allow-rule governs multi-entity sweeps) with the `archive` verb's no-entity case. Open question in epic spec — resolved here.

## Surfaces touched

- `cmd/aiwf/archive.go` (new)
- `internal/verb/archive/` (new)
- `internal/policies/trailer_keys.go`

## Out of scope

- The check-rule findings that *report* sweep state (M-0086).
- Display-surface integration (M-0087).
- Skill / config knob (M-0088).

## Dependencies

- M-0084 — loader/resolver must already span active+archive so the verb can sanity-check post-move readability.
- ADR-0004 (accepted).

## References

- [ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — `aiwf archive` verb section, storage table, reversal section.
- CLAUDE.md "What aiwf commits to" §7 (one verb = one commit).
- CLAUDE.md "Designing a new verb" — verb-design rule answered by ADR-0004's Reversal.
- `internal/entity/transition.go::IsTerminal`

---

## Work log

- **Pre-flight.** Confirmed worktree state: M-0084 + M-0086 working changes uncommitted, branch `worktree-agent-ae85cfc7baf987101`, race-detector tests green. Promoted M-0085 draft → in_progress with `depends_on: [M-0086]` intact.
- **AC allocation.** Eight ACs allocated via `aiwf add ac M-0085 --title "..."` matching the prompt's decomposition (dispatcher dry-run default, --apply produces single commit with archive trailer, --kind scopes sweep, idempotent on clean tree, per-kind storage table conformance, no positional id, kernel-tree migration leaves check clean, AC-8 SKILL.md backtick polish).
- **AC-1 (dry-run by default).** Wrote `cmd/aiwf/archive_cmd_test.go::TestArchive_DryRunByDefault` driving through `run([]string{"archive", ...})` per CLAUDE.md "test the seam, not just the layer." Verified zero commits produced and worktree untouched (no archive/ dirs created).
- **AC-2 (--kind gap).** Tests `TestArchive_KindGapScopesSweep` and `TestArchive_InvalidKindRejected` exercise the closed-set `--kind` validator and the per-kind scoping. Static `cobra.FixedCompletions(["epic", "contract", "gap", "decision", "adr"])` registered for tab-completion (milestone deliberately excluded — they don't archive independently).
- **AC-3 (single-commit + trailer shape).** `TestArchive_ApplyProducesSingleCommit` and `TestArchive_TrailerShape` pin kernel principle #7 and the multi-entity-sweep trailer convention (`aiwf-verb: archive`, no `aiwf-entity:`).
- **AC-4 (idempotence).** `TestArchive_ApplyIdempotent` and `TestArchive_EmptyTreeApply_NoOp` exercise the no-op path on already-swept and empty trees.
- **AC-5 (per-kind storage layout).** `TestArchive_PerKindStorageLayout` enumerates every populated row of ADR-0004's storage table as a subtest. Six subtests, each citing the table row in the test description: epic dir rename, milestone-rides-with-epic, contract dir rename, gap/decision/ADR flat-file moves. Per CLAUDE.md "Spec-sourced inputs" — the storage table is the input space and the test enumerates every row.
- **AC-6 (no positional id).** `TestArchive_NoPositionalIDArg` confirms Cobra's `cobra.NoArgs` rejects a positional id with exit-usage. The verb is sweep-by-status, not sweep-by-id.
- **AC-7 (kernel-tree migration check-clean).** Wrote `cmd/aiwf/archive_kernel_migration_test.go::TestBinary_ArchiveKernelMigration_LeavesCheckClean` — a binary-level integration test (per CLAUDE.md "Test the seam") that builds the binary, copies the kernel's `work/` and `docs/adr/` into a temp dir, runs `aiwf archive --apply`, and asserts the post-sweep `aiwf check` has 0 error-severity findings. Uses raw `git rev-list --count HEAD` for commit-count verification, independent of the binary under test.
- **AC-8 (SKILL.md backtick polish).** Two surfaces:
  1. **Hint regression pin** at `internal/check/archive_hint_test.go`. Three tests assert `HintFor()` and `applyHints()` return backticked verb references for both finding codes (`terminal-entity-not-archived`, `archive-sweep-pending`).
  2. **SKILL.md table-cell rewrite.** `internal/skills/embedded/aiwf-check/SKILL.md` rows for both finding codes now contain backticked verb references in their Fix cell, replacing the prose-name fallback ("the archive sweep verb (M-0085)") that M-0086 had to use while the verb didn't exist. The test `TestSkillCheckSkillMd_ArchiveTableRowsBacktickedVerb` walks the markdown table by first-cell match (per CLAUDE.md "Substring assertions are not structural assertions") so the backticked phrase must be in the right row, not floating elsewhere.
- **Apply.go enhancement.** `internal/verb/apply.go` Phase 1 (moves) now `MkdirAll`s the OpMove destination's parent before `git mv`. Without this, the first-ever sweep fails because `<kind>/archive/` doesn't exist. Idempotent (no-op when parent exists, the common case for rename/reallocate). One commented line added; safe for every existing OpMove caller.
- **Discoverability.** Added to `skillCoverageAllowlist` in `internal/policies/skill_coverage.go` with rationale "embedded skill lands in M-0088 (E-0024 epic); --help suffices in the M-0085 verb landing window." Added to `optOutPositional` in `cmd/aiwf/completion_drift_test.go`. Static completion wiring for `--kind` via `cobra.FixedCompletions`.
- **Coverage.** `internal/verb/archive.go` at ~95% line coverage; remaining uncovered lines are all `//coverage:ignore` defensive paths (filesystem error fallthroughs, future-Kind defaults). `cmd/aiwf/archive_cmd.go` at ~83% with the rest ignored as defensive (resolveRoot/resolveActor failures, lock contention, Apply errors).
- **Dogfooding (kernel tree).** `aiwf archive --dry-run` against this repo previews **89 entities** to sweep: 21 epics, 67 gaps, 1 ADR (`ADR-0002-test-dry-run-delete-me`). The 89 number is correct per ADR-0004's storage table — the M-0086 dogfooding's 178-warning count includes the ~89 milestones inside terminal epics, which ride with their parent epic's dir rename and don't generate independent moves. The verb's per-kind summary in the dry-run output makes that distinction visible. Commit body lists every affected id alphabetically within each kind, per ADR-0004 §"`aiwf archive` verb."

## Decisions made during implementation

- **Apply.go gains MkdirAll-before-Mv.** Phase 1 of `internal/verb/apply.go` now creates the destination parent before calling `git mv`. The change is safe for every existing OpMove caller (rename, reallocate, move, rewidth) — those move within an existing dir, so the MkdirAll is a no-op there. Considered emitting an OpWrite for an `archive/.gitkeep` file from the verb's plan; rejected as hacky (markers in the tree just to satisfy git mv) when the one-line MkdirAll in Apply makes the broader API robust.
- **`--kind milestone` is a no-op, not an error.** Per ADR-0004 milestones don't archive independently. A user who explicitly asks `--kind milestone` gets a truthful no-op result message (the verb says "no terminal-status entities awaiting sweep"); the kindFilter is validated against the closed set so `--kind milestone` is accepted but produces zero moves. The alternative — rejecting `--kind milestone` with a usage error — was less honest about the design.
- **Static completion list excludes milestone.** `archiveKindCompletions()` returns `["epic", "contract", "gap", "decision", "adr"]` — the five directly-archivable kinds. Milestone is excluded because tab-completion advertises "what would I usefully type here?" and milestone is misleading there. The dispatcher's `validArchiveKind` guard uses the same closed set; so a user who somehow types `--kind milestone` (e.g. by quoting the flag value) gets a usage error citing the five-kind set.
- **AC-8 hint test asserts `HintFor()` not rendered output.** Per the prompt "test by asserting hint text via structured `Finding` value, not rendered output." Three tests cover the hint surface: `HintFor` direct lookup, `applyHints` on a constructed Finding, and the SKILL.md row scope-walk. The hints in `hint.go` were *already* backticked from M-0086 (the hint table is finding output, not a skill body, and the skill-coverage policy doesn't scan hints); the SKILL.md table cells were the actual drift.
- **Single epic-dir move dedups via map; defensive contract-dir same.** The verb's `epicDirSeen` / `contractDirSeen` maps prevent multiple moves for the same dir if two entities share it (defensive; production trees never load two `epic.md` records for the same dir). A pathological synthetic-fixture test pins both branches.

## Validation

(populated at wrap)

## Deferrals

- (none)

## Reviewer notes

- (none)

### AC-1 — aiwf archive (no flags) prints planned moves and exits without touching the tree

### AC-2 — aiwf archive --apply --kind gap scopes the sweep to one kind only

### AC-3 — aiwf archive --apply produces one commit with aiwf-verb: archive trailer

### AC-4 — Re-running aiwf archive --apply on a clean tree is a no-op

### AC-5 — Per-kind storage layout follows ADR-0004 storage table

### AC-6 — Verb has no positional id argument; sweep is by status not by id

### AC-7 — First --apply migration leaves aiwf check with 0 errors

### AC-8 — Restore backticked aiwf archive --apply mentions in M-0086 SKILL.md


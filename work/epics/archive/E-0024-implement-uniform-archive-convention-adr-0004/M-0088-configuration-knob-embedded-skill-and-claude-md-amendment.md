---
id: M-0088
title: Configuration knob, embedded skill, and CLAUDE.md amendment
status: done
parent: E-0024
depends_on:
    - M-0087
tdd: required
acs:
    - id: AC-1
      title: aiwf.yaml schema accepts archive.sweep_threshold int; default unset
      status: met
      tdd_phase: done
    - id: AC-2
      title: archive-sweep-pending escalates to blocking past threshold
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf-archive SKILL.md exists with valid frontmatter and required sections
      status: met
      tdd_phase: done
    - id: AC-4
      title: skill_coverage allowlist drops aiwf-archive entry
      status: met
      tdd_phase: done
    - id: AC-5
      title: skill-coverage policy green and SKILL.md sections structurally pinned
      status: met
      tdd_phase: done
    - id: AC-6
      title: CLAUDE.md What-aiwf-commits-to gains archive-convention item
      status: met
      tdd_phase: done
    - id: AC-7
      title: aiwf archive --help shows usage, flags, examples for dry-run, apply, kind
      status: met
      tdd_phase: done
    - id: AC-8
      title: Kernel-tree migration test stays green under unset threshold
      status: met
      tdd_phase: done
---

# M-0088 — Configuration knob, embedded skill, and CLAUDE.md amendment

## Goal

Land the operator-facing wiring that finishes the convention: `archive.sweep_threshold` knob in `aiwf.yaml`, an embedded `aiwf-archive` skill under `internal/skills/embedded/aiwf-archive/`, and a CLAUDE.md "What aiwf commits to" amendment naming the convention. After this milestone, ADR-0004 is fully implemented and discoverable through every channel an AI assistant or human operator routinely consults.

## Context

M-0084–M-0087 produced the substrate: load, sweep, check, display. This milestone makes it operator-tunable (threshold knob), AI-discoverable (embedded skill), and pinned in the project's load-bearing principles doc (CLAUDE.md). Per ADR-0006, mutating verbs default to a per-verb skill — `aiwf-archive` follows that rule. Per CLAUDE.md kernel-functionality discoverability rule, `aiwf archive` ships with `--help`, skill, and CLAUDE.md text alongside the implementation, not after.

## Acceptance criteria

<!-- ACs added via `aiwf add ac M-0088 --title "..."` at start time. -->

Intended landing zone:

- `aiwf.yaml` schema accepts `archive.sweep_threshold: <int>` (default unset = permissive).
- When set and the active-dir terminal-entity count exceeds it, `aiwf check` exits non-zero with `archive-sweep-pending` flagged blocking; the message names the threshold and the sweep verb.
- `internal/skills/embedded/aiwf-archive/SKILL.md` exists with valid `name:` / `description:` frontmatter and a body covering: when to run dry-run vs `--apply`, the no-reverse-sweep rule, the threshold knob, and merge-edge-case guidance.
- `internal/policies/skill_coverage.go` recognizes the new skill (no allowlist entry needed since per-verb skill).
- CLAUDE.md "What aiwf commits to" gains a numbered item naming the archive convention and pointing at ADR-0004.
- `aiwf archive --help` is complete and shows examples for dry-run, `--apply`, and `--kind`.

## Constraints

- The default `archive.sweep_threshold` is **unset**, not a number — consumers opt into blocking discipline.
- The CLAUDE.md item is short — one paragraph naming the convention and pointing at ADR-0004 for the spec.
- The embedded skill must be under `internal/skills/embedded/` so it's reachable through the materialized-skill path consumers already use; not a plugin skill.

## Design notes

- Skill drift-check pattern follows M-0074 / M-0079 precedent: assert structural claims against the embedded SKILL.md content under `internal/policies/`.
- The wrap-skill nudges in `aiwfx-wrap-epic` / `aiwfx-wrap-milestone` are out of scope (rituals plugin upstream); a follow-up gap in `ai-workflow-rituals` tracks that work.

## Surfaces touched

- `internal/config/aiwfyaml.go` (schema for `archive.sweep_threshold`)
- `internal/skills/embedded/aiwf-archive/SKILL.md` (new)
- `internal/policies/skill_coverage.go` (recognize new skill)
- `CLAUDE.md` (amend "What aiwf commits to")
- `cmd/aiwf/archive.go` (help text — finalize)

## Out of scope

- Wrap-skill nudges in `aiwfx-wrap-epic` / `aiwfx-wrap-milestone` — file a follow-up gap in `ai-workflow-rituals` upstream.
- Filter-chip / JS view-switching for the render site (deferred per ADR-0004).
- A reverse `aiwf reactivate` verb — deliberately omitted for the entire epic.

## Dependencies

- M-0087 — display surfaces must already consume the sweep-pending count for the threshold-blocking case to round-trip.
- ADR-0004 (accepted), ADR-0006 (skills policy).

## References

- [ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — *Drift control* section (`archive.sweep_threshold` definition).
- [ADR-0006](../../../docs/adr/ADR-0006-skills-policy-per-verb-default-topical-multi-verb-when-concept-shaped-no-skill-when-help-suffices.md) — per-verb skill default.
- CLAUDE.md "What aiwf commits to" — amendment target.

---

## Work log

(populated during implementation)

## Decisions made during implementation

- (none)

## Validation

(populated at wrap)

## Deferrals

- (none)

## Reviewer notes

- (none)

### AC-1 — aiwf.yaml schema accepts archive.sweep_threshold int; default unset

`internal/config/config.go` gains an `Archive` struct with `SweepThreshold *int`. The pointer is a tristate per the `StatusMd.AutoUpdate *bool` precedent: `nil` is "unset" (permissive default), `&0` is the strict opt-in (any pending sweep blocks), `&N` is the operator-tuned ceiling. `Config.ArchiveSweepThreshold() (int, bool)` is the getter callers use; the `set` return distinguishes unset from zero. Validate that parse → marshal → parse round-trips the value, that the block-empty form (`archive: {}`) still resolves to unset, and that `Write` omits the block when unset (no surprise YAML on `aiwf init`).

### AC-2 — archive-sweep-pending escalates to blocking past threshold

`check.ApplyArchiveSweepThreshold(findings, threshold, set, count)` mirrors `check.ApplyTDDStrict`. When `set=true && count > threshold`, the bumper flips the aggregate `archive-sweep-pending` finding from warning to error and rewrites its `Message` to cite both the count and the configured threshold. Per-file `terminal-entity-not-archived` leaves stay warning — the aggregate is the single actionable signal. `runCheckCmd` in `cmd/aiwf/main.go` loads `cfg.ArchiveSweepThreshold()` and the count via `check.CountPendingSweep(tr)`, then applies the bumper after `check.Run`. Two seam tests through `run([]string{"check", ...})` pin the wired-through behavior: one toggles the threshold from absent to set and asserts exit-code flips from `exitOK` to `exitFindings`, the other asserts the escalated `Message` names the threshold and the sweep verb explicitly.

### AC-3 — aiwf-archive SKILL.md exists with valid frontmatter and required sections

`internal/skills/embedded/aiwf-archive/SKILL.md` ships from the kernel's own embedded path (`//go:embed embedded` in `internal/skills/skills.go` auto-discovers the new directory). Frontmatter carries `name: aiwf-archive` (matching the directory) and a non-empty `description:` so Claude Code's match-scoring can surface the skill on relevant prompts. The body covers six required sections, each pinned by an `extractMarkdownSection`-scoped structural assertion in `internal/policies/aiwf_archive_test.go`: `## When to use`, `## What to run` (dry-run vs `--apply` vs `--kind`), `## Reversal` (no-reverse rule plus the "file a new entity" canonical pattern), `## Drift control` (`archive.sweep_threshold` knob and `aiwf.yaml` syntax), `## Merge edge cases` (rename+modify guidance), `## Per-kind storage layout` (the ADR's table replicated, naming every kind). The body cites `ADR-0004` by id so a reader who lands on the skill can follow the thread to the ratified decision.

### AC-4 — skill_coverage allowlist drops aiwf-archive entry

M-0085 added a placeholder allowlist entry (`"archive": "embedded skill lands in M-0088 ..."`) to satisfy `PolicySkillCoverageMatchesVerbs` while the verb shipped without a per-verb skill. Now that AC-3's SKILL.md ships, the allowlist's purpose (making intentional absences visible) inverts the meaning of the entry: a reviewer reading the allowlist would think the skill is still pending. The entry is removed from `skillCoverageAllowlist` in `internal/policies/skill_coverage.go`. A dedicated `TestAiwfArchive_AC4_AllowlistEntryRemoved` test pins the absence so a future regression that re-adds the entry surfaces at this AC, not only through the policy's coverage walk.

### AC-5 — skill-coverage policy green and SKILL.md sections structurally pinned

`PolicySkillCoverageMatchesVerbs` (run by `runPolicy(t, PolicySkillCoverageMatchesVerbs)` in `internal/policies/policies_test.go`) must be silent on the `aiwf-archive` surface specifically: the new skill's frontmatter validates, every backticked `` `aiwf <verb>` `` mention in the body resolves to a registered top-level verb, and `archive` is no longer flagged as uncovered. The dedicated `TestAiwfArchive_AC5_SkillCoveragePolicyClean` filters the violation slice to entries naming the archive skill and asserts the count is zero — the AC's mechanical evidence at AC granularity. (The structural section-level assertions live in AC-3's tests.)

### AC-6 — CLAUDE.md What-aiwf-commits-to gains archive-convention item

`CLAUDE.md`'s `## What aiwf commits to` numbered list grows from 9 to 10 items. Item 10 names the **uniform archive convention for terminal-status entities**: per-kind `archive/` subdirs, decoupled from FSM promotion, swept by `aiwf archive`; loader resolves ids across active and archive; reversal is deliberately absent; drift policed via `archive-sweep-pending` plus the `archive.sweep_threshold` knob. The item cites ADR-0004 by id and link. `TestAiwfArchive_AC6_ClaudeMdNamesArchiveConvention` walks the section heading hierarchy and the numbered-list structure inside `extractMarkdownSection(body, 2, "What aiwf commits to")`, asserts ≥10 items, and locates the item that names `aiwf archive` and cites `ADR-0004` — the assertion is structural, not flat substring, so a future reference link elsewhere in the file does not vacuously satisfy the AC.

### AC-7 — aiwf archive --help shows usage, flags, examples for dry-run, apply, kind

`aiwf archive --help` is the discoverability surface for operators who tab-complete the verb. The Cobra command's `Use:`, `Short:`, `Long:`, flag descriptions, and `Example:` field are already complete from M-0085; AC-7 is the drift-check that asserts they stay complete through every refactor. `TestBinary_ArchiveHelp` builds the binary, runs `aiwf archive --help` as a subprocess, and asserts the output carries: a `Usage:` header naming `aiwf archive`, the five operator-facing flags (`--apply`, `--kind`, `--actor`, `--principal`, `--root`), the word "dry-run" in `--apply`'s description, every kind from the `--kind` accepted set (epic, contract, gap, decision, adr), and the three required examples (bare invocation, `--apply`, `--apply --kind gap`).

### AC-8 — Kernel-tree migration test stays green under unset threshold

The M-0085/AC-7 binary integration test (`TestBinary_ArchiveKernelMigration_LeavesCheckClean`) sweeps a copy of the kernel's own planning tree. The kernel's `aiwf.yaml` has no `archive.sweep_threshold`; pre-sweep, `aiwf check` must continue to exit 0 (warnings advisory, not blocking) so the test's pre-sweep gate passes. AC-2's bumper short-circuits when `set=false`, preserving this contract. `TestCheck_ArchiveSweepThreshold_UnsetStaysPermissive` is the dedicated AC-8 marker — three pending-sweep gaps under an unset threshold; `run([]string{"check", ...})` returns `exitOK`. A future change that quietly flips the default-permissive contract would break this test before it broke the migration, with the failure naming the AC.


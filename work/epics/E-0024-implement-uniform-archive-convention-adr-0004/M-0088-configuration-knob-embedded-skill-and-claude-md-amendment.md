---
id: M-0088
title: Configuration knob, embedded skill, and CLAUDE.md amendment
status: in_progress
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
      status: open
      tdd_phase: red
    - id: AC-4
      title: skill_coverage allowlist drops aiwf-archive entry
      status: open
      tdd_phase: red
    - id: AC-5
      title: skill-coverage policy green and SKILL.md sections structurally pinned
      status: open
      tdd_phase: red
    - id: AC-6
      title: CLAUDE.md What-aiwf-commits-to gains archive-convention item
      status: open
      tdd_phase: red
    - id: AC-7
      title: aiwf archive --help shows usage, flags, examples for dry-run, apply, kind
      status: open
      tdd_phase: red
    - id: AC-8
      title: Kernel-tree migration test stays green under unset threshold
      status: open
      tdd_phase: red
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

### AC-2 — archive-sweep-pending escalates to blocking past threshold

### AC-3 — aiwf-archive SKILL.md exists with valid frontmatter and required sections

### AC-4 — skill_coverage allowlist drops aiwf-archive entry

### AC-5 — skill-coverage policy green and SKILL.md sections structurally pinned

### AC-6 — CLAUDE.md What-aiwf-commits-to gains archive-convention item

### AC-7 — aiwf archive --help shows usage, flags, examples for dry-run, apply, kind

### AC-8 — Kernel-tree migration test stays green under unset threshold


---
id: G-0245
title: embedded ritual agents still instruct the retired work/tracking doc convention
status: addressed
addressed_by_commit:
    - 38ff079c
---
## What's missing

The embedded ritual snapshot (`internal/skills/embedded-rituals/`, the single source of truth per ADR-0016) retired the v1 separate tracking-doc convention in some artifacts but never swept the others. The retirement side is explicit — `aiwfx-start-milestone/SKILL.md:8` ("the v1 separate tracking-doc convention is gone") and `:103`, `aiwfx-wrap-milestone/SKILL.md:49`, and `templates/milestone-spec.md:85,117` (the spec sections "replace the v1 `tracking-doc.md`"). Against that, thirteen stale lines across five artifacts still instruct the retired convention:

- `agents/builder.md:41` — "Tracking doc at `work/tracking/M-NNN-<slug>.md` (scaffolded at start, finalized at wrap)" — plus lines 16, 23, 26, 35 (maintain / scaffold / finalize / read prior tracking docs).
- `agents/reviewer.md:10,16,30` — the reviewer is told to reconcile "the tracking doc" against the spec and roadmap.
- `skills/aiwfx-record-decision/SKILL.md:113,134` — mid-flight decisions recorded "in the tracking doc", a file the current convention never creates, so the agent either resurrects the convention or improvises.
- `skills/aiwfx-wrap-milestone/SKILL.md:3,8` — the frontmatter `description:` and intro say "finalizes/reconciles the tracking doc", contradicting the same file's line 49. The description line is what Claude Code surfaces in skill discovery in every consumer session.
- `skills/aiwfx-wrap-epic/SKILL.md:14` — "tracking docs" listed among wrap artefacts (mildest of the set; reads as historical).

The contradiction was inherited, not introduced: both sides arrived together in the vendored snapshot at M-0148 (commit `8a2c8acd`) — upstream shipped it self-contradictory and the vendoring froze it. With upstream archived per ADR-0016, the fix is a direct edit to the embedded snapshot in this repo.

## Why it matters

A builder agent following its own instructions recreates the retired `work/tracking/` directory, and nothing mechanical objects: verified 2026-06-12 in a scratch consumer repo (current-source kernel) that a stray milestone-named file under `work/tracking/` produces zero findings from both `aiwf check --shape-only` and full `aiwf check`. Which convention an agent follows therefore depends entirely on which embedded artifact it happened to read — the "guarantee depends on the LLM's behavior" failure class the kernel principles forbid, here in self-contradictory documentation form. Consumers report the live symptom: a builder regenerating the directory a project just retired, with the materialized copies byte-refreshed from the embed so no consumer-side fix can stick. G-0224 is the same defect class (embedded ritual content citing a retired convention) but nit-level; this instance is behavior-shaping, and the class has now recurred — the stated threshold for adding a mechanical chokepoint.

## Fix shape

1. Sweep all thirteen stale lines to point at the in-spec replacements: frontmatter `acs[]` plus the milestone spec's `## Work log` and `## Decisions made during implementation` sections.
2. Chokepoint: a policy test under `internal/policies/` asserting the embedded ritual bytes contain no `work/tracking/` path reference (and no instruction-shaped "tracking doc" phrasing outside explicit v1-historical context), so the retired convention cannot be reintroduced by a future ritual edit. Same shape as the existing embedded-content pins via the snapshot path constants.
3. Patch-release-worthy once swept; consumers pick it up via `aiwf upgrade` + `aiwf update` (byte-refresh of the materialized copies).

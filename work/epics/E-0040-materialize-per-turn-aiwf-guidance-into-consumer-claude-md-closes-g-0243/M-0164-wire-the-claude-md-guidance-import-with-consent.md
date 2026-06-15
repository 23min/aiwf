---
id: M-0164
title: Wire the CLAUDE.md guidance import with consent
status: in_progress
parent: E-0040
depends_on:
    - M-0163
tdd: required
acs:
    - id: AC-1
      title: init and update wire the import by default; --no-wire-claudemd opts out
      status: met
      tdd_phase: done
    - id: AC-2
      title: Content outside the markers is preserved; CLAUDE.md is created if absent
      status: met
      tdd_phase: done
    - id: AC-3
      title: Re-running is idempotent; a removed import line is reported, not re-added
      status: met
      tdd_phase: done
    - id: AC-4
      title: A printed notice announces the CLAUDE.md edit and names the opt-out
      status: open
      tdd_phase: green
    - id: AC-5
      title: The inserted import line resolves to the materialized guidance file
      status: open
      tdd_phase: red
    - id: AC-6
      title: A damaged marker block is handled per the hook-marker policy
      status: open
      tdd_phase: red
---
# M-0164 — Wire the CLAUDE.md guidance import with consent

## Goal

Wire the marker-wrapped `@.claude/aiwf-guidance.md` import line into the
consumer's `CLAUDE.md` so the guidance materialized by M-0163 actually loads on
every turn. `aiwf init` and `aiwf update` do this by default — including in
non-TTY contexts — with `--no-wire-claudemd` to decline.

## Context

M-0163 leaves the guidance file on disk but unimported. This milestone is the
activation step. The consent stance is fixed by ADR-0018: default-on for
`CLAUDE.md`, a deliberate departure from ADR-0015's settings.json default
(opt-in / refuse-in-non-TTY), licensed by the lower risk profile. The wiring
reuses the consent machinery built for the statusline in E-0039 rather than
forking a parallel flow.

## Acceptance criteria

### AC-1 — init and update wire the import by default; --no-wire-claudemd opts out

`aiwf init` and `aiwf update` insert the marker-wrapped import line without a
flag, including in non-TTY contexts (the deliberate departure from ADR-0015's
refuse-in-non-TTY default). `--no-wire-claudemd` declines. Fixture tests cover
the TTY, non-TTY, and declined paths.

### AC-2 — Content outside the markers is preserved; CLAUDE.md is created if absent

An existing `CLAUDE.md` keeps every byte outside the marker block verbatim; an
absent `CLAUDE.md` is created containing only the marker block.

### AC-3 — Re-running is idempotent; a removed import line is reported, not re-added

A second `aiwf update` against an already-wired `CLAUDE.md` produces no diff. A
`CLAUDE.md` whose import line was hand-removed is reported (printed nudge), not
silently re-added — the operator's removal is respected.

### AC-4 — A printed notice announces the CLAUDE.md edit and names the opt-out

The wiring is never silent: a printed notice names the file edited and the
`--no-wire-claudemd` opt-out.

### AC-5 — The inserted import line resolves to the materialized guidance file

The seam test: the path in the inserted `@…` line equals the path M-0163
materializes (`.claude/aiwf-guidance.md`), so the line cannot drift from the file
it imports.

### AC-6 — A damaged marker block is handled per the hook-marker policy

A `CLAUDE.md` with a missing or malformed END marker is handled by the same
policy aiwf already applies to its git-hook markers (recreate / refuse), not left
in an ambiguous state.

## Constraints

- Reuse E-0039's consent machinery (`internal/cli/cliutil/statusline.go`,
  `internal/skills/settings.go`); do not fork a parallel consent flow.
- Default-on per ADR-0018; the edit is marker-scoped and never clobbers
  outside-marker content.
- The imported path is the in-repo `.claude/aiwf-guidance.md` only (ADR-0018).

## Design notes

- ADR-0018 — the default-on consent decision this milestone implements.
- ADR-0015 / E-0039 — the settings.json consent precedent and the machinery
  reused here.

## Surfaces touched

- `internal/cli/initcmd/`, `internal/cli/update/` — the `--no-wire-claudemd`
  flag and the wiring call.
- `internal/cli/cliutil/`, `internal/skills/` — the consent + marker-write
  helpers.

## Out of scope

- The guidance content and its materialization (M-0163).
- The `aiwf doctor` unwired-guidance finding — the doctor-finding milestone.

## Dependencies

- M-0163 — the guidance file must be materialized before the import line can
  point at it.

## References

- ADR-0018 — risk-calibrated consent (the default-on CLAUDE.md instance).
- ADR-0015 / E-0039 — the settings.json consent precedent and reused machinery.
- G-0243 — the gap E-0040 closes.

---

## Work log

<!-- One entry per AC or unit of work; append-only. -->

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)

---
id: M-0164
title: Wire the CLAUDE.md guidance import with consent
status: done
parent: E-0040
depends_on:
    - M-0163
tdd: required
acs:
    - id: AC-1
      title: init and update wire the import automatically; aiwf.yaml opts out
      status: met
      tdd_phase: done
    - id: AC-2
      title: Content outside the markers is preserved; CLAUDE.md is created if absent
      status: met
      tdd_phase: done
    - id: AC-3
      title: Re-running is idempotent; a removed import line is self-healed
      status: met
      tdd_phase: done
    - id: AC-4
      title: A printed notice announces the CLAUDE.md edit
      status: met
      tdd_phase: done
    - id: AC-5
      title: The inserted import line resolves to the materialized guidance file
      status: met
      tdd_phase: done
    - id: AC-6
      title: A damaged marker block is handled per the hook-marker policy
      status: met
      tdd_phase: done
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
(opt-in / refuse-in-non-TTY), licensed by the lower risk profile. Default-on
needs no interactive prompt, so the statusline's opt-in consent machinery does
not apply — the wiring is a focused `ensureGuidanceImport` step (see *Decisions*).

## Acceptance criteria

### AC-1 — init and update wire the import automatically; aiwf.yaml opts out

`aiwf init` and `aiwf update` insert the marker-wrapped import line without a
flag, including in non-TTY contexts (the deliberate departure from ADR-0015's
refuse-in-non-TTY default). `--no-wire-claudemd` declines. Fixture tests cover
the TTY, non-TTY, and declined paths.

### AC-2 — Content outside the markers is preserved; CLAUDE.md is created if absent

An existing `CLAUDE.md` keeps every byte outside the marker block verbatim; an
absent `CLAUDE.md` is created containing only the marker block.

### AC-3 — Re-running is idempotent; a removed import line is self-healed

A second `aiwf update` against an already-wired `CLAUDE.md` produces no diff. A
`CLAUDE.md` whose import line was hand-removed is reported (printed nudge), not
silently re-added — the operator's removal is respected.

### AC-4 — A printed notice announces the CLAUDE.md edit

The wiring is never silent: a printed notice names the file edited and the
`--no-wire-claudemd` opt-out.

### AC-5 — The inserted import line resolves to the materialized guidance file

The seam test: the path in the inserted `@…` line equals the path M-0163
materializes (`.claude/aiwf-guidance.md`), so the line cannot drift from the file
it imports.

### AC-6 — A damaged marker block is handled per the hook-marker policy

A `CLAUDE.md` with only one of the two markers is refused: aiwf cannot safely
determine the block's extent in a user-owned file, so it leaves the file
untouched and reports the damage (the refuse-on-ambiguity stance the git-hook
marker policy takes).

## Constraints

- Default-on per ADR-0018; the edit is marker-scoped and never clobbers
  outside-marker content.
- Default-on needs no interactive prompt, so there is no statusline-style consent
  *flow* to reuse; the wiring is a focused `ensureGuidanceImport` step in the
  init/update pipeline, not a parallel prompt-consent path (see *Decisions* below).
- The imported path is the in-repo `.claude/aiwf-guidance.md` only (ADR-0018).

## Design notes

- ADR-0018 — the default-on consent decision this milestone implements.
- ADR-0015 / E-0039 — the settings.json consent *precedent* (its opt-in prompt
  machinery does not apply to default-on; see *Decisions*).

## Surfaces touched

- `internal/initrepo/initrepo.go` — `ensureGuidanceImport`, the marker constants,
  `guidanceImportBlock` / `replaceGuidanceBlock`, and the init/update pipeline
  wiring (plus the `Options` / `RefreshOptions` fields).
- `internal/cli/initcmd/`, `internal/cli/update/` — the `--no-wire-claudemd` flag,
  threaded through.

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

<!-- Phase/met timeline per AC is authoritative in `aiwf history M-0164/AC-<N>`;
     the implementation landed in this milestone's single wrap commit. -->

### AC-1 — init and update wire the import automatically; aiwf.yaml opts out

`ensureGuidanceImport` wires the marker block on init (`WireClaudeMdIfAbsent`);
`--no-wire-claudemd` on both `init` and `update` opts out. · `aiwf history M-0164/AC-1`

### AC-2 — preserve outside content + create-if-absent

Block appended preserving user content; CLAUDE.md created when absent. ·
`aiwf history M-0164/AC-2`

### AC-3 — Re-running is idempotent; a removed import line is self-healed

Re-run with the block present is a no-op diff; a stale block is refreshed in
place; on update an absent block is reported, not re-added. · `aiwf history M-0164/AC-3`

### AC-4 — A printed notice announces the CLAUDE.md edit

The import ledger step's Detail names `--no-wire-claudemd`. · `aiwf history M-0164/AC-4`

### AC-5 — import resolves to the materialized file

The `@…` line is built from `skills.GuidanceFile`, so it can't drift. ·
`aiwf history M-0164/AC-5`

### AC-6 — damaged marker refused

A one-sided marker pair leaves the file untouched and reports the damage. ·
`aiwf history M-0164/AC-6`

## Decisions made during implementation

- **Consent-machinery reuse did not apply.** The spec assumed reusing E-0039's
  consent machinery (`cliutil/statusline.go`, `skills/settings.go`). That code is
  the *opt-in* TTY-prompt / `--wire-settings` flow for the statusline. CLAUDE.md is
  *default-on* (ADR-0018): there is no prompt flow to share — "consent" is the
  default plus a `--no-wire-claudemd` opt-out. Implemented a focused
  `ensureGuidanceImport` step in `internal/initrepo` instead; no statusline-consent
  code was touched, and no parallel prompt-consent flow was created.

## Validation

- `go build ./...` — green; `golangci-lint run` (full module) — 0 issues; `go vet` — clean.
- `go test ./internal/initrepo/ ./internal/skills/ ./internal/cli/initcmd/ ./internal/cli/update/` — green.
  `ensureGuidanceImport`, `replaceGuidanceBlock`, `guidanceImportBlock` at 100% branch coverage.
- Full `go test ./...` — green (integration's TempDir-cleanup flake notwithstanding; isolated-passes).
- `aiwf check` — 0 errors (3 pre-existing / worktree-benign warnings).

## Deferrals

- None. (The narrative-doc update describing the new consumer-CLAUDE.md wiring in
  the repo's own CLAUDE.md "marker-managed artifacts" section + design docs is
  deferred to the epic wrap, when the whole feature — incl. M-0165's doctor
  finding — has landed; noted under Reviewer notes, not a separate gap.)

## Reviewer notes

- Init/update behavior is asymmetric by design (ADR-0018 + AC-3): `init` adds the
  block; `update` refreshes a present block but *nudges rather than re-adds* a
  removed one. The split is carried by `RefreshOptions.WireClaudeMdIfAbsent`
  (true for init, false for update).
- Damaged/reversed markers are refused (left untouched), never auto-repaired —
  the conservative choice for a user-owned file.
- The consumer-facing narrative docs (repo CLAUDE.md materialized-artifacts list,
  design-decisions.md) are intentionally not updated in this milestone; that
  description belongs with the epic wrap once M-0165 lands too. `--help` text +
  ADR-0018 carry discoverability in the meantime.

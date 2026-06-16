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

Maintain a marker-wrapped `@.claude/aiwf-guidance.md` import in the consumer's
root `CLAUDE.md` so the guidance materialized by M-0163 loads on every turn.
`aiwf init` and `aiwf update` do this **automatically and self-heal it** — added
when absent, refreshed when present, re-added if removed — exactly like skill/hook
materialization. Default-on; a consumer opts out via `aiwf.yaml`
`guidance.wire_claudemd: false` (ADR-0018).

## Context

M-0163 leaves the guidance file on disk but unimported; this milestone activates
it. ADR-0018 fixes the consent stance: automatic and self-maintaining, with no
CLI flag — adopting aiwf (running `init`/`update`) is the consent, and the edit is
marker-scoped, announced, reversible, and confined to aiwf's own region. The
opt-out is a single set-once `aiwf.yaml` knob mirroring `status_md.auto_update`.

## Acceptance criteria

### AC-1 — init and update wire the import automatically; aiwf.yaml opts out

`aiwf init` and `aiwf update` insert/refresh the marker-wrapped import with no
flag. `guidance.wire_claudemd: false` in `aiwf.yaml` opts out. Fixture tests and a
verb-seam test (`cli.Execute`) cover the wire and opt-out paths.

### AC-2 — Content outside the markers is preserved; CLAUDE.md is created if absent

An existing `CLAUDE.md` keeps every byte outside the marker block verbatim
(detection is line-anchored); an absent `CLAUDE.md` is created containing the
marker block.

### AC-3 — Re-running is idempotent; a removed import line is self-healed

A second run with the block present produces no diff. A `CLAUDE.md` whose block
was removed is **re-added** on the next `init`/`update` (self-healing) — the
automagical replacement for the earlier nudge-not-re-add design, changed in
review per ADR-0018.

### AC-4 — A printed notice announces the CLAUDE.md edit

The wiring is never silent: it surfaces as a ledger step that `init`/`update`
print (created / updated / preserved).

### AC-5 — The inserted import line resolves to the materialized guidance file

The `@…` line is built from `skills.GuidanceFile`, the same constant M-0163
materializes, so the line cannot drift from the file it imports.

### AC-6 — A damaged marker block is handled per the hook-marker policy

A one-sided or reversed marker pair is refused (the file is left untouched and the
damage reported) — the refuse-on-ambiguity stance the git-hook marker policy
takes. Detection is line-anchored, so marker text inside user prose is inert; a
pre-existing bare import line is wrapped in markers rather than duplicated.

## Constraints

- Default-on per ADR-0018; the edit is marker-scoped, line-anchored, and never
  clobbers outside-marker content.
- No CLI flag: the wiring is automatic with a single `aiwf.yaml` opt-out
  (`guidance.wire_claudemd`), the focused `ensureGuidanceImport` step in the
  init/update pipeline.
- The imported path is the in-repo `.claude/aiwf-guidance.md` only (ADR-0018).

## Design notes

- ADR-0018 — the automatic, self-maintaining, config-opt-out decision this
  milestone implements.
- The `guidance.wire_claudemd` knob mirrors `status_md.auto_update` (tristate
  `*bool`, default-on getter).

## Surfaces touched

- `internal/initrepo/initrepo.go` — `ensureGuidanceImport`, marker constants,
  `guidanceMarkerLineIdx` / `spliceGuidanceLines` / `writeGuidanceImport`,
  pipeline wiring, `RefreshOptions.WireClaudeMd`, `loadWireClaudeMd`.
- `internal/config/config.go` — the `Guidance` block + `WireClaudeMd` getter.
- `internal/cli/initcmd`, `internal/cli/update` — config-derived wiring (no flag).

## Out of scope

- The guidance content and its materialization (M-0163).
- The `aiwf doctor` unwired-guidance finding (M-0165).

## Dependencies

- M-0163 — the guidance file must be materialized before the import line can
  point at it.

## References

- ADR-0018 — automatic CLAUDE.md wiring + the fragment inclusion principle.
- ADR-0015 / E-0039 — the settings.json consent precedent (opt-in prompt
  machinery, which does not apply to the automatic case).
- G-0243 — the gap E-0040 closes.

---

## Work log

<!-- Phase/met timeline per AC is authoritative in `aiwf history M-0164/AC-<N>`. -->

### AC-1 — automatic wiring + config opt-out

`ensureGuidanceImport` wires/refreshes the block on init+update; `aiwf.yaml`
`guidance.wire_claudemd: false` opts out. · `aiwf history M-0164/AC-1`

### AC-2 — preserve outside content + create-if-absent

Block added preserving user content (line-anchored); created when absent. ·
`aiwf history M-0164/AC-2`

### AC-3 — idempotent / self-heal

Re-run with the block present is a no-op diff; a removed block is re-added. ·
`aiwf history M-0164/AC-3`

### AC-4 — announced as a ledger step

The import step prints created/updated/preserved. · `aiwf history M-0164/AC-4`

### AC-5 — import resolves to the materialized file

Built from `skills.GuidanceFile`; can't drift. · `aiwf history M-0164/AC-5`

### AC-6 — damaged/ambiguous markers refused; prose inert; bare line wrapped

One-sided/reversed markers left untouched; marker text in prose ignored;
pre-existing bare import line wrapped. · `aiwf history M-0164/AC-6`

## Decisions made during implementation

- **Design changed in review: flag → automatic.** The original spec (and an
  earlier draft of ADR-0018) used a `--no-wire-claudemd` flag with init-adds /
  update-nudges-not-re-adds. E-0040 review reworked this to automatic +
  self-healing with an `aiwf.yaml` opt-out and no CLI flag; ADR-0018 was rewritten
  accordingly. The assumed reuse of E-0039's consent machinery did not apply —
  default-on has no prompt flow — so a focused `ensureGuidanceImport` was written
  instead (no parallel prompt-consent path).
- **Line-anchored marker detection (review hardening).** Substring matching risked
  clobbering user prose that mentions the markers and duplicating a pre-existing
  bare import line; detection now matches a line that, trimmed, equals a marker,
  upholding ADR-0018's "clobbers nothing" guarantee.

## Validation

- `go build ./...` — green; `golangci-lint run` (full module) — 0 issues.
- `go test ./internal/initrepo/ ./internal/config/ ./internal/cli/...` — green;
  `ensureGuidanceImport`, its helpers, `loadWireClaudeMd`, and `Config.WireClaudeMd`
  at 100% coverage; a `cli.Execute` verb-seam test covers init-wire / update-self-heal / opt-out.
- `aiwf check` — 0 errors.

## Deferrals

- None. (The narrative design-docs update lands at the epic wrap; the CLAUDE.md
  "what aiwf materializes" entry already describes the write-channel + opt-out.)

## Reviewer notes

- Automatic + self-healing by design (ADR-0018): init/update always maintain the
  block; a removed block returns on the next `update`; the only opt-out is the
  set-once `aiwf.yaml` knob.
- Damaged/reversed markers and prose mentions of the markers are handled
  conservatively (refuse / inert) — a user-owned file is never clobbered.
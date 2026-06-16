---
id: M-0165
title: Surface unwired CLAUDE.md guidance in aiwf doctor
status: done
parent: E-0040
depends_on:
    - M-0164
tdd: required
acs:
    - id: AC-1
      title: Unwired tree yields an advisory finding naming the exact fix command
      status: met
      tdd_phase: done
    - id: AC-2
      title: A wired tree or an absent guidance file yields no finding
      status: met
      tdd_phase: done
    - id: AC-3
      title: Fixtures cover the wired, unwired, and file-absent states
      status: met
      tdd_phase: done
---
# M-0165 — Surface unwired CLAUDE.md guidance in aiwf doctor

## Goal

Add an advisory `aiwf doctor` finding, `claudemd-guidance-unwired`, that fires
when `.claude/aiwf-guidance.md` exists but the consumer's `CLAUDE.md` does not
import it — naming the exact fix (`aiwf update`, which self-heals the import). It
is the surface that tells an operator their tree is currently unwired, before the
next `update` fixes it.

## Context

M-0164 wires and self-heals the import automatically (default-on; opt out via
`aiwf.yaml` `guidance.wire_claudemd: false`). Between a hand-removal and the next
`aiwf update`, or on a fresh clone before `update` regenerates the gitignored
fragment, a tree can be transiently unwired. `aiwf doctor` surfaces that state
(advisory) and names `aiwf update` as the fix — the pressure level ADR-0018 chose
over silence or a blocking error. When the consumer has opted out, doctor stays
silent.

## Acceptance criteria

### AC-1 — Unwired tree yields an advisory finding naming the exact fix command

When `.claude/aiwf-guidance.md` exists and `CLAUDE.md` does not import it
(line-anchored check), `aiwf doctor` emits an advisory `claudemd-guidance-unwired`
finding whose hint names the exact command — `aiwf update` (per G-0199) — not a
vague "wire it up".

### AC-2 — A wired tree or an absent guidance file yields no finding

When `CLAUDE.md` imports the guidance, or the fragment is absent (nothing to
wire), or the consumer opted out via `aiwf.yaml`, the finding does not fire.

### AC-3 — Fixtures cover the wired, unwired, and file-absent states

A table test exercises unwired / wired / absent (plus the CLAUDE.md-absent and
opt-out branches); `appendGuidanceImportReport` is at 100% coverage.

## Constraints

- Advisory severity — the finding never blocks `aiwf check`'s push gate.
- The hint names the exact command (G-0199).
- Respects the `aiwf.yaml` opt-out (`guidance.wire_claudemd: false` → no finding).
- Text output only; an `aiwf doctor --format=json` envelope is tracked separately
  (G-0070) and is out of scope here.

## Design notes

- ADR-0018 — the finding is the safety-net surface for the automatic wiring stance.
- G-0199 — findings must name the exact remediation command.

## Surfaces touched

- `internal/cli/doctor/guidance.go` — `appendGuidanceImportReport` (opt-out check,
  line-anchored detection, `aiwf update` remediation) and its fixtures.

## Out of scope

- The `aiwf doctor --format=json` envelope (G-0070).
- The wiring itself (M-0164).

## Dependencies

- M-0164 — the marker/import-line shape this finding inspects.

## References

- ADR-0018 — automatic CLAUDE.md wiring; the finding as safety net.
- G-0199 — exact-remediation-command rule.
- G-0243 — the gap E-0040 closes.

---

## Work log

<!-- Phase/met timeline per AC is authoritative in `aiwf history M-0165/AC-<N>`. -->

### AC-1 — unwired → advisory naming `aiwf update`

`appendGuidanceImportReport` emits the advisory with the exact `aiwf update`
remediation. · `aiwf history M-0165/AC-1`

### AC-2 — wired / absent / opted-out → no finding

`guidance: ok` when wired; nothing when the fragment is absent or wiring is
disabled. · `aiwf history M-0165/AC-2`

### AC-3 — fixtures cover all states

Table test over unwired / wired / absent (+ CLAUDE.md-absent + opt-out); 100%
coverage. · `aiwf history M-0165/AC-3`

## Decisions made during implementation

- **The remediation command is `aiwf update`, not `aiwf init`.** Under the
  automagical model (M-0164), `update` re-adds a removed block, so it is the
  correct idempotent-friendly fix. This supersedes the flag-era design where
  `update` nudged and the finding had to name `aiwf init` (changed in E-0040
  review along with the rest of the automatic-wiring rework).
- **The finding respects the opt-out.** When `guidance.wire_claudemd` is false the
  consumer chose not to wire, so doctor does not nag.

## Validation

- `go build ./...` — green; `golangci-lint run` (full module) — 0 issues.
- `go test ./internal/cli/doctor/` — green; `appendGuidanceImportReport` and
  `guidanceImportLinePresent` at 100% coverage (unwired / wired / absent /
  CLAUDE.md-absent / opt-out).
- Rendered output human-verified via a worktree-built binary: unwired prints the
  advisory + `aiwf update`; wired prints `guidance: ok`.
- `aiwf check` — 0 errors.

## Deferrals

- None. (The `aiwf doctor --format=json` envelope remains out of scope, tracked by
  the pre-existing G-0070.)

## Reviewer notes

- Advisory only: the finding never increments doctor's problem count / exit code.
- Respects the `aiwf.yaml` opt-out; detection is line-anchored, consistent with
  `ensureGuidanceImport`, so a prose mention of the import path is not counted as
  wired.
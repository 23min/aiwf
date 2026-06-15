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
import it — naming the exact command to fix it. This is the recurring safety net
that makes a declined or hand-removed wiring self-healing without aiwf
re-fighting the operator's choice.

## Context

M-0164 wires the import by default, but an operator can decline
(`--no-wire-claudemd`) or later remove the line, and a `git pull` of a tree whose
guidance file is gitignored arrives unwired until the next `aiwf update`. A
silent unwired state would defeat E-0040's purpose; an advisory `doctor` finding
surfaces it on the next routine run — the pressure level ADR-0018 chose over
either silence or a blocking error.

## Acceptance criteria

### AC-1 — Unwired tree yields an advisory finding naming the exact fix command

When `.claude/aiwf-guidance.md` exists and `CLAUDE.md` lacks the import marker,
`aiwf doctor` emits an advisory `claudemd-guidance-unwired` finding whose hint
names the exact remediation command (per G-0199), not a vague "wire it up".

### AC-2 — A wired tree or an absent guidance file yields no finding

When `CLAUDE.md` already imports the guidance, or the guidance file is absent
(nothing to wire), the finding does not fire.

### AC-3 — Fixtures cover the wired, unwired, and file-absent states

A fixture-tree test exercises all three states, so the present and both
absent-finding branches are each traversed.

## Constraints

- Advisory severity — the finding never blocks `aiwf check`'s push gate.
- The hint names the exact command (G-0199).
- Text output only; an `aiwf doctor --format=json` envelope is tracked separately
  (G-0070) and is out of scope here.

## Design notes

- ADR-0018 — the finding is the recurring safety net for the default-on consent
  stance.
- G-0199 — findings must name the exact remediation command.

## Surfaces touched

- `internal/cli/doctor/` — the new finding and its fixtures.

## Out of scope

- The `aiwf doctor --format=json` envelope (G-0070).
- The wiring itself (M-0164).

## Dependencies

- M-0164 — the marker/import-line shape this finding inspects.

## References

- ADR-0018 — risk-calibrated consent; the finding as safety net.
- G-0199 — exact-remediation-command rule.
- G-0243 — the gap E-0040 closes.

---

## Work log

<!-- Phase/met timeline per AC is authoritative in `aiwf history M-0165/AC-<N>`;
     the implementation landed in this milestone's single wrap commit. -->

### AC-1 — unwired tree → advisory naming the fix

`appendGuidanceImportReport` emits `guidance: claudemd-guidance-unwired:
advisory — … run \`aiwf init\` to wire it` when the fragment exists but CLAUDE.md
lacks the import line. · `aiwf history M-0165/AC-1`

### AC-2 — wired / absent → no finding

Wired → `guidance: ok`; fragment absent → no line. · `aiwf history M-0165/AC-2`

### AC-3 — fixtures cover all states

A table test exercises unwired / wired / absent (plus the CLAUDE.md-absent
branch); `appendGuidanceImportReport` at 100% coverage. · `aiwf history M-0165/AC-3`

## Decisions made during implementation

- **The remediation command is `aiwf init`, not `aiwf update`.** M-0164's
  `update` *nudges* rather than re-adds a removed import block (AC-3), so it
  cannot re-wire; `init` adds by default. The finding therefore names `aiwf init`
  as the exact fix (per G-0199). If a lighter re-wire affordance (e.g.
  `aiwf update --wire-claudemd`) is wanted later, that's a follow-up.

## Validation

- `go build ./...` — green; `golangci-lint run` (full module) — 0 issues; `go vet` — clean.
- `go test ./internal/cli/doctor/` — green; `appendGuidanceImportReport` at 100%
  coverage (every branch: absent / wired / unwired / CLAUDE.md-absent).
- Rendered output human-verified via a worktree-built binary: unwired prints the
  advisory + `aiwf init`; wired prints `guidance: ok`.
- `aiwf check` — 0 errors (pre-existing / worktree-benign warnings only).

## Deferrals

- None. (The `aiwf doctor --format=json` envelope remains out of scope, tracked
  by the pre-existing G-0070.)

## Reviewer notes

- Advisory only: the finding never increments doctor's problem count / exit code,
  matching the ADR-0018 stance (surface, don't block).
- The wired state emits an informational `guidance: ok` row (not a finding),
  consistent with the other doctor rows; the absent state emits nothing.
- Detection keys on the import *line* (`@.claude/aiwf-guidance.md`, built from
  `skills.GuidanceFile`) rather than importing M-0164's package-private marker
  constants — the line is the user-visible contract and avoids cross-package
  coupling.

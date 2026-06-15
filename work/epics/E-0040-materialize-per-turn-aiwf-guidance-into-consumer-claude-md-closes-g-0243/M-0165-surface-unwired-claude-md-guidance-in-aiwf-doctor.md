---
id: M-0165
title: Surface unwired CLAUDE.md guidance in aiwf doctor
status: in_progress
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
      status: open
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

<!-- One entry per AC or unit of work; append-only. -->

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)

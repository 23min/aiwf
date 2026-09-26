---
id: M-0355
title: Take aiwf-only rules and aiwf ids out of aiwf check
status: draft
parent: E-0096
depends_on:
    - M-0354
tdd: advisory
---
## Goal

`aiwf check` carries no rule that exists only for this repository, and nothing it
prints cites an aiwf id.

## Closes

- (none)

## Context

M-0354 checks every embedded file for what `skill-body-id` and
`skill-body-claude-md-section` check. Those two rules live in `internal/check`,
which is compiled into the shipped binary, where they are inert. `internal/check`
also prints aiwf ids: at `fde32ce8a`, 13 string literals in `hint.go` and 6 in five
other files cite one. The CLI-text policy's scope does not include `internal/check`.

## Acceptance criteria

## Constraints

- Starts after E-0092 lands: `internal/check/hint.go` belongs to that epic until
  then.
- Deleting the two rules loses no coverage: every shape they report, the gate
  reports.

## Out of scope

- Rewording finding messages beyond removing ids: E-0092 owns message wording.

## Dependencies

- M-0354.
- E-0092 landing.

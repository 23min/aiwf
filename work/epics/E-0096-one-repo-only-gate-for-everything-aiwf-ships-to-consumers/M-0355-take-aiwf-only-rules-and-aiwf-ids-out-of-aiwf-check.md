---
id: M-0355
title: Take aiwf-only rules and aiwf ids out of aiwf check
status: draft
parent: E-0096
depends_on:
    - M-0354
tdd: advisory
acs:
    - id: AC-1
      title: aiwf check carries no rule that exists only for this repo
      status: open
    - id: AC-2
      title: Nothing aiwf check prints cites an aiwf id
      status: open
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

### AC-1 — aiwf check carries no rule that exists only for this repo

`skill-body-id` and `skill-body-claude-md-section` are deleted from `internal/check`,
together with their hints, their rows in the shipped `aiwf-check` skill and their
tests. **Pass criterion**: neither code appears in `internal/check` or in any shipped
file, and the gate's tests cover every shape the deleted rules' tests covered.
**Edge cases**: `body-prose-id`, which reads a consumer's own entity bodies, keeps
working on the shared id classifier. **Code references**: `internal/check/check.go`,
`internal/check/skill_body_id.go`, `internal/check/skill_body_claude_md.go`,
`internal/check/hint.go`.

### AC-2 — Nothing aiwf check prints cites an aiwf id

The CLI-text policy's scope includes `internal/check`, and every literal it then
reports is rewritten without the id. **Pass criterion**: the policy reports nothing
on the tree with `internal/check` in scope. **Edge cases**: a malformed spelling shown
as the thing a rule reports, such as the `body-prose-id` hints' `M-1`, is an
illustration rather than a citation and stays (the distinction G-0538 names).
**Code references**: `inOperatorTextScope` in
`internal/policies/cli_text_internal_ids.go`, `internal/check/hint.go`, and the five
other files carrying an id.

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

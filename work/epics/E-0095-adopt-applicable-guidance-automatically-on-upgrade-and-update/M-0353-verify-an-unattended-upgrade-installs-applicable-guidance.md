---
id: M-0353
title: Verify an unattended upgrade installs applicable guidance
status: draft
parent: E-0095
depends_on:
    - M-0351
    - M-0352
tdd: advisory
---
## Goal

Show that an unattended `aiwf upgrade` — no terminal, no prior guidance configuration — leaves a real consumer repository with its applicable guidance installed, routed and reported.

## Closes

- (none)

## Context

The previous milestones pin adoption and reporting against fixtures and a stand-in catalogue. The failure that motivated this epic happened in a consumer repository reached through `aiwf upgrade`, with the published catalogue and a personal-bootstrap installation present. This milestone checks that path end to end, hermetically where a test can reach it and by recorded observation where it cannot.

## Acceptance criteria

## Constraints

- The observation runs against the released binary for the epic, not a development build.
- The observation records the command, the expected result, the observed output and the environment together in this milestone.
- The observation does not commit in the consumer repository.

## Design notes

- The hermetic path uses the existing upgrade integration harness with a fake `go` binary and a local catalogue source.

## Surfaces touched

- `internal/cli/integration/upgrade_cmd_test.go`

## Out of scope

- Fixing consumer-side legacy delivery, which the personal-bootstrap maintainer owns under ADR-0052.

## Dependencies

- Both earlier milestones in this epic.
- A tagged release containing them, for the observation.

## References

- ADR-0054, ADR-0052.

---

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)

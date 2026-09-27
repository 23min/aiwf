---
id: M-0353
title: Verify an unattended upgrade installs applicable guidance
status: in_progress
parent: E-0095
depends_on:
    - M-0351
    - M-0352
tdd: none
acs:
    - id: AC-1
      title: Unattended upgrade adopts guidance in a hermetic consumer
      status: met
    - id: AC-2
      title: Unattended upgrade observed in a real consumer repository
      status: deferred
---
## Goal

Show that an unattended `aiwf upgrade` — no terminal, no prior guidance configuration — leaves a real consumer repository with its applicable guidance installed, routed and reported.

## Closes

- (none)

## Context

The previous milestones pin adoption and reporting against fixtures and a stand-in catalogue. The failure that motivated this epic happened in a consumer repository reached through `aiwf upgrade`, with the published catalogue and a personal-bootstrap installation present. This milestone checks that path end to end, hermetically where a test can reach it and by recorded observation where it cannot.

## Acceptance criteria

### AC-1 — Unattended upgrade adopts guidance in a hermetic consumer

**Pass criterion**: `aiwf upgrade` run with stdin not a terminal, in a repository with an `aiwf.yaml` carrying no `guidance` block, installs a new binary through the fake `go` binary, re-executes `update`, and leaves `guidance.packs`, `.guidance/` and host routing populated for the matching packs of a local catalogue, with the guidance section in its output. **Edge cases**: the catalogue unreachable, where the upgrade still succeeds and the report names the retrieval failure. **Code references**: `internal/cli/integration/upgrade_cmd_test.go`.

### AC-2 — Unattended upgrade observed in a real consumer repository

**Pass criterion**: an observation record in this milestone — command, expected result, observed output, environment — of `aiwf upgrade` run non-interactively against the published catalogue in a consumer repository that has no guidance configuration and an ai-dotfiles installation, showing either installed guidance with routing and a report naming the adopted packs, or a report naming a handover blocker and its fix. **Edge cases**: a handwritten legacy import in the consumer's `CLAUDE.md`, recorded as the blocker it produces. **Code references**: none; this criterion is met by the record.

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

No user-visible change.

## Decisions made during implementation

- The milestone's TDD policy is `none`: both criteria verify behaviour the earlier milestones built, so no test could fail before its code existed.

## Validation

`make ci` at `0b673c71f`, in the devcontainer (Linux): exit 0. golangci-lint reported `0 issues.`, the race and coverage suite and the profile-driven gates passed, and `aiwf doctor --self-check` reported `self-check passed (29 steps).`

Mutation probes against `TestBinary_UnattendedUpgradeAdoptsGuidance`, each applied to a working copy and then restored byte for byte:

- Replacing the `projectguidance.Adopt` call in `internal/initrepo/project_guidance.go` with the unchanged selection fails the reachable case: `guidance.packs = <nil>, want [sample/base]`.
- Re-executing `version` instead of `update` in `internal/cli/upgrade/upgrade.go` fails both cases: `output has 0 guidance sections, want 1`.
- Dropping the source from the fetch error in `internal/projectguidance/retrieve.go` fails the unreachable case: `report does not name the retrieval failure`.

## Deferrals

- M-0353/AC-2 → G-0722. The observation must run against a released binary containing this epic, and no release can contain it until the epic wraps.

## Reviewer notes

- The upgrade harness's isolation of `HOME`, `CODEX_HOME`, `CLAUDE_CONFIG_DIR` and the Git global configuration has no dedicated test. Pinning it would take a serial `t.Setenv` test and a further binary build; the explicit environment entries in `hostUpgradeRun` are the guarantee.
- `TestBinary_UpgradeOutputEndsWithTheGuidanceSection` is kept beside the new test. It covers upgrade with a configured `guidance.source`, and M-0352 cites it as evidence for M-0352/AC-5.
- The redirect of the default catalogue URL is kept in the unreachable case although the test harness already refuses the `https` transport, so that case does not depend on the harness's transport policy.
- The milestone adds no module, abstraction or data model; there was nothing for a design review.

---
id: M-0352
title: Report guidance outcomes at the end of init, update and upgrade
status: draft
parent: E-0095
depends_on:
    - M-0351
tdd: required
acs:
    - id: AC-1
      title: Update ends with a guidance section of adopted, installed and ignored packs
      status: open
    - id: AC-2
      title: Blocked installation is reported with its cause and fix
      status: open
---
## Goal

Every `aiwf init`, `aiwf update` and `aiwf upgrade` run ends with a guidance section that states what happened to project guidance and what, if anything, the maintainer must do.

## Closes

- (none)

## Context

Guidance outcomes surface today as one step-ledger line whose wording ("no project guidance selected; existing delivery retained") does not say whether guidance is adopted, and whose blocked cases read as `guidance incomplete: …` among unrelated steps. `aiwf doctor` describes the same state in clearer terms from its own code in `internal/cli/doctor/project_guidance.go`. With adoption automatic after the previous milestone, the report is the maintainer's only per-run view of what changed.

## Acceptance criteria

### AC-1 — Update ends with a guidance section of adopted, installed and ignored packs

**Pass criterion**: a successful `aiwf update` ends with a guidance section listing the packs adopted in this run, the installed packs with the installed source commit, and the ignored packs, each list present even when empty. **Edge cases**: first adoption; a run adopting nothing new; a repository matching no pack; `guidance.enabled: false`, reported as maintenance disabled with the installed set unchanged. **Code references**: the report renderer beside `internal/cli/cliutil/guidance_report.go`; binary-level tests in `internal/cli/update`.

### AC-2 — Blocked installation is reported with its cause and fix

**Pass criterion**: when guidance cannot be installed, the report names the cause and the action that clears it for each blocker the installer can return: catalogue retrieval failure, incompatible personal-bootstrap delivery, a handwritten legacy import in a host instruction file, and a locally edited owned file; unrelated update steps still complete and the exit code is unchanged from today. **Edge cases**: a blocker on first adoption (nothing installed) and on refresh (installed set preserved). **Code references**: the blocker cases in `internal/cli/update/project_guidance_failures_test.go` and `project_guidance_handover_test.go`, asserting the report section.

## Constraints

- The report and `aiwf doctor` derive their guidance-state wording from one shared source.
- The report is plain text on stdout in the verbs' existing output style; no new output format.
- Every blocker named in the report carries the action that clears it.

## Design notes

- ADR-0054 requires the report on every run.
- `upgrade` re-executes `update` in place, so the report reaches an upgrade's output through the re-executed `update` unless the re-exec is skipped; the skipped case prints nothing about guidance, since no refresh ran.

## Surfaces touched

- `internal/cli/update/update.go`, `internal/cli/initcmd/initcmd.go`
- `internal/cli/cliutil/guidance_report.go`
- `internal/cli/doctor/project_guidance.go`
- `internal/initrepo/project_guidance.go`

## Out of scope

- `--format=json` output for `update`.
- Changing what `aiwf doctor` checks, as opposed to how it words guidance state.

## Dependencies

- The adoption milestone in this epic, whose adoption result the report prints.

## References

- ADR-0054, E-0094.

---

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)

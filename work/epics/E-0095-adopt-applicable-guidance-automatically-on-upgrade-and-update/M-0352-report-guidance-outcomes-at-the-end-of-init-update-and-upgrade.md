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
    - id: AC-3
      title: Report and doctor word guidance state identically
      status: open
    - id: AC-4
      title: Init ends with the same guidance section as update
      status: open
    - id: AC-5
      title: Upgrade output carries the guidance section
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

### AC-3 — Report and doctor word guidance state identically

**Pass criterion**: for each guidance state — selection unset, explicitly empty, populated, maintenance disabled, installation pending — the state line in the update report equals the corresponding line `aiwf doctor` prints for the same repository. **Edge cases**: pending installation after an interrupted run. **Code references**: the shared wording source used by `internal/cli/doctor/project_guidance.go` and the report; a test deriving both outputs by running the two verbs against one fixture.

### AC-4 — Init ends with the same guidance section as update

**Pass criterion**: `aiwf init` ends with the same guidance section as `aiwf update` for the same outcome. **Edge cases**: `init` on a repository with an existing `aiwf.yaml` that carries `guidance.ignored`. **Code references**: `internal/cli/initcmd/initcmd.go`; the init binary tests.

### AC-5 — Upgrade output carries the guidance section

**Pass criterion**: the output of `aiwf upgrade`, with the re-exec into `update` enabled, contains the guidance section. **Edge cases**: `AIWF_NO_REEXEC` set, where the output states that no refresh ran and carries no guidance section. **Code references**: the upgrade integration tests in `internal/cli/integration/upgrade_cmd_test.go`, which stand in a fake `go` binary through `AIWF_GO_BIN`.

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

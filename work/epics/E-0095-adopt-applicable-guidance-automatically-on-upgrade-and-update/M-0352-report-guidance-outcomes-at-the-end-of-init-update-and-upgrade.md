---
id: M-0352
title: Report guidance outcomes at the end of init, update and upgrade
status: in_progress
parent: E-0095
depends_on:
    - M-0351
tdd: required
acs:
    - id: AC-1
      title: Update ends with a guidance section of adopted, installed and ignored packs
      status: met
      tdd_phase: done
    - id: AC-2
      title: Blocked installation is reported with its cause and fix
      status: met
      tdd_phase: done
    - id: AC-3
      title: Report and doctor word guidance state identically
      status: met
      tdd_phase: done
    - id: AC-4
      title: Init ends with the same guidance section as update
      status: met
      tdd_phase: done
    - id: AC-5
      title: Upgrade output carries the guidance section
      status: met
      tdd_phase: done
---
## Goal

Every `aiwf init`, `aiwf update` and `aiwf upgrade` run ends with a guidance section that states what happened to project guidance and what, if anything, the maintainer must do.

## Closes

- (none)

## Context

Guidance outcomes surface today as one step-ledger line whose wording ("no project guidance selected; existing delivery retained") does not say whether guidance is adopted, and whose blocked cases read as `guidance incomplete: …` among unrelated steps. `aiwf doctor` describes the same state in clearer terms from its own code in `internal/cli/doctor/project_guidance.go`. With adoption automatic after the previous milestone, the report is the maintainer's only per-run view of what changed.

## Acceptance criteria

### AC-1 — Update ends with a guidance section of adopted, installed and ignored packs

**Pass criterion**: a successful `aiwf update` ends with a guidance section listing the packs adopted in this run, the installed packs with the installed source commit, and the ignored packs, each list present even when empty, and any selected pack the catalogue no longer lists, with the fix (remove it from `guidance.packs` or move it to `guidance.ignored`). **Edge cases**: first adoption; a run adopting nothing new; a repository matching no pack; `guidance.enabled: false`, reported as maintenance disabled with the installed set unchanged. **Code references**: a structured guidance outcome returned by `internal/initrepo/project_guidance.go` in place of the free-text step detail; the report renderer in `internal/cli/cliutil`; binary-level tests in `internal/cli/update`.

### AC-2 — Blocked installation is reported with its cause and fix

**Pass criterion**: when guidance cannot be installed, the report names the cause and the action that clears it for each blocker the installer can return: catalogue retrieval failure, incompatible personal-bootstrap delivery, a handwritten legacy import in a host instruction file, and a locally edited owned file; unrelated update steps still complete and the exit code is unchanged from today. **Edge cases**: a blocker on first adoption (nothing installed) and on refresh (installed set preserved). **Code references**: the blocker cases in `internal/cli/update/project_guidance_failures_test.go` and `project_guidance_handover_test.go`, asserting the report section.

### AC-3 — Report and doctor word guidance state identically

**Pass criterion**: for each guidance state — selection unset, explicitly empty, populated, maintenance disabled, installation pending, a selected pack not installed — the state line in the update report equals the corresponding line `aiwf doctor` prints for the same repository. **Edge cases**: pending installation after an interrupted run; a selected pack the catalogue no longer lists, which `aiwf doctor` detects offline as selected but absent from the installed index. **Code references**: the shared wording source used by `internal/cli/doctor/project_guidance.go` and the report; a test deriving both outputs by running the two verbs against one fixture.

### AC-4 — Init ends with the same guidance section as update

**Pass criterion**: `aiwf init` ends with the same guidance section as `aiwf update` for the same outcome. **Edge cases**: `init` on a repository with an existing `aiwf.yaml` that carries `guidance.ignored`. **Code references**: `internal/cli/initcmd/initcmd.go`; the init binary tests.

### AC-5 — Upgrade output carries the guidance section

**Pass criterion**: the output of `aiwf upgrade`, with the re-exec into `update` enabled, contains the guidance section. **Edge cases**: `AIWF_NO_REEXEC` set, where the output states that no refresh ran and carries no guidance section. **Code references**: the upgrade integration tests in `internal/cli/integration/upgrade_cmd_test.go`, which stand in a fake `go` binary through `AIWF_GO_BIN`.

## Constraints

- The report and `aiwf doctor` derive their guidance-state wording from one shared source.
- The report is plain text on stdout in the verbs' existing output style; no new output format.
- Every blocker of the four classes M-0352/AC-2 names carries the action that clears it; the other installer errors are tracked in G-0714.

## Design notes

- ADR-0054 requires the report on every run.
- `upgrade` re-executes `update` in place, so the report reaches an upgrade's output through the re-executed `update` unless the re-exec is skipped; the skipped case prints nothing about guidance, since no refresh ran.

## Surfaces touched

- `internal/cli/update/update.go`, `internal/cli/initcmd/initcmd.go`
- `internal/cli/cliutil` (report renderer)
- `internal/projectguidance/inspect.go`
- `internal/cli/doctor/project_guidance.go`
- `internal/initrepo/project_guidance.go`

## Out of scope

- `--format=json` output for `update`.
- Changing what `aiwf doctor` checks, beyond the per-pack not-installed warning M-0352/AC-3 requires.

## Dependencies

- The adoption milestone in this epic, whose adoption result the report prints.

## References

- ADR-0054, D-0103, E-0094.

---

## Release note

- `aiwf init`, `aiwf update` and `aiwf upgrade` now end with a `Guidance:` section, including runs that stop on a hook-chain collision; a dry run prints none. It lists the selection (or that it is unset or explicitly empty), the packs adopted and recorded in this run, the installed packs with their source commit (or that they are unknown when the installed index is not intact), and the ignored packs. When they apply it adds that maintenance is disabled, a `pending:` line for an interrupted installation, a `blocked:` line naming why installation did not complete and, where the installer names one, the action that clears it, and a `missing:` line with its fix for each selected pack the installed index does not list. A blocked installation leaves the exit status and the other steps unchanged.
- `aiwf doctor` warns about each selected guidance pack an intact installed index does not list, and words the selection, disabled maintenance, a pending installation and missing packs as the report does.
- Guidance refusals for an unreadable source catalogue or pack document, a source with no commit, and unreadable personal-instruction paths now name the action that clears them.

## Decisions made during implementation

- None recorded as entities; where the work settled a scope edge, Constraints and Out of scope above state it.

## Validation

`make ci` at `055655827` on the M-0352 milestone branch, in the Linux devcontainer: exit 0. golangci-lint `0 issues.`; `go test -race` passed; total statement coverage `91.9%`; the diff-scoped coverage gate and policy suite passed; `./bin/aiwf doctor --self-check` reported `self-check passed (29 steps).`; `aiwf check` reported `0 errors, 10 warnings`, none on M-0352.

## Deferrals

- G-0714 — installer errors for corrupt records and file-system faults print on the `blocked:` line without an action; M-0352/AC-2 covers the four blocker classes it names.

## Reviewer notes

- (none)

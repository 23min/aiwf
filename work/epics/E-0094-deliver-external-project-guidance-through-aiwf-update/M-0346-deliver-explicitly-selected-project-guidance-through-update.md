---
id: M-0346
title: Deliver explicitly selected project guidance through update
status: in_progress
parent: E-0094
depends_on:
    - M-0344
    - M-0345
tdd: required
acs:
    - id: AC-1
      title: Explicit configuration controls maintenance without adopting policy
      status: met
      tdd_phase: done
    - id: AC-2
      title: Retrieval validates current guidance without a persistent cache
      status: met
      tdd_phase: done
    - id: AC-3
      title: Installation preserves ownership and produces tracked project files
      status: met
      tdd_phase: done
    - id: AC-4
      title: Failures and removal preserve a coherent installed selection
      status: met
      tdd_phase: done
    - id: AC-5
      title: Legacy handover never activates a second guidance corpus
      status: met
      tdd_phase: done
    - id: AC-6
      title: Both hosts and local diagnostics expose the installed policy
      status: met
      tdd_phase: done
---
## Goal

Ship a complete init/update workflow for packs explicitly listed in `aiwf.yaml`, including tracked outputs, host routing, safe refresh/removal, and legacy handover.

## Closes

- (none)

## Context

The external corpus and compatible ai-dotfiles delivery exist. This milestone serves explicit configuration directly; automatic detection and interactive selection follow separately. It must have a real CLI caller when it lands.

## Acceptance criteria

### AC-1 — Explicit configuration controls maintenance without adopting policy

Default to the agreed upstream with maintenance enabled; support one source override, selected packs, ignored pack ids, and maintenance opt-out while preserving existing guidance fields and unrelated YAML. Distinguish absent selection from an explicitly adopted empty selection so an unconfigured noninteractive run cannot withdraw legacy guidance. Reject invalid or overlapping selected/ignored state with a remedy. References: `internal/config/config.go`, schema, examples, and init/update integration tests.

### AC-2 — Retrieval validates current guidance without a persistent cache

Use a temporary Git clone of the source's default branch through existing credentials, validate the catalogue and every selected document, and clean temporary data on success, failure, and cancellation. Return validated selected content with the exact fetched source commit; repeated retrieval reflects changes to upstream. References: guidance retrieval for `internal/initrepo/` and local-Git fixtures. Test unavailable sources, missing selections, malformed content, and unsafe paths without live network dependencies.

### AC-3 — Installation preserves ownership and produces tracked project files

Through `aiwf init` and `aiwf update`, materialize the retrieved selected documents and the index under `.guidance/`, preserve handwritten `project.md`, and generate concise routing for the selected hosts through existing wiring controls. Record the exact installed source commit in the index only on successful installation; integration tests verify that the next update installs changed upstream content and its revision. Project overrides take precedence; unchanged inputs cause no diff. Reject foreign or edited generated outputs before replacement, including collisions, malformed managed blocks, and symlinks. References: `internal/skills/ownership.go`, `internal/initrepo/agents_guidance.go`, and refresh fixtures; reuse their suitable primitives without assuming the existing filename restrictions fit namespaced packs.

### AC-4 — Failures and removal preserve a coherent installed selection

Fetch, validation, missing-pack, and conflict failures leave all installed guidance and its recorded revision unchanged, while unrelated update work proceeds with visible diagnostics. Removing a selection removes only its unmodified owned outputs; disabling maintenance preserves installed files/routing and makes no guidance network call. Exercise first-install failure and interrupted-write recovery, including config/index disagreement, without claiming multi-file crash atomicity. References: refresh and ownership fault-injection tests.

### AC-5 — Legacy handover never activates a second guidance corpus

Compatible setups and machines without ai-dotfiles can adopt project guidance. Incompatible personal delivery blocks handover with actionable remediation and preserves legacy delivery; binary upgrade and unrelated refresh work can continue. Successful handover replaces recognized legacy managed imports with usable project routing in the same host file. When no legacy imports need replacement, both host routes may remain disabled while selected guidance files and the index are installed. Test explicitly empty selections and retry after interruption. References: init/update integration and the compatibility signal established by the preceding delivery.

### AC-6 — Both hosts and local diagnostics expose the installed policy

Claude and Codex route to project overrides, the index, and relevant packs without concatenating the corpus. Local diagnostics report selected/installed state, missing or modified artifacts, and the last installed revision without claiming network freshness. Check host opt-outs, neither host, retained unselected host artifacts, and portable clones/worktrees. References: `internal/cli/doctor/`, `internal/initrepo/`, and shared rendering fixtures. Help and configuration documentation describe the complete explicit-selection workflow; update never commits or pushes.

## Constraints

Use required TDD and repository build/lint/race/selfcheck gates. Resolve write ordering and recovery before shipping. No persistent cache, lockfile, package solver, extra upgrade command, or automatic personal-file rewrites. Do not add an index that disables legacy guidance merely because defaults enable maintenance.

## Design notes

E-0094 fixes user-visible behavior. The configuration's representation of unadopted versus explicitly empty selection is an implementation choice to settle against existing YAML handling. This distinction adds no new user-facing mode.

## Surfaces touched

Configuration/schema; shared init refresh; update and upgrade integration; ownership/rendering; doctor and user documentation.

## Out of scope

Automatic applicability detection, selection prompts, live migration of aiwf itself, and language content embedded in the binary.

## Dependencies

- M-0344 — Publish the external engineering guidance corpus.
- M-0345 — Preserve legacy guidance through repository-aware routing.

Corpus and compatible ai-dotfiles routing, including the ownership/preflight boundary. Host behavior is inherited from E-0093.

## References

- E-0094 — delivery scope and compatibility requirements.

---

## Release note

Projects can explicitly select external engineering-guidance packs in `aiwf.yaml` and refresh them through `aiwf init` or `aiwf update`. Guidance is downloaded on demand and materialized as repository files with Claude and Codex routing, project overrides, ownership checks, and interrupted-update recovery. Compatible legacy handover replaces recognized imports without enabling a second corpus. `aiwf doctor` reports local installation state and damage without claiming upstream freshness. Unconfigured projects retain their existing guidance delivery; updates never commit or push the generated files.

## Decisions made during implementation

- D-0098 — Finish interrupted guidance installation on the next update.
- Preserve an edited engineering route for an unselected host without blocking shared-pack refresh. Doctor warns about the retained edit; reselecting the host requires reconciliation. Legacy imports and unsafe instruction paths still block handover.

## Validation

Observed on 2026-09-22 in the Linux development container, on the milestone branch with implementation through `e830eae99`. Commands below ran from the milestone checkout. The implementation gate is reused because no production code changed after it; subsequent test-strengthening changes passed focused tests, lint, and the package race run below.

| Command | Expected | Observed |
| --- | --- | --- |
| `make check-fast` | Vet, full configured lint, and the full test suite pass. | Exit 0; lint reported `0 issues.`; test packages passed. |
| `make lint` | Final test additions satisfy the same lint gate. | Exit 0; `0 issues.` |
| `go test -race ./internal/config ./internal/gitops ./internal/pathutil ./internal/projectguidance ./internal/initrepo ./internal/skills ./internal/cli/update ./internal/cli/doctor` | Changed delivery packages pass with race instrumentation. | Exit 0; every named package reported `ok`. |
| `go build -o /tmp/aiwf-m0346-wrap ./cmd/aiwf` | Current CLI builds. | Exit 0. |
| `/tmp/aiwf-m0346-wrap doctor --self-check` | CLI lifecycle checks pass in a throwaway repository. | Exit 0; `self-check passed (29 steps).` |
| `/tmp/aiwf-m0346-wrap check --since origin/main` | No error findings; provenance scope explicit. | Exit 0; `15 findings (0 errors, 15 warnings)`; warnings concern advisory TDD records and archival backlog. |

The executable evidence covers these claims:

- Configuration round trips distinguish omitted and empty selection, preserve opt-outs, and reject invalid or overlapping ids (`internal/config/project_guidance_test.go`).
- Local-Git retrieval fixtures verify exact source revision, updated upstream content, catalogue/document validation, cancellation, and temporary-data cleanup (`internal/projectguidance/retrieve_test.go`, `catalogue_test.go`).
- Installer and update fixtures verify unchanged-input convergence, override preservation, ownership conflicts, removal, and recovery after interrupted publication (`internal/projectguidance/install_test.go`, `install_faults_test.go`, and `internal/cli/update/project_guidance_failures_test.go`).
- Compatibility and handover fixtures distinguish supported installed delivery from incompatible or ambiguous legacy setups, preserve blocked installations, and replace recognized imports in the same host file (`internal/projectguidance/compatibility_test.go`, `handover_test.go`, and `internal/cli/update/project_guidance_handover_test.go`).
- Host-routing and diagnostic fixtures verify individual host selection and opt-outs, retained edited routes, local revision/damage reporting, and tracked guidance in clones and worktrees (`internal/initrepo/project_guidance_test.go`, `internal/cli/doctor/project_guidance_test.go`, and `internal/projectguidance/retained_host_test.go`).

These are local fixture and generated-artifact checks. Fresh assistant-session observations and migration of aiwf itself remain the claims of M-0348; this milestone does not establish that a running assistant read the generated instructions. The full `make ci` integration gate remains for integration into mainline or a push, per repository validation cadence.

## Deferrals

- G-0702 — The configuration reference incorrectly requires `aiwf_version`.

## Reviewer notes

- (none)

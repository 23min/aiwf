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
      status: open
      tdd_phase: done
    - id: AC-4
      title: Failures and removal preserve a coherent installed selection
      status: open
    - id: AC-5
      title: Legacy handover never activates a second guidance corpus
      status: open
    - id: AC-6
      title: Both hosts and local diagnostics expose the installed policy
      status: open
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

Compatible setups and machines without ai-dotfiles can adopt project guidance. Incompatible personal delivery blocks handover with actionable remediation and preserves legacy delivery; binary upgrade and unrelated refresh work can continue. Successful handover removes recognized legacy managed imports and records project ownership only with usable replacement routing. Test explicitly empty selections and retry after interruption. References: init/update integration and the compatibility signal established by the preceding delivery.

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

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- G-0702 — The configuration reference incorrectly requires `aiwf_version`.

## Reviewer notes

- (none)

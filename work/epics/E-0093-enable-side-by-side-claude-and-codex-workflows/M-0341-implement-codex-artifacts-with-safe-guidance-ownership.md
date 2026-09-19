---
id: M-0341
title: Implement Codex artifacts with safe guidance ownership
status: in_progress
parent: E-0093
depends_on:
    - M-0340
tdd: required
acs:
    - id: AC-1
      title: Codex artifacts use the selected native and aiwf-owned support paths
      status: met
      tdd_phase: done
    - id: AC-2
      title: Managed AGENTS guidance preserves user content and converges on refresh
      status: met
      tdd_phase: done
    - id: AC-3
      title: Instruction-file symlinks and aliases are preserved and diagnosed
      status: met
      tdd_phase: done
    - id: AC-4
      title: Ownership conflicts and unsafe paths cannot overwrite foreign artifacts
      status: open
    - id: AC-5
      title: Operational fragments are selected by host and resolve workflow references
      status: open
---
## Goal

Provide complete, tested Codex artifact operations and preserve user-owned guidance for both hosts before public automatic detection is enabled.

## Closes

- G-0501 — Preserve symlinked instruction files during init and update, report skipped guidance with remediation, and audit both instruction-file creation and managed-guidance writers.

## Context

The preceding milestone establishes explicit rendering and Claude compatibility evidence. This milestone adds the Codex adapter and safe instruction-file handling behind the internal boundary. The existing AtomicWriteFile contract deliberately replaces a destination path; guidance callers must prevent unintended symlink replacement.

## Acceptance criteria

The criteria below define the observable completion contract.

### AC-1 — Codex artifacts use the selected native and aiwf-owned support paths

Render the canonical skill corpus under .agents/skills and templates under .agents/aiwf/templates, with ownership metadata for each generated family. References in the generated skills resolve to the generated support files. Validate skill metadata and output inventory through observable files. Do not create .agents/agents, Claude role Markdown as Codex agents, Claude settings, or a fictitious .agents/templates convention. Ordinary Claude output continues to match its baseline.

### AC-2 — Managed AGENTS guidance preserves user content and converges on refresh

Create a missing root AGENTS.md or add/update only aiwf's marked block in an existing regular file. Preserve surrounding bytes and applicable existing file permissions; exercise absent files, empty files, no final newline, existing blocks, duplicate/conflicting markers, one-sided or reversed markers, marker text quoted in ordinary prose, and persistent opt-out. Refuse ambiguous edits with actionable results. Repeating a successful update is byte-idempotent. The generated block contains native instructions, not an assumed Claude-style import.

### AC-3 — Instruction-file symlinks and aliases are preserved and diagnosed

Inspect root CLAUDE.md and AGENTS.md without following links before writing guidance. Skip linked instruction paths and report the path, current condition, and remediation. Preserve the link and its target for broken, external, looping, and ordinary links. When the two host paths alias the same underlying file, diagnose the conflict before either host updates it. Test that unrelated artifacts can still be handled as documented and that the result identifies guidance as incomplete. Apply the guard to Claude as the deliberate G-0501 correction while keeping AtomicWriteFile unchanged.

### AC-4 — Ownership conflicts and unsafe paths cannot overwrite foreign artifacts

Exercise first installation, refresh of owned output, an unowned file or directory occupying a desired Codex name, obsolete owned entries, and foreign siblings. Preserve content not established as aiwf-owned; report collisions instead of assuming a name proves ownership. Reject unsafe manifest paths and artifact-directory links that escape the permitted output location. An invalid ownership record must not trigger arbitrary deletion. Re-running after a controlled filesystem-boundary failure converges to the complete artifact set without consuming foreign content. Characterize any shared-writer change affecting existing Claude collision behavior explicitly.

### AC-5 — Operational fragments are selected by host and resolve workflow references

Given each supported host binding, verify that rendering selects its named skill-invocation, worktree-entry, and review-dispatch fragments, with shared workflow content preserved outside those slots. Derive expectations from the selected fragment source, and verify referenced local artifacts and valid rendered metadata rather than asserting prose phrases. Revalidate the fragment interfaces against current official documentation during implementation and review their semantics. Successful live delegation is a separate observation in the final milestone; this criterion does not claim that renderer tests prove a model can execute the workflow.

## Constraints

- Codex output consists of skills, aiwf-owned template support files, and native guidance. Do not invent Codex role-agent directories or copy Claude settings.
- Keep canonical sources single-owned and the shared AtomicWriteFile contract unchanged.
- Guidance defaults to automatic maintenance when its host is selected, with a persistent opt-out that stops maintenance without silently deleting existing content.
- Required independent review cannot become self-review when delegation is unavailable.
- Use per-file atomic writes with idempotent recovery; do not claim an all-files transaction. Public Codex host enablement belongs to the lifecycle milestone.

## Design notes

- E-0093 and docs/initiatives/agent-host-artifact-adapters.md define the approved scope and selected host contracts.
- D-0073 keeps planning on main and implementation in an isolated worktree.

## Surfaces touched

- internal/skills
- internal/initrepo
- internal/config guidance options
- embedded host fragments and template references

## Out of scope

- Executable detection and public Codex selection.
- Codex custom-agent TOML, lifecycle hooks, cloud execution, and statusline parity.
- Changing the generic atomic writer or following instruction-file symlinks.

## Dependencies

- Preserve Claude output through explicit host rendering

## References

- E-0093
- ADR-0018
- G-0178
- G-0501
- D-0070
- D-0095

## Release note



## Decisions made during implementation

- (none)

## Validation



## Deferrals

- (none)

## Reviewer notes

- (none)

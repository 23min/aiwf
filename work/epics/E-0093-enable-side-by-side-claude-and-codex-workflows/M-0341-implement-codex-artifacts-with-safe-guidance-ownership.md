---
id: M-0341
title: Implement Codex artifacts with safe guidance ownership
status: done
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
      status: met
      tdd_phase: done
    - id: AC-5
      title: Operational fragments are selected by host and resolve workflow references
      status: met
      tdd_phase: done
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

Claude setup and refresh preserve linked or aliased instruction files and report incomplete guidance with remediation. Artifact refresh refuses unowned name collisions, preserves supplemental user files when retiring generated skills, and supports retry after partial writes. Codex skills, templates, managed AGENTS guidance, and native workflow instructions are implemented internally; automatic detection and public host selection arrive in M-0342.

## Decisions made during implementation

- (none)

## Validation

Validated on 2026-09-19 in the Linux devcontainer, on the isolated milestone branch. All five ACs are met with TDD phase done. Implementation commits: AC-1 `c2e1bf6ec`, AC-2 `853188e90`, AC-3 `a195023ca`, AC-4 `88e6aa2c1`, AC-5 `e874bb7f0`.

- `make check-fast`: exit 0; default, stress-tagged, and testpins-tagged vet passed; full configured lint reported `0 issues.`; the full untagged test suite passed, including CLI integration and policy tests.
- `make diag-aiwf`: exit 0; the worktree-local diagnostic binary built successfully.
- Race tests for `internal/skills` and `internal/initrepo`: passed after the final mutation probes and source restoration. The full repository race/coverage/self-check gate, `make ci`, has not been run for this milestone; the repository requires it at epic-to-main and push boundaries.
- `bin/aiwf-diag check --since epic/E-0093-enable-side-by-side-claude-and-codex-workflows`: exit 0; `3 findings (0 errors, 3 warnings)`. The warnings concern archive cleanup for G-0464 and G-0691, not this milestone.

AC evidence:

- **AC-1** — `go test -parallel 8 ./internal/skills -run 'TestMaterializeTo_|TestMaterializeArtifacts_'`: Complete Codex inventory, valid metadata, resolving support references, independent host roots, and preservation of foreign siblings.
- **AC-2** — `go test -parallel 8 ./internal/initrepo ./internal/config -run 'TestSpliceAgentsGuidance_|TestEnsureAgentsGuidance_|TestWireAgentsMd_'`: Exact surrounding bytes and modes, marker ambiguity refusal, idempotent refresh, persistent opt-out, dry-run and error behavior.
- **AC-3** — `go test -parallel 8 ./internal/initrepo -run 'TestInstruction|TestInitAndRefresh_|TestClaudeScaffold_'`: All three instruction writers preserve symlinks and targets, refuse aliases in either order, and diagnose incomplete guidance while unrelated artifacts continue.
- **AC-4** — `go test -parallel 8 ./internal/skills -run TestArtifactOwnership_`: Collision refusal, validated ownership and paths, conservative retirement, preserved user content, and recovery across partial writes and changed sources.
- **AC-5** — `go test -parallel 8 ./internal/skills -run 'TestHostFragments_|TestRender'`: Native fragment selection, unchanged shared bytes, concrete local references, and missing-binding refusal.

The full-suite `TestClaudeArtifacts_MatchBaseline` comparison passes for init, update, and generated worktrees in all six scenarios. The independently captured inventory and its narrowly approved configuration and recovery-ignore exceptions are documented in `internal/cli/integration/testdata/claude-baseline/README.md`. Native-fragment extraction changes no Claude output bytes.

Changed branches received manual branch walks and coverage inspection. Assertion-strength checks caught all 48 bounded manual mutations across the five ACs (9, 10, 10, 10, and 9 respectively), with exact source restoration. Gremlins was unavailable, so this is sampled mutation evidence, not an exhaustive mutation score.

The host-fragment interface review is recorded in `docs/design/design-decisions.md`: official host documentation plus local `codex --version` (0.155.0) and `codex --help` confirmed the cited entry points. Renderer tests do not establish live skill discovery, handoff, or independent delegation; those observations belong to M-0343. Filesystem safety covers inspected paths and per-file atomic writes with retry, not concurrent path replacement or an all-files transaction.

## Deferrals

- No ACs deferred.
- G-0698 — The shared planning rituals retain a ritual branch and merge step contrary to accepted D-0073. Correcting both planning workflows changes shared behavior and the frozen Claude baseline; it requires a separate workflow change and review.
- Public host detection and lifecycle selection remain the planned scope of M-0342; fresh-session discovery, cross-host handoff, parallel terminals, independent Codex review, and container persistence remain the planned observations in M-0343.

## Reviewer notes

- Independent code review: approve. Design review: keep the artifact ownership/recovery boundary and the instruction-file safety/block-splicing boundary. No blocking findings remain.
- Keep separate preflight, ownership, and recovery states: each protects a distinct preservation or retry obligation. The smaller measured alternative loses clarity and replaces linear membership lookups with quadratic scans.
- Retain the four private ownership-helper cancellation branches under the tested helper contract and Go context convention. The public materialization entry supplies `context.Background()`; this does not establish public cancellation support.
- Scoped doc-lint: clean across the 24 changed Markdown files. Shared planning-workflow drift is recorded in G-0698 under Deferrals.

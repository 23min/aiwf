---
id: M-0256
title: Bulk-input verb coverage backfill
status: in_progress
parent: E-0064
depends_on:
    - M-0252
tdd: required
acs:
    - id: AC-1
      title: Every bulk-input verb group branch tested or ignored
      status: met
      tdd_phase: done
    - id: AC-2
      title: Scoped coverage-gate reports zero findings
      status: met
      tdd_phase: done
---

## Goal

Clear every branch `branch-coverage-audit` currently flags in the
bulk-input verb group — `importcmd`, `render`, `check`+`check/provenance`
— plus `initcmd`, using the shared failure fixtures M-0252 builds.

## Context

M-0252 lands the reusable fixtures for the failure modes these guards
share. This group's verbs read larger, more structurally varied input
(imported entity files, render targets, the full tree under `aiwf check`)
than the other three consumer groups, so a couple of its flagged sites may
need a fixture beyond M-0252's five shared ones — a malformed import
source, or a render-target-specific failure — surfaced during
implementation rather than pre-designed here.

`initcmd` doesn't fit this milestone's bulk-input theme any better than it
fits any other milestone's — it wasn't assigned to any of E-0064's five
milestones during planning. Folding its 4 remaining flagged lines in here
(rather than a sixth milestone) keeps the epic's "zero findings" success
criterion reachable without adding a milestone for one file, mirroring the
same call M-0255 made for `archive`/`authorize`.

## Acceptance criteria

### AC-1 — Every bulk-input verb group branch tested or ignored

Every branch `branch-coverage-audit` flags (base = the commit before
M-0238/AC-3's rename, `2ac84846^`) within `internal/cli/{importcmd,
render,check,initcmd}` (including `check/provenance.go`) carries either a
passing test (reusing M-0252's fixtures where the failure mode matches,
or a new fixture where it doesn't) or a `//coverage:ignore <reason>`.

### AC-2 — Scoped coverage-gate reports zero findings

`make coverage-gate`'s underlying policy test, run with `AIWF_COVERAGE_BASE`
set to the pre-M-0238 commit, reports zero findings for the files listed
in AC-1.

## Constraints

- Reuse M-0252's fixtures for shared failure modes rather than
  reimplementing them per file; a genuinely new fixture (e.g. malformed
  import source) is scoped to this milestone only.
- Per-site judgment only: real test where triggerable, honest
  `//coverage:ignore <reason>` otherwise.

## Out of scope

- Entity-lifecycle, contract, diagnostic, and non-CLI infra files —
  M-0252, M-0253, and M-0255's job.
- Any change to error-handling behavior beyond what's needed to make a
  branch testable.

## Dependencies

- M-0252 — its shared fixtures must exist before this milestone starts.

## References

- **E-0064** — parent epic.
- **M-0252** — shared fixtures this milestone consumes.

## Work log

### AC-1 — Every bulk-input verb group branch tested or ignored

21 new tests plus 17 `//coverage:ignore` annotations, closing all 41
branch-coverage-audit findings across check+provenance, importcmd,
render, and initcmd · commit 639f5eca · tests 21/21

check.go/provenance.go needed real triggers for the --pretty warning,
a malformed aiwf.yaml (LoadTreeWithTrunk), a malformed `contracts:`
block (LoadContractsBlock), and tree.Load's `ctx.Err()` fatal path —
the last one exercised directly via the unexported runShapeOnly/
runFast with a canceled context, which also surfaced and cleared a
stale, factually-inaccurate ignore comment on runFast's own sibling
branch (it claimed the branch was untestable; a canceled context
proves otherwise). importcmd.go's `--on-collision` guard turned out
genuinely reachable through the CLI despite Cobra's `FixedCompletions`
only hinting the shell-completion set, not validating the flag value.
render.go and initcmd.go each needed one read-only-directory fixture
(AtomicWriteFile / htmlrender.Render's MkdirAll) and initcmd.go reused
internal/initrepo's own G45 hook-migration-collision fixture shape at
the CLI entry point.

### AC-2 — Scoped coverage-gate reports zero findings

Validation-only, no new commit. Re-ran the scoped
`TestPolicy_BranchCoverageAudit` policy test with
`AIWF_COVERAGE_BASE=2ac84846^` against a full-repo coverage profile
generated after AC-1's tests landed: zero findings across all 5 files
in this milestone's scope.

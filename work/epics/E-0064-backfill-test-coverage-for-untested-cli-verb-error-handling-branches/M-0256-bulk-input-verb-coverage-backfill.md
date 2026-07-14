---
id: M-0256
title: Bulk-input verb coverage backfill
status: done
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

23 new tests plus 15 `//coverage:ignore` annotations, closing all 41
branch-coverage-audit findings across check+provenance, importcmd,
render, and initcmd · commits 639f5eca, 96a79655 · tests 23/23

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

Corrective commit from wrap review: `fix(cli): correct ResolveRoot
ignore wording, test render's unreadable-root path` · commit 96a79655.
Reworded four `cliutil.ResolveRoot` ignores that claimed a failure
mode ("missing aiwf.yaml + non-existent --root path") ResolveRoot
doesn't actually have, to the accurate characterization already used
for initcmd's own resolver. Added two real tests
(`TestRunRoadmap_TreeLoadFailure` / `TestRunSite_TreeLoadFailure`, a
0o000 unreadable root) for two render.go `tree.Load` guards that had
been ignored on an incomplete rationale — a canceled context isn't
reachable through the public API, but an unreadable directory is —
removing both ignores.

### AC-2 — Scoped coverage-gate reports zero findings

Validation-only, no new commit. Re-ran the scoped
`TestPolicy_BranchCoverageAudit` policy test with
`AIWF_COVERAGE_BASE=2ac84846^` against a full-repo coverage profile
generated after AC-1's tests landed: zero findings across all 5 files
in this milestone's scope. Re-confirmed after the corrective commit
(96a79655) against a freshly regenerated profile: still zero findings.

## Decisions made during implementation

- (none)

## Validation

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `make lint` (full CI-parity set) — 0 issues.
- `make test-race` (full repo, `-race -parallel 8`) — all packages pass.
- Scoped `TestPolicy_BranchCoverageAudit` with `AIWF_COVERAGE_BASE=2ac84846^` — zero
  findings across all 5 files in this milestone's scope, re-confirmed after the
  corrective commit.
- Independent fresh-context code-quality review (`wf-review-code`) — first pass
  **REQUEST-CHANGES** (two blocking findings, see Reviewer notes); both fixed in a
  corrective commit; a second, narrowly-scoped independent pass **APPROVE**d the
  fix. No design-quality (`wf-rethink`) pass — this milestone introduces no new
  module boundary, abstraction, or data model, only test files and comment-only
  production edits.
- `wf-doc-lint` (scoped to the milestone's change-set) — clean; the change-set is
  entirely Go source/test files plus this spec and the epic's Milestones-list
  note, none intersecting the doc-lint scope.

## Deferrals

- (none) — the second review's one repeat-instance observation (the same
  imprecise `ResolveRoot` ignore wording B1 corrected here still survives at
  ~20 pre-existing sites from sibling M-0253/M-0254/M-0255 commits) is already
  tracked by the open **G-0412**, not a new deferral.

## Reviewer notes

- The first independent review (`wf-review-code`) returned **REQUEST-CHANGES**
  with two blocking findings, both genuine defects in the AC-1 commit rather than
  reviewer false positives:
  - **B1**: four `//coverage:ignore` comments on `cliutil.ResolveRoot` guards
    claimed a failure mode ("missing aiwf.yaml + non-existent --root path")
    `ResolveRoot` does not actually have — the explicit-root path never stats
    anything, and a missing `aiwf.yaml` with no `--root` falls back to `cwd`
    rather than erroring. Notably, this wording is the same class G-0412 already
    tracks as pervasive-but-imprecise across ~20 pre-existing sites from earlier
    E-0064 milestones — but these four instances were newly authored *this*
    milestone, and the correct wording was sitting right next to them in the same
    commit (`initcmd.go`'s `resolveInitRoot` ignore), so this was a "propagate a
    new instance without checking" mistake, not a G-0412 repeat — it got fixed
    inline rather than deferred.
  - **B2**: two `render.go` `tree.Load` guards were ignored on the claim that
    only a canceled `context.Context` could trigger them (correct, but
    incomplete) — an unreadable root directory also hits the same fatal
    `os.Stat`-error path and IS reachable through the public API. Fixed with two
    new tests (a `0o000` root, empirically distinct from the neighboring `0o500`
    write-failure fixtures, which retain enough permission for `tree.Load` to
    succeed).
  - Both fixes landed as commit 96a79655. A second, narrowly-scoped independent
    review verified the corrective commit directly against `resolveroot.go` and
    `tree.go` (not just re-reading the diff), re-ran the two new tests and
    confirmed they hit the intended line via their actual stderr output, re-ran
    the scoped coverage audit, and confirmed the commit's scope was tight with
    clean build/vet — **APPROVE**.
- The second review's repeat-instance observation (the imprecise `ResolveRoot`
  wording B1 corrected here still exists at ~20 sites introduced by M-0253's
  `add.go`, M-0254's `contract/bind.go`, and other sibling milestone commits) is
  the same issue **G-0412** already tracks repo-wide (deliberately deferred as a
  future sweep, not a per-milestone fix) — consistent with the same observation
  M-0254's and M-0255's own reviewers made.

---
id: M-0253
title: Entity-lifecycle verb coverage backfill
status: in_progress
parent: E-0064
depends_on:
    - M-0252
tdd: required
acs:
    - id: AC-1
      title: Entity-lifecycle flagged branches are tested or documented
      status: met
      tdd_phase: done
    - id: AC-2
      title: Coverage gate is clean for the entity-lifecycle group
      status: met
      tdd_phase: done
---

## Goal

Clear every branch `branch-coverage-audit` currently flags in the
entity-lifecycle verb group — `add`, `promote`, `retitle`, `rename`,
`reallocate`, `cancel`, `milestone`, `update`, `rewidth`, `editbody` —
using the shared failure fixtures M-0252 builds.

## Context

M-0252 lands the reusable fixtures for the failure modes these guards
share (root-resolution, actor-resolution, lock contention, malformed tree,
bad `--format`). Entity-lifecycle verbs carry the largest concentration of
flagged sites among the four consumer groups, since every one of them
shares the same `ResolveRoot`/`ResolveActor`/`AcquireRepoLock`/`tree.Load`
guard shape ahead of its verb-specific logic.

## Acceptance criteria

### AC-1 — Entity-lifecycle flagged branches are tested or documented

Every branch `branch-coverage-audit` flags (base = the commit before
M-0238/AC-3's rename) within `internal/cli/{add,promote,retitle,rename,
reallocate,cancel,milestone,update,rewidth,editbody}` carries either a
passing test (reusing M-0252's fixtures where the failure mode matches)
or a `//coverage:ignore <reason>` naming why the branch isn't
triggerable, mirroring `internal/cli/archive/archive.go:120`.

### AC-2 — Coverage gate is clean for the entity-lifecycle group

`make coverage-gate`, run with `AIWF_COVERAGE_BASE` set to the
pre-M-0238 commit, reports zero findings for the files listed in AC-1.

## Constraints

- Reuse M-0252's fixtures for shared failure modes rather than
  reimplementing them per file; only build a new fixture for a
  verb-specific failure mode not already covered.
- Per-site judgment only: real test where triggerable, honest
  `//coverage:ignore <reason>` otherwise.

## Out of scope

- Contract, diagnostic, bulk-input, and non-CLI infra files — M-0252,
  M-0254, M-0255, and M-0256's job.
- Any change to error-handling behavior beyond what's needed to make a
  branch testable.

## Dependencies

- M-0252 — its shared fixtures must exist before this milestone starts.

## References

- **E-0064** — parent epic.
- **M-0252** — shared fixtures this milestone consumes.

## Work log

### AC-2 — Coverage gate is clean for the entity-lifecycle group

Two independent confirmations against the committed AC-1 state: (1)
`TestPolicy_BranchCoverageAudit`, run directly with `AIWF_COVERAGE_BASE`
set to the pre-M-0238 commit (`2ac84846^`), scoped-grepped to all ten
entity-lifecycle files — zero matching lines. (2) the standard
`make coverage-gate` (default base = `git merge-base origin/main HEAD`)
— all four policies (`BranchCoverageAudit`, `FiringFixturePresence`,
`FiringFixtureNoStaleAllowlist`, `SkillEditStructuralTestBackstop`)
pass clean. No new code in this AC — it's the closing verification
that AC-1's three waves together satisfy the milestone's stated
success criterion.

### AC-1 — Entity-lifecycle flagged branches tested or documented

Implemented in three sequential waves (48 flagged branches across ten
files, all resolved with per-site judgment — no blanket suppression):

- **Wave 1** — `add`, `cancel`, `editbody` (22 branches). Landed as
  16 real tests + 6 honest `//coverage:ignore`s (commit `d716478c`);
  independent review (see Reviewer notes) found two of those six
  ignores dishonest, converted to real tests in a corrective commit
  (`d3b147c8`) — final tally 18 real tests, 4 honest ignores.
  `add.go`'s `runAC` subcommand needed its own verb-specific triggers
  (title-count mismatch, stdin batch mutex, frontmatter-leading body
  rejection) beyond the shared root/actor/lock/tree guard shape.
- **Wave 2** — `promote`, `reallocate`, `rename`, `retitle` (19
  branches). 11 real tests, 8 honest ignores. `promote.go` carried the
  largest single concentration (10 branches) — its own `--phase`/
  positional-status mutex, `--force`/`--audit-only` gating, and
  resolver-flag (`--by`/`--by-commit`/`--superseded-by`) mutex checks,
  each independently cross-checked against the actual source during
  review. Commit `7db16b47`.
- **Wave 3** (final) — `milestone`, `update`, `rewidth` (7 branches). 4
  real tests, 3 honest ignores. `update.go`'s two flagged lines are
  NOT actor-shaped (the verb takes no `actor`/`principal` params) — one
  is the standard `ResolveRoot` ignore, the other is a verb-specific
  G45 hook-chain-collision guard needing its own fixture. `rewidth.go`'s
  two flagged lines are `out.JSON()` output-format branches, not error
  guards — a different shape than every other flagged line in this
  milestone, both trivially real-tested rather than ignored. Commit
  `a09b1edb`.

Final scoped rerun (`branch-coverage-audit`, base = the pre-M-0238
commit, all ten files, against the fully-corrected state): zero
findings.

## Decisions made during implementation

- None beyond the per-wave judgment calls recorded above — no decision
  rose to the level of a tracked `D-NNN`/`ADR-NNNN`.

## Validation

`go build ./...`, `go vet ./...`, `gofumpt -l` (clean except one
pre-existing, untouched-by-this-milestone formatting drift in
`internal/cli/add/add.go` lines ~516-563), `golangci-lint run` (0
issues) — all clean against the final committed state. Full suite:
`go test -race -parallel 8 ./...` green throughout every wave and
after the corrective fix. `aiwf check` — 0 errors, 3 pre-existing
warnings unrelated to this diff. `make coverage-gate` (standard base)
— all four policies pass. Doc-lint (scoped to this milestone's
change-set): no findings — the diff touches no file under `docs/`,
`README.md`, or `CONTRIBUTING.md`.

Three independent fresh-context reviewers (code-quality lens, one per
wave, no authorship attachment) verified every load-bearing claim by
measurement — reran the branch-coverage audit per wave, traced test
claims against actual source, cross-checked `//coverage:ignore`
rationales against the real guarded code. Verdicts: wave 1
**request-changes** (see Reviewer notes), waves 2 and 3 **approve**.
A fourth, narrower reviewer pass confirmed the wave-1 corrective fix.
The design-quality (`wf-rethink`) lens was skipped by operator
decision — this milestone introduces no new module/package boundary,
core abstraction, or data model, only mechanical coverage backfill.

## Deferrals

- **G-0412** — the blessed `ResolveRoot` `//coverage:ignore` rationale
  text (inherited from `archive.go:127`, now copied into nine files
  across M-0252/M-0253) is factually imprecise about what actually
  makes `ResolveRoot` fail, though the ignore itself is legitimate.
  Flagged independently by two reviewers; tracked for a repo-wide
  wording sweep rather than a piecemeal fix mid-epic — genuinely
  deferred, since it spans files outside this milestone.

## Reviewer notes

Wave 1 (`add`/`cancel`/`editbody`, commit `d716478c`) came back
**request-changes**: the `//coverage:ignore` on both files'
`LoadTreeWithTrunk` guard claimed the branch "only fails on
filesystem/git IO failure," which is false — `LoadTreeWithTrunk` calls
`config.Load`, and a syntactically malformed `aiwf.yaml` produces a
real, deterministically-reproducible parse error that propagates
through it. The reviewer proved this by triggering the branch in a
throwaway unit test. This was exactly the kind of untested error path
E-0064 exists to catch, hidden behind an inaccurate ignore rather than
genuinely absent. Fixed in commit `d3b147c8`: both ignores removed,
replaced with `TestRun_LoadTreeWithTrunkConfigParseFailure` in each
package, using a malformed `aiwf.yaml` to force the real parse-error
path. A fourth, narrowly-scoped reviewer pass independently confirmed
the fix resolves the finding with no new issues.

Wave 2's reviewer separately found `promote.go:128` (the
`gateFlag = "--audit-only"` reassignment inside the `--force`/
`--audit-only` `--reason` gate) genuinely reachable and untested —
outside AC-1's own charter since the line predates the audit's diff
base, so the mechanical gate never flagged it. Filed as G-0411 and
closed inline in this same milestone (commit `3c20ac67`, a
table-driven extension of `TestRun_ForceRequiresReason`) rather than
via a standalone `wf-patch`: the patch would have had nothing to
extend, since `promote_error_paths_test.go` doesn't exist on `main`
yet — only on this branch — so a trunk-based patch right now would
have had to recreate the file from scratch, duplicating work about to
land anyway at epic wrap. Fixing it in-context, while the file and
review context were already loaded, was the smaller change.

Wave 3 came back a clean approval with only a non-blocking note
(tracked as G-0412 above). No deliberate trade-off or rejected
approach beyond what's recorded in Decisions and Deferrals.

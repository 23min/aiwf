---
id: G-0386
title: Backfill test coverage for ~194 untested CLI verb error-handling branches
status: open
discovered_in: M-0238
---
## What's missing

Mechanical test coverage for ~194 pre-existing, never-exercised error-handling
branches across 31 files under `internal/cli/`, surfaced by M-0238/AC-3's
mechanical bare-print-call migration (every `fmt.Fprintf(os.Stderr, ...)` /
`fmt.Println(...)` / etc. call site rewritten to route through the `cliutil`
text-output wrapper set). The migration changed only the print-call text —
same stream, same bytes, no logic change — but the enclosing coverage block
(typically an `if err != nil { <print>; return ... }` guard around
`cliutil.ResolveRoot`, `cliutil.ResolveActor`, `cliutil.AcquireRepoLock`,
`tree.Load`, or a verb-specific validation step) had never been exercised by
any test before the rename touched a line inside it. The diff-scoped
`branch-coverage-audit` policy (`internal/policies/branch_coverage_audit.go`,
run via `make coverage-gate` against `git merge-base origin/main HEAD`) flags
the block's start line whenever any statement inside it changes, regardless
of whether the change altered behavior — which is what surfaced this now.

Affected files and line counts (block-start lines the audit names):

- `internal/cli/add/add.go` (17), `internal/cli/authorize/authorize.go` (8),
  `internal/cli/cancel/cancel.go` (6), `internal/cli/check/check.go` (10),
  `internal/cli/cliutil/statusline.go` (7), `internal/cli/contract/bind.go` (4),
  `internal/cli/contract/recipes.go` (19), `internal/cli/contract/unbind.go` (3),
  `internal/cli/contract/verify.go` (5), `internal/cli/doctor/doctor.go` (1),
  `internal/cli/doctor/selfcheck.go` (15), `internal/cli/editbody/editbody.go` (4),
  `internal/cli/history/history.go` (6), `internal/cli/importcmd/importcmd.go` (9),
  `internal/cli/initcmd/initcmd.go` (4), `internal/cli/list/list.go` (4),
  `internal/cli/milestone/milestone.go` (3), `internal/cli/move/move.go` (4),
  `internal/cli/promote/promote.go` (10), `internal/cli/reallocate/reallocate.go` (3),
  `internal/cli/rename/rename.go` (3), `internal/cli/render/render.go` (10),
  `internal/cli/retitle/retitle.go` (3), `internal/cli/root.go` (1),
  `internal/cli/schema/schema.go` (2), `internal/cli/show/show.go` (6),
  `internal/cli/status/status.go` (8), `internal/cli/template/template.go` (2),
  `internal/cli/update/update.go` (3), `internal/cli/upgrade/upgrade.go` (13),
  `internal/cli/whoami/whoami.go` (1).

The exact line numbers are reproducible on demand: `make coverage-gate` on
the `epic/E-0061-diagnostic-logging-and-correlation` branch (or any branch
carrying M-0238's commits) names every one.

## Why it matters

These branches will block the eventual `E-0061` epic-to-`main` push: the
coverage gate's base ref is `origin/main`, not the epic's own fork point, so
the finding persists regardless of which branch carries the print-call
rename. Left alone, whoever wraps `E-0061` inherits an undiagnosed wall of
~194 findings at the least convenient moment (mixed in with whatever
M-0239 also changes).

More importantly, the underlying gap is real independent of M-0238: a large
fraction of this codebase's CLI-verb infrastructure-failure paths (root
resolution, actor resolution, repo-lock contention, tree-load failure) have
no test asserting they print the right message and return the right exit
code. Some are genuinely difficult to trigger deterministically — this repo
already has precedent for that exact judgment call at
`internal/cli/archive/archive.go:120` (`//coverage:ignore
cliutil.ResolveRoot only fails on missing aiwf.yaml + non-existent --root
path`) — but most of the 31 affected files never went through that exercise
at all.

Recommended remediation: a new small epic (not `E-0061`, whose own Context
section says it was scoped separately specifically so it could ship on its
own schedule), branched from `main` directly — independent of and in
parallel with `E-0061`. Since the affected lines exist on `main` today in
their pre-rename form, this can proceed without waiting on `E-0061`, and
because the fix and M-0238's rename touch different lines within the same
guards, there is no merge conflict either direction. If it lands on `main`
before `E-0061` pushes, `E-0061` inherits the resolution automatically via
the coverage gate's merge-base recomputation — no coordination required.
The work itself: per flagged site, either a real test (where the failure
condition is genuinely triggerable — a malformed entity file, a bad
`--format` flag) or an honest `//coverage:ignore` naming why it isn't
(mirroring the `archive.go` precedent), never a blanket suppression without
that judgment call.

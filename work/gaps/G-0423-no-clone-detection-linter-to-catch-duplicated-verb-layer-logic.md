---
id: G-0423
title: No clone-detection linter to catch duplicated verb-layer logic
status: open
priority: medium
---
## What's missing

No linter or policy in this repo detects structurally duplicated code across
functions, files, or packages. `.golangci.yml`'s enabled set is `bodyclose,
errcheck, errorlint, forbidigo, gocritic, gosec, govet, ineffassign,
misspell, revive, staticcheck, thelper, unconvert, unused` — `gocritic`'s
duplicate-detection checks (e.g. `dupSubExpr`) are statement-local within a
single function; none of the enabled linters compare structurally similar
function bodies across files or packages. A dedicated clone detector
(`dupl` is the standard golangci-lint plugin for this) is not enabled.

This class of defect is invisible to every other guardrail in the repo by
construction: a duplicated helper can be independently correct, independently
well-tested, and pass 100% branch coverage and mutation-kill rate in both
copies — none of those tools compare one function against another.

## Evidence

A single 2026-07-18 audit
([`docs/initiatives/verb-layer-cleanup.md`](../../docs/initiatives/verb-layer-cleanup.md))
found this pattern recurring at least four times, none caught before the
audit:

- `internal/verb/rename.go:139-182` (`renamePaths`/`substituteSlug`) and
  `internal/verb/reallocate.go:435-471` (`reallocatePaths`/`substituteID`)
  are structurally identical (finding F2); `reallocate.go:460` already
  comments "same shape as substituteSlug."
- The same `failX`/`emitXEnvelope`/`withCommitSHA` triad is independently
  reimplemented three times: `internal/cli/archive/archive.go` (helpers
  spanning roughly lines 243-291), `internal/cli/rewidth/rewidth.go:214,232,250`,
  and `internal/cli/importcmd/importcmd.go:257,275,293` (finding F4) — each
  admitted by its own comments as a mirror of the others, and verified as a
  real gap in `cliutil.FinishVerb`'s contract (no dry-run/multi-`Plan`
  support), not mere copy-paste laziness.
- `internal/check/reflog_walk.go:138` duplicates
  `internal/cli/check/isolation_escape_oracle.go:324`'s ref-listing
  instead of both calling `gitops.LocalBranchRefs` (finding F7).
- `internal/cli/doctor/doctor.go:488,605,690,741` hardcodes hook-marker
  strings that `internal/initrepo/initrepo.go:1412-1422,1579` already
  exports specifically so doctor wouldn't have to, and
  `internal/initrepo/initrepo.go:850`/`internal/cli/doctor/guidance.go:46`
  independently implement the same CLAUDE.md-marker check (finding F9).

Each instance individually was small enough to never justify its own
investigation; a mechanical clone detector would have flagged all four at
the PR that introduced the second copy.

## Direction

Enable `dupl` (or equivalent) in `.golangci.yml`, tuned to a threshold that
catches whole-function-body clones like the four above without excessive
noise. Expect and document legitimate exceptions up front — e.g.
`isolation_escape_oracle.go`'s ref-listing variant has a documented perf
reason to diverge (it batches in `%(objectname)`) and should be an explicit
exclusion, not a false-positive fire on every CI run. Treat this as an
ongoing lint tripwire against future recurrence, not a one-time cleanup
pass — the four existing instances are tracked for cleanup separately in
the initiative doc's scoped cleanup list (F2, F4, F7, F9).

## Provenance

Surfaced during a 2026-07-18 verb-layer call-graph audit
([`docs/initiatives/verb-layer-cleanup.md`](../../docs/initiatives/verb-layer-cleanup.md)),
then generalized in a follow-up discussion about why no existing test,
lint, or policy had caught any of the four instances.

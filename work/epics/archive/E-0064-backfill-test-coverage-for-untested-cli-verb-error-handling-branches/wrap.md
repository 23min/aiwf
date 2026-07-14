# Epic wrap — E-0064

**Date:** 2026-07-14
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0064-backfill-test-coverage-for-untested-cli-verb-error-handling-branches
**Merge commit:** 39cc011d

## Milestones delivered

- M-0252 — Shared CLI-verb failure fixtures and non-CLI infra coverage backfill (merged 8f362b7d)
- M-0253 — Entity-lifecycle verb coverage backfill (merged b72b4f73)
- M-0254 — Contract subsystem coverage backfill (merged f3e8e65d)
- M-0255 — Diagnostic and introspection verb coverage backfill (merged 621d1d6b)
- M-0256 — Bulk-input verb coverage backfill (merged e7e490c2)

## Summary

Closes G-0386: every CLI-verb error-handling branch M-0238/AC-3's mechanical
print-call migration surfaced as untested — 188 branches live across 39 files
at epic-open — now carries either a real regression test or an honest
`//coverage:ignore <reason>` naming the specific condition that makes it
untestable. M-0252 built the shared failure fixtures (root/actor-resolution
failure, repo-lock contention, malformed/corrupt tree, a bad `--format` flag)
the four consumer milestones reused; M-0253 through M-0256 split the
remaining verb groups by shared guard shape (entity-lifecycle, contract,
diagnostic/introspection, bulk-input), each closing its group's branches
against the same `AIWF_COVERAGE_BASE=2ac84846^` scoped audit. Two files
(`archive`/`authorize`, folded into M-0255; `initcmd`, folded into M-0256)
surfaced during implementation as unassigned to any milestone from planning —
both folded into the nearest milestone rather than opening a sixth, keeping
the epic's zero-findings bar reachable without disproportionate ceremony for
a handful of lines. `make coverage-gate` run with the pre-M-0238 base now
reports zero findings across the full 39-file scope.

## Doc findings

Scoped to the epic's full change-set against `main` (85 files, `internal/**`
Go source/tests plus entity markdown under `work/`). No file under `docs/`,
`README.md`, or `CONTRIBUTING.md` intersects this change-set, so every
`wf-doc-lint` check (broken code references, removed-feature docs, orphan
files, TODOs, broken links, stale CLI invocations, structural issues) is
scope-empty by construction — clean, no findings.

## ADRs ratified

- none

## Decisions captured

- none — this epic's scope required no new architectural decisions; the two
  mid-flight scope adjustments (folding `archive`/`authorize` into M-0255,
  `initcmd` into M-0256) were tactical planning calls surfaced and confirmed
  in-conversation, not the kind of durable "why we did it this way" reasoning
  an ADR/D-NNNN exists to preserve.

## Follow-ups carried forward

- G-0412 — ResolveRoot `//coverage:ignore` rationale text is inaccurate
  across multiple files (discovered during M-0254; a milestone-0256 reviewer
  independently re-derived the same finding for four newly-authored sites,
  which got fixed in place since they were fresh, but the ~20 pre-existing
  sites from M-0253/M-0254/M-0255 remain — deliberately deferred as a
  repo-wide sweep, not a per-milestone fix)

G-0411 (an untested reachable branch in `promote.go`'s audit-only `--reason`
gate, discovered during M-0253/M-0254) was addressed within the epic and does
not carry forward.

## Handoff

G-0386's named debt is closed: the diff-scoped `branch-coverage-audit` policy
no longer fires against any pre-M-0238 baseline finding, and the standard
`origin/main`-scoped `make coverage-gate` stays clean through every commit in
this epic. The reusable failure fixtures M-0252 built
(`internal/cli/cliutil/testutil`) are now a standing asset for any future
CLI-verb work needing the same triggers. The one deliberately open item is
G-0412's repo-wide sweep — next epic to reach for it, not blocking anything
today.

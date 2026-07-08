# Epic wrap — E-0061

**Date:** 2026-07-08
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0061-diagnostic-logging-and-correlation
**Merge commit:** 870f88bc

## Milestones delivered

- M-0237 — Logger core: internal/logger package and concurrent-append safety (merged a56da95b)
- M-0238 — Migrate bare-stderr call sites; forbidigo chokepoint (merged 0f4f1114)
- M-0239 — Correlation id wiring; ratify ADR-0017 (merged 075bcfec)

## Summary

aiwf now has a retrace-ready diagnostic surface. `internal/logger` wraps `log/slog`
behind an opt-in, default-off resolution chain (`AIWF_LOG`/`AIWF_LOG_FORMAT`/
`AIWF_LOG_FILE`, then `aiwf.yaml`'s schema-registered `logging:` block, then silent
discard), writing structured, concurrent-safe (`O_APPEND`, one `Write()` per record)
records to a daily-rotated file under `$XDG_STATE_HOME` (M-0237). Every named
bare-stderr call site migrated to the bound logger or the `cliutil` text-output
wrappers, backed by both a `forbidigo` lint rule and an independent AST policy so the
discipline survives even if the linter is ever disabled (M-0238). A single
per-invocation `correlation_id` now threads from the Cobra root through every mutating
verb into its JSON envelope, matching the same invocation's log `run_id`; every
mutating verb reports its own verb-appropriate metadata in that envelope; an operator
can pass `--trace` to see per-phase timings without configuring logging first; and
ADR-0017 — the design this epic implements — now reads `accepted`, with CLAUDE.md's
own CLI-conventions text corrected to match the shipped behavior (M-0239).

## ADRs ratified

- ADR-0017 — Opt-in slog diagnostic logging, default off, XDG state-home file route
  (`proposed -> accepted`, M-0239/AC-5 — the epic's own closing act, not a new
  decision this epic introduced)

## Decisions captured

- none as separate ADR/D entities. The one cross-cutting correction this epic made —
  the `logging:` block's parsing/validation lives in `internal/logger`, not
  `internal/aiwfyaml` as originally named — was folded directly into ADR-0017's own
  Consequences section rather than spawned as a new entity, since it corrects that
  ADR's own prior text. The remaining mid-flight decisions (M-0237's
  `OpenDestination` empty-`HOME` refusal; M-0238's AC-5/AC-6 milestone-scope call and
  `upgrade`'s `verb.completed`-on-success-not-exit-code timing) are narrow,
  implementation-scoped calls, adequately captured in their originating milestone
  spec's own `## Decisions made during implementation` section — none crosses the bar
  of a strategy reversal, a new cross-cutting default, or an ADR supersession that a
  future reader would need a dedicated entity to find.

## Follow-ups carried forward

- G-0223, G-0232, G-0382 — promoted `addressed` at this wrap (fully implemented by
  this epic's own milestones; see each gap's `addressed_by` for the resolving
  milestone).
- G-0383 — carried forward, still `open`. M-0238 deliberately decided that none of
  the five migrated call sites logs a path-shaped value outside the three fields
  `WithVerb` already scrubs (verb/entity/actor), so no current call site needs
  broader `os.Args` scrubbing — and corrected ADR-0017's own wording, which had
  overstated the guarantee. But `scrubHomePaths` remains unexported and reachable
  only through `WithVerb`; the underlying risk (a *future* call site logging a path
  under a different key, with no mechanical guard) is unresolved, not eliminated.
- G-0386 — carried forward, still `open`. ~194 pre-existing, never-exercised
  `internal/cli/*` error-handling branches, incidentally surfaced (not introduced) by
  M-0238's mechanical print-call migration. Deliberately out of scope for this epic
  (disproportionate to a diagnostic-logging epic); recommended remediation is a
  separate epic branched from `main`. `make coverage-gate` on this epic branch fails
  exclusively on this pre-existing set — confirmed by diffing against the gap's own
  enumerated file list, not introduced by any of this epic's three milestones.
- G-0387 — carried forward, still `open`. The `verb.completed`/`verb.failed`
  diagnostic event (M-0238/AC-5, AC-6) carries `verb`/`entity`/`actor`/`run_id` and,
  for `cancel`/`move`, a `sha` — but no duration, and there is no `verb.started`
  event to measure one from. Filed as a gap candidate rather than added to M-0238's
  own scope; its own text names M-0239 as "a natural candidate" since M-0239 touches
  the same `EmitVerbOutcome` seam for `correlation_id` wiring, but M-0239 did not
  pick it up — deliberately deferred, not overlooked.

## Doc findings

Scoped to the epic's own change-set (164 files, 12 touched markdown docs) computed
against current `main` — mainline was reconciled into the epic branch before this
sweep, so the diff isolates exactly what E-0061 itself adds, not mainline's own
unrelated commits that happened to land in the same window.

- **Broken code references:** none live. Every backticked `internal/*.go` path
  across the 12 touched docs was checked against the current tree and resolves,
  with two expected exceptions, both historical prose rather than live references:
  M-0237's own Work log line names `internal/cli/output/outputformat.go` only to
  say it was "found and fixed" (past tense) to the current `internal/cli/cliutil/
  outputformat.go` path — that fix already happened. G-0223's "What's missing"
  section (frozen prose from before this epic started) names the pre-migration
  call sites (`internal/cli/statusline.go`, `internal/verb/cancel.go`,
  `internal/verb/upgrade.go`) that M-0238 subsequently moved as part of the very
  fix G-0223 asked for — a gap's problem statement describing the pre-fix state
  it was filed against, not a live reference expected to track the tree.
- **Removed-feature docs:** none — this epic only added behavior.
- **Orphan files:** the four new gap entities (G-0382, G-0383, G-0386, G-0387) are
  out of scope for this check — they're aiwf-native entities discoverable via
  `aiwf list --kind gap` / `aiwf show`, each carrying its own `discovered_in`
  back-reference, not narrative docs that need an inbound link to stay reachable.
- **Documentation TODOs:** none. Three `TODO`/`FIXME` string hits across the
  touched docs are all prose mentioning the word ("Stubs and TODOs in shipped
  code are a smell," two "no TODOs found" doc-lint self-reports) — no actual
  work-tracking marker.

Clean — 0 actionable findings.

## Handoff

`internal/logger`, `cliutil.ResolveLogger`/`ResolveTraceLogger`, `verb.Result.Metadata`,
and `OutputFormat.CorrelationID`/`Metadata` are now real, tested infrastructure — a
future diagnostic-logging need (a new bound field, a new per-verb metadata shape) is
an addition to an existing seam, not new plumbing. Epic 2 (the correctness stress
harness named in `docs/initiatives/robustness-correctness-stress-testing.md`) can now
build its own RCA value on top of the shipped `AIWF_LOG*`/`correlation_id` surface, per
this epic's own dependency note. G-0383 and G-0386 remain open, deliberately not
folded into this epic's own scope.

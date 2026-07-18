---
id: G-0422
title: No documented allowlist for verbs exempt from projectionFindings
status: addressed
priority: medium
addressed_by_commit:
    - 796c0d0
---
## What's missing

`internal/verb/verb.go`'s package doc states, unconditionally, that every
verb runs the projection check (`projectionFindings` →
`internal/verb/common.go:123`) before writing. That's not true today, by
design: `setarea.go`, `setpriority.go`, `renamearea.go`, `archive.go`,
`rewidth.go`, `authorize.go`, `acknowledgeillegal.go`,
`acknowledgemistag.go`, `linkrewrite.go`, and `contractrecipe.go` all skip
it, correctly, because `check.Run` (what `projectionFindings` wraps) only
covers rules computable from in-memory tree state — the fields these verbs
mutate are validated only by CLI-composed, git-history-dependent rules
(e.g. `area-mistag`/`area-unknown`/`area-overlap`, which need a
`touchedByEntity` map built by scanning commit history) that can never fire
inside an in-memory projection regardless of which verb calls it. Those
rules are gated by the pre-push hook's full `aiwf check` run instead, by
design — `rewidth.go` already states this rationale explicitly for its own
case.

The actual, narrower invariant (which verbs must call `projectionFindings`,
and why the rest legitimately don't) exists only as tribal knowledge spread
across individual verb-file comments and one design doc
(`docs/pocv3/design/design-decisions.md:251`, which scopes the guarantee to
a named "current set"). There is no single place that states the rule, and
no check that the exempt set is exactly the verbs that should be exempt —
an accidental omission (a new verb that mutates a `check.Run`-reachable
field but skips `projectionFindings`) would look identical to one of the
legitimate, git-history-dependent exemptions, and nothing would catch the
difference.

## Evidence

- Confirmed by grepping every `internal/verb/*.go` file for
  `projectionFindings(`: present in `ac.go`, `add.go`, `editbody.go`,
  `import.go`, `milestone_depends_on.go`, `move.go`, `promote.go`,
  `reallocate.go`, `rename.go`, `retitle.go` — absent from the ten files
  listed above.
- `internal/check/check.go:109-158` (`check.Run`'s rule composition) does
  not include `AreaMistag`/`AreaUnknown`/`AreaOverlap` — those are
  assembled only in `internal/cli/check/check.go:245,266,295`, fed
  `touchedByEntity`, which is derived from a commit-history scan no verb
  performs.
- `area-mistag` is documented as warning-only, never escalating;
  `area-unknown`/`area-overlap` escalate to error only via
  `ApplyAreaRequiredStrict`, itself CLI-composed, never part of
  `check.Run`.

## Direction

This gap was originally scoped as "add a policy requiring
`projectionFindings` everywhere," on the mistaken premise that the omission
in `setarea.go`/`setpriority.go`/`renamearea.go` was an unenforced
invariant violation. Adversarial verification (2026-07-18) refuted that
premise — see the corrected direction below.

1. **Immediate fix**: correct `internal/verb/verb.go`'s package doc to
   state the actual rule — which verbs call `projectionFindings`, and the
   explicit reason the rest don't (git-history-dependent validation,
   gated at pre-push instead) — rather than the current unconditional
   "every verb" claim.
2. **Optional, more robust fix**: encode the exempt set as an explicit,
   reviewed allowlist (e.g. a small table in `internal/verb/verb.go` or a
   policy fixture) and add an AST policy — same walk-every-exported-verb
   shape as `internal/policies/verbs_validate_then_write.go`, opposite
   polarity — that asserts `projectionFindings(` is present in every
   exported `internal/verb/*.go` function returning `(*Plan, error)`
   *unless* it's on the allowlist. This doesn't change any verb's
   behavior; it makes the exemption set explicit and reviewable instead of
   implicit, so a future accidental omission (versus a legitimate,
   git-history-dependent one) is visible at the allowlist diff instead of
   requiring another full audit to rediscover.

## Provenance

Surfaced during a 2026-07-18 verb-layer call-graph audit
([`docs/initiatives/verb-layer-cleanup.md`](../../docs/initiatives/verb-layer-cleanup.md),
originally finding F1), initially scoped as "no presence check for
`projectionFindings`." A follow-up adversarial-verification pass the same
day refuted the original premise — `check.Run`'s area rules are
git-history-dependent and unreachable from `projectionFindings` regardless
of which verbs call it — and rescoped this gap to the documentation/
allowlist gap that survives the correction.

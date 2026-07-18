---
id: G-0422
title: No presence check that structural verbs call projectionFindings
status: open
priority: high
---
## What's missing

`internal/policies/verbs_validate_then_write.go` proves a *negative*
invariant: it walks every exported `internal/verb/*.go` function and asserts
a ban-list of write primitives is absent from the body. There is no
companion policy proving the matching *positive* invariant — that every
structural mutator's body contains a call to `projectionFindings`
(`internal/verb/common.go:123`) before it returns a `Plan`. Nothing catches
a verb that writes without validating; only three did (see Evidence), and
none were caught by the existing gate, by CI, by coverage, or by mutation
testing.

## Evidence

- `internal/verb/setarea.go`, `internal/verb/setpriority.go`, and
  `internal/verb/renamearea.go` each validate their new field value inline
  and go straight to `plan(&Plan{...})`, with no call to
  `projectionFindings`. Confirmed by grepping every `internal/verb/*.go`
  file for `projectionFindings(`: present in `ac.go`, `add.go`,
  `editbody.go`, `import.go`, `milestone_depends_on.go`, `move.go`,
  `promote.go`, `reallocate.go`, `rename.go`, `retitle.go` — absent from
  the three above.
- `check` has dedicated `area-mistag`/`area-unknown`/`area-overlap` rules
  that operate on exactly the fields these three verbs mutate, so a
  mistagged or unknown area introduced by `set-area`/`rename-area` is never
  caught at write time — only on the next unrelated `aiwf check` run, if
  anyone remembers to run one.
- Why nothing already caught this: branch-coverage gates and mutation
  testing (`mutate-hunt`) can only exercise/perturb code that exists — they
  have no way to demand a call that was never written. `internal/verb`'s
  own package doc states unconditionally that every verb runs the
  projection check before writing; that claim was never mechanically
  enforced, only asserted in a comment.

## Direction

Two independent pieces, sequenced:

1. **Immediate fix**: route `SetArea`, `SetPriority`, and `RenameArea`
   through `projectionFindings` like every other structural verb, closing
   the three known instances.
2. **Prevent recurrence**: add a companion AST policy alongside
   `PolicyVerbsValidateThenWrite` — same walk-every-exported-verb-function
   shape, opposite polarity: assert `projectionFindings(` (or an
   equivalent gate) *is present* in the body of every function on a
   maintained "structural mutator" list (or, more robustly, every
   exported `internal/verb/*.go` function whose signature takes a
   `*tree.Tree`/`tree.Tree` and returns `(*Plan, error)`, minus a narrow,
   named allowlist for verbs with a documented reason to skip it, e.g.
   `Cancel`'s and `Archive`'s own bespoke guards). This is the same
   presence-policy pattern the codebase already uses elsewhere
   (`internal/policies/test_setup_presence.go`,
   `internal/policies/skill_coverage.go`,
   `internal/policies/firing_fixture_presence.go`) — no new tooling
   category, just an uncovered instance of an existing one.

## Provenance

Surfaced during a 2026-07-18 verb-layer call-graph audit
([`docs/initiatives/verb-layer-cleanup.md`](../../docs/initiatives/verb-layer-cleanup.md),
finding F1), then traced to its root cause in a follow-up discussion about
why no existing test, policy, or mutation-testing pass had caught it.

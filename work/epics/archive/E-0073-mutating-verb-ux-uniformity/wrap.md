# Epic wrap — E-0073

**Date:** 2026-07-30
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0073-mutating-verb-ux-uniformity
**Merge commit:** b84bbb916

## Milestones delivered

- M-0281 — Same-state mutating-verb inputs return NoOp (merged c2fb192c)

## Summary

A mutating verb handed input that already equals current state now returns a
NoOp — exit 0, a descriptive message, and zero commits — instead of a Go error.
The convention was already live in `archive`, `rewidth`, `contract bind` and
`contract recipe install`; this epic finishes rolling it out and pins it with an
AST policy so it cannot rot back to one-of as new verbs land.

Scope grew from the six verbs the goal named to twelve operator-facing verbs
across sixteen code entry points. That growth is the premise being confirmed
rather than creep: the AC-6 policy invariant surfaced five verbs the original
source scan had missed, and a convention honoured by some verbs and not others
is exactly the half-rolled-out discipline the epic was filed against.

Three of the converging paths were **correctness** fixes, not ergonomics.
`acknowledge illegal` appended a duplicate audit record on every re-run, and
`milestone tdd`, `milestone depends-on` and `edit-body --body-file` each landed a
commit with an empty diff — aiwf does not refuse a same-tree commit, so a
byte-identical write is one file op that commits cleanly. The reframing that
made this tractable is in the two ADRs: converge on the verb's *declared effect*,
after resolving every argument against something real.

## ADRs ratified

- ADR-0036 — same-status FSM transitions converge to NoOp, not refusal
- ADR-0037 — retitle re-derives the slug only while it tracks the title

## Decisions captured

None as `D-NNNN` entities. The two decisions worth keeping were architectural and
became the ADRs above; the dry-run deferrals are recorded in this epic's
*Out of scope* section and mirrored into G-0230's own.

## Follow-ups carried forward

Every gap below was discovered while delivering M-0281 and is deliberately left
open. None is a precondition for anything this epic shipped.

- G-0458 — AC `tdd_phase` same-phase input: converge or keep refusing
- G-0459 — event-shaped verbs append duplicate records on identical re-run
- G-0460 — repeat `authorize` leaves two active scopes on one entity, no finding
- G-0461 — composite `--for-entity` acks never suppress the rule they target
- G-0462 — intermittent test failures: ETXTBSY on exec, golangci-lint cache contention
- G-0463 — `edit-body --body-file` is not body-only; frontmatter drift rides the commit
- G-0464 — check predicates treat deferred ACs as non-terminal
- G-0465 — no chokepoint catches shipped surfaces drifting from verb behavior
- G-0466 — structured-state verbs commit a hand-edited frontmatter field as their own
- G-0471 — no chokepoint detects a verb run by a binary older than the worktree's source

G-0462 originally also described timing-dependent stress-scenario oracles. That
account is superseded by G-0468, filed independently with a stronger analysis —
four scenarios across three oracle-defect shapes — so G-0462 now covers only the
two mechanisms G-0468 does not.

## Handoff

**Ready.** The convergence convention is documented in CLAUDE.md
§"Same-state convergence — resolve, then converge" as two ordered rules: resolve
every argument before asking whether the request is satisfied, then converge when
the verb's declared effect already holds. `verb_result_noop_invariant` enforces
that a new mutating verb either carries a same-state NoOp assertion or an
allowlist entry with a stated reason, so the next verb author is told rather than
trusted.

**Deliberately open.** The dry-run half of G-0230 is decided, not pending —
`reallocate` dry-run is YAGNI until a collision incident occurs, and `rename`
dry-run is rejected because dry-run-by-default is a regression for a hot
interactive single-entity verb. Refile against a real incident rather than
re-deriving the argument.

**Worth knowing for the next epic.** Two of the follow-ups above are about
chokepoints this epic's own work depended on and found missing: G-0465 (nothing
catches a shipped surface drifting from the verb it documents) and G-0471
(nothing catches a verb run by a stale binary). Both were surfaced by reading and
by accident respectively, which is the argument for each — neither detector
exists, so neither scales.

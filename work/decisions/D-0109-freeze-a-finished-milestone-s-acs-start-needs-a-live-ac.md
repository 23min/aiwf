---
id: D-0109
title: Freeze a finished milestone's ACs; start needs a live AC
status: proposed
relates_to:
    - D-0039
    - D-0004
---
> **Date:** 2026-11-16 · **Decided by:** human/peter

## Question

Two cases no record settles:

1. May an AC change status after its milestone is `done` or `cancelled`?
   `docs/design/design-decisions.md` lets a `met` AC move to `deferred` or `cancelled` "if scope
   changes after the fact", and the comment on `acTransitions` in `internal/entity/transition.go`
   and the spec table in `internal/workflows/spec/rules.go` say the same; none names the
   milestone's status. Pass B's R-FP-0064 and D-0004 say ACs "cannot outlive the milestone", and
   the same `acTransitions` comment calls descoping a met AC legitimate "while its parent
   milestone runs".
2. For D-0039's refusal of a zero-AC start, does a milestone whose every AC is `deferred` or
   `cancelled` count as having no ACs? D-0039 speaks of a milestone with no AC at all, M-0268 of
   an empty `acs[]`, and a cancelled AC stays in `acs[]` at its position.

## Decision

- Freeze a finished milestone's ACs: once the milestone is `done` or `cancelled`, refuse every
  change to an AC's status, so a `met` AC is not descoped after its milestone closed.
- Except an `open` AC on a finished milestone: allow it to be promoted to `met`, `deferred` or
  `cancelled`. That state is already a `milestone-done-incomplete-acs` or
  `milestone-cancelled-incomplete-acs` error, which a merge or a hand edit produces, and closing
  the AC is its repair.
- Keep same-status convergence: a request for the status an AC already has succeeds with no
  commit (ADR-0036), frozen or not.
- Require an `open` or `met` AC to start: refuse `draft → in_progress` when every AC is
  `deferred` or `cancelled`, as D-0039 refuses a milestone with none, under the same
  `--force --reason` override.

## Reasoning

**The freeze.** The `acTransitions` comment places descoping inside the milestone's run, D-0004
says ACs "cannot outlive the milestone", and E-0033 gave "you can't promote an AC to met after
the parent milestone is done" as its example of a cross-verb sequence rule. A finished
milestone's record then stays what it was when it closed, and a later change of scope becomes a
new entity that references it, as for every other terminal entity (ADR-0004). It is also what
happened in practice. On main at 2de605437, replaying in order every commit that
`git log --reverse --grep='aiwf-entity: M-'` lists shows no AC status changed after its
milestone finished. When a finished milestone needed more work, it was reopened with `--force`
instead: `git log --grep='done -> in_progress'` lists seven such commits, on M-0072, M-0073,
M-0074, M-0078 and M-0106. Leaving the moves allowed rests on the "after the fact" wording in
design-decisions.md, the `acTransitions` comment and the spec table, none of which names the
milestone's status.

**The exception.** No verb leaves a finished milestone with an open AC, but a merge does with no
one at fault (G-0575), and a total freeze would leave reopening with `--force`, a sovereign act,
as the only exit. The "Milestone-done implies AC progress" bullet in design-decisions.md, the
finding's hint in `internal/check/hint.go` and the `aiwf-promote` skill already expect the open
ACs to be closed after the milestone finished. This departs from E-0033's example only for that
error state. The repair works today only while one AC is open; with two, each close is refused
(G-0574).

**A live AC to start.** D-0039 refuses a zero-AC start because such a milestone can traverse its
lifecycle "without ever substantiating anything", and rejects a warning because the discipline
could be "dodged by omitting the AC entirely". Adding an AC and cancelling it is the same dodge
one step longer. The comment on `requireNonEmptyACsAtMilestoneStart` in `internal/verb/promote.go`
gives the guard's purpose: with zero ACs there is "no contract yet" for what done means. A
`deferred` or `cancelled` AC is "off the milestone's contract" (the comments in
`internal/entity/transition.go` and `internal/verb/ac.go`; G-0464 and G-0695 say likewise).
Counting entries, the literal reading, keeps the dodge open. Counting a `deferred` AC as live
was rejected too: deferred work is not delivered in this milestone either. In the same replay no
milestone ever started with every AC deferred or cancelled, so the rule changes nothing that
has happened.

## Consequences

- The kernel allows both today; the gaps that cite this decision track the changes.
- D-0039 stays in force. This decision refines what its point 1 counts as an AC.

---
id: G-0638
title: The epic branch guard's --force bypass is recorded nowhere and gated by nothing
status: open
---
## What's missing

`--force` bypasses the epic branch guard, and that bypass is recorded nowhere
and gated by nothing. The comment above the guard says the opposite.

`internal/verb/add_branch_guard.go:44-45`:

> *opts.Force is the bypass, already sovereign and human-only on this verb.*

The guard it documents,
`refuseEpicCreationOnRitualBranch`, returns early on `kind != entity.KindEpic
|| opts.Force`, so the flag does bypass a real refusal. Neither half of what
the comment claims about it holds:

- **No record.** `internal/verb/add.go` stamps `aiwf-force:` and carries
  `--reason` into the commit body only when `gateBypassed` is true, and
  `requireNonEmptyBornCompleteBody` sets that only for
  `entity.IsBornComplete(kind)` — ADR, gap, decision, contract
  (`internal/entity/entity.go:74-81`). Epic is not among them, so a forced
  epic creation on a ritual branch commits with no force trailer and no
  reason.
- **Not human-only.** `CheckForceTrailerCoherence`
  (`internal/verb/coherence.go:317`) reads the assembled trailer set. With no
  force trailer there is nothing for it to refuse.

Two comments in the same package disagree. `add.go:100-101` describes the flag
correctly for the gate it was built for — *"Has no effect on kinds the gate
doesn't apply to (epic, milestone) — passing it there is inert"* — and
`add.go:216-231` reasons carefully that a no-op force must not fabricate a
sovereign-override record. That reasoning is right for the born-complete gate.
The branch guard, added later, reuses the same flag for a case where it is not
a no-op, and inherits provenance machinery scoped to the other gate.

## Why it matters

A guard worth adding is worth knowing when someone stepped over. The branch
guard exists because an epic created on a ritual branch could not be activated
at all; overriding it is a deliberate act with consequences for whoever finds
the entity later. `aiwf history` cannot show that it happened, and the
`provenance-force-non-human` audit, which walks history for commits no verb
produced, has no trailer to walk.

The comment is the sharper half. A reader deciding whether the guard needs an
actor check finds one asserted, and stops. That is how a false comment costs
more than an absent one.

Worth measuring before sizing the fix, not assumed here: whether an `ai/` actor
is blocked from this path anyway by the scope guards, which fire independently
of `--force`. The command is `aiwf add epic --title T --force --reason R
--actor ai/claude` on a ritual branch in a disposable repo, with and without an
active authorize scope. If those guards already refuse it, the exposure is the
missing record rather than an unguarded override, and the fix shrinks
accordingly.

## Resolution shape

Two independent decisions, and the first does not wait on the second.

**Correct the comment.** It states a guarantee the code does not provide. That
is true regardless of what the guard should do, and it is a one-line edit.

**Then decide whether the bypass should record itself.** Stamping the force
trailer when the branch guard is bypassed would make it visible to
`aiwf history` and subject it to the human-only rule, at the cost of extending
a trailer whose current contract — as `add.go:216-231` argues at length — is
that it marks an override that actually happened. That argument supports
stamping here: this bypass *is* an override that happened. The question is
whether `--reason` should then become required for a forced epic add, as it is
where the born-complete gate fires, and that is a CLI-surface change rather
than a comment fix.

## Where to fix

- `internal/verb/add_branch_guard.go:44-45` — the false claim.
- `internal/verb/add.go:210-235` — where the trailer decision lives, if the
  second question resolves toward recording the bypass.
- `internal/verb/add.go:95-102` — the `Force` field's own comment, which is
  accurate for the born-complete gate and silent about the branch guard now
  sharing the flag.

---
id: G-0638
title: The epic branch guard's --force bypass is recorded nowhere and gated by nothing
status: open
priority: medium
---
## What's missing

`--force` bypasses the epic branch guard, the bypass is recorded nowhere, and
the human-only rule for `--force` never sees it. The comment above the guard
says the opposite.

`internal/verb/add_branch_guard.go:44-45`:

> *opts.Force is the bypass, already sovereign and human-only on this verb.*

The guard it documents,
`refuseEpicCreationOnRitualBranch`, returns early on `kind != entity.KindEpic
|| opts.Force`, so the flag does bypass a real refusal. Neither half of what
the comment claims about it holds:

- **No record.** `internal/verb/add.go` stamps `aiwf-force:` and carries
  `--reason` into the commit body only when `gateBypassed` is true, and
  `requireCompleteBody` sets that only when the body would otherwise have
  failed the body-completeness gate: a required section absent, or, for a
  born-complete kind, present and empty. An epic created from its scaffold
  passes that gate, so a forced epic creation on a ritual branch commits with
  no force trailer and no reason.
- **Not human-only at the verb.** `CheckForceTrailerCoherence`
  (`internal/verb/coherence.go:317`) reads the assembled trailer set, and with
  no force trailer there is nothing for it to refuse. The scope check stops an
  `ai/` actor with no open scope, or with one the new epic's references do not
  reach, but not one whose `--relates-to` names the scope entity. That forced
  create commits, and the check-time `provenance-authorization-out-of-scope`
  rule reports it at error severity on the next full `aiwf check`, the pre-push
  hook's included, because that rule judges the created epic rather than its
  references.

Measured at `3c5bd7546` (git 2.51.1, Linux) in a throwaway repository on the
ritual branch `patch/G-0001-x`, where `aiwf add epic --title T` exits 2 with
the guard's refusal:

- `aiwf add epic --title T --force --reason R` exits 0, and its commit carries
  only the `aiwf-verb`, `aiwf-entity` and `aiwf-actor` trailers: no
  `aiwf-force:`, and no reason in the body.
- The same command with a `--body-file` holding only `## Goal` exits 0 with
  `aiwf-force: R` and the reason in the body, because that body fails the
  body-completeness gate, which `--force` then bypasses.
- With `--actor ai/claude --principal human/t` it exits 1 with
  `provenance-no-active-scope`. On the branch of another epic whose scope
  `aiwf authorize` opened for `ai/claude`, it exits 1 with
  `provenance-authorization-out-of-scope`; adding `--relates-to` naming that
  epic, it exits 0, and the commit carries `aiwf-actor: ai/claude` and
  `aiwf-authorized-by:` but no `aiwf-force:`. `aiwf check`, with no upstream
  configured, then exits 1 with an error-severity
  `provenance-authorization-out-of-scope` finding on that commit.

Comments in the same package disagree. The `Force` field's comment
(`add.go:95-103`) calls the flag inert "where the resolved body would have
passed the gate anyway, which includes every create that takes the kind's
scaffold", and the `--force` help in `internal/cli/add/add.go` says the same;
on a ritual branch a forced epic creation from its scaffold is not inert. The
comment on the trailer decision (`add.go:216-236`) reasons carefully that a
no-op force must not fabricate a sovereign-override record, and still names
epic and milestone as kinds with no gate to bypass, although
`requireCompleteBody` refuses an absent section for every kind. That
reasoning is right for the body-completeness gate. The branch guard reuses
the same flag for a case where it is not a no-op, and inherits provenance
machinery scoped to the other gate.

## Why it matters

A guard worth adding is worth knowing when someone stepped over. The branch
guard exists because an epic created on a ritual branch cannot be activated,
short of a forced promote, until that branch merges; overriding it is a
deliberate act with consequences for whoever finds the entity later.
`aiwf history` cannot show that it happened, and the
`provenance-force-non-human` audit, which walks history for commits no verb
produced, has no trailer to walk.

The comment is the sharper half. A reader deciding whether the bypass needs
a record or an actor check finds both asserted, and stops. That is how a
false comment costs more than an absent one.

The exposure is more than the missing record. An `ai/` actor inside a scope
can take the bypass and commit, and what reports it is a scope rule judging
the epic's reach after the commit, not a refusal at the verb.

## Resolution shape

Two independent decisions, and the first does not wait on the second.

**Correct the comments.** The guard's comment states a guarantee the code does
not provide, and the `Force` field's comment and the `--force` help call the
flag inert where it acts. That is true regardless of what the guard should do.

**Then decide whether the bypass should record itself.** Stamping the force
trailer when the branch guard is bypassed would make it visible to
`aiwf history` and subject it to the human-only rule, at the cost of extending
a trailer whose current contract — as `add.go:216-236` argues at length — is
that it marks an override that actually happened. That argument supports
stamping here: this bypass *is* an override that happened. `--reason` is
already required whenever `--force` is set (`aiwf add epic --title T --force`
exits 2: `--reason "..." is required when --force is set`), so stamping
changes no CLI surface: it records the reason the command demands and now
discards.

## Where to fix

- `internal/verb/add_branch_guard.go:44-45` — the false claim.
- `internal/verb/add.go:212-236` — where the trailer decision lives, if the
  second question resolves toward recording the bypass.
- `internal/verb/add.go:95-103` and the `--force` help in
  `internal/cli/add/add.go` — both call the flag inert for a scaffold create,
  which the branch guard makes false for an epic on a ritual branch. The help
  also says the actor must be human, which holds only where a force trailer is
  stamped.

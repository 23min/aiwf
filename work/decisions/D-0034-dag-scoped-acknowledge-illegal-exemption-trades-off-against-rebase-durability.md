---
id: D-0034
title: DAG-scoped acknowledge-illegal exemption trades off against rebase durability
status: proposed
relates_to:
    - E-0062
    - M-0244
---
# D-0034 — DAG-scoped acknowledge-illegal exemption trades off against rebase durability

> **Date:** 2026-07-09 · **Decided by:** human/peter

## Question

G-0395 found that `aiwf acknowledge illegal`'s exemption — a lookup over HEAD's
reachable history for a commit carrying `aiwf-force-for: <sha>` — silently
reappears if a later history rewrite (typically a rebase) drops just the
acknowledgment commit while leaving the originally-flagged commit reachable.
Should this be fixed by making the exemption durable against history rewrites,
or is the current behavior an acceptable trade-off?

## Decision

Keep the HEAD-reachable-only exemption lookup exactly as it is. The reappearing
finding is a real, observed property, not a defect to eliminate — the property
that makes it possible (checking current reachability, not a persisted record)
is the same property that gives the exemption its DAG-scoping guarantee: an
acknowledgment made on one branch correctly never leaks into exempting an
unrelated branch carrying the same-shaped violation. The two properties cannot
both hold under a plain git-log walk; DAG-scoping is the more valuable one to
keep.

The residual risk this trade-off carries is bounded by an existing chokepoint:
`illegal-transition` is error-severity, and the shipped pre-push hook's `exec
"$AIWF" check` means a non-zero exit blocks the push. Confirmed empirically
(M-0244/AC-2): reproducing G-0395's exact sequence and checking `aiwf check`'s
raw exit code at each step shows the revived finding still exits non-zero,
so the corrupted history cannot leave the machine via a normal push — only a
deliberate `git push --no-verify` bypasses it, which is a visible, sovereign
act this repo's own conventions already discourage without explicit
instruction. Checking this repo's own history additionally confirmed the
compound failure (an ack commit later dropped by a rebase) has never actually
occurred: all 56 real `acknowledge illegal` commits in this repo remain
reachable from current HEAD.

The one real, addressable gap was purely diagnostic: a revived finding looks
identical to one that was never acknowledged. `internal/check.findDanglingAckHint`
closes that gap without persisting any new state — it searches git's own
dangling-object store (`git fsck --unreachable --no-reflogs`) for a commit
still carrying the dropped `aiwf-force-for:` trailer, best-effort, only on the
already-failing path, and sets the finding's `Hint` when found.

## Reasoning

Two persistence-based alternatives were considered and rejected:

- **Record acknowledgments independent of git reachability** (e.g. a
  `.aiwf/acks.json` ledger, git notes, or similar). Rejected: this repo commits
  to "`aiwf history` reads git log; no separate event log" as one of its
  load-bearing properties (see `CLAUDE.md` §"What aiwf commits to", item 4),
  and explicitly excludes an events-log-shaped mechanism from its design
  (`CLAUDE.md` §"What is not in scope"). A persisted ack ledger is exactly
  that shape, and would need its own ADR and design work disproportionate to
  a diagnostic-only gap with zero confirmed real-world occurrences.
- **Detect reappearance against a last-known-good baseline.** Rejected for
  the same reason — it requires persisting a baseline outside git log, which
  this repo's design deliberately does not do.

Given the sharing-boundary risk is already closed by an existing chokepoint
(the pre-push gate), and the verb itself has never actually lost an
acknowledgment this way in 56 real uses, building persisted infrastructure to
close a purely diagnostic, forensic gap would cost far more than the residual
risk justifies. The dangling-object hint gets the practical value (naming the
dropped acknowledgment for the one operator who can actually still see it —
themselves, moments after their own accidental rebase, on the same local
clone) without touching the architecture.

## Consequences

- G-0395 closes referencing this decision plus the `findDanglingAckHint`
  diagnostic (M-0244/AC-2) — not a persistence-mechanism fix.
- `internal/stresstest/force_override_durability.go`'s ack-revocation-by-rebase
  scenario no longer treats the revival itself as a violation; it asserts the
  revival happens (unchanged) and additionally asserts the diagnostic hint
  fires when it does.
- If a future incident ever shows the DAG-scoping vs. rebase-durability
  trade-off is no longer acceptable — e.g., the compound failure actually
  occurs and causes real harm — that would be new information superseding
  this decision, not evidence this decision was wrong at the time it was made.

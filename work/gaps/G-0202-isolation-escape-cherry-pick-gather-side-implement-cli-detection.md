---
id: G-0202
title: 'isolation-escape cherry-pick gather-side: implement CLI detection'
status: addressed
discovered_in: M-0106
addressed_by:
    - M-0159
---
M-0106's `isolation-escape` rule supports cherry-pick suppression
via a `cherryPicked map[string]bool` parameter on `RunIsolationEscape`
([`internal/check/isolation_escape.go`](../../internal/check/isolation_escape.go)).
The rule logic is complete and sabotage-verified; the CLI gather-
layer half is NOT yet implemented.

`RunProvenanceCheck` at
[`internal/cli/check/provenance.go`](../../internal/cli/check/provenance.go)
passes `nil` for the `cherryPicked` argument. As a result, AC-6
suppression fires in unit tests (where fixtures populate the map)
but does NOT fire against real git history — every cherry-pick
re-author would surface as an `isolation-escape` warning when the
rule lands in the pre-push pipeline.

The CLAUDE.md "test the seam" rule applies: M-0106 lands the
rule-side seam tested, but the production CLI seam is unwired.

## What's needed

A gather-side helper that, for each candidate commit SHA in the
provenance commit window, computes:

1. The actor's expected email (per the same role-to-email mapping
   `cliutil.ResolveActor` uses — extract the `human/`/`ai/`/`bot/`
   prefix from `aiwf-actor:`, look up the configured email).
2. The commit's actual committer email (via `git log --format=%ce`).
3. Whether the commit body contains the regex
   `\(cherry picked from commit [0-9a-f]{7,40}\)` (via
   `git log --format=%B`).

When (committer ≠ expected actor) AND (cherry-pick marker present),
add the SHA to the `cherryPicked` set passed to
`RunIsolationEscape`.

## Why parked

The rule's logic is complete and verified. The gather-side
implementation is a separate concern with its own design
decisions (how to map AI actor roles to expected emails, how to
handle multi-author commits with Co-Authored-By trailers,
whether to short-circuit when the actor email is unconfigured).
Landing it inside M-0106 would inflate the milestone scope
beyond the kernel-finding deliverable; landing it as a follow-up
keeps each piece reviewable.

Once landed, the existing rule-side tests automatically gain
their end-to-end seam (cherry-pick commits produced by `git cherry-pick -x`
in integration fixtures will populate `cherryPicked` and the rule
will suppress as expected).

## Out of scope for the gap

The rule-side logic itself — that's M-0106's delivered scope and
won't change.

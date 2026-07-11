---
id: G-0395
title: acknowledge illegal is revoked when the ack commit becomes unreachable
status: addressed
discovered_in: M-0243
addressed_by_commit:
    - d9f32dc3
---
## What's missing

`aiwf acknowledge illegal`'s exemption is looked up by scanning HEAD's
reachable history for `aiwf-force-for:` trailers (the CLI gather layer at
`internal/cli/check/check.go`) — the acknowledgment commit itself, not
just the originally-flagged commit, must stay reachable for the exemption
to apply. A history rewrite that drops the acknowledgment commit while
leaving the originally-flagged commit in place (e.g. an interactive
rebase or `git rebase --onto` that excludes just that one commit from an
otherwise-unchanged range) silently revives the finding the acknowledgment
was meant to suppress — with no distinguishing signal telling the
acknowledgment ever existed. The revived finding looks identical to one
that was never acknowledged in the first place.

Confirmed directly: an illegal-transition finding, once acknowledged and
confirmed suppressed, reappears verbatim after a rebase that surgically
drops only the acknowledgment commit (keeping the flagged commit and a
later, unrelated commit both reachable). No warning, no distinct code,
no trace that an acknowledgment once covered it.

A real force-push producing the same reachability effect (the historical
SHA the acknowledgment targets becomes unreachable, or — this variant —
the acknowledgment commit itself does) is exactly the risk item 5 of the
data-loss audit named as a future-epic concern.

## Why it matters

`acknowledge illegal` exists precisely so a human's documented rationale
for an exceptional state survives in the audit trail. A revocation this
silent defeats that purpose: an operator who force-pushes, rebases, or
otherwise rewrites history near an acknowledgment commit has no way to
know they've undone someone else's considered exception — the tree just
looks freshly broken again. A mechanism that recorded the acknowledgment
independent of raw reachability (or that surfaced a distinct finding when
a previously-covered offense reappears with no matching acknowledgment in
the git-log-since-last-known-good) would close this gap.

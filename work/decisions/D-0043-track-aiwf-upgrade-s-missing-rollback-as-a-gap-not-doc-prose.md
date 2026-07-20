---
id: D-0043
title: Track aiwf upgrade's missing rollback as a gap, not doc prose
status: proposed
relates_to:
    - M-0270
---

# D-0043 — Track aiwf upgrade's missing rollback as a gap, not doc prose

> **Date:** 2026-07-20 · **Decided by:** human/peter

## Question

F12 (`docs/initiatives/verb-layer-cleanup.md`) observed that `aiwf upgrade`
delegates its fetch/verify/place sequence to a single `go install` call,
with no aiwf-level backup of the previous binary — so a broken newly
installed binary has no automated rollback path. The finding's own framing
suggested naming this in "the release-process docs," but where does that
fact actually need to live to reach the people who need it?

## Decision

Track the absence as a first-class gap entity (`G-0430`, `discovered_in:
M-0270`) rather than as static prose in CLAUDE.md or `aiwf upgrade --help`.

## Reasoning

Two prose locations were considered and rejected:

- **CLAUDE.md's `### Release process` section** — this file is
  repo-development-only and never ships to a consumer repo (per this
  repo's own audience-split rule for consumer-operating vs
  repo-development guidance). A fact about `aiwf upgrade`'s runtime
  behavior needs to reach the operators who actually run that command in
  their own repos, not just aiwf's own maintainers cutting a release.
- **`aiwf upgrade --help`'s `Long` description** — would state what the
  tool does *not* do, without a corresponding flag to point at. Per this
  repo's own skill-coverage allowlist, `--help` is `aiwf upgrade`'s
  designated shipped discoverability channel, but preemptively documenting
  the absence of a `--rollback` flag that doesn't exist reads as
  aspirational rather than descriptive, and drifts the moment a real
  decision gets made about whether to build one.

A gap avoids both problems: it's a first-class, queryable record
(`aiwf show G-0430`) that doesn't presuppose an answer to "should
`aiwf upgrade --rollback` eventually exist?" — it just makes the current
absence a tracked fact. If a future milestone builds rollback, it closes
the gap and updates `--help` in the same change, which is the natural
coupling ("once it exists, help should be updated") rather than
`--help` documenting a feature ahead of its existence.

## Consequences

`G-0430` stays open until either a rollback capability ships (closing the
gap alongside a real `--help` update) or a future decision explicitly
judges minimalism the right permanent call (closing the gap by
referencing that decision, per this repo's "you'd open a new entity for
the inverse" reversal convention).
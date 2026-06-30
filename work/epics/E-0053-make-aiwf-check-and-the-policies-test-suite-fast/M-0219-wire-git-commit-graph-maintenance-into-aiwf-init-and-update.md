---
id: M-0219
title: Wire git commit-graph maintenance into aiwf init and update
status: cancelled
parent: E-0053
tdd: required
---

## Goal

Addresses G-0322. Git's `commit-graph` is a traversal-acceleration cache that
the kernel repo currently lacks; writing it cut a full `aiwf check` from ~35s to
~25s in the M-0216 measurements (~9s), with zero correctness impact — it is git's
own cache, so findings stay byte-identical. The win compounds the M-0216
blob-object-id read path and accelerates every history walk the check performs
(the isolation-escape oracle's per-branch `rev-list`, the `--all` DAG/log walks,
the reflog reads).

The deliverable is to have `aiwf init` / `aiwf update` ensure the commit-graph is
maintained for the consumer repo — most likely by setting `gc.writeCommitGraph =
true` (and/or `fetch.writeCommitGraph`), or wiring `git maintenance`, so the
graph is refreshed as commits land rather than going stale. Settings/hook
materialization follows the existing marker-managed, consent-gated artifact
conventions (ADR-0015 for any `settings.json` touch; the git-config edit is
repo-local). The acceptance criteria are authored when the milestone starts.

## Acceptance criteria

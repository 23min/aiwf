---
id: G-0322
title: Maintain git commit-graph on init/update to accelerate aiwf check
status: open
discovered_in: M-0216
---

## What's missing

`aiwf check` repeatedly walks ~5,500 commits (`git log --all --raw`,
`rev-list`, `merge-base`). Git's `commit-graph` cache makes those walks much
cheaper, but aiwf never writes or refreshes it in a consumer repo. Wire
`git commit-graph write --reachable` (and/or `fetch.writeCommitGraph` /
`gc.writeCommitGraph`) into `aiwf init` / `aiwf update`.

## Why it matters

Measured in M-0216 — with a commit-graph present, check ran 35.4s→25.3s vs
48.8s→37.3s without it: a near-free ~9s, the cheapest lever. Addressed by
milestone M-0219.

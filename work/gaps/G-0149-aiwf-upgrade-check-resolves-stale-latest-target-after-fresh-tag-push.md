---
id: G-0149
title: aiwf upgrade --check resolves stale latest target after fresh tag push
status: open
---
## Problem

`aiwf upgrade --check` on the host (binary built from pseudo-version `v0.8.1-0.20260516161658-02c349f629d7`) reports `target: v0.8.0 (tagged)` for several minutes after `v0.8.1` is pushed to the remote — even though the Go proxy is already serving `v0.8.1` on both endpoints:

```
$ curl -s https://proxy.golang.org/github.com/23min/aiwf/@latest
{"Version":"v0.8.1","Time":"2026-05-21T20:17:11Z", ...}

$ curl -s https://proxy.golang.org/github.com/23min/aiwf/@v/list
... v0.8.0
v0.8.1
```

Observed from a Linux devcontainer; the host that reported `target: v0.8.0` is macOS. The kernel's `Latest()` function (`internal/version/version.go:265`) uses `/@v/list`-first per the documented anti-pattern fix, which should pick up `v0.8.1` from the list.

## Hypotheses (unverified)

1. **Edge-cache propagation lag.** `proxy.golang.org` is a CDN; different edges propagate the list response at different rates. My devcontainer's curl hits a different POP than the user's host. Several minutes is plausible but not documented anywhere.
2. **GOPROXY chain on host.** If host has a corporate proxy or `GOPROXY=https://goproxy.cn,direct`, that proxy may be staler than the default. Need to check `go env GOPROXY` on host.
3. **Host's resolver doesn't actually call `/@v/list`.** Some path other than `internal/version/Latest()` is being used (e.g. an older entry point the upgrade verb calls).
4. **Local module-download cache on host.** `~/go/pkg/mod/cache/download/github.com/23min/aiwf/@v/list` may have been written by an earlier `go install` and is being served stale by a local-cache layer between the binary and the proxy.

Resolution likely needs at least one round of host-side diagnostics: `aiwf upgrade --check` with `-v` (if available), `go env GOPROXY`, direct `curl` from the host to the proxy, and inspection of `~/go/pkg/mod/cache/download/`.

## Why it matters

`aiwf upgrade` is the documented consumer-side upgrade path (CLAUDE.md *Release process*). If a release lands cleanly on the remote but consumers can't see it for an indeterminate time, the "release → consumer picks it up" loop is broken at the user-facing surface — not at the publishing surface. The `--worktrees` motivating example from this session is the canonical symptom.

## Reproduction

1. Cut and push a new tag (e.g. `v0.8.1` on top of v0.8.0).
2. Verify proxy has it: `curl -s https://proxy.golang.org/github.com/23min/aiwf/@v/list`.
3. From a host with the previous binary installed: `aiwf upgrade --check`.
4. Expected: `target: v0.8.1`. Observed: `target: v0.8.0`.

Time-bound the wait — at minute 0 fail is expected; at minute T (TBD) it should succeed.

## Out-of-scope for this gap

- The CI-side `go.yml` red state since 2026-05-19 (separate concern; "aiwf binary not found on PATH" pattern).

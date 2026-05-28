---
id: G-0149
title: aiwf upgrade --check resolves stale latest target after fresh tag push
status: addressed
addressed_by_commit:
    - "99583366"
---
## Problem (original observation)

`aiwf upgrade --check` on the host (binary built from pseudo-version
`v0.8.1-0.20260516161658-02c349f629d7`) reported `target: v0.8.0 (tagged)`
for several minutes after `v0.8.1` was pushed to the remote — even though
the Go proxy was already serving `v0.8.1` on both endpoints:

```
$ curl -s https://proxy.golang.org/github.com/23min/aiwf/@latest
{"Version":"v0.8.1","Time":"2026-05-21T20:17:11Z", ...}

$ curl -s https://proxy.golang.org/github.com/23min/aiwf/@v/list
... v0.8.0
v0.8.1
```

Observed from a Linux devcontainer; the host that reported `target: v0.8.0`
was macOS. The kernel's `Latest()` function had already been switched to
`/@v/list`-first resolution in `32672cda` (2026-05-03), so the resolution
strategy was already correct at the time of the observation.

## Investigation (2026-05-28)

### Hypotheses verdicts

The four hypotheses from the original filing were investigated as follows.

| # | Hypothesis | Verdict |
|---|---|---|
| 1 | Edge-cache propagation lag on `proxy.golang.org`'s CDN | Confirmed plausible; **already handled** by `proxyStaleHint()` (see *Existing remediation* below). |
| 2 | Host's `GOPROXY` chain serves a staler view than the default | Confirmed plausible; **already handled** by `proxyStaleHint()` (suggests `GOPROXY=direct` to bypass). |
| 3 | A code path other than `Latest()` is being used for resolution | **Disproven by code audit.** `version.Latest()` is the only resolver. It is called from exactly two sites: `cli/upgrade/upgrade.go` (`ResolveTarget("latest")` at line 224) and `cli/doctor/doctor.go` (line 728). No alternate path exists. |
| 4 | Local `~/go/pkg/mod/cache/download/.../@v/list` is being served stale by a cache layer between the binary and the proxy | **Disproven by code audit.** `latestFor()` uses `http.DefaultClient` to hit `proxy.golang.org` over raw HTTP. The Go module-download cache is populated only by `go install` / `go mod download`; our resolver never touches it. |

### Existing remediation

The fix for the observed scenario shipped in `99583366`
(`fix(upgrade): hint on proxy CDN stale-tag scenario (closes G-0149)`),
23 minutes after this gap was filed. It added `proxyStaleHint()` in
`internal/cli/upgrade/upgrade.go`: when the running binary's pseudo-version
base is newer than the resolved `--check` target — the exact signature of a
freshly-pushed tag that the proxy CDN has not yet propagated to every edge —
the verb prints:

```
hint:     pseudo-base <X> is newer than target <Y>; the Go module
          proxy CDN may not have propagated the freshest tag yet.
          retry in a few minutes, or set GOPROXY=direct to bypass.
```

That covers both environmental root causes (CDN propagation lag and
GOPROXY-chain staleness) with the only two remedies the consumer-side
binary can offer: wait, or bypass the proxy chain.

## Why no further code change

The residual scenarios — edge-cache propagation and operator GOPROXY
configuration — are outside the binary's control. The kernel already does
what it reasonably can:

- Correct resolver (`/@v/list`-first, anti-pattern-fix in place).
- Diagnostic hint when the stale-tag signature is detected (`99583366`).

A `--verbose` / `-v` diagnostic flag on `aiwf upgrade --check` (printing the
resolved proxy URL, the raw response, and the chosen version) would be a
reasonable follow-on if this pattern recurs and remote diagnosis becomes
necessary, but it's not warranted today: live verification at close-out
confirmed the proxy is consistent and the resolver picks up tags correctly.

This gap stayed `open` only because its status was never promoted after
`99583366`'s "(closes G-0149)" subject — the same status-hygiene class as
the E-0036 scope-gaps the whiteboard surfaced on 2026-05-26.

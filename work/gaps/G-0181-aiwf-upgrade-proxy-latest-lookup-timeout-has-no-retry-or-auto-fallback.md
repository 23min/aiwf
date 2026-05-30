---
id: G-0181
title: 'aiwf upgrade: proxy latest-lookup timeout has no retry or auto-fallback'
status: open
---
## What

`aiwf upgrade` resolves the target version by querying the Go module
proxy's `/@v/list` (`version.Latest` → `ResolveTarget`). When that single
HTTP GET times out (`context deadline exceeded` against
`proxy.golang.org`, common right after a fresh tag or on a slow/restricted
network), the upgrade has **no retry and no automatic fallback** — it
prints `target: latest (proxy lookup failed: …)` and proceeds to
`go install …@latest`, which may also fail the same way, leaving the
operator stuck with no resolved target.

## Evidence

Reported immediately after the v0.10.0 release from a consumer repo:

    current:  v0.9.0 (tagged)
    target:   latest (proxy lookup failed: querying
              https://proxy.golang.org/github.com/23min/aiwf/@v/list:
              … context deadline exceeded)

The proxy *did* have v0.10.0 indexed at the time (verified via curl), so
the failure was a transient network/proxy-warmth timeout, not a missing
version — exactly the class a retry or a `,direct` fallback would absorb.

## Already shipped (the quick patch)

A focused remediation **hint** now prints on the proxy-lookup-failure
branch (M-… / `proxyLookupFailedHint`): it tells the operator the
subsequent `go install` may still succeed via GOPROXY's `,direct`
fallback, and if not, to retry / pin `@vX.Y.Z` / `GOPROXY=direct aiwf
upgrade`. That removes the dead-end but does not make the lookup itself
resilient.

## Desired resolution (the fuller fix)

1. **Retry the proxy lookup** on transient errors (timeout / 5xx) — a
   small bounded retry with backoff in `version.Latest`, since the most
   common failure (a just-pushed tag the CDN hasn't propagated, or a
   one-off timeout) clears on a second attempt.
2. **Honor GOPROXY's `,direct` fallback for the lookup too** — today the
   lookup hits the first proxy host directly; if it fails it should walk
   the GOPROXY list (proxy → direct) the way `go install` does, so the
   pre-flight target matches what the install will actually resolve.
3. Per CLAUDE.md § "Contract tests for upstream-cached systems": any
   change here must keep the resolution-correctness test (derive the
   expected latest via an independent `/@v/list` fetch), and the retry
   must be gated under `-short` so offline CI still passes.

## References

- `internal/cli/upgrade/upgrade.go` (`Run`, `ResolveTarget`, `proxyLookupFailedHint`); `internal/version/version.go` (`Latest`, `highestTaggedFromList`).
- CLAUDE.md § "Contract tests for upstream-cached systems", § "Spec-sourced inputs".

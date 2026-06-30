---
id: G-0330
title: internal/cli/integration (~128s) is the test-suite critical path
status: open
discovered_in: E-0053
---
## Context

The M-0218 review (E-0053) measured the full `go test ./...` per-package
wall-times and found `internal/cli/integration` ~= **128s** — roughly 10x the
next package (`internal/policies` ~= 12.6s) and the **critical path** of the
whole test suite. Because Go runs packages concurrently, the suite wall-clock —
and thus `make ci`, `make test`, and the CI test job — is bounded by this one
package.

This is the finding that made M-0218 (policies ~9s -> ~5s) not worth doing: the
policies suite runs fully overlapped *behind* integration, so optimizing it
changes nothing anyone waits on. The integration package is where the real
leverage is — it is not overlapped, so reducing it cuts real wall-time on every
`make ci` / CI run.

## The gap

`internal/cli/integration` is the dominant test-suite cost and has not been
profiled for reducible vs inherent cost. Unverified candidates:

- per-test `aiwf` binary builds or repeated full repo-fixture setup;
- subprocess churn (each scenario shells the built binary end-to-end);
- redundant `git init` / tree materialization per test;
- serial execution or a low `-parallel` ceiling;
- genuinely-inherent end-to-end cost (building + driving the real binary).

## Proposed resolution

1. **Measure first** (the M-0218 lesson): per-test timings (`go test -json` /
   `-v`) over `internal/cli/integration`, warm and cold, at the pinned
   `-parallel 8`; rank the cost centers; separate reducible from inherent.
2. Where leverage justifies the risk, reduce the dominant cost: confirm the
   once-built-binary helper (`AiwfBinary` `sync.Once`) is used everywhere;
   share read-only fixtures once; raise parallelism where the G-0097 race-flake
   cap allows; cut redundant per-test repo setup.
3. Behavior-preserving only — these are end-to-end correctness tests; no faking
   the integration assertions. A "largely inherent, not worth it" conclusion is
   an acceptable, recorded outcome.

## Acceptance sketch

- A measured per-test breakdown of the ~128s with a reducible-vs-inherent split.
- If pursued: a named, measured wall-time reduction with all tests green.

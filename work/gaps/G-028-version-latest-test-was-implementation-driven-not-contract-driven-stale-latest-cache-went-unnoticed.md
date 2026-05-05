---
id: G-028
title: '`version.Latest()` test was implementation-driven, not contract-driven — stale `/@latest` cache went unnoticed'
status: addressed
addressed_by_commit:
  - f810a86
---

Resolved in commit `f810a86` (test(aiwf): close G27/G28/G29 — seam, contract, spec-sourced tests). `TestLatest_RealProxy` ("version is non-empty") replaced by `TestLatest_RealProxy_ContractTest` which fetches `/@v/list` directly via raw `net/http` (not through `version.Latest`), computes the expected highest semver via a test-side reference implementation, then asserts `version.Latest()` returns that exact value. The reference implementation is deliberately not imported from the version package so a future regression can't be hidden by a matching regression in the helper. New `TestLatest_PrereleaseExcludedFromHighestSelection` pins the pre-release-skipping invariant offline via httptest.

The policy text in `CLAUDE.md`'s Testing section ("Contract tests for upstream-cached systems") is the durable rule.

---

<details><summary>Original entry (open)</summary>

The v0.1.0 shipped with `aiwf doctor --check-latest` displaying a stale pseudo-version instead of `v0.1.0`. Root cause: `version.Latest()` queried the proxy's `/@latest` endpoint and unit tests served whatever JSON the implementation expected. The real proxy behavior — that `/@latest` and `/@v/list` are cached independently, and `/@latest` can serve a pre-tag pseudo-version answer for hours after the first tag lands — was not modeled. The Go toolchain's own resolver uses `/@v/list` first for exactly this reason; we re-discovered the lesson by shipping the wrong endpoint and noticing in v0.1.0 verification.

The existing real-proxy integration test (`TestLatest_RealProxy`) queries `gopkg.in/yaml.v3` and only asserts the version is non-empty. It would have passed with either implementation choice (`/@latest` returning yaml.v3's tag happens to work because nobody queried that module's `/@latest` before tags existed). The test was *cooperative* — it tested the parsing round-trip, not the resolution semantics.

**Resolution path:** Policy added to `CLAUDE.md`'s Testing section ("Contract tests for upstream-cached systems") in the same commit that files this gap. The Latest() resolution itself was fixed in v0.1.1 (commit `32672cd`); G28's residual work is the *test* that pins the contract:

1. Tighten `TestLatest_RealProxy` to derive the expected version through an **independent** code path. Concretely: the test fetches `https://proxy.golang.org/<known-tagged-module>/@v/list` directly (without going through `version.Latest`), parses the response, computes the highest semver triple, and asserts `version.Latest()` returns that exact value. Today's "version is non-empty" assertion is replaced by "version matches the independently-derived expected value."
2. Add a multi-tag fixture test using the existing httptest seam to pin the highest-of-N selection logic without network: serve a `/@v/list` body with three or four tags including a pre-release, assert the highest non-pre-release wins.

Severity: Medium. Same class as G27 — the implementation has been fixed; the policy + the contract test are what stop the next instance.

</details>

---

<a id="g29"></a>

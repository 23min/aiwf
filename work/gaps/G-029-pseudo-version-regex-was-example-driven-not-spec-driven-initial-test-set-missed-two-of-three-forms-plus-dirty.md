---
id: G-029
title: Pseudo-version regex was example-driven, not spec-driven — initial test set missed two of three forms plus `+dirty`
status: addressed
addressed_by_commit:
  - f810a86
---

Resolved in commit `f810a86` (test(aiwf): close G27/G28/G29 — seam, contract, spec-sourced tests). `TestParse`, `TestProxyBase`, and `pseudoVersionRE`'s doc comment now cite the upstream specs (`go.dev/ref/mod#pseudo-versions`, `semver.org`, `go.dev/ref/mod#environment-variables`); `TestParse` cases now cover all three pseudo-version forms explicitly (was: form 1 + form 3 only) plus the `+dirty` stamping case for both base shapes. The citations make spec-drift detectable: a future Go-toolchain change to pseudo-version grammar will be flagged by anyone reading the spec, rather than missed because tests were example-driven.

The policy text in `CLAUDE.md`'s Testing section ("Spec-sourced inputs for upstream-defined input spaces") is the durable rule.

---

<details><summary>Original entry (open)</summary>

The first pass of `version.isTagged` had a `pseudoVersionRE` that only matched the basic `v0.0.0-DATE-SHA` shape. The Go module spec defines three pseudo-version forms (basic, post-tag `vX.Y.(Z+1)-0.DATE-SHA`, pre-release-base `vX.Y.Z-pre.0.DATE-SHA`) and Go's VCS stamping adds the `+dirty` suffix on working-tree builds with uncommitted changes. The regex caught only the first form; the other three were missed.

The bug was caught mid-implementation by a smoke test (the working-tree build of aiwf reported `"v0.0.0-...-...+dirty (tagged)"`), not by the unit-test pass that immediately preceded it. Root cause: test cases were sourced from "the example I had in mind" rather than from the spec. A spec-sourced enumeration would have listed all four shapes from `https://go.dev/ref/mod#pseudo-versions` plus the VCS-stamping behavior on first writing.

**Resolution path:** Policy added to `CLAUDE.md`'s Testing section ("Spec-sourced inputs for upstream-defined input spaces") in the same commit that files this gap. The implementation already covers the cases (regex updated mid-step-2 to `[-.]\d{14}-[0-9a-f]{12}$` and `+dirty` checked separately); G29's residual work is small:

1. Add a `// per https://go.dev/ref/mod#pseudo-versions` comment above the test data in `version_test.go` so the spec-sourcing is visible to future readers.
2. Audit other test sets that enumerate upstream-defined input spaces (frontmatter shapes against YAML 1.2; commit-trailer shapes against `git interpret-trailers`; semver against the semver.org grammar) for analogous unsourced enumerations, and either add the citation or document the omission.

Severity: Low. Bug already resolved; the policy + the citation are the durable defense. The audit pass is one read-through, not a refactor.

</details>

---

<a id="g30"></a>

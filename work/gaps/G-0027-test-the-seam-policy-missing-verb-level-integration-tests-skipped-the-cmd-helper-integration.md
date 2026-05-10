---
id: G-0027
title: Test-the-seam policy missing — verb-level integration tests skipped the cmd → helper integration
status: addressed
addressed_by_commit:
  - f810a86
---

Resolved in commit `f810a86` (test(aiwf): close G27/G28/G29 — seam, contract, spec-sourced tests). New `cmd/aiwf/binary_integration_test.go` builds the cmd binary to a tempfile and subprocesses it; two test cases pin (a) ldflags-stamped Version reaches the verb output (`make install` path) and (b) without ldflags, `aiwf version` and `aiwf doctor`'s `binary:` row report the same value (the seam G27 was filed against). Companion fix in `cmd/aiwf/main.go`: `resolvedVersion`'s no-ldflags fallback now returns `version.Current().Version` directly, byte-coherent with the doctor row. Reverse-validated: restoring the v0.1.0 bug shape (`fmt.Println(Version)` printing the unstamped global) fails the fallback test with the exact "literal sentinel" + "seam mismatch" messages.

The policy text in `CLAUDE.md`'s Testing section ("Test the seam, not just the layer") is the durable rule that should prevent the next instance.

---

<details><summary>Original entry (open)</summary>

The v0.1.0 shipped with `aiwf version` returning `"dev"` despite a working `version.Current()` helper. Root cause: tests covered the new helper in isolation but no test exercised the verb that was supposed to use it. The verb's body kept printing an unrelated package-global (`Version`, the ldflags-stamped value defaulting to `"dev"`); the helper was wired into `aiwf doctor`'s `binary:` row but not into the `version` verb. Two parallel sources of truth for "what version am I" coexisted; the test surface covered only the new one.

The bug was caught by a manual smoke test against the v0.1.0 binary post-publish — exactly the wrong place for it to surface. The pattern generalizes: any time a new helper is added that an existing verb *should* adopt, a verb-level test must assert the verb's output reflects the helper's contract. Without that, a future refactor that introduces a new helper alongside an unrelated existing path repeats the bug.

**Resolution path:** Policy added to `CLAUDE.md`'s Testing section ("Test the seam, not just the layer") in the same commit that files this gap. Implementation work to retrofit existing verbs:

1. Add a binary-level integration test (`cmd/aiwf/binary_integration_test.go` or similar): `go build -o $TMP/aiwf ./cmd/aiwf` then run `aiwf version`, `aiwf doctor` as subprocesses, assert their output. This catches the v0.1.0 bug class for every verb whose output depends on `runtime/debug.ReadBuildInfo`, `os.Args[0]`, `os.Executable()`, or `-ldflags`-stamped globals.
2. Audit each existing verb that consumes a shared helper (`version.Current`, `version.Latest`, `entity.SchemaForKind`, etc.) and confirm there is at least one verb-level test asserting the helper is the actual source of truth.

A future `aiwf check`-style policy could detect "exported helper imported by `cmd/aiwf` but no test in `cmd/aiwf` references it" — overkill for the PoC, but the policy framework already exists (G21/G26) and a fourth policy in that family would be cheap.

Severity: Medium. The class of bug is high-impact (shipped correctness regression), and the policy is the durable defense; the implementation work is small.

</details>

---

<a id="g28"></a>

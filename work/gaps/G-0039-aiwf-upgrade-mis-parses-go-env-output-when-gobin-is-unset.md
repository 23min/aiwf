---
id: G-0039
title: '`aiwf upgrade` mis-parses `go env` output when GOBIN is unset'
status: addressed
addressed_by_commit:
  - 9a06c74
---

Resolved in commit `(this commit)` (fix(aiwf): G39 — upgrade flow's go env parser fails when GOBIN is unset). The post-install lookup in `goBinDir` now queries `go env GOBIN` and `go env GOPATH` in two separate calls instead of one combined call. The combined call returns one line per name, and an unset GOBIN renders as a leading blank line — `strings.TrimSpace` was eating that blank, leaving a 1-element slice that tripped the `len(lines) < 2` guard. Anyone with stock Go install (no GOBIN exported, GOPATH at default) hit a non-zero exit immediately after `go install` succeeded, with the operator-facing message `"unexpected `go env` output: \"\n/home/.../go\n\""` and a generic "run aiwf update manually" hint. The fix removes the multi-line parser entirely; each call returns at most one value, so there is no shape to mis-parse.

Companion UX upgrade: when locating the new binary still fails for any reason, `runUpgrade` now prints a concrete fallback path derived from `$GOBIN`, `$GOPATH/bin`, or `$HOME/go/bin` so the user can recover with one command (`<path> update --root <root>`) instead of guessing where `go install` writes.

Test coverage added in the same commit:

- `TestGoBinDir_Matrix` — table test driving `goBinDir` through the shim across the four GOBIN/GOPATH shape combinations (gobin set, gobin empty + gopath set, both set, both empty). The "gobin empty, gopath set" row is the case this gap was filed for.
- `TestRunUpgrade_FullFlow_GOBINUnset` — verb-level seam test mirroring the pre-existing `TestRunUpgrade_FullFlow_NoReexec` but with `AIWF_TEST_GOBIN=""`, asserting the resolution falls through to GOPATH/bin and that `env GOPATH` is queried after `env GOBIN` returns empty.
- `TestInstallLocationHint` — covers the env-var precedence of the new fallback hint helper.

The pre-existing test seam parameterized the shim's `env` arm with hard-coded non-empty paths, so the empty-GOBIN shape was never exercised. This is a recurrence of G27's pattern (helper covered in isolation, integration shape not covered) — `CLAUDE.md`'s "Test the seam, not just the layer" rule predates this gap and explicitly calls out the pattern. The lesson is that "drive the helper through the shim" is necessary but not sufficient: the shim's input space must enumerate the upstream tool's real output shapes (per G29's spec-sourced-inputs rule applied to runtime tools, not just data formats).

Severity: **High**. Operator-facing regression on the most common Go install setup; would have blocked any tagged-release upgrader on a fresh devcontainer or stock workstation. Caught by the user during a real upgrade attempt against `v0.2.3`.

---

---
id: G-046
title: '`aiwf upgrade` fails opaquely when the install package path changes between releases'
status: addressed
---

Resolved in commit `93a3e2b` (feat(aiwf): G46 — structured remediation when go install reports the package-path-change failure). `runGoInstall` now tees stderr to a captured buffer (no UX change — the user still sees the live stream, the buffer just lets us introspect after the fact). New `pathChangedFromStderr` matches the Go toolchain's `module .* found .*, but does not contain package <subpath>` signature, captures the missing subpath, and `printPackagePathChangedHint` surfaces a kernel-friendly remediation: the install path may have changed, here's the CHANGELOG link, here's the manual `go install <new-path>@<target>` to recover, follow with `aiwf update` to refresh artifacts. False-positive guard: unrelated `go install` failures (network, invalid version, permission) do not trigger the hint. Tests pin both the table-driven detector cases and the runtime path through a stderr-emitting shim.

The fix can't help v0.3.x consumers retroactively — their binary's upgrade verb is frozen. But every release from v0.5.0 forward will produce a structured remediation if a future release relocates the cmd package again.

`aiwf upgrade` invokes `go install <pkg>@<target>` where `<pkg>` is the install path the running binary was built from — hard-coded in `internal/version` (the `pkg` constant in `Latest()` and consumed by the upgrade verb's shell-out). When a release relocates the cmd package within the module — exactly what `v0.4.0` did, moving `cmd/aiwf` from `tools/cmd/aiwf` to `cmd/aiwf` as part of the Go-conventional reorg — the upgrade verb on the *prior* binary (v0.3.x) tries `go install github.com/23min/ai-workflow-v2/tools/cmd/aiwf@latest`, the module proxy resolves the module fine, but the subpath no longer exists in the new tag. `go install` exits 1 with `module ... found (v0.4.0), but does not contain package .../tools/cmd/aiwf`. `aiwf upgrade` surfaces the raw exit-1 to the user with no remediation hint.

**Concrete reproducer (real, today):** a consumer running `aiwf v0.3.0` runs `aiwf upgrade` after `v0.4.0` ships. The error message names the missing subpath; nothing in the output tells the consumer that the install path moved or that the recovery is one manual `go install` against the new path.

**Why this matters now:**

1. *We just shipped the break.* `v0.4.0` is the trigger. Any consumer upgrading hits it once.
2. *The fix can't be retroactive.* The v0.3.x binary is already shipped; its `aiwf upgrade` logic is frozen. Whatever we do here improves *future* path-change resilience, not the v0.3.x → v0.4.0 transition.
3. *Path changes are rare but not theoretical.* If we ever rename the binary directory again (e.g., split `aiwf` from a future `aiwf-server`, or move under an aiwf/aiwf org), the same failure mode recurs. The v0.4.0+ binary should handle the next break gracefully.

**Proposed fix (for v0.4.x or later):**

`aiwf upgrade` learns to detect "module found but subpath missing" specifically and either:

- *Print a structured remediation* — "the install path may have changed in `<target>`; check the CHANGELOG at https://github.com/23min/ai-workflow-v2/blob/main/CHANGELOG.md and re-install manually with `go install <module>/<new-subpath>@<target>`." Doesn't try to be clever; tells the user what to do.
- *Try a small set of known-alternate paths.* If `go install <module>/tools/cmd/aiwf` fails with the specific error, retry with `<module>/cmd/aiwf`. Hardcoded fallback list — three entries max, documented in source. Cleaner UX but couples the binary to past path layouts.

Lean: option 1 (structured remediation). YAGNI on the fallback list — we hope to never rename again, and if we do, the next break we know about can ship its own one-time message in the next release notes. The structured remediation generalizes; the fallback list bakes in path archaeology.

**Detection shape:** parse `go install` stderr for `module .* found .*, but does not contain package`. That's the exact phrasing the Go toolchain uses for this case (see `cmd/go/internal/modload/import.go`); pinning the regex to that line is reliable.

**Severity:** Medium. One-time stumble per consumer per path-change release. Doesn't corrupt state, just confuses the user. Filed as a follow-up to `v0.4.0`'s release pain, not as a `v0.4.0` blocker.

---

<a id="g48"></a>

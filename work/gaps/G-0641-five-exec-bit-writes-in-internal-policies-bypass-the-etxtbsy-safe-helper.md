---
id: G-0641
title: Five exec-bit writes in internal/policies bypass the ETXTBSY-safe helper
status: addressed
priority: medium
addressed_by_commit:
    - 940bddd7dd1c4439e5d6f1aebd2ee9318febdeee
---
## What's missing

`internal/policies` writes five executable files through a bare
`os.WriteFile(..., 0o755)` rather than `testsupport.WriteExecutable`. Measured
2026-08-26; none carries an `//exec:ok` escape:

- `internal/policies/coverage_gate_wiring_test.go:127` — a `go` stub
- `internal/policies/m0155_statusline_scaffold_test.go:72` — a statusline script
- `internal/policies/prepush_lint_hook_test.go:396` and `:474` — a hook stub
- `internal/policies/statusline_behavioral_test.go:121` — a `gh` stub

Four of the five sit in files that call `exec.Command`.

A plain write holds a writable descriptor for as long as the file is open; a fork
concurrent with that window inherits it, and `execve` against a file some descriptor
still holds open for writing fails with `ETXTBSY`. `testsupport.WriteExecutable`
holds `syscall.ForkLock` across the write, which excludes exactly those forks.
G-0491 established the mechanism and the remedy.

Neither half of the chokepoint reaches these lines. `test-executable-write` is
diff-scoped, so a call site no change has touched stays forgiven. Its whole-file
sibling `TestAdoptedPackagesRouteThroughWriteExecutable` walks `adoptedPackages`,
which holds `internal/stresstest` and `internal/contractverify` — the package where
the fault was measured and the one where it recurred. `internal/policies` is in
neither list.

Adopting the package is one available remedy, and it carries a standing cost: every
future executable write in it would have to route through the helper.

## Why it matters

`internal/policies` runs on every push, so the flake is reachable from
`make check-fast`, `make ci` and CI alike. What it costs is not a wrong verdict but
a lost one — the fixture step fails, so the property the test exists to pin is never
evaluated, and the run reads as a red suite rather than as an untested invariant.

Because the failure needs a fork inside the write window, it surfaces under whatever
concurrency a run happens to have and hides otherwise. A green re-run is not evidence
against it. G-0491 measured it in one package and it recurred in a second; this is a
third that no rule watches.

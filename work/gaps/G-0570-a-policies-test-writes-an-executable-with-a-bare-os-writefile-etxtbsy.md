---
id: G-0570
title: A policies test writes an executable with a bare os.WriteFile (ETXTBSY)
status: open
priority: low
discovered_in: M-0300
---
## What's missing

`internal/policies/worktree_rituals_check_hook_test.go` writes an executable
stand-in with a bare `os.WriteFile(path, ..., 0o755)` rather than through
`testsupport.WriteExecutable`. That is the shape G-0491 records: a plain write
holds a writable descriptor open, a concurrent fork inherits it, and `execve`
against a file held open for writing fails with `ETXTBSY`.

Observed once during a review run of `go test ./internal/policies/`, as
`fork/exec ...: text file busy`; the package passed on re-run and passes in
isolation, which is the signature of the race rather than a stable failure.

## Why it matters

The `test-executable-write` chokepoint is diff-scoped, so it forgives every line
a change does not touch — this one has not been touched since it was written, and
so is invisible to it. `internal/stresstest` and `internal/contractverify` are
additionally held clean whole-file by
`TestAdoptedPackagesRouteThroughWriteExecutable`; `internal/policies` is not in
that set.

The cost of the flake is not a failed assertion but an unevaluated one: the
fixture step fails, so the property the test exists to pin is never exercised,
and the failure reads as infrastructure noise. Either route the call through the
helper, or add `internal/policies` to the whole-file adopted set.

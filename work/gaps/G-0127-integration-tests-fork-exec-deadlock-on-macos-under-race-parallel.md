---
id: G-0127
title: Integration tests fork/exec deadlock on macOS under -race + parallel
status: open
discovered_in: M-0120
---
## What's wrong

On macOS, `go test -race -parallel 8 ./...` against this repo intermittently leaves test-spawned shell processes stuck — alive, no CPU, never advancing past their first command. The stuck processes accumulate across test runs; ~55 such orphans were observed in one diagnostic snapshot, the oldest 25+ hours old. Each test invocation that triggers the bug hangs the test that forked it, eventually triggering the Go test framework's 10-minute timeout and a SIGQUIT kill. The orphans persist after the kill and re-parent to launchd.

## How to recognize it

After `go test -race -parallel 8 ./...` times out:

```sh
ps -ef | grep -iE 'aiwf-stub|aiwf-int-build|post-commit|pre-commit\.sh|succeed-shim' | grep -v grep | wc -l
```

A non-zero count of long-lived sleeping processes is the signature.

## Diagnostic record (M-0120 preflight, 2026-05-18)

Test run timed out at 11 minutes per package across `internal/policies`, `internal/recipe`, `internal/render`, `internal/repolock`, `internal/roadmap`. Investigation found:

- Stuck child processes in state `S` (interruptible sleep), 0% CPU, WCHAN empty
- Parent shells stuck in `__wait4()` (libsystem_kernel) waiting on children
- Children that have not executed their first command — `STATUS.md.tmp` was 0 bytes despite the script's only action being `printf '# regen content\n'`
- No held shared resources (`.git/aiwf.lock` flocks were all released; only Spotlight had read handles on the lockfiles)
- 33 of the 55 orphans were re-parented to launchd (PPID = 1)
- The exact same hook + stub pattern invoked from a plain shell prompt completed reliably 10/10 times in under 300ms each — confirming the bug is NOT in the script content or `/bin/sh` itself

The combination that triggers the bug:

- Go runtime fork/exec on macOS (known-fragile under high parallelism)
- `-race` (doubles instrumentation cost, widens race windows)
- `-parallel 8` (concurrent test goroutines, each doing multiple `exec.Command` invocations)
- Test fixtures that invoke `exec.Command("sh", hookPath)` (extra fork layer)

## Mitigations (immediate)

1. **Don't run `-race` on macOS.** CI runs `-race` on Linux where Go's fork/exec is solid. Local dev on macOS uses `go test -parallel 8 ./...` without `-race`. Race detection is a CI concern.
2. **Document this in CLAUDE.md.** The existing `-parallel 8` cap note is upstream of this finding; add a complementary note that race detection is Linux-only.
3. **Makefile**: optionally gate `test-race` behind a `RACE=1` env var or skip on macOS with a clear message.

## Root-cause investigation (deferred)

Whether the bug is in:

- macOS bash 3.2's signal-handler setup post-fork
- Go's fork/exec interaction with macOS pthread state
- A specific kqueue/poll wakeup miss
- Something else entirely

…is a separate investigation worth doing properly. Possible avenues:

- Build a minimal Go reproducer (`exec.Command("sh", "-c", "printf hi") .CombinedOutput()` in a tight loop with -race) and observe whether the bug surfaces outside aiwf
- Try bash 5 instead of /bin/sh (install via Homebrew, point the hook scripts at it explicitly)
- Try `exec.Command(hookPath)` (let kernel handle shebang) instead of `exec.Command("sh", hookPath)`
- Try `setsid` / process-group cleanup in the test framework so the SIGQUIT propagates to descendants
- File upstream if the bug reproduces in a tiny Go example

## Related decisions to ratify

A devcontainer (Linux-in-Docker on macOS host) would side-step this entire class of issue but introduces other tradeoffs (filesystem perf, worktree mapping, Claude Code integration overhead). Worth a separate ADR + epic if the team decides to commit. Not blocking this gap.

---
id: G-0462
title: 'Intermittent test failures: ETXTBSY on exec and golangci-lint lock contention'
status: addressed
priority: high
discovered_in: M-0281
addressed_by_commit:
    - d85733910
---
## What's missing

`go test ./...` failed intermittently for reasons unrelated to the change under
test. Two mechanisms were tracked here; both presented the same way — a red gate
that went green on an immediate re-run with no code change — and they were worth
fixing together, because what made them expensive was the shared symptom, not
the separate causes.

Timing-dependent stress-scenario oracles are a third mechanism of the same
class, tracked separately as **G-0468**, whose remedy is independent of the two
here.

### Mechanism 1 — `ETXTBSY` on exec of a just-written file

Tests that write an executable and then run it failed with
`fork/exec …: text file busy`. A script or binary is written into a temp
directory and exec'd shortly afterwards while the package's other tests run in
parallel. `ETXTBSY` is what the kernel returns when a file is exec'd while some
process still holds it open for writing, which a concurrent `fork` can produce
even after the writing goroutine has closed its own descriptor — the child
inherits the descriptor across the fork window.

Fixed under **G-0491**: `testsupport.WriteExecutable` holds `syscall.ForkLock`
across the write, excluding exactly those forks, and every executable stand-in
routes through it. Writing to a temp name and renaming does not help —
`ETXTBSY` is enforced against the inode, which the rename carries along.

### Mechanism 2 — a concurrent golangci-lint anywhere on the machine

`internal/policies` — `TestGolangciConfigRulesFire` shells out to a real
`golangci-lint` per guarded rule, to prove the rule fires rather than merely
appearing in the enable list. It passed the inherited environment, so it
competed for golangci-lint's start-up lock with every other instance on the
machine.

golangci-lint acquires that lock unless `--allow-parallel-runners` is passed. It
waits briefly for it and then exits with `Error: parallel golangci-lint is
running`. The harness asserts on the child's *output* and deliberately ignores
its exit code, because findings are expected, so the refusal read as a missing
finding and the test reported:

```
rule forbidigo-panic did not fire: golangci-lint output lacked "(forbidigo)"
 — the config rule is dormant, disabled, or dropped from the enable list
```

Any concurrent golangci-lint sufficed: another worktree's `make lint`, a
pre-push hook, an editor integration. This repo routinely carries several
worktrees at once, so the condition was ordinary rather than exotic.

The refusal was a defect in the *diagnostic* as much as in the timing: the
condition is external and harmless, but the message accused the lint
configuration of exactly the defect the harness exists to detect, so a reader
who trusted it went looking for a dormant rule that was working fine.

**The lock is keyed to the temp directory, not to the lint cache.** It lives at
`os.TempDir()/golangci-lint.lock`, so scoping `GOLANGCI_LINT_CACHE` — the
per-worktree measure `make lint` applies under G-0179 — does not avoid it.
Measured on v2.12.2: two concurrent runs with *different* cache directories and
a shared temp directory still produce the refusal.

Resolved by passing `--allow-parallel-runners`, which is the documented way to
decline the lock and is a public contract, where the lock's location is an
implementation detail that is documented nowhere. The harness additionally runs
against a private `GOLANGCI_LINT_CACHE` so each run is hermetic. A refusal that
reaches a reader anyway is now reported as a refusal, naming the lock and
stating that the run ended before the config was applied and so is no evidence
about any rule.

`--allow-serial-runners` queues on the lock rather than refusing; it was
rejected because it trades a spurious failure for an unbounded wait on however
many runners the machine happens to be running.

### Found alongside — the harness contained a row that could not fail

Each row asserted that a substring appeared anywhere in golangci-lint's output,
and golangci-lint echoes the fixture's path, which `t.TempDir()` derives from
the subtest name. The `gocritic-filepathJoin` row asserted only `filepathJoin`,
so it was satisfied by the directory the test was running in: removing
`gocritic` from the enabled linters entirely left the row passing. The same
mechanism partly hollowed the `forbidigo-panic` row, whose `panic` element was
likewise matched by its own path.

This is the G-0264 condition the harness was built to prevent, reproduced
inside the harness itself. Assertions now run against the message half of each
finding line, with the echoed path stripped, so a row can only pass on
something a linter actually reported.

## Why it mattered

`go test ./...` is not an advisory signal here. It runs inside `make check-fast`,
`make ci`, and `make coverage-gate`, and CI runs it on every push, so these
failures landed on the gate that is supposed to decide whether work is safe to
integrate.

A gate that is occasionally red for reasons unrelated to the change under test
teaches readers to re-run rather than read. That is the expensive part: the next
genuine failure arrives looking exactly like the last spurious one. The repo
leans hard on mechanical chokepoints over vigilance, and this is the failure
mode that erodes them.

---
id: G-0485
title: Gpg-signing test fixture leaks a gpg-agent and GNUPGHOME per test binary
status: addressed
priority: medium
addressed_by_commit:
    - 3690153703f9930f1c8f3eb8a9458cdba016a5cf
---
## What's missing

`buildGPGSignFixture` (`internal/gitops/committree_gpgsign_test.go`) creates a throwaway `GNUPGHOME` with `os.MkdirTemp` and generates an ephemeral signing key inside it. Generating the key starts a `gpg-agent` daemon rooted at that directory. Neither the directory nor the daemon is ever torn down.

The fixture is package-level behind a `sync.Once` — the shared read-only fixture shape the test-discipline rules call for, built once per test binary and never regenerated. That shape is correct, and it is also why the ordinary cleanup affordances do not reach it: `t.TempDir` and `t.Cleanup` are per-test, while the fixture outlives every test that uses it. The only teardown point matching the fixture's lifetime is `TestMain`, and this package's `TestMain` returns straight through `os.Exit(m.Run())`, which runs no deferred cleanup.

The sibling helper `emptyGPGWrapper` in the same file does use `t.TempDir`, and its key-less home spawns no persistent agent. The leak is specific to the key-generating path.

## Why it matters

Each `go test` invocation that touches `internal/gitops` leaks exactly one `gpg-agent` process and one `/tmp/aiwf-gpgsign-home-*` directory. Nothing reaps either: the agents idle indefinitely, and the directories persist until the machine's temp store is cleared.

Measured on a devcontainer up for roughly a day: 215 live agents holding 753 MB of resident memory, plus 8.4 MB of directories, the oldest agent idle for over six hours. The count tracks test *invocations*, not failures, so an ordinary inner-loop cadence accumulates them faster than anyone notices — every `make check-fast` adds one.

CI never sees this. Its runners are discarded after the job, so the leak is bounded there by construction. The exposure is entirely on long-lived development environments — the devcontainer and a contributor's own machine — which are precisely where hundreds of megabytes of idle daemons compete with the test suite that keeps spawning them.

Distinct from G-0375, which occupies the same neighborhood in the opposite direction: that gap is about ambient `commit.gpgsign` leaking *into* fixtures from the invoking machine's global config. This one is a fixture leaking daemons *out* onto the machine. Resolving either does nothing for the other.

## Resolution shape

Give the fixture a teardown matching its lifetime. Capture the `GNUPGHOME` path in a package-level variable alongside the existing `gpgSignProgram` / `gpgSignFingerprint`, and have `TestMain` run `gpgconf --homedir <home> --kill all` followed by `os.RemoveAll(home)` once `m.Run()` returns.

The `os.Exit` detail is load-bearing: deferred calls do not run through it, so the teardown has to sit between a captured exit code and the exit call rather than in a `defer`.

Teardown must be inert when the fixture was never built. The `sync.Once` body does not run when gpg is absent (every caller skips first) or when no test in the selected run touches gpg signing, so an empty path variable means there is nothing to clean. Killing the agent before removing the directory is the required order — the agent holds sockets inside it. `gpgconf --homedir <home> --kill all` terminates a live fixture agent on gpg 2.2.40, exiting 0.

Pinning the fix is awkward from inside the package, because the teardown runs after every test in the binary has finished and so no test in that binary can observe it. Two honest options: a subprocess test that runs the package's own test binary and asserts no `aiwf-gpgsign-home-*` directory survives it, or a structural assertion that `TestMain` routes through the cleanup rather than calling `os.Exit(m.Run())` directly. The first pins the behavior and costs a nested test-binary run; the second is cheap and pins shape rather than effect.

## Where to fix

- `internal/gitops/committree_gpgsign_test.go` — `buildGPGSignFixture` retains the homedir path; add the teardown helper.
- `internal/gitops/setup_test.go` — `TestMain` captures the code from `m.Run()`, tears down, then exits.

---
id: G-0693
title: Makefile passes the Darwin signing wrapper on Linux, defeating the test cache
status: open
---
## What's missing

`Makefile:24` sets `TEST_EXEC := $(CURDIR)/scripts/sign-and-run.sh` unconditionally, and
eleven `go test` recipes pass it as `-exec=$(TEST_EXEC)`. `scripts/sign-and-run.sh:7`
signs only on Darwin and otherwise `exec "$@"` — a no-op. But `-exec` sits outside the
flag set `go help test` lists as cacheable, so passing it defeats Go's test cache on
every host, including the devcontainer `CLAUDE.md` documents as the primary path.

Measured in the devcontainer (Linux 6.12.76, go 1.25), second run of each pair:

```
$ go test ./internal/version/
ok  github.com/23min/aiwf/internal/version  (cached)
$ go test -exec=$(pwd)/scripts/sign-and-run.sh ./internal/version/
ok  github.com/23min/aiwf/internal/version  0.227s
```

An empty value is accepted and stays cacheable: `go test -exec= ./internal/version/`
reports `(cached)` on a repeat, and `go test -exec= -count=1 ./internal/version/` runs
the tests in 0.238s. Coverage runs cache too, profile included — two
`-covermode=atomic -coverprofile=… -coverpkg=./internal/...` runs over that package
reported `0.226s` then `(cached)`, and both profiles were 121 lines.

The cost is the whole suite, every time. In this container `make ci` took 214s and a
repeat `make check-fast` over an unchanged tree took 225s, of which `make lint` is 7s.

Three places state the no-cache property as a whole-repo fact and would stop being
true on Linux: `CLAUDE.md:88`, `Makefile:164`, and
`internal/policies/coverage_gate_wiring_test.go:88`. `CLAUDE.md`'s devcontainer
subsection separately claims `go test` "runs unwrapped" there, which `make test` does
not honor; `PolicyM0134ClaudeMDTestRunningSections` pins that claim's presence, not
its truth.

## Why it matters

Every local test run pays a full uncached suite, so wall-clock scales with the number
of runs rather than with what changed. The review loop `wf-patch` step 6 mandates
re-runs the gate once per round of findings, and that is where it is felt.

Go's cache does not observe inputs a test reads through a subprocess, so a policy test
that shells out to git can serve a stale green locally once caching is live. The
profile-driven gate targets already pass `-count=1`, and the workflows pass the
wrapper by an absolute path of their own, so neither is affected.

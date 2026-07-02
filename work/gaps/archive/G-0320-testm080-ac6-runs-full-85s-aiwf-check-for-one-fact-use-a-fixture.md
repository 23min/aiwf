---
id: G-0320
title: TestM080_AC6 runs full 85s aiwf check for one fact; use a fixture
status: addressed
discovered_in: M-0196
addressed_by_commit:
    - 1c3339b6
---
## What's missing

The `internal/policies` test suite takes ~84s wall, and a single test —
`TestM080_AC6_NoUnexpectedTreeFileWarning` (`internal/policies/m080_test.go`) —
is **82.18s of that 84.24s (97%)**. Every other test in the package is under
4.5s and runs in parallel; this one dominates the wall clock alone.

The test builds the `aiwf` binary (cached at `/tmp/aiwf-m080`, so the build is
*excluded* from the 82s) and then runs the **full `aiwf check --format=json`
subprocess against the entire live repo tree** — solely to assert one narrow
fact: that `aiwf check` emits no `unexpected-tree-file` finding for
`criticalPathMdPath`. It is the textbook "exercise the whole system to verify
one fact" anti-pattern.

## Why it matters

That ~84s is paid on **every Go-touching commit** (the `pre-commit.local`
G-0280 gate runs `go test ./internal/policies/...`) and in the **CI policy-test
job**. It is a large share of the wrap+push slowness the operator flagged after
the M-0196 wrap. The cost is structural, not incidental: the test asserts a
narrow rule-level fact via the slowest possible path.

## Proposed fix shape

Decouple the assertion from the live tree. Assert the `unexpected-tree-file`
rule's behavior on a **minimal fixture**: build a temp tree containing just the
critical path file (and whatever frontmatter/placement makes it *expected*),
then either call the rule's check function directly or run `check.Run` over the
fixture, and assert no `unexpected-tree-file` finding for that path. Sub-second,
no binary build, no 85s full-tree scan. It pins the *same* behavior more
precisely (the narrow path, not "the whole repo happens to be clean").

This is the cheap, test-level decoupling. It does **not** wait on the deeper
kernel fix (the 85s `aiwf check` itself — tracked in G-0319); even after check
is fast, asserting a rule on a fixture is the correct shape.

Sanity-check the rest of the suite after: with this test fixtured, the suite
should drop from ~84s to ~5s, making the per-commit gate and CI policy job
cheap.

## Discovered in

M-0196 — profiling the `internal/policies` suite (`go test -v` per-test
timing) while investigating the wrap+push slowness the operator flagged.

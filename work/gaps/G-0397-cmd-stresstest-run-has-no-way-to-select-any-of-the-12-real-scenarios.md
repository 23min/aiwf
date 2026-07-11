---
id: G-0397
title: cmd/stresstest run has no way to select any of the 12 real scenarios
status: addressed
discovered_in: M-0244
addressed_by_commit:
    - 930d391e
---
## What's missing

`cmd/stresstest run`'s scenario selection is still hardcoded to
`placeholderScenario` (`cmd/stresstest/run.go`'s `newScenario` closure) —
the trivial "git init + `aiwf check`, always passes" stand-in M-0240 built
with its own comment: "no real catalog scenario ships until M-0241+."
`cmd/stresstest`'s command tree (`root.go`) offers only `run` and
`compose`; there is no `--scenario` flag, no name→constructor registry,
and no way to select any of the 12 real scenarios built across
M-0241 through M-0244 (`ConcurrentIDAllocationScenario`,
`ParallelBranchReallocateScenario`, `ConcurrentWriterAtScaleScenario`,
etc.) through the actual CLI binary. Confirmed directly: none of the 12
scenario constructors are referenced anywhere under `cmd/`.

## Why it matters

E-0062's own Scope section commits to "a harness driving the real,
compiled `aiwf` binary as a subprocess... on-demand invocation only: a
script/binary a human runs when they want it... lives in its own tree."
Today the only way to invoke any real scenario is
`go test ./internal/stresstest/... -run <TestName>` — the dedicated
`cmd/stresstest` binary this epic built specifically for on-demand
invocation cannot run any of them. The underlying scenario properties
(real subprocess/git, deterministic pass/fail oracle via each scenario's
own classify function, preserved-repo-state-on-failure) are all
genuinely present and verified — what's missing is only the
single-binary convenience surface the epic's own framing describes.
Discovered during M-0244/AC-3's walk of E-0062's success criteria.

---
id: G-0399
title: VerbSequenceScenario isn't registered in cmd/stresstest's catalog
status: open
discovered_in: M-0249
---
## What's missing

`internal/stresstest/verb_sequence.go`'s `VerbSequenceScenario`
(M-0241/AC-1 — a property-style FSM random-walk driving real `aiwf`
subprocesses through legal-transition sequences) is not registered in
`cmd/stresstest/registry.go`'s catalog. `cmd/stresstest list` and
`--scenario all` cannot reach it; only `go test -run TestVerbSequence...`
can.

## Why it matters

M-0249's own AC-1 explicitly enumerates exactly 12 scenario names, and
its Out-of-scope section says "does not add a 13th" — so this is not a
defect against M-0249, which is met exactly as specified. The open
question is whether E-0062's own "run the whole catalog on demand" framing
(the epic's own success criteria) is understood to mean these 12
concurrency/fault-injection scenarios specifically, with `VerbSequence`
staying a `go test`-only property probe by design — or whether the
exclusion was simply never revisited once the registry existed.
`VerbSequence` is a different shape (a general FSM random-walk, not a
targeted concurrency/fault probe), so the exclusion may well be
intentional; this just makes it a conscious call rather than a silent
gap once the epic closes.

## Direction

A one-line confirmation resolves this: either register `VerbSequenceScenario`
under a 13th catalog name (cheap — it already implements the `Scenario`
interface), or explicitly note in the epic's own wrap-up that the on-demand
catalog is scoped to the 12 concurrency/fault-injection scenarios by
design, and `VerbSequence` remains a `go test`-only property probe.

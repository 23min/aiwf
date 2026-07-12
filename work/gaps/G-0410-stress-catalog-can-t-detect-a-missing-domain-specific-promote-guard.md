---
id: G-0410
title: stress catalog can't detect a missing domain-specific promote guard
status: open
---
## What's missing

`cmd/stresstest`'s `verb-sequence` scenario drives a random walk of `aiwf promote` calls against every entity kind and judges each outcome against `entity.ValidateTransition`'s bare FSM legality (`classifyVerbSequenceStep`, `internal/stresstest/verb_sequence.go`). The oracle asserts only two things: an FSM-illegal transition must always refuse, and a successful transition must land exactly one commit. It explicitly tolerates any FSM-legal transition that gets refused for an orthogonal business rule — without ever asserting that the business rule actually fires when its precondition holds. A verb-time guard that is missing entirely (not merely present-but-buggy) is invisible to it: a silent bypass looks identical to a legitimate success.

Confirmed empirically against G-0335 (`aiwf promote <milestone> cancelled` used to bypass the open-AC guard `aiwf cancel` already enforced): running the `verb-sequence` scenario 30x against the pre-fix binary passed 30/30 with zero violations, and running it 20x against the fixed binary also passes clean — the scenario cannot distinguish the two.

## Why it matters

The stress catalog is the repo's only automated surface driving `aiwf` end-to-end under randomized/concurrent state; a reader could reasonably assume a clean catalog run means promote/cancel behavior is consistent across guards. It doesn't test that. A guard added to one verb surface (`cancel`) but not its sibling (`promote`) for the same domain invariant — exactly G-0335's shape — ships silently and stays undetected by the catalog indefinitely, caught only if a human notices the asymmetry or the separate spec-catalog audit (M-0123/M-0124/M-0125) happens to enumerate the transition. Two candidate fix directions: extend `classifyVerbSequenceStep`'s oracle to also check the domain-specific preconditions it can already observe from the fixture (open ACs, non-terminal children, resolver requirements), or explicitly document the scenario's scope as FSM-shape regressions only, with cross-verb guard consistency called out as deliberately out of scope.
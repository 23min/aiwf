---
id: G-0211
title: Combinatorial verb-composition scenarios untested at branch-choreography E2E
status: open
discovered_in: M-0159
---
## What's missing

The branch-choreography surface (M-0102..M-0106 + M-0158 chokepoints) is pinned at the rule-unit level only — synthetic `scope.Commit` fixtures driving `RunIsolationEscape` directly, plus authorize-time integration tests in `internal/cli/integration/authorize_cmd_test.go`. The check-time surface (`isolation-escape` rule + the BranchOracle interface + the CLI gather layer that populates it) has no real-git integration test that exercises a verb-composition scenario end-to-end.

Specifically missing: a scenario-table integration test that builds the aiwf binary, sets up a real git repo, runs a sequence of verbs (authorize → AI commits via real git → check), and asserts the actual envelope output. Same gap for every M-0106 path: bound-branch silent, paused silent, cherry-pick silent (currently the unit test injects the `cherryPicked` map directly), force-amend silent (currently the unit test uses a pre-baked human/actor + aiwf-force fixture), scope-ended silent, worktree-mismatch fires, per-commit firing.

The kernel principle "test the seam, not just the layer" (CLAUDE.md Go conventions §"Test the seam, not just the layer") was codified after M-0106 itself shipped with the rule effectively disabled — the CLI passed `nil` for the oracle for four implementation cycles (F-1 from the M-0106 retrospective). That shipped-disabled incident IS the evidence that unit-level coverage is insufficient for this surface.

## Why it matters

Without combinatorial real-git E2E coverage:

- Verb-composition bugs (sequences of authorize+commit+pause+resume+cherry-pick that combine in surprising ways) silently work or silently fail in production while unit tests stay green.
- The CLI gather layer that populates the BranchOracle has no integration test — the same layer where M-0106's "oracle passed nil" bug lived undetected through four implementation cycles.
- Operator workflows that compose existing verbs in unexpected ways have no guarantee of correctness. The user's framing during M-0159 planning: "verbs can be composed in any way for any reason" — that combinatorial space is currently untested.

Drives M-0159's primary deliverable: a combinatorial scenario-table integration test framework under `internal/cli/integration/`, with helpers for realistic scenarios (shallow clone, force-push, branch rename, cherry-pick, amend, merge, detached HEAD) that other M-0159..M-0161 ACs reuse.

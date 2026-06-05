---
id: G-0213
title: Cellcoverage fixture writes fictional aiwf-branch values (latent landmine)
status: open
discovered_in: M-0159
---
## What's missing

`internal/cellcoverage/authorized_scope.go:34` writes a fictional `aiwf-branch` trailer value into every authorized-scope cell-coverage fixture:

```go
CurrentBranch: "epic/" + entityID + "-cellcoverage-fixture",
```

The branch by that name does not exist in the test repo or anywhere on disk. The fixture stamps the value into the authorize commit's `aiwf-branch:` trailer purely to satisfy the trailer-shape rule's "non-empty when actor is ai/" requirement.

Today no kernel rule reads `aiwf-branch:` against a "this ref must resolve to a real branch" check. Tests pass because the fictional value is structurally well-formed (matches `epic/E-NNNN-<slug>` shape) and no rule validates resolvability.

The moment M-0159 or M-0161 lands a rule that polices "bound branch resolves to a real ref in the BranchOracle's index" — exactly the M-0158/T-6 case formalized — **every M-0125 positive cell test breaks in the same PR**. The fixture's fictional branch will not resolve, the rule will fire, and the M-0125 cell-coverage tests will fail simultaneously.

## Why it matters

This is a sequencing constraint, not a stale-fixture cleanup task. Any milestone that adds a branch-resolution rule must address the fixture in the same commit set, or every consumer of `cellcoverage.AuthorizedScope` breaks.

Options for the fix:

1. Make the fixture actually create the branch (`git branch <ref>` in the fixture setup).
2. Mark the fixture's authorize commit with a sentinel trailer (e.g., `aiwf-cellcoverage-fixture: true`) that the new rule recognizes as exempt.
3. Make the new rule fail-open when the BranchOracle has no entry at all (vs partial entries), so empty-oracle tests stay silent.

Discovered by the history-mining subagent investigation §4a during M-0159 planning (2026-06-02). The fixture has shipped since M-0125 (the cell-coverage discipline epic); it has been a latent landmine across multiple milestones.

Must address before any branch-resolution rule lands in M-0159 or M-0161.

---
id: G-0154
title: epic-branch worktree mislabeled as milestone via post-merge trailer cascade
status: addressed
discovered_in: M-0124
addressed_by_commit:
    - a6b6e339
---
## What's missing

`aiwf status --worktrees` mislabels every epic-branch worktree as a milestone-driver as soon as a child milestone is merged into the epic branch. The misattribution is silent — the worktree section renders confidently as `→ M-NNNN (driven)` with full milestone context, while the epic the worktree was created to drive disappears from the driver row entirely.

The cause sits in `correlateBranchToEntity` (internal/cli/status/worktrees.go, around line 240). The hybrid cascade walks three steps in order:

1. **Scope-defining events** from `aiwf-verb`/`aiwf-entity`/`aiwf-to` trailers ahead of trunk (`authorize`, `promote → active`, `promote → in_progress`, phase promotes).
2. **Most-recent `aiwf-entity` trailer** as a fallback.
3. **Branch-name parsing** via `branchEntityPattern`.

When a milestone branch (`milestone/M-NNNN-...`) merges into an epic branch (`epic/E-NNNN-...`), the merge commit pulls the milestone's wrap-time `aiwf-verb: promote aiwf-entity: M-NNNN aiwf-to: done` trailers onto the epic branch. The cascade sees those at step 1, returns the milestone id, and never reaches the branch-name parser that would have correctly returned the epic id.

The cascade was originally designed for the dual case: an operator working on M-NNNN inside a generically-named worktree should still be recognized as driving M-NNNN via their authorize trailer. That case is real and the cascade handles it correctly. The bug is that the cascade gives trailer-events precedence over an *explicit* branch-name signal — even when the operator named the branch `epic/E-NNNN-...` to declare their intent.

## Live evidence

Captured 2026-05-22 on this consumer repo, after M-0124 wrapped on its branch and was merged into the E-0033 epic branch (epic itself not yet merged to trunk):

    Worktree: /workspaces/aiwf-epic-E-0033
      ⎇ epic/E-0033-pin-legal-kernel-verb-workflows-mechanically  •  last commit 46m ago
      last entity touch 1h ago
      E-0033 — Pin legal kernel-verb workflows mechanically [proposed]
      → M-0124 — Positive cell coverage: legal workflows succeed with expected post-state [done]  (driven)
        depends on:
          ...
        ACs:
          ...
        Surfaced gaps:
          ...
      WRAP PENDING — driver done but branch ahead of trunk by 248 commits; merge to trunk before removing

Both the parent-epic breadcrumb (E-0033) and the driver row reference E-0033 implicitly, but the `→ (driven)` marker is on M-0124. The worktree is *named* for E-0033 and *should* render with E-0033 as its driver and its milestones (including M-0124) in epic-expansion. Instead it duplicates the M-0124 milestone-driver view that already lives on the `/workspaces/aiwf-M-0124-...` worktree.

## Why it matters

The visible symptoms compound:

1. **Two worktrees show the same driver.** Looking at the side-by-side, the E-0033 and M-0124 sections are nearly identical — only the worktree paths differ. The operator can't tell at a glance which session is doing what.
2. **Epic-expansion never renders.** The whole point of an epic-driver worktree section is to summarize the epic's milestones, closes-gaps, surfaced-gaps. None of that appears because the cascade picked a milestone driver.
3. **The fix-iteration loop is broken.** When the operator's current task is "drive the E-0033 wrap step", they expect their worktree to be labeled accordingly. The misattribution makes the planning surface lie about what the session is for.

This is the same failure-mode pattern aiwf's "framework correctness must not depend on the LLM's behavior" rule is meant to eliminate. Branch naming is a deliberate, conventional signal; the kernel should respect it.

## Fix shape

Reorder the cascade to make branch-name parsing the primary signal and the trailer cascade the fallback. The conventional ritual branch shapes (`epic/E-NNN-...`, `milestone/M-NNN-...`, `patch/[Gg]-NNN-...`) carry deliberate operator intent; the trailer cascade is only needed when the branch name does not encode an entity.

```go
func correlateBranchToEntity(ctx context.Context, rootDir, branch string) string {
    if branch == "main" { return "" }
    // Branch name is the primary signal when it follows the ritual
    // shape. The operator named the branch deliberately; trust it.
    if id := parseEntityFromBranch(branch); id != "" {
        return id
    }
    // Non-ritual branches: fall back to trailer cascade as before.
    events := branchAiwfEvents(ctx, rootDir, branch)
    if id := scopeDefiningEntity(events); id != "" {
        return id
    }
    return mostRecentEntity(events)
}
```

Three-line change, no new helpers, no schema additions.

ACs (suggested):

- AC-1: An `epic/E-NNN-...` worktree whose recent commits include a `promote M-NNN done` trailer (e.g., from a milestone merge) correlates to E-NNN, not M-NNN. Direct regression test against the bug.
- AC-2: A `milestone/M-NNN-...` worktree whose recent commits include a `promote E-NNN active` trailer (e.g., an unrelated epic-activation from a parallel session) correlates to M-NNN, not E-NNN. Mirror case.
- AC-3: A non-ritual branch (e.g., `fix/some-thing`) with a `promote M-NNN in_progress` trailer still correlates to M-NNN via the cascade. Confirms the fallback path is preserved.
- AC-4: A non-ritual branch with no aiwf trailers correlates to "" (no driver). No-signal case.

## Alternative considered (and rejected)

Option 2 from the design discussion: filter merge-introduced trailers out of the cascade. Recognize when a scope event's commit is a merge that brought the trailer in (vs an authored promote on the branch itself), and exclude those.

Rejected because the bug class is broader than merges. An authored commit on the epic branch with `aiwf-entity: M-NNN` (e.g., `fix(check): tighten M-NNN AC enforcement` referencing the child milestone) would still mislead the cascade. Branch-name precedence cuts the knot for every variant.

## Test approach

Unit tests on `correlateBranchToEntity` using hand-constructed `branchAiwfEvents` outputs — the function already has its events-walking logic factored into `scopeDefiningEntity` and `mostRecentEntity`, both of which are unit-tested. Add a new `TestCorrelateBranchToEntity_BranchNamePrecedence` that drives the four AC cases by passing crafted event slices through the (now-reordered) cascade logic.

The git-shell step (`branchAiwfEvents` reading actual git log output) is already covered indirectly by the live integration. The unit tests pin the cascade ordering.

## History

Surfaced during the M-0124 worktree wrap pass, when the operator noticed two worktrees (`epic/E-0033-...` and `milestone/M-0124-...`) rendering identical milestone-driver sections in `aiwf status --worktrees`. Diagnosed against the worktrees.go cascade. Captured alongside G-0153 (which closed the misleading-cleanup-hint bug) since both surfaced in the same live wrap-pending scenario.

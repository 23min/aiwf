---
id: G-0435
title: Declared-sequence gate's shape list reads as exhaustive, blocking batching
status: addressed
priority: medium
discovered_in: M-0126
addressed_by_commit:
    - 090241a
---
## Problem

CLAUDE.md's "Gate discipline survives compaction" section states the declared-sequence gate as a general capability — *"a single gate MAY cover a sequence of local, reversible mutations at one moment provided the gate enumerates every action verbatim"* — but immediately follows that sentence with a `Batchable:` list of specific named shapes: promotes, an `aiwf archive` sweep, a local merge to mainline, a tracker-closure promote, local branch/worktree deletion, and the wrap rituals' named terminal sequence. The shipped `internal/skills/embedded-guidance/aiwf-guidance.md` fragment repeats the same shape.

In practice the list reads as the operative policy, not as illustrative examples of the general test that precedes it. When a set of mutations mixes verb types that individually resemble list entries but weren't grouped together before — an `aiwf add gap`, a plain-file git commit, and an `aiwf edit-body`, discovered together while filing one gap — the combination doesn't obviously match any named shape. The conservative reading wins: gate each mutation separately, defeating the batching the general clause is supposed to allow.

This is not a new problem. G-0295 already generalized the declared-sequence gate from a wf-patch-only mechanic into a stated general capability for "any sequence of local, reversible mutations." G-0341 separately pushed the shipped guidance's framing toward neutral, operator-owned wording. Both landed, and the friction still recurs — the general clause and the illustrative list send conflicting signals, and nothing in the current wording forces the general reading to win over the narrower, safer one.

Surfaced most recently while working under an open `aiwf authorize` scope (E-0034/M-0126), but the mechanism has nothing to do with authorize or delegation — the same conservative-reading trap recurs in any session, delegated or not, where local/reversible mutations of different verb types are discovered together.

## Direction

Reframe the rule's list from an enumerated closed set to explicit examples of a general test: any set of local, reversible mutations discovered together within one coherent unit of work (one AC, one bug investigation, one gap-filing pass, one milestone wrap) may be presented as a single declared-sequence gate — regardless of verb-type mixing — provided:

- the gate enumerates every action verbatim,
- the human can approve a subset,
- any deviation (conflict, finding, unexpected dirty state, an action outside the enumerated list) aborts and re-gates, and
- neither sovereign acts nor outward/irreversible actions are ever included, no matter how the unit of work is scoped.

Apply the reworded rule in both CLAUDE.md's "Gate discipline survives compaction" section and the shipped `internal/skills/embedded-guidance/aiwf-guidance.md` fragment, preserving the "ships vs. stays" split (this rule already ships). `PolicyM0211GuidanceOperatingAnchors`'s `gate-per-mutation` anchor pins only the fragments `"each mutating action"` and `"approval gate"` — it does not pin the `Batchable:` list wording, so the reword is free to land without touching that policy.

Open question for whoever picks this up: does "one coherent unit of work" need a crisper boundary (tied to an AC, a single investigation, or a wall-clock/turn window), or is the enumerate-and-approve mechanic itself sufficient protection regardless of how loosely the unit is scoped?

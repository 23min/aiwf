---
id: G-0059
title: 'Branch model: no canonical mapping from entity hierarchy to git branches; epic/milestone work lands on whichever branch is current'
status: open
discovered_in: M-0069
---
## What's missing

aiwf has no canonical mapping from its entity hierarchy (epic → milestone → AC) to git branches. Today, every verb commits on whatever branch is current, with no surface that nudges the operator (human or AI) to consider branch isolation before delegated multi-commit work begins.

The shape this gap proposes (without locking in):

```
main
└── epic/E-NN-slug          (epic = integration branch)
    └── milestone/M-NNN-slug (milestone work; merges into epic on done)
```

Rules implied by that shape:

- Entity *creation* lands on the parent's integration branch — epic created on `main`, milestone created on its epic branch. Just the `aiwf add` commit.
- Entity *work* (`promote`, `cancel`, `edit-body`, AC phase walks, `authorize`/`pause`/`resume`) happens on the entity's own branch, not on the parent's.
- Milestone branch merges into its epic branch when the milestone reaches `done`.
- Epic branch merges into `main` when the epic reaches `done`.
- ACs do not get their own branches — they ride on the milestone branch alongside the test/code commits that satisfy them.

This is **not** the same gap as "AI agents should branch before milestone work" (a workflow preference). The actual missing piece is a *kernel-level surface* that makes branch context visible at the natural ceremony points — `aiwf authorize`, the first `aiwf promote --phase` of a milestone, the `aiwf-authorize` and `aiwf-promote` skills. Per CLAUDE.md "framework correctness must not depend on the LLM's behavior", an advisory in CLAUDE.md alone is not a guarantee; the chokepoint must be reachable through channels an AI assistant routinely consults.

## Why it matters

Per-mutation atomicity (CLAUDE.md design decision §7) gives one commit per verb. That commit-level guarantee is what `aiwf history` projects against. It says nothing about how to compose mutations into a *unit of merge*. There is no "milestone branch" or "epic branch" concept that the kernel surfaces, even though the planning hierarchy is already structured (epic owns milestones owns ACs) in a way that maps cleanly to a branch hierarchy.

Concrete evidence: M-0069 (this gap's `discovered_in`). The milestone produced 23 commits (7 test/code commits + 7 `aiwf edit-body` AC-prose commits + 14 `aiwf promote --phase` commits) all on `poc/aiwf-v3`. The PoC branch is intentionally free-form per CLAUDE.md "commit directly on the branch; no PR ceremony" — but that license was meant for solo human work, not AC-shaped delegated AI work that produces 20+ commits per milestone. Bisecting M-0069's history, reverting a single AC, or pausing mid-milestone is materially harder than it would have been on an isolated `milestone/M-069-tdd-retrofit-e14` branch with a single merge point.

The cost when this drift happens is not catastrophic but it compounds:

- **AI delegation amplifies the drift.** A human delegating "drive M-0069 through TDD" gets 23 commits without any prompt to consider isolation. The AI can't know to ask without a kernel-level surface telling it the branch context matters.
- **Authorization scopes are the natural attachment point and currently silent.** `aiwf authorize <id> --to <agent>` already exists as the ceremony where the human says "this entity, this agent, autonomous." It says nothing about *which branch* the work runs on, even though that is exactly the kind of decision the human is making implicitly when they hand off.
- **The pre-existing scope FSM (`active | paused | ended`) couples cleanly to a branch lifecycle**, if we choose to pursue that. Open scope on `aiwf authorize` → create-or-attach milestone branch. End scope on terminal-status → merge or close branch. The hooks line up.

## Resolution shape (open)

The gap captures the question; the answer is downstream and likely a multi-step ladder rather than a single fix. Possibilities, roughly in increasing strength:

1. **CLAUDE.md advisory** — document the recommended branch hierarchy without enforcement. Cheapest. Per the kernel's "correctness must not depend on LLM behavior" principle, this alone is insufficient as a guarantee but useful as a starting layer.
2. **Skill-level surfacing** — `aiwf-authorize` and `aiwf-promote` skills surface branch context ("current branch: <name>, <N> ahead of origin/<base>, last unrelated commit <SHA>") at the entry point of multi-commit work. AI-discoverable per the kernel rule.
3. **Verb-level branch context in `aiwf authorize`** — the verb itself prints a branch-context line and offers `--branch <name>` to record an explicit branch with the scope. Couples the scope FSM to a named branch.
4. **Verb-level branch creation** — `aiwf authorize` (or first `promote --phase` of a tdd-required milestone) creates the canonical branch if absent. Stronger; harder to roll back.
5. **`aiwf check` finding** — flag commits whose `aiwf-entity` trailer points at a milestone whose work spans more than N branches, or whose milestone-scoped commits are interleaved with non-aiwf commits. Strongest; narrowest signal-to-noise question.

Patches are out of scope for this gap. They follow a different shape and are tracked separately.

The fix should preserve the PoC branch's permissive ethos for solo human work — branch isolation is the *delegated multi-commit* case, not every commit. A future ADR may be the right place to record the chosen ladder steps and their rationale.

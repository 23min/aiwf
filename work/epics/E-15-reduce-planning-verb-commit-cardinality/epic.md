---
id: E-15
title: Reduce planning-verb commit cardinality
status: proposed
---

## Goal

Add batching capabilities to the `aiwf add` family so a planning session produces one commit per logical mutation rather than one per entity. Closes G-051 (commit-count explosion in planning sessions) and G-052 (skill/check policy contradiction over plain-git body edits) by giving the verb routes the same expressive power that plain `git commit` currently has, while preserving the kernel's atomicity guarantee.

## Scope

- `--body-file` (and `--body -` for stdin) on every `aiwf add` variant — epic, milestone, gap, decision, ADR, contract.
- Repeated `--title` on `aiwf add ac <milestone-id>` — N ACs in one atomic commit.
- A new verb `aiwf edit-body <id> --body-file <path>` for post-creation body edits.
- `aiwf-add` skill text revised to remove the plain-git body-edit carve-out, now that verb routes cover both creation and post-creation cases.

## Out of scope

- `aiwf plan apply <file>` — the declarative whole-plan-in-one-commit verb. Real value but a separate decision; the in-scope items get a planning session from ~42 commits to ~8, which clears the user-stated friction threshold.
- Nested batching (e.g., `aiwf add epic` accepting inline `--milestone` definitions). Defer until in-scope work proves itself.
- Changes to the `provenance-untrailered-entity-commit` policy. The policy is correct as-is; once verb routes cover all body-edit cases, it stops firing under normal use and stands as a backstop against accidental hand-edits.

## Note on the meta-irony

This epic plans the fix for friction observed during E-14's planning. By design, creating the epic itself triggers the same friction one more time — accepted as the cost of capturing the design rationale durably before the fix lands. After M-056 ships, this epic's body and ACs become the first thing to use the new `--body-file` flag.

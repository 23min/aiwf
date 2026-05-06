---
id: E-15
title: Reduce planning-verb commit cardinality
status: active
---

## Goal

Add batching capabilities to the `aiwf add` family so a planning session produces one commit per logical mutation rather than one per entity, and close the verb-route gaps that today force users to hand-edit frontmatter. Closes G-051 (commit-count explosion in planning sessions), G-052 (skill/check policy contradiction over plain-git body edits), and G-053 (no verb-flag for resolver-pointer fields on status transitions) by giving the verb routes the same expressive power that plain `git commit` currently has, while preserving the kernel's atomicity guarantee.

## Scope

- `--body-file` (and `--body -` for stdin) on every `aiwf add` variant — epic, milestone, gap, decision, ADR, contract.
- Repeated `--title` on `aiwf add ac <milestone-id>` — N ACs in one atomic commit.
- A new verb `aiwf edit-body <id> --body-file <path>` for post-creation body edits.
- Resolver-pointer flags on status-transition verbs — `aiwf promote <gap> addressed --by <id|sha>`, `aiwf promote <adr> superseded --superseded-by <ADR-id>`, generalized for future pointer-requiring transitions.
- `aiwf-add` skill text revised to remove the plain-git body-edit carve-out, now that verb routes cover both creation and post-creation cases.

## Out of scope

- `aiwf plan apply <file>` — the declarative whole-plan-in-one-commit verb. Real value but a separate decision; the in-scope items get a planning session from ~42 commits to ~8, which clears the user-stated friction threshold.
- Nested batching (e.g., `aiwf add epic` accepting inline `--milestone` definitions). Defer until in-scope work proves itself.
- Changes to the `provenance-untrailered-entity-commit` policy. The policy is correct as-is; once verb routes cover all body-edit cases, it stops firing under normal use and stands as a backstop against accidental hand-edits.

## Implementation order

Bootstrap dependency: G-051, G-052, and G-053 cannot be promoted to `addressed` until **M-059** ships, because closing a gap as `addressed` requires `addressed_by` to be set, and M-059 is the milestone that adds the `--by` flag to do that without hand-editing frontmatter. So:

- **Recommended:** ship **M-059 first**. Cheapest path to unlock the gap-close chain. The other three milestones can then ship in any order.
- **Alternative:** ship **M-056 first** for the largest day-to-day ergonomic win (`--body-file` is the highest-leverage flag), and only do M-059 once you're ready to close the gaps. Acceptable if you don't mind the gaps staying `open` in the meantime.

After M-059 ships, close the three gaps:

```bash
aiwf promote G-051 addressed --by M-056   # or whichever milestone delivered the relevant fix
aiwf promote G-052 addressed --by M-058
aiwf promote G-053 addressed --by M-059
```

M-057 (batched `--ac-title`) and M-058 (`aiwf edit-body` + skill update) have no ordering constraints with the other milestones — pick by whichever is most useful next.

## Note on the meta-irony

This epic plans the fix for friction observed during E-14's planning. By design, creating the epic itself triggers the same friction one more time — accepted as the cost of capturing the design rationale durably before the fix lands. After M-056 ships, this epic's body and ACs become the first thing to use the new `--body-file` flag.

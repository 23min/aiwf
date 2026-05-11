---
id: G-0100
title: aiwfx-wrap-epic emits untrailered merge commits; ritual should produce aiwf-verb/entity/actor trailers on the merge so provenance is self-describing
status: addressed
discovered_in: M-0089
addressed_by_commit:
    - 104b416
---

## What's missing

`aiwfx-wrap-epic`'s current merge step runs `git merge --no-ff <branch>`, producing a merge commit with no `aiwf-verb`, `aiwf-entity`, or `aiwf-actor` trailers. The merge commit touches the epic's entity files (epic.md, milestone specs, wrap.md), which fires the kernel's `provenance-untrailered-entity-commit` rule once per affected file. After the merge of E-0024 and E-0026, this repo's `aiwf check` carries `provenance-untrailered-entity-commit (warning) × 4` from the two wrap-epic merges — and the count will rise by ~4 on every future epic wrap until the ritual emits trailers.

The fix is to **emit trailered merge commits from the ritual**, not to scope the kernel rule. Clean idiom: `git merge --no-ff --no-commit <branch>` followed by `git commit -m "Merge <branch>: <title> (E-NNNN)" --trailer "aiwf-verb: wrap-epic" --trailer "aiwf-entity: E-NNNN" --trailer "aiwf-actor: human/<id>"`. The merge commit becomes self-describing; `aiwf history E-NNNN` surfaces the merge event explicitly.

## Why it matters

This was originally filed from a symptom-side framing — "rule should skip merge commits (parent count > 1)." That route would hide a real signal: a merge commit that touches entity files outside a proper wrap ritual would also pass silently. Per CLAUDE.md "framework correctness must not depend on the LLM's behavior" — the kernel rule is the mechanical chokepoint. Loosening it to ignore an entire class of commits trades long-term defect-detection for short-term warning suppression.

The cause-side fix preserves the chokepoint and aligns the ritual with the rule: every entity-touching commit, including epic-wrap merges, carries its provenance trailers. If a future agent or hand-edit forgets the trailers on a merge, the existing rule fires and the regression is caught at the next push. No new mechanism, no rule scoping; just bring the ritual into compliance.

The historical 4 warnings (from E-0024 and E-0026's already-merged commits) stay as advisory artifacts — no history rewrite per CLAUDE.md's no-backwards-compat-hacks rule. The cause-side fix prevents new instances from this point forward.

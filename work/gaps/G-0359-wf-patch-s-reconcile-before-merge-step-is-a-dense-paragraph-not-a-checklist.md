---
id: G-0359
title: wf-patch's reconcile-before-merge step is a dense paragraph, not a checklist
status: open
---
## Problem

`wf-patch`'s step 9 ("Wrap gate") bundles the reconcile-before-merge requirement into a single ~180-word prose paragraph nested inside its first bullet ("Merge to mainline"), mixing the imperative instruction (what to run, what to check) with rationale and argumentation (why the ordering matters, what failure mode it avoids, an aside about PR-flow projects). It is not its own numbered step, and it is not a checklist.

Concretely, during the wrap of `patch/G-0271-start-ritual-fixes` (closing G-0271 and G-0224), an AI assistant executing this ritual:

- Omitted the reconcile-before-merge check entirely from its first draft of the wrap gate.
- When it added the check, ran `git merge-base --is-ancestor` with the arguments reversed — checking whether the patch branch was an ancestor of mainline instead of the reverse, even though the paragraph names the exact correct command.

The assistant had read and even quoted the paragraph correctly earlier in the same session, then still executed it wrong later — consistent with working from a compressed recollection of a dense paragraph rather than re-reading a short, literal checklist at the point of use.

## Why it matters

This ritual is followed mechanically, often by an AI assistant, precisely at the highest-stakes step (the merge to mainline). A step written as argued prose is easy to skim, compress, and misremember; a step written as a short imperative checklist is not. The paragraph's actual content is correct — the problem is presentation, not substance.

## Fix shape — deferred to Q&A, not prescribed here

Rewrite the reconcile-before-merge guidance as a brief, clearly-separated checklist: what to do, what to check, in order — no rationale, no argumentation, no embedded aside about PR-flow projects (which can stay as its own separate bullet, as it already is one level out). The exact wording, step boundaries, and how much (if any) rationale to retain elsewhere is intentionally left open — draft it via Q&A when this gap is picked up, rather than pre-deciding it here.

## Open question — does the merge mechanism belong in the skill itself?

Step 9 currently defers the merge *mechanism* (fast-forward vs. `--no-ff` merge commit vs. rebase-and-merge) entirely to "the consuming project's `CLAUDE.md` §'Working in this repo' policy." In this repo, that section is silent on the mechanism for patches specifically — it only states trunk-based, commit-directly, no-PR-ceremony. The `--no-ff` merge-commit convention (with a descriptive `Merge patch/<branch>: <summary>` message) had to be reverse-engineered from git-log precedent (prior patches such as G-0293's and G-0355's), not read from a written rule.

Worth discussing when this gap is drafted: should `wf-patch` name a default mechanism (e.g. `--no-ff`, so the patch stays a single identifiable commit in history) with the project override still available, rather than fully punting to a project policy that may not actually exist? Not decided here — flagged for the same Q&A pass as the paragraph rewrite.

## Discovered while

Wrapping `patch/G-0271-start-ritual-fixes` (closing G-0271, G-0224) in the aiwf kernel repo itself.

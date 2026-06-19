---
id: G-0263
title: require independent two-lens review before milestone wrap and patch commit
status: addressed
addressed_by_commit:
    - 0fabc0b5
---
## What's missing

The pre-commit review in the engineering rituals is **self-administered** — the author grades their own work — and is therefore structurally weak and non-persistent:

- `wf-patch` step 5 ("self-review the diff") *is* the `wf-review-code` checklist, run by the author who just wrote the code.
- `wf-rethink`'s own skill admits it: *"it is self-graded ... still partly trusts the agent's own judgment — the very thing it is trying to discipline. Treat that as a known limit, not a solved problem."*

This is the kernel's own anti-pattern turned inward: a guarantee that depends on the LLM reasoning completely about its own work is not a guarantee. Worse, any "discipline" the agent adopts mid-session dies at the session boundary — there is no next-session memory of having been burned.

**Evidence (dogfooded):** during the G-0259 patch, the author's self-review reported "nothing"; a fresh independent reviewer agent, handed an adversarial brief, found three items — one a real blocker (an uncovered branch that would have failed the patch's own coverage gate on push). Self-review *reasoned* about coverage; the independent pass *measured* it.

## The fix

Require an **independent** (fresh-context agent, no authorship attachment) review through **two lenses** before committing milestone/patch work:

- **Code-quality lens** (`wf-review-code`): correctness, completeness, conventions — the defect class self-review misses.
- **Design-quality lens** (`wf-rethink`): is this the right shape at all — run independently, which also repairs wf-rethink's self-graded limit.

Placement (the asymmetry is intentional, because the unit differs):

- **`wf-patch`:** independent review before the **commit gate** (one small diff).
- **Milestone:** independent review before the **wrap** — it gates milestone *closure*, not the per-commit work. The implementation commits are already in; findings become corrective commits *inside* the milestone (no follow-up-gap ceremony), before any AC flips to `met`. Per-commit review is explicitly **not** wanted (too heavy); the mechanical gates (CI, G-0067, `aiwf check`) carry the per-commit mechanical defects, and the wrap review is the judgment gate (design soundness, holistic AC satisfaction, a code-quality sweep over the whole diff).

The independent review is a step that **feeds the human wrap/commit gate**, not a replacement for it.

## Design constraints to honor (these decide whether it works)

1. **Independence is only as strong as the brief — and the author writes the brief.** A lazy "review this diff" from a fresh agent is still shallow. The ritual must *mandate the briefing shape*: enumerate the load-bearing claims, instruct "verify by measuring, not reasoning," name the risk areas. Otherwise we get independence of context without independence of rigor, and it rots back to self-review-with-extra-steps. This is the real lever.
2. **The two lenses scale differently.** Code review scales to the full diff — but for a large milestone, slice it by concern/unit (one agent over thousands of lines goes shallow, the exact failure we are avoiding). `wf-rethink` is **per-unit by its own rule** ("never the whole codebase at once"), so the wrap step needs a sub-step that *names the design unit(s) the milestone introduced* and runs rethink only on those — which is precisely G-0261's trigger-detection problem.
3. **Close the loop.** Findings → corrective commits → mechanical re-verify (re-dispatch a fresh reviewer for judgment-level findings) → the human gate sees the review outcome *and* the fixes. Leaving this open bit the G-0259 patch (the original reviewer could not be re-engaged for fix-confirmation).

## Relationships / scope

- Sibling to **G-0261** ("auto-invoke `wf-rethink` when a unit introduces a non-trivial design") — this widens it to run *independently*. Widening G-0261 is a **separate** change so the first patch stays atomic.
- **G-0260** (make `wf-vacuity` a required ritual step): the independent reviewer naturally folds in a vacuity check (it mutation-tested the new tests during G-0259), so vacuity rides along — probably not a distinct third lens.
- First, atomic patch: the independent two-lens review step in the wrap-milestone ritual + flipping `wf-patch`'s "self-review" to "independent review." Ritual-content edits to the embedded snapshot (no kernel code, no AC ceremony).

## Honest limit

The reviewer is also an LLM and can miss things — independence is the **floor, not a ceiling**. The step stays **advisory** (you cannot mechanize "is this good"); the durable property is that it is **reloaded into every session via the ritual**, so unlike a per-session resolution it does not evaporate. The mechanical chokepoints (CI, hooks, `aiwf check`) remain the authoritative net for mechanical properties.

## Source

Surfaced when an independent review of the G-0259 firing-fixture patch caught a blocker that the author's self-review missed — the proposal is dogfooded by its own origin.

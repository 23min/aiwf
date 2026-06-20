---
id: G-0260
title: make wf-vacuity a required invocation step in the engineering rituals
status: addressed
addressed_by_commit:
    - 3fc831da
---
## What's missing

`wf-vacuity` is advisory and fires only on explicit invocation or agent whim. The G-0259 audit showed the cost: 75% of `internal/policies/` had never been vacuity-checked, because nothing in the engineering rituals requires the check. `wf-vacuity` should be a **required invocation step** in the test-bearing rituals — its *output* stays advisory (probe 2 is LLM-judged and cannot be a hard gate), but its *invocation* becomes non-optional.

## Where it wires in

- **`wf-tdd-cycle`** — immediately after the branch-coverage audit. Coverage proves the line ran; vacuity is the missing sufficiency check (would the assertion catch the bug).
- **`wf-patch`** — at the self-review step, post-green and before the commit gate, on the unit just built.

## Design questions

- **Gate or inform?** Recommendation: invocation mandatory, output surfaced at the commit gate for the human to weigh — not an automatic block. A `mutate-hunt` survivor or a probe-2 finding becomes a commit-gate input, not a hard fail. The hard mechanical floor (firing-fixture meta-chokepoint + a gating `mutate-hunt` step) is tracked in G-0259, not here.
- **Scope:** the unit just built, never the whole tree (per the skill).
- **Defer to the tool:** where `mutate-hunt` covers the unit, probe 1 defers to it; the manual probe is the stop-gap.

## Why it matters

The mechanical floor (G-0259) catches "never fires." The assertion-shape reasoning (probe 2) catches "fires but passes for the wrong reason" — and that can only ever be advisory, so the only lever to make it reliable is to make invocation a non-skippable step. Without the wiring, vacuity-checking depends on the agent remembering — the exact dependency the framework forbids.

## Source

G-0259 (vacuous-policy finding), G-0258 (the `wf-vacuity` ritual). A sibling gap wires `wf-rethink` by the same invocation mechanism at a different trigger (post-green test verification here vs pre-commit design review there); the two are kept separate because the rethink trigger is its own design question.

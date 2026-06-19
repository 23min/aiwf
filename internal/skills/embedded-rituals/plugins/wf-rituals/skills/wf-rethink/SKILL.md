---
name: wf-rethink
description: Re-evaluate one unit's design (a file, module, function, or decision) by reconstructing it from intent against a list of pinned obligations, then keep or rewrite — biased to keep, with any rewrite gated behind explicit human approval. Use when code works but feels over-complex or shaped by its edit history, before committing a non-trivial design, or when the user invokes wf-rethink.
---

# wf-rethink

A design-quality audit for one unit. An LLM coding agent behaves like a greedy optimizer: at each step it takes the first adequate move, so a working unit accretes into a **local optimum** — correct, but carrying incidental complexity, single-caller abstractions, and defensive layers that exist only because of the order in which things were built. This ritual re-derives the unit from intent to separate that path-dependent residue from genuinely load-bearing structure, and keeps whichever design is simpler — but only when it provably preserves what the current code does.

This is a **design-quality ritual, not a correctness gate.** Passing a rethink says nothing about whether the code satisfies a stated specification — that is an orthogonal property a test suite or verifier checks. The ritual's only safety mechanism is the obligation list you pin in step 1; it is self-graded, so its strength is bounded by how completely you can name what the unit owes. On code with no tests and no written invariants it still partly trusts the agent's own judgment — the very thing it is trying to discipline. Treat that as a known limit, not a solved problem.

## Independence — the author cannot reconstruct from intent

This ritual is worth far more run by a **fresh agent that has not seen the implementation** than by the author who wrote the unit — and the reason is sharper than it is for review (`wf-review-code` §"Independence"). Step 2 asks you to reconstruct the design *from intent, deliberately not referencing the current structure* — and the author cannot. They hold the implementation in working memory; their "from-scratch" design paraphrases what they already built (the exact failure the workflow warns against), because the same greedy optimizer that accreted the local optimum is now being asked to imagine away its own residue. Self-rethink reliably returns `keep` — not because the design is sound, but because the reconstruction never escaped it.

Independence fixes the central step. Hand a fresh agent the **obligation list from step 1** (behavior, interface, invariants, tests) and the underlying problem — but *not* the implementation — and let it reconstruct cold; only then show it the current code for the step-3 diff. The author's job narrows to the part they *can* do honestly: pinning a trustworthy obligation list. That division is the point — the author owns the obligations, the fresh agent owns the reconstruction.

But independence does not move the safety mechanism. The rewrite is still gated by the obligation list alone — step 1's 🛑 stands (obligations you cannot pin from tests, types, or written invariants → `keep`), and that list is self-graded no matter who reconstructs. A fresh agent with a weak obligation list is still an ungated rewrite: independence sharpens the *reconstruction*, the obligation gate still governs the *rewrite*. Don't let a confident outsider substitute for the gate.

A calling ritual that invokes `wf-rethink` should dispatch it to a fresh-context subagent — handing over the named unit and the obligations, not running the reconstruction in the author's own head. Resource that agent to the stakes: a design rewrite is expensive to get wrong, so reach for the most capable reasoner the host offers (don't name a model — identifiers age, and consumers run different tiers). A dispatched subagent inherits the orchestrator's model by default, so the session's own capability is already the floor.

## When to use

- The user invokes `wf-rethink`, or names a unit and asks you to reconsider its design.
- Code works but feels over-complex, accreted, or shaped by its edit history rather than by its problem.
- Before committing a **non-trivial design**, while it is still cheap to change — see §"The non-trivial-design trigger" for what qualifies. This is the criterion the engineering rituals auto-invoke on.

Don't reach for it for naming or micro-style (that is review — see `wf-review-code`), and never run it over the whole codebase at once.

## The non-trivial-design trigger

`wf-rethink` is worth running — and the engineering rituals **auto-invoke** it — when a change introduces a **new design surface**, concretely one of:

- a **new module or package boundary**;
- a **core abstraction** — a new type or interface other code is meant to build on;
- a **data model** — the shape of the state a unit owns.

It is **not** warranted when the change is mechanical or local: a bug fix, a config nudge, a dependency bump, a single-call-site tweak, a test-only change, a rename, or prose. Auto-invoking on those is the churn this trigger exists to avoid — the same instinct as the skill's own "never run it over the whole codebase at once." When in doubt, ask *"would I have to explain this design to a reviewer?"* — if there is nothing to explain, there is nothing to rethink.

The calling rituals fire `wf-rethink` on exactly this trigger, each on the **named unit** the change introduced:

- `wf-patch` — at its commit gate (step 5), when the patch introduced one of the surfaces above.
- `aiwfx-wrap-milestone` — at its pre-wrap review (step 2), on the design unit(s) the milestone introduced.

It is deliberately **not** wired per-AC inside `wf-tdd-cycle`: mid-cycle the design is still in flux, so a rethink there is premature and churny. The before-wrap review is the right moment — the design is settled, but still cheap to change.

## Scope

Operate on **one unit** named by the user — a file, module, function, or design decision. If none is given, infer it from the just-completed work and state your choice in one line before proceeding. When the rethink is dispatched independently (§"Independence"), that naming is the author's or calling ritual's act: the fresh agent reconstructs the unit it is handed, it does not pick its own from whatever it happens to recall. The bound is the point: a rethink with no edges is a rewrite-everything licence.

## Workflow

### 1. Pin the obligations — before looking at the current structure

List what the unit must keep true regardless of design:

- **Observable behavior** — what callers depend on.
- **Public interface / signature** — what it exposes.
- **Invariants** — what must always hold.
- **Tests** — the cases it must still pass.

Do this *first*, from intent and from callers — not by reading the current implementation top to bottom. Obligations back-fitted to the code that exists are worthless; they rubber-stamp whatever that code happens to do. Flag any obligation that is currently **unstated** — no test, no type, no written invariant pins it. Those are the riskiest to break and the ones the gate cannot protect.

🛑 **If the obligations cannot be enumerated from tests, types, or written invariants — i.e. you are inferring them from the same code you are about to judge — the correct verdict is `keep`.** Say so, and stop. A rewrite gated by an obligation list you cannot trust is not gated at all.

### 2. Reconstruct from intent

Describe how you would build this from scratch to satisfy the underlying problem, deliberately *not* referencing the current structure while you do it. Work at the level of data model, control flow, and the core abstraction — not naming or micro-style. The point is a genuinely non-local view; a reconstruction that paraphrases the current code has skipped the step. This is the step the unit's author cannot perform honestly (see §"Independence"): when the rethink is dispatched independently, the fresh agent reconstructs from the obligation list alone, with the implementation out of view.

### 3. Diff essential vs. incidental

Compare the from-scratch design to what exists. Name concretely:

- **Incidental** — what the current code carries only because of how it grew: path-dependent state, defensive layers, single-caller abstractions, unused capability, dead config knobs.
- **Essential** — what is genuinely load-bearing and must survive any design.

### 4. Decide, biased to keep

Adopt the from-scratch design only if it is **both** simpler/clearer **and** preserves every obligation from step 1. Otherwise keep the current code. **"No change warranted" is a correct and common outcome — a rethink that changes nothing is a successful audit, not a failure.** The agent's bias is toward action and churn; this step is the counter-pressure.

### 5. 🛑 Report, then gate the rewrite

Emit the report (below) first — the verdict and the concrete win, *not* an applied rewrite. If the verdict is `rewrite`, **stop and wait for explicit human approval before touching the code.** Deciding "rewrite" and rewriting in the same breath collapses the decision and the action into one step; the human owns the decision to replace working code. A `keep` verdict needs no gate — there is nothing to approve.

### 6. After approval — implement and verify against the obligations

Only after the human approves: implement the rewrite, then re-check **each** obligation from step 1 — run the tests, confirm the interface and the invariants still hold. Never declare success from the design alone. If an obligation now fails, the rewrite is wrong, not the obligation — revert or fix; don't relax the obligation.

If the project records design decisions (an ADR, a work log, the `aiwfx-record-decision` ritual), capture a non-trivial rewrite — or a deliberate keep-despite-smell — there, with the incidental complexity you found.

## Output format

```markdown
# Rethink — <unit>

**Verdict:** keep | rewrite (awaiting approval)

## Obligations
- <behavior / interface / invariant / test the unit must preserve>
- <… mark any that are currently unstated>

## From-scratch design
<a few lines — data model, control flow, core abstraction only>

## Delta
- Incidental: <path-dependent structure the current code carries needlessly>
- Essential: <load-bearing parts confirmed in both designs>

## Decision
<keep: why the current design already wins, or why the obligations can't be trusted>
<rewrite: the concrete simplification win and the obligations it preserves —
 awaiting approval before any code changes>
```

## Anti-patterns

- *Judging "cleaner" with the same optimizer that built the mess.* Without the obligation gate, a from-scratch rewrite is just a *differently*-local optimum shipped with a fresh regression. The gate is the whole point.
- *Reading the implementation first, then "deriving" obligations from it.* That back-fits the gate to the code and defeats it.
- *Self-rethink — reconstructing your own unit from memory.* The author holds the implementation in mind, so the "from-scratch" design paraphrases it and the verdict is a false `keep`. Dispatch a fresh agent (see §"Independence").
- *Auto-applying a `rewrite` verdict.* The rewrite is gated; report and wait.
- *Weakening or dropping an obligation to make the rewrite look better.* If an obligation seems wrong, flag it for the human — never silently relax it.
- *Rethinking the whole codebase.* One bounded unit at a time.
- *Treating a `keep` as a wasted pass.* A rethink that confirms the current design is a successful audit.

## Constraints

- 🛑 A `rewrite` verdict is never self-applied. Report the verdict and the win, then wait for explicit human approval before changing code.
- 🛑 Obligations that cannot be pinned from tests, types, or written invariants → `keep`. An ungated rewrite is not a rethink.
- Never weaken, drop, or delete an obligation — a behavior, an invariant, a test — to make a rewrite pass. Flag a wrong-looking obligation; don't relax it.
- Preserve the public interface unless the rethink is explicitly about the interface.
- Default to keep. One unit at a time. Verify every obligation after a rewrite — never declare success from the design alone.

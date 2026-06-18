---
name: wf-rethink
description: Re-evaluate one unit's design (a file, module, function, or decision) by reconstructing it from intent against a list of pinned obligations, then keep or rewrite — biased to keep, with any rewrite gated behind explicit human approval. Use when code works but feels over-complex or shaped by its edit history, before committing a non-trivial design, or when the user invokes wf-rethink.
---

# wf-rethink

A design-quality audit for one unit. An LLM coding agent behaves like a greedy optimizer: at each step it takes the first adequate move, so a working unit accretes into a **local optimum** — correct, but carrying incidental complexity, single-caller abstractions, and defensive layers that exist only because of the order in which things were built. This ritual re-derives the unit from intent to separate that path-dependent residue from genuinely load-bearing structure, and keeps whichever design is simpler — but only when it provably preserves what the current code does.

This is a **design-quality ritual, not a correctness gate.** Passing a rethink says nothing about whether the code satisfies a stated specification — that is an orthogonal property a test suite or verifier checks. The ritual's only safety mechanism is the obligation list you pin in step 1; it is self-graded, so its strength is bounded by how completely you can name what the unit owes. On code with no tests and no written invariants it still partly trusts the agent's own judgment — the very thing it is trying to discipline. Treat that as a known limit, not a solved problem.

## When to use

- The user invokes `wf-rethink`, or names a unit and asks you to reconsider its design.
- Code works but feels over-complex, accreted, or shaped by its edit history rather than by its problem.
- Before committing a non-trivial design — a new module boundary, a core abstraction, a data model — while it is still cheap to change.

Don't reach for it for naming or micro-style (that is review — see `wf-review-code`), and never run it over the whole codebase at once.

## Scope

Operate on **one unit** named by the user — a file, module, function, or design decision. If none is given, infer it from the just-completed work and state your choice in one line before proceeding. The bound is the point: a rethink with no edges is a rewrite-everything licence.

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

Describe how you would build this from scratch to satisfy the underlying problem, deliberately *not* referencing the current structure while you do it. Work at the level of data model, control flow, and the core abstraction — not naming or micro-style. The point is a genuinely non-local view; a reconstruction that paraphrases the current code has skipped the step.

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

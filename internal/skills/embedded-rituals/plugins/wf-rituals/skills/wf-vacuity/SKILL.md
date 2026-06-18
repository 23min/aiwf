---
name: wf-vacuity
description: Adversarially audit whether a unit's existing tests and assertions can actually fail — break the implementation and confirm a test goes red, and flag tautological or over-narrowed assertions. Use after tests exist and before declaring done, when you don't trust an LLM-authored suite, or when the user invokes wf-vacuity.
---

# wf-vacuity

Coverage proves a line *ran*; it does not prove an assertion would *catch a bug* on that line. A fully-covered test suite can still be vacuous — tautological checks, over-narrowed antecedents, properties that can't fail. This ritual audits the assertions a unit already carries by attacking them: break the implementation and see whether anything goes red, and read each assertion for "could this pass for the wrong reason?"

The failure this targets is **endogenous, not capability**: the same agent wrote the implementation *and* its tests and is graded on the tests passing, so the tests are suspect by construction. `wf-vacuity` is the adversarial counter-stance. Its honest limit: the audit is itself LLM-judged — unlike `wf-property-test`, whose artifact is a CI gate, this produces a report a human reads. Where a real mutation-testing tool exists, defer to it; the manual probe is the stop-gap for code and repos without one.

## When to use

- After tests exist (post-green), before declaring a unit done or proposing it for merge.
- You don't trust an LLM-authored suite — especially when the same author wrote the code and the tests.
- Right after `wf-tdd-cycle`'s branch-coverage audit: coverage is *necessary* (the line ran), vacuity is the missing *sufficiency* check (an assertion would catch the bug).
- The user invokes `wf-vacuity`, or asks "do these tests actually test anything?"

It needs assertions to audit — run it on a unit that already has tests, not before they're written (that's `wf-tdd-cycle` / `wf-property-test`).

## Defer to a real tool first

If the project has a mutation-testing harness — the repo's `mutate-hunt` workflow / gremlins, or Stryker / mutmut / PIT — run it and read its survivors. It is the mechanical version of probe 1, and it is the stronger signal. The manual probes below are the stop-gap for the units, languages, or repos where no such tool is wired up, plus the assertion-shape reasoning (probe 2) that mutation tools don't do.

## Workflow

### 1. Mutation probe — can the tests fail at all?

Pick the unit's core logic. Introduce a deliberate, realistic bug and run the tests:

- Negate a guard or boundary (`<` → `<=`, `==` → `!=`).
- Off-by-one a loop or index.
- Return a constant / the input unchanged / nil.
- Drop a step (skip the validation, skip the write).

Confirm at least one test goes red for each mutation. A mutation that leaves the suite green is a **surviving mutant** — a hole where the assertions don't constrain the behaviour. Record it.

Mutate one thing at a time; revert between mutations. The implementation must be back to its original state when you finish.

### 2. Tautology / narrowing probe — do the assertions mean anything?

Read each assertion (and each property) and ask "could this pass for the wrong reason?":

- **Tautological** — asserts existence, not value: `result != nil`, `len(out) >= 0`, `err == err`. Strengthen to assert the actual expected value.
- **Over-narrowed antecedent** — the test exercises only the input where the bug can't appear (the empty list, the happy path, the one hard-coded fixture). Widen the input or add the adversarial case.
- **Can't-fail property** — `∀x. true` in disguise: a property whose body holds for any implementation. (This is probe 1 applied to a property test.)
- **Asserting on a mock** — the test verifies the mock was called, not that the real behaviour is correct.

### 3. Report and route

Emit the report (below). Each weak assertion is paired with the mutant that survives it or the reason it's tautological. Route findings to the human, or back into `wf-tdd-cycle` / `wf-property-test` to strengthen — `wf-vacuity` does not rewrite the tests itself.

## Output format

```markdown
# Vacuity audit — <unit>

## Surviving mutants (tests stayed green under a real bug)
- `ledger.go:42` — negated the `if balance < 0` guard; no test went red.
  Missing: a negative-balance rejection test.

## Weak assertions
- `foo_test.go:18` — asserts `result != nil`; passes for any non-nil value.
  Strengthen to the expected value.
- `parse_test.go:30` — only exercises empty input; the bug lives in the
  multi-token path.

## Clean
- <assertions / properties confirmed to constrain behaviour — a real bug
  makes them fail>

## Summary
- <N surviving mutants, M weak assertions> — <one-line takeaway>
```

If nothing is weak, the report says so: every mutation was caught and every assertion constrains a value.

## Anti-patterns

- *Leaving a mutation in place.* Every deliberate bug is reverted; the implementation ends exactly as it started. A vacuity audit that ships a mutant is worse than none.
- *Treating coverage as the answer.* "100% covered" says the lines ran, not that a bug would be caught. Coverage and vacuity are different axes.
- *Rewriting the tests inside the audit.* `wf-vacuity` reports; strengthening happens in `wf-tdd-cycle` / `wf-property-test` as its own reviewed change.
- *Auditing tests that don't exist yet.* This is a checking ritual — it needs assertions to attack.
- *Trusting the probe as proof.* The manual audit is LLM-judged; a clean report lowers risk, it doesn't certify correctness. Where a mutation tool exists, it is the stronger signal.

## Constraints

- 🛑 Revert every mutation. The implementation is byte-identical before and after the audit.
- 🛑 `wf-vacuity` reports; it never rewrites the unit's tests. Strengthening is a separate, reviewed change.
- Where a real mutation-testing tool is wired up, run it and read survivors — the manual probe is the stop-gap, not the preferred path.
- The audit is LLM-judged; size the claim to that. A clean report is "no weakness found," not "tests verified correct."

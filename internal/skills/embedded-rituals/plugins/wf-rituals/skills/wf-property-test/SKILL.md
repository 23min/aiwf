---
name: wf-property-test
description: Turn a unit's crisp invariant into a generative property or metamorphic test — one that fabricates many inputs and asserts the property across all of them, not a single example. Use while implementing parsers, ledgers, state machines, allocators, or serializers, when hardening invariant-bearing code, or when the user invokes wf-property-test.
---

# wf-property-test

An LLM agent reaches for example-based tests — one input, one expected output — which check the case the author already had in mind and miss the ones they didn't. For the subset of code with a *crisp invariant* — a property that must hold for **every** valid input — a far stronger test fabricates many inputs and asserts the property across all of them. This ritual turns an English invariant into that generative test.

Unlike the other verification rituals (`wf-rethink` for design quality, `wf-vacuity` for assertion strength), the artifact this produces is a **real mechanical gate**: the property test runs in CI and fails loudly, so once written it does not depend on anyone's judgment. The honest limit is that it **samples** the input space — it does not **prove** the invariant for all inputs; that exhaustive guarantee is what a verifier provides. Sampled-and-mechanical still beats unsampled-and-hand-judged; size every claim to that.

## When to use

- While implementing (test-first) a unit with a crisp invariant — a parser, ledger, state machine, allocator, serializer, comparator, id allocator.
- Hardening existing invariant-bearing code that currently has only example tests.
- The user invokes `wf-property-test`, or asks for property / fuzz / metamorphic tests.

Don't reach for it on glue, IO, or UI — code with no sharp invariant. Forcing a property there is high-cost, low-yield (the classic formal-methods objection). Rule of thumb: if you can't state the invariant in one sentence, it isn't crisp enough — use `wf-tdd-cycle` instead.

## Scope: does this unit have a crisp invariant?

Decide before writing anything. A crisp invariant holds for *every* valid input and can be stated in one sentence without enumerating cases. If yours can't, stop — example-based `wf-tdd-cycle` is the right tool, not this one.

## The property families

Most useful invariants fall into a handful of shapes. Pick the one(s) that fit:

- **Conservation** — a quantity is preserved. `sum(balances before) == sum(balances after)`; element count preserved across a reshuffle.
- **Round-trip** — `decode(encode(x)) == x`; `parse(render(x)) == x`.
- **Idempotence** — `f(f(x)) == f(x)`: slugify, normalize, dedupe, a convergent migration.
- **Order-independence / commutativity** — the result doesn't depend on input order.
- **Monotonicity** — adding input never decreases (or never increases) the output.
- **Invariant-preservation** — a state machine never leaves the set of valid states; an allocator never double-allocates or exceeds capacity.
- **Metamorphic / oracle** — relate two runs when you have no full oracle: `sort(x)` and `sort(shuffle(x))` agree; an optimized path agrees with a slow reference path.

## Workflow

### 1. Pin the invariant in one sentence

State the property in English first, in one of the families above. If it takes a paragraph or a list of special cases, it isn't crisp — reconsider scope before writing code.

### 2. Choose the generator

Use the language's property / fuzz framework — never hand-roll one. Go: `testing/quick` or native `Fuzz*` (this repo's convention; seeds under `testdata/fuzz/`). Python: Hypothesis. JS/TS: fast-check. Rust: proptest. Haskell/Scala: QuickCheck / ScalaCheck.

### 3. Write the property as a test

Generate inputs, exercise the unit, assert the invariant. Keep the generator **broad** — the value is in the inputs you didn't think of. Constrain it only enough to stay valid (a parser property feeds arbitrary bytes; a ledger property feeds well-formed transfers).

### 4. Run it green, then make it fail on purpose

Run it green. Then briefly break the implementation — negate a guard, drop a step — and confirm the property goes red with a shrunk counterexample. A property that stays green under a real bug is testing nothing; this is a vacuity check applied to the property itself (the dedicated audit is `wf-vacuity`).

### 5. Commit the test and any seed corpus

The test is the deliverable and the gate. Commit discovered counterexamples as regression seeds (Go: `testdata/fuzz/Fuzz<Name>/`) so the failure is pinned and can't silently return.

## Examples

An idempotence property, Go shape:

```go
// idempotence: slugifying a slug is a no-op
func FuzzSlugify_Idempotent(f *testing.F) {
    f.Add("Hello World")
    f.Fuzz(func(t *testing.T, s string) {
        if once := Slugify(s); once != Slugify(once) {
            t.Errorf("not idempotent: %q -> %q -> %q", s, once, Slugify(once))
        }
    })
}
```

A conservation property, any language:

```
// a transfer neither creates nor destroys value, and leaves no negative balance
for any ledger L and any transfer T valid in L:
    sum(balances(apply(T, L))) == sum(balances(L))
    and every balance in apply(T, L) is >= 0
```

## Anti-patterns

- *Forcing a property onto glue / IO / UI.* No crisp invariant → no property test. Don't manufacture one to look thorough.
- *A generator so constrained it only emits the happy path.* The unthought-of inputs are the whole point; over-constraining throws the value away.
- *A property that can't fail.* If breaking the implementation leaves it green, it asserts nothing — `∀x. true` in disguise.
- *Claiming proof.* A passing property test samples; it does not prove the invariant for all inputs. Say "checked over generated inputs," not "verified."
- *Replacing example tests entirely.* Properties and examples are complementary — a named regression example still documents intent. Keep both.

## Constraints

- 🛑 Only for units with a one-sentence crisp invariant. If you can't state it, use `wf-tdd-cycle` — don't manufacture a property.
- 🛑 Confirm the property can fail (break the implementation, watch it go red) before declaring done. A property that survives a real bug is vacuous.
- Use the language's property / fuzz framework; never hand-roll a generator.
- Commit discovered counterexamples as regression seeds.
- A property test samples, it does not prove. Size every claim to that.

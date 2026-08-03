# Oracles — what decides whether the work is right

**Status:** living document, Normative tier. An oracle inventory drifts as
chokepoints are added and retired; update it when one lands or goes away.

Companion reading: [`growth.md`](growth.md) (what the apparatus costs — this doc
is the other half of that question: what it *buys*), and
[`design-decisions.md`](design-decisions.md) (what the kernel commits to).

---

## What an oracle is

An **oracle** is a procedure that, given a system's behavior on an input,
decides whether that behavior is acceptable. The term comes from testing theory,
where the *oracle problem* names the asymmetry that generating inputs is usually
easy while deciding correctness is usually hard.

For an AI-assisted workflow the framing is sharper. **An assistant without an
oracle is generating; an assistant with one is searching.** The ceiling on an
agent's output is set by the oracle available to it, not by the model — so the
design question for any automated loop is not "how capable is the agent" but
"what tells it that it is wrong, how fast, and how specifically".

## The properties that matter

**Independence.** The oracle must not derive from what it judges. An expectation
computed by the code under test, or written by the same author from the same
misreading, is a mirror rather than an oracle. This is the property AI workflows
lose first: a model that writes both the implementation and its test from one
flawed reading of the spec produces a green suite that establishes nothing.

**A verdict, not a vibe.** Pass or fail, or a score against a threshold fixed in
advance. "Looks right" cannot close a loop, because an agent will rationalize
either way.

**Cost proportional to the frequency you need.** Cost decides frequency and
frequency decides whether the oracle is a gradient or a ceremony. One that runs
at every save shapes the work; one that runs at the end reports on it.

**Diagnostic locality.** It must say where and why. A bare red is one bit, so the
next attempt is close to a random restart; a failure naming the location and the
violated expectation makes the next attempt informed. Feedback *bandwidth*, not
merely feedback presence, is what produces convergence.

**Stability under irrelevant change.** An oracle that fires on behavior-preserving
refactors becomes noise, and noise gets suppressed. A suppressed oracle is worse
than an absent one, because it is still believed.

**A deliberately chosen failure asymmetry.** Two error modes, unequal:
*false accept* passes bad behavior and manufactures confidence; *false reject*
fails good behavior and drives suppression. No oracle minimizes both. Decide
which way each one leans, and write the choice down.

## How an oracle is obtained

Four classes, cheapest first. The working rule is to **use the cheapest class
that can catch the defect class** — a human judging what a type could decide is
the common waste, and a mechanical check standing in for a judgment is the
common self-deception.

**Implicit.** No specification required; detects universally-wrong behavior.
Crashes, hangs, panics, data races, sanitizer trips, type errors. Free, broad,
shallow. Turn these on before anything else — they cost nothing per subject.

**Specified.** The property is stated and a machine checks it: assertions, types,
schemas, contracts, state machines, invariants. Strongest independence, since the
specification exists apart from the implementation. Limited by what can be
articulated.

**Derived.** The expectation comes from another artifact.
- *Regression / golden*: the previous version's output. Cheap, catches drift,
  silent on whether the original was right.
- *Differential*: a second independent implementation. Very strong, since two
  implementations rarely share a defect.
- *Metamorphic*: the correct output is unknown, but a relation between outputs is
  known — `f(x)` against `f(transform(x))`. The escape hatch when correctness is
  unknowable, and the most underused technique in the set.

**Human.** A person judges. Expensive, non-repeatable, irreplaceable for design
soundness, prose coherence, and whether the problem is the right one.

A fifth class has become practical: a **model-judged** oracle, where an assistant
evaluates output. Independence is weak by default — shared training, shared blind
spots — so it must be enforced structurally: separate context, adversarial
framing that asks the judge to refute rather than confirm, and where the stakes
justify it, agreement across several distinct lenses rather than repetition of
one.

## Where an oracle sits

Position matters more than sophistication. Each rung is roughly an order of
magnitude more expensive to act on than the one above it:

| position | latency |
|---|---|
| type checker, editor diagnostics | milliseconds |
| unit test | seconds |
| pre-commit | seconds |
| pre-push | ~a minute |
| CI | minutes |
| review | hours |
| production | days |

**Fire as early as the class allows.** An oracle that only runs in CI is a
backstop, not a gradient: by the time it speaks, the context that would have made
the fix cheap is gone.

## The meta-problem

An oracle that cannot fail is worse than none. This is *vacuity*, and it is
endemic wherever tests are generated from the same understanding as the code,
since such assertions tend to restate the implementation. The only reliable
answer is mutation: break the code deliberately and confirm the oracle goes red.
An oracle nothing has ever seen fail is decorative until proven otherwise.

## Background — where the idea comes from

The name is borrowed from the oracle at Delphi, and the borrowing carries an
irony worth noticing: a classical oracle answers authoritatively and explains
nothing. That is precisely the property a *test* oracle must not have. An
authoritative "no" with no account of why is the low-bandwidth failure the
diagnostic-locality property above exists to rule out — so the metaphor names
the verdict and inverts the requirement.

The concept was formalized in testing theory during the 1970s; **W. E. Howden's**
work on the theory and empirics of program testing (IEEE *Transactions on
Software Engineering*, 1978) is among the early treatments that separate
generating a test input from deciding whether its result is correct.

The paper that named the hard case is **Elaine Weyuker's "On Testing
Non-Testable Programs"** (*The Computer Journal*, 1982). A program is
*non-testable*, in her sense, when no oracle is available for it — either
because the program was written precisely to compute something nobody already
knows, or because a human simply cannot tell a correct output from a plausible
wrong one. Her point was that such programs are not rare edge cases; they are
ordinary, and testing theory had been quietly assuming them away.

That framing transfers to AI-assisted work almost without modification. **You ask
a model to do something because doing it yourself is expensive — which is the
same reason verifying it is expensive.** Weyuker's non-testable program and the
prompt whose output you cannot readily check are the same problem, forty years
apart, and the responses that worked then are the ones that work now: find an
independent source of truth, or find a relation between outputs that must hold
even when no single output is known.

Three of those responses became the practical toolkit:

- **Executable specification.** Bertrand Meyer's *Design by Contract* (Eiffel,
  from the mid-1980s) made preconditions, postconditions, and invariants part of
  the program text, so the specification and the oracle became the same artifact.
- **Differential testing.** William McKeeman's account (*Digital Technical
  Journal*, 1998) formalized running two independent implementations against the
  same input and treating disagreement as the verdict — no specification needed,
  and very hard to fool, since independent implementations rarely share a defect.
- **Metamorphic testing.** T. Y. Chen and colleagues (technical report, HKUST,
  1998) supplied the escape hatch for Weyuker's hard case: you may not know what
  `f(x)` should be, but you often know how `f(x)` and `f(transform(x))` must
  relate. It remains the most underused technique in the set.

Property-based testing — **Claessen and Hughes's QuickCheck** (2000) — is what
made specified oracles practical at scale, by generating the inputs and letting
the property, rather than a hand-written expectation, deliver the verdict. It is
the direct ancestor of this repo's property tests and fuzz targets.

The taxonomy this document uses — implicit, specified, derived, human — follows
the consolidation in **Barr, Harman, McMinn, Shahbaz and Yoo, "The Oracle Problem
in Software Testing: A Survey"** (IEEE *Transactions on Software Engineering*,
2015), which is the standard reference and the place to start for anything
beyond this summary.

---

## aiwf's oracles

| oracle | class | judges | fires at |
|---|---|---|---|
| `aiwf check` finding rules | specified | planning-tree consistency | pre-commit (shape), pre-push (full) |
| the six kind FSMs | specified | transition legality | verb time |
| `internal/policies/` chokepoints | specified | repo structural invariants | CI, `make ci` |
| stress-catalog scenarios | specified | workflow legality, concurrency safety | on demand |
| golden files under `testdata/` | derived (regression) | render and format drift | test |
| property tests, fuzz targets | specified, metamorphic | FSM invariants, parser totality | test, fuzz workflow |
| race detector, `go vet`, panics | implicit | universally-wrong behavior | every race-enabled run |
| `aiwf contract verify` | specified, user-declared | schema against fixtures | on demand |
| `aiwf doctor --self-check` | derived (end-to-end) | the built binary behaves | `make ci` |
| dispatched reviewer subagent | model-judged | correctness, design, prose | milestone wrap, patch |
| the human gate | human | whether this is the right thing | every mutation |

### Meta-oracles

Chokepoints that judge whether the oracles themselves are real. Most codebases
have none; the presence of these is why the ones above can be trusted.

| meta-oracle | judges |
|---|---|
| diff-scoped coverage gate | whether an oracle exists for each changed line |
| firing-fixture presence | whether each policy has a test that makes it fire |
| `wf-vacuity`, `mutate-hunt` | whether an oracle goes red when the code breaks |
| contract `invalid/` fixtures | whether the schema actually rejects |

The contract design is the sharpest instance. Requiring every fixture under
`invalid/` to *fail* is what distinguishes a strict schema from a permissive one
that merely looks strict; a validation setup without that half is silently
vacuous.

### Two structural properties

**Independence is architectural, not conventional.** The kernel loads
inconsistent state and reports on it separately — validation is a distinct axis
from loading — so the checker does not share a code path with the thing it
judges.

**The strongest oracles here are on the structural axis.** The planning tree's
shape, the legality of transitions, and the repo's own invariants are well
covered. What decides whether a *specification* is any good — whether an
acceptance criterion states an observable condition, whether a gap is
actionable — has no mechanical oracle, and the presence-shaped rules that exist
(a required section is non-empty; frontmatter matches body headings) should not
be mistaken for one.

## Known gaps

Tracked as entities; this list points at them rather than restating their
content, so the entity stays the single source.

- The duplication detector is enabled on production code and excluded across the
  test corpus. G-0473 covers the production-file exclusion catalogue, including
  entries that no longer correspond to any clone; the test-corpus exclusion,
  which is the larger lever, is not covered by it.
- Spec adequacy at authoring time has no oracle. The rule requiring mechanical
  evidence before an acceptance criterion is promoted is real, but it is
  evaluated after implementation rather than while the criterion is written.
- Whether a unit of work closes more than it opens is measured only advisorily,
  by [`growth.md`](growth.md)'s reporting script, and produces no verdict at any
  review point.

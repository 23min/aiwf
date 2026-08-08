---
id: M-0300
title: Assert read-path agreement and ref-less verdict stability
status: in_progress
parent: E-0080
tdd: required
acs:
    - id: AC-1
      title: No two read paths contradict each other on the same bytes
      status: open
      tdd_phase: red
    - id: AC-2
      title: A verdict does not depend on refs a fresh clone would not have
      status: open
      tdd_phase: red
    - id: AC-3
      title: Each property is demonstrated failing, real states before stand-ins
      status: open
      tdd_phase: red
---
## Goal

Give the stress harness an invariant oracle — properties that must hold in every
reachable state — starting with the two that judge a single tree: every read
path agrees on a verdict, and a verdict survives the absence of refs the tree
does not need.

## Context

The `verb-sequence` walker in `cmd/stresstest` composes real multi-step verb
chains and re-checks after every step, which is the right shape. Its oracle
asserts only that `aiwf check` never regresses, and monotonicity cannot catch a
finding carrying the wrong severity for its state.

D-0063 settles the direction: judge with properties that hold in every reachable
state rather than with an expected end state the harness computes. This
milestone builds that oracle seam and the two properties that need no branch
machinery. It deliberately does not widen what a sequence may mutate — D-0063 is
explicit that a wider state space under a monotonic oracle buys reachability
without judgment, so the oracle comes first.

Two fixes landed after D-0063 was accepted and both shape what agreement means
here. G-0558 gave a ref-less surface a way to say it did not build the
cross-branch tier, via the `unresolved-unverified` subcode at warning severity.
G-0556 split a cross-branch hit by whether its branch is published, so the full
check reports `cross-branch-local-only` at error severity where it previously
reported a warning. A property demanding byte-identical finding sets across
surfaces would therefore fire on a correct tree.

## Approach

Add an invariant-oracle seam to the walker: after every step of a sequence,
evaluate a set of registered properties against the repository that step
produced. Two properties land here. Read-path agreement runs each surface that
renders a verdict over the same bytes and looks for contradiction between them.
Ref-less stability recomputes the verdict on a copy stripped of refs the tree
does not need, and compares.

Neither property knows the correct verdict. Both compare two observations of the
same bytes, which is what makes them cost nothing per axis once the mutation
space widens.

## Acceptance criteria

Each criterion below is asserted by the harness itself, not by review.

### AC-1 — No two read paths contradict each other on the same bytes

After each step of a composed sequence, the harness runs every verdict-rendering
read path over the repository that step produced and fails when two of them
**contradict** each other about the same subject. The failure names both
surfaces and the subject they disagree on.

Agreement is non-contradiction, not identity of finding sets. A surface
reporting `unresolved-unverified` is declining to judge — it is saying it did
not build the tier that would settle the question — and a declined judgment
contradicts nothing. Two surfaces contradict when both make a substantive claim
about one subject and those claims cannot both hold: one blocks while the other
is clean, or they classify the same subject differently.

The distinction is load-bearing rather than pedantic. On today's main, a
cross-branch reference draws `cross-branch-local-only` at error severity from
the full check and `unresolved-unverified` at warning severity from a ref-less
one. That is correct behavior, and an identity-shaped property would report it
as a defect on every run.

Silence carries information in one direction only. A cheaper surface may be
silent because it never ran the rule, so the gate blocking where a cheaper
surface says nothing is not a contradiction. The reverse is: a cheaper surface
that blocks where the authoritative gate does not is claiming a push should fail
that the gate would let through, which is the shape G-0558 recorded, and it must
never be excused as a rule-set difference. The asymmetry is a fact about which
surface is authoritative, not a model of what any rule does.

A subject is finer than one rule on one entity. A rule fires once per offending
reference or prose token, and two surfaces can agree about one token while
disagreeing about another under the same code. The comparison carries whatever
the envelope offers to locate a finding inside an entity, so that "saw one more
token" stays distinguishable from "classified a token we both saw differently".

The comparison stays between observations. The harness never states which
surface is right, so the property holds no model of the tier rules and needs no
update when they change — which is what let it survive G-0556's classification
change arriving between this milestone's planning and its implementation.


### AC-2 — A verdict does not depend on refs a fresh clone would not have

For a repository produced by a sequence, the harness computes the verdict twice
— once on the working checkout, once on a copy stripped of every ref a fresh
clone would not receive — and requires the two runs to agree on whether each
subject blocks.

The defect this catches is the one G-0556 records: a reference resolving only
against refs the author's machine holds passes locally and fails in every clone.
What makes a ref removable is therefore clone visibility, not whether some tier
consults it. A published branch survives the strip through its remote-tracking
ref, so the verdict is unchanged; an unpublished one does not, and a verdict
that turns on it is reporting on one machine's ref graph.

Clone visibility is a git-structural fact, which is what keeps this property free
of the tier rules it must hold no model of. A rule naming the refs some tier
consults would *be* that model, and would go stale exactly as those rules evolve.

Agreement is disposition, not identity. Stripping an unpublished branch that
carries a cited id legitimately moves the classification from
`cross-branch-local-only` to `unresolved`, and both block. The violation is a
subject that blocks in one run and not the other — **in either direction**,
including the one where the working checkout is silent and the stripped copy
blocks, which is precisely "passes locally, fails in every clone" and is the
case the property exists for.

The property has no meaning in a repository with no remote, because "what a
clone would receive" is undefined there. It declines to judge rather than
treating every local branch as removable.

The stripped copy is a copy. A repository whose `.git` is a file rather than a
directory names an admin directory elsewhere, so a filesystem copy of it aliases
the original and writes through to it; the property refuses such a repository
rather than operating on it.

### AC-3 — Each property is demonstrated failing, real states before stand-ins

For each property, a test drives it into reporting a violation, and prefers a
repository state the real surfaces produce over one a stand-in fabricates. A
stand-in is admissible only where no real state reaches the property, and the
test names which case it is.

A property observed only passing cannot be told apart from one that cannot fail.
But a fabricated failure is weaker evidence than it looks: it proves the
comparison core's failing branch is reachable, not that the property has purchase
on the kernel. A fixture calibrated on one variant of a disagreement passes while
the property stays blind to another, and the blindness is invisible precisely
because building the fake replaced the search for a real state.

So the search comes first. Where a real repository makes a real surface violate
the property, that state is the evidence and a stand-in is not admissible in its
place. Where the kernel is correct on every reachable state, a stand-in is what
is left — and the test records that as a finding about the kernel rather than
implying the state space was never examined.

## Constraints

- **The oracle stays invariant-shaped.** No property computes an expected
  verdict, and none embeds the reference-resolution tier rules of ADR-0030 or
  ADR-0041. A second implementation of the kernel inside the harness would drift
  against the first and be wrong in a way no test catches, because it *is* the
  test.
- **No property measures the runner** — not how many actors get through, not how
  fast (G-0468).
- **Agreement is non-contradiction.** A property demanding identical finding
  sets across surfaces would fire on correct trees, because a ref-less surface
  legitimately declines to judge where a full one classifies. Identity is the
  shape to refuse at review, on the same footing as an exact-verdict oracle.
- **A green run is not evidence, and a fabricated failure is weaker evidence
  than it looks.** A property that has only ever passed cannot be told apart
  from one that cannot fail; a property that has only ever failed against a
  stand-in has been shown to have a reachable failing branch, which is not the
  same as having purchase on the kernel. AC-3 carries the standard.

## Design notes

- D-0063 is the accepted direction. Its Decision names these properties; its
  Reasoning explains why an invariant oracle costs nothing per new axis while an
  exact-verdict oracle needs a model of the right answer.
- The surfaces read-path agreement compares are the ones G-0558 measured as
  disagreeing: `aiwf check`, `aiwf check --fast`, `aiwf check --shape-only`, and
  `aiwf status`. `aiwf show`, `aiwf render`, and `aiwf doctor` also render a
  verdict off the same rule pass and are **not** observed here. Each renders
  through a report shape of its own, so each costs a decoder; the milestone buys
  the three that share one envelope plus the one whose blocking count is
  comparable, and the exclusion is named rather than left to read as coverage.
- `aiwf status` states a blocking count and warning rows carrying neither
  subcode nor severity, so it is compared on the count. That comparison is sound
  in one direction only, and its soundness rests on a conjunction the harness
  does not own: status runs `check.Run` alone, every gate-side config pass
  escalates rather than downgrades, and every tier-dependent classification is
  enumerated in the kernel's own downgrade switch. A gate-side suppression knob
  would break it.
- The subcodes that make a declined judgment legible are
  `refs-resolve/unresolved-unverified` and `body-prose-id/unresolved-unverified`
  (G-0558), both at warning severity. The property treats them as
  non-substantive; every other classification is substantive.

## Surfaces touched

- `cmd/stresstest` — the walker and the new oracle seam
- `internal/stresstest` — scenario drivers

## Out of scope

- Widening what a sequence may mutate.
- The branch-independence property. It is unreachable until a sequence can cross
  a branch boundary, so it ships with the scenario that makes it reachable
  rather than as a test no generated condition ever exercises.
- Repairing any disagreement these properties detect. G-0558 and G-0556 fixed
  the two known instances; a new one is filed, not fixed here.

## Dependencies

- D-0063, accepted.
- No milestone dependency. G-0558 and G-0556 are both `addressed` on main, so
  the properties are written against a tree where the known instances are
  already repaired — which is why AC-3 carries the whole non-vacuity burden.

## References

- D-0063 — widen the stress walker; keep its oracle invariant-shaped.
- G-0558 — the measured read-path disagreement, and the
  `unresolved-unverified` subcode that makes a declined judgment legible.
- G-0556 — the published-versus-local split that makes the full check's
  severity differ from a ref-less surface's on the same subject.
- G-0468 — the precedent for holding an oracle to a shape rather than a subject.
- G-0121, E-0062 — the parent gap and the epic that built the walker.

## Work log

The first cycle's three implementation commits are below; an independent review
refuted the properties they delivered, and the second cycle's entries follow as
it lands. `aiwf history M-0300/AC-<N>` carries the ladder for both, including
the forced reverts and their reasons.

### Cycle 1 — refuted at wrap review

- AC-1 · commit cbf75aedb — invariant seam plus the read-path agreement
  property, with the pre-existing list-vs-ground-truth assertion routed through
  the same seam.
- AC-2 · commit 5dee2e38d — ref-less stability registered on the seam.
- AC-3 · commit 13030a678 — registry-driven vacuity chokepoint.

## Decisions made during implementation

- **Each surface is compared at the granularity it speaks in.** `aiwf check`,
  `--fast`, and `--shape-only` emit the findings envelope, so they are compared
  claim for claim. `aiwf status` renders the same in-memory rule pass into a
  report carrying a blocking count and warning rows with neither subcode nor
  severity; it is compared on the count alone. Inferring a subcode for it would
  be the harness deciding what a surface meant rather than reading what it said,
  which is the same failure as modelling the tier rules, one layer down.

## Validation

Pending the second cycle. The first cycle's gates — full suite, lint, and the
diff-scoped coverage audit — all reported green against properties an
independent review then refuted, which is the record worth keeping: every
mechanical gate this repo runs passed over both defects, because none of them
asks whether a property can detect what it claims to.

## Deferrals

- (none)

## Reviewer notes

- (none)

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
      title: A verdict is stable under refs the tree does not need
      status: open
      tdd_phase: red
    - id: AC-3
      title: Each property is demonstrated failing against a constructed violation
      status: open
      tdd_phase: done
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

The comparison stays between observations. The harness never states which
surface is right, so the property holds no model of the tier rules and needs no
update when they change — which is what let it survive G-0556's classification
change arriving between this milestone's planning and its implementation.


### AC-2 — A verdict is stable under refs the tree does not need

For a repository produced by a sequence, the harness computes the verdict twice
— once on the working checkout, once on a copy from which the refs the tree does
not need have been removed — and requires the two to agree.

A verdict that changes when an unrelated ref disappears is reporting on the
repository's ref graph rather than on the tree, which is the shape G-0556
records: a reference resolving against refs only the author's machine holds
passes locally and fails in every clone.

Which refs a tree does not need is decided by what the verdict claims, not by a
fixed list, so the property does not go stale as the tier rules evolve.

### AC-3 — Each property is demonstrated failing against a constructed violation

For each property, a test constructs a repository state that violates it and
asserts the property reports the violation, naming the observations that
diverged.

A property observed only passing is not evidence that it can fail. E-0080's
success criteria require the failing direction explicitly, and this is where it
is discharged — at plan time, rather than as a review finding once the
implementation is already written.

The constructed violation is the only non-vacuity evidence available. The
defects that motivated these properties are fixed on main — G-0558 and G-0556
both landed — so neither property fails against the tree as it stands, and a
green run proves nothing on its own.

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
- **A green run is not evidence.** Both properties pass against main as it
  stands, so AC-3's constructed violation is what proves either can fail.

## Design notes

- D-0063 is the accepted direction. Its Decision names these properties; its
  Reasoning explains why an invariant oracle costs nothing per new axis while an
  exact-verdict oracle needs a model of the right answer.
- The surfaces read-path agreement compares are the ones G-0558 measured as
  disagreeing: `aiwf check`, `aiwf check --fast`, `aiwf check --shape-only`, and
  `aiwf status`.
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

### AC-1 — No two read paths contradict each other on the same bytes

Invariant seam added and the read-path agreement property registered on it;
the pre-existing list-vs-ground-truth assertion routed through the same seam
rather than staying an inline call · commit cbf75aedb · tests 24/24

### AC-2 — A verdict is stable under refs the tree does not need

Property registered on the seam, short-circuiting when the repository holds
no ref it does not need · commit 5dee2e38d · tests 22/22

### AC-3 — Each property is demonstrated failing against a constructed violation

Registry-driven chokepoint: a property registered with no state that makes it
report fails by name · commit 13030a678 · tests 1/1

## Decisions made during implementation

- **Each surface is compared at the granularity it speaks in.** `aiwf check`,
  `--fast`, and `--shape-only` emit the findings envelope, so they are compared
  claim for claim. `aiwf status` renders the same in-memory rule pass into a
  report carrying a blocking count and warning rows with neither subcode nor
  severity; it is compared on the count alone. Inferring a subcode for it would
  be the harness deciding what a surface meant rather than reading what it said,
  which is the same failure as modelling the tier rules, one layer down.
- **A subject carries a set of classifications, compared by containment.** A
  rule fires once per offending reference or prose token, so one subject can
  hold several. Containment rather than equality is what lets a surface that
  classified more of the same subject agree with one that classified less —
  the absence rule at per-classification granularity.
- **"Refs the tree does not need" is read as "not derived from the tier
  rules", not as "computed from the verdict's own claims".** Three refs are
  kept: the branch HEAD is on, its upstream (which defines the provenance audit
  range), and the configured trunk ref (which the uniqueness check reads). The
  literal alternative — treat a ref as needed when the verdict cites an id it
  carries — defeats the property, because the ref carrying a cited id is
  precisely the one whose removal the G-0556 shape has to survive. The set here
  is derived from git structure and one config knob, so it does not go stale as
  ADR-0030 or ADR-0041 evolve, which is what the criterion asks for.

## Validation

- `go test ./... -count=1` green; `internal/stresstest` carries 242 passing
  tests, 47 of them added here.
- `make lint` reports 0 issues.
- Diff-scoped branch-coverage audit green against `main`
  (`AIWF_COVERAGE_BASE=main make coverage-gate`).
- Both agreement properties measured against a real repository that produces
  the G-0556 shape — a reference to an id carried only by an unpublished local
  branch, with a remote configured so the published-versus-local split has
  something to read. The full check reports `body-prose-id` /
  `cross-branch-local-only` at error severity; the ref-less surface reports
  `unresolved-unverified` at warning; stripping the branch moves the full
  check's own verdict to `unresolved`, still at error. Neither property reports
  anything, which is correct, and an identity-shaped one would report a defect
  on every run.
- Per-push cost of the two new properties, measured on this machine over the
  walker's own tests: 28.3 / 32.1 / 38.9 s with them registered against
  20.4 / 21.0 / 26.8 s without. Read-path agreement accounts for essentially
  all of it, at four extra subprocesses per walk step; ref-less stability
  short-circuits on a single-branch repository and costs one `git
  for-each-ref`. It starts costing more when a scenario crosses a branch
  boundary, which is the point at which it has something to judge.

## Deferrals

- (none)

## Reviewer notes

- (none)

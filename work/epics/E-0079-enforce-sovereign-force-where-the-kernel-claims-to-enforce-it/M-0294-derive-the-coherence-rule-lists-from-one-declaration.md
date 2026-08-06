---
id: M-0294
title: Derive the coherence rule lists from one declaration
status: done
parent: E-0079
depends_on:
    - M-0291
tdd: required
acs:
    - id: AC-1
      title: Three hand-maintained rule lists derive from one declaration
      status: met
      tdd_phase: done
    - id: AC-2
      title: The declaration's per-rule claims are verified against behavior
      status: met
      tdd_phase: done
    - id: AC-3
      title: Declared rules and firing rules are in bijection, failing by name
      status: met
      tdd_phase: done
---

## Goal

Collapse the three hand-maintained lists that describe the coherence rule set into
one declaration beside the rules, so a rule added without it fails by name rather
than reading as covered while nothing exercises it.

## Context

M-0291 pinned the rule set across a generated domain: every actor role crossed
with every subset of the presence-bearing trailers, a golden verdict at each
point, design-doc invariants each carrying a non-vacuity guard, a reachability
assertion over the declared rules, and a bidirectional property for the subset
`verb.Apply` enforces.

Three lists still describe that rule set by hand, and each sits in a different
file from the rules it describes: the rule roster, the seam's force-predicated
subset, and the domain's own trailer axis. The first two already carry comments
naming this milestone as their retirement.

The axis is the one nobody named, and it is the one that matters most. A rule
predicated on a trailer outside the five the domain varies fires at no point in
that domain — so the golden never moves, the reachability assertion never sees
it, and the rule reads as covered while nothing exercises it. Retiring the two
named lists leaves that open; deriving the axis from the same source closes it,
because a rule declaring a new input widens the domain by construction.

D-0062 records why this is a declaration rather than the Pin-and-bijection
registry the epic originally scoped.

## Acceptance criteria

### AC-1 — Three hand-maintained rule lists derive from one declaration

One table beside the rules declares each rule: the presence-bearing trailers it
turns on, and whether it fires only when a force trailer is present. The rule
roster, the seam's force-predicated subset, and the domain's trailer axis all
derive from it.

The hand-maintained copies are deleted rather than left alongside. A derivation
that coexists with the list it replaces is a second source of truth, and the two
answer the same question differently the first time one is edited.

Evidence: each derived value asserted equal to the list it replaces before that
list is removed, so the change is measured behavior-preserving rather than
assumed; the removal is what makes the derivation load-bearing.

### AC-2 — The declaration's per-rule claims are verified against behavior

The declaration is data a human writes, so a test that reads its claims and
believes them proves nothing about the rules. Each entry is asserted against what
its rule actually does across the domain.

The force claim is checked in both directions: a rule declaring it needs a force
trailer must fire at no point lacking one, and a rule not declaring it must fire
at some point lacking one. Checking only the first direction would let the claim
be dropped from a rule that needs it, which narrows what the seam enforces
without anything saying so.

Each declared input is checked for relevance: the domain must hold two points
differing only in that trailer where the rule is the reported verdict at one and
not the other. That is weaker than "the rule's condition reads this trailer",
and first-violation ordering is why — an earlier rule shadowing this one moves
the reported verdict when the trailer toggles, without this rule's condition
touching it. So the check establishes that a declared input is load-bearing for
the domain, which is what the field exists for, and not that the rule reads it.
A characterization test pins that boundary, so a reader meets it rather than
discovering it against a declaration they trusted.

Completeness of the declared inputs — that no trailer outside them affects the
rule — is not asserted, and is not observable through the verdict for the same
reason. Establishing it needs a different instrument: an AST policy walking each
rule's condition back to the trailer lookups it depends on, of the shape
`internal/policies/` already runs elsewhere. Declined here — the trailer lookups
are hoisted above the conditions they serve, so per-rule attribution would mean
restructuring the guard into table-driven evaluation, which D-0062 rejects on
its own grounds. AC-3 covers what completeness was wanted for: a rule whose
input is missing from the axis fires nowhere in the domain, and the bijection
reports it by name.

Evidence: both force directions and the relevance check asserted across the
generated domain, each with a fixture whose deliberately mis-declared entry makes
the assertion fail.

### AC-3 — Declared rules and firing rules are in bijection, failing by name

Every declared rule fires somewhere in the domain, and every rule that fires is
declared. The failure names the rule; one reporting only that a mismatch exists
sends the reader back to the search the check just performed.

Evidence: both directions, each with a fixture — a declared rule that fires
nowhere, and a rule firing under a name absent from the declaration — asserting
the message names it.

## Design notes

- **The declaration lives beside the rules, not in a test file.** Not because a
  same-package test file would drift — the checks run identically from either —
  but because `coherence.go`'s own doc comment already enumerates this rule set
  in prose, and putting the structural declaration elsewhere forks the rules'
  documentation across two files. The house pattern agrees: `internal/workflows/spec`
  is production code whose cells are consumed almost entirely from tests.
- **The seam's membership criterion is not recorded here.** `FiresOnlyWithForce`
  states a property of the rule. `verb.Apply` enforces exactly the rules carrying
  it today, but membership is decided by satisfiability (D-0060), and keying the
  declaration on a property that merely coincides with it would hand the next
  author the wrong test.
- **Declaring inputs, not just names, is what closes the axis.** A roster of rule
  names retires two lists and leaves the third — the one whose absence lets a rule
  go unexercised.
- **Adding a rule still means adding its entry.** That obligation is real and
  worth naming: it is one line beside the rule rather than three edits in another
  file, and AC-3 catches the omission for every rule that fires within the current
  axis. What it does not catch is a rule both undeclared and predicated on a new
  trailer — stated here rather than closed, per D-0062.

## Surfaces touched

- `internal/verb/coherence.go` — the declaration and the three derivations.
- `internal/verb/coherence_domain_test.go` — axis, roster and force subset now
  derived; case names and golden lines sorted.
- `internal/verb/coherence_claims_test.go` — AC-2.
- `internal/verb/coherence_bijection_test.go` — AC-3.
- `internal/verb/testdata/coherence_domain.golden` — regenerated.

## Out of scope

- Table-driven evaluation — dispatching the rules from the declaration so an
  undeclared rule cannot run. Rejected in D-0062; it refactors a working guard for
  the residual named above.
- The branch spec's Pin registry and any change to it. Different rule space,
  different key.
- Which rules `verb.Apply` enforces. Membership stays satisfiability per D-0060;
  this milestone changes only where that subset is written down.

## Dependencies

- M-0291 — its generated domain, and the two lists that name this milestone as
  their retirement.

## References

- D-0062 — why a declaration rather than a cell registry.
- D-0060 — the satisfiability criterion for the seam's subset, unchanged here.

## Work log

### AC-1 — Three hand-maintained rule lists derive from one declaration

The roster, the seam's force-predicated subset, and the domain's trailer axis all
derive from `coherenceRuleSpecs`; the three hand-maintained copies are gone ·
commit 9d327705d

The equality between each derivation and the list it replaced was asserted while
both existed and removed with them — a transitional proof rather than a durable
pin. What stays durable is the golden: regenerated and verified a pure
permutation, so every actor-and-trailer point maps to the verdict it did before.
Case names and golden lines are now sorted, which decouples the golden from the
axis order, so a rule widening the axis adds lines instead of rewriting all of
them.

### AC-2 — The declaration's per-rule claims are verified against behavior

Both force directions and the per-input relevance check assert against the
verdicts the rules actually produce, with a mis-declaration fixture for each ·
commit 772516caf

Building this falsified the criterion as first written. It required that a rule
not fire where every input it declares is absent, which
`principal-missing-for-non-human-actor` contradicts: it fires precisely when
`aiwf-principal` is absent, and declares that trailer. The criterion was
corrected before the AC was promoted. Input completeness is not asserted and is
not observable through the verdict — only the first violation is reported, so an
undeclared trailer can change which rule reports without any condition having
consulted it.

### AC-3 — Declared rules and firing rules are in bijection, failing by name

Both directions asserted, each with a fixture, the failure naming the offending
rule · commit 5ad260301

`TestCheckTrailerCoherence_EveryRuleIsReachable` was removed rather than kept
alongside: direction one asserts what it asserted, over the same domain, and its
shadowing rationale moved into the new check's failure message. The mutation that
survived at AC-1 — dropping a rule's entry while another rule keeps its trailer
on the axis — now fails naming `audit-only-non-human`.

## Decisions made during implementation

- **D-0062 — a declaration rather than a Pin-and-bijection cell registry.** Taken
  before implementation. Its reasoning was corrected at wrap review: what
  separates this rule space from the branch spec's is *enumerability*, not
  vocabulary size, and the declaration is now held to the same owner-and-
  retirement bar the decision applied to the registry it rejected.
- **Sorted case names and golden lines.** The derived axis orders differently
  from the hand-written one it replaced, which alone would have rewritten every
  golden line. Sorting both decouples the golden from axis order for good: a rule
  widening the axis now adds lines instead of rewriting them, measured
  insert-only.
- **`TestCheckTrailerCoherence_EveryRuleIsReachable` removed, not kept.**
  Direction one of the bijection asserts the same property over the same domain;
  keeping both would be two tests answering one question.
- **`FiresOnlyWithForce`, not `RequiresForce`.** Named for what it states about
  the rule, so it cannot be read as recording seam membership. Membership is
  decided by satisfiability (D-0060) — a different criterion that selects the
  same three rules today, and would not have to.
- **Completeness of the declared inputs left unasserted.** An AST policy over
  each rule's condition would establish it. Declined: the trailer lookups are
  hoisted above the conditions they serve, so per-rule attribution needs the
  table-driven restructure D-0062 rejects on its own grounds.

## Validation

`make check-fast` (full suite, `go vet`, golangci-lint) exit 0.
`AIWF_COVERAGE_BASE=epic/E-0079-… make coverage-gate` exit 0.
`go build ./...` green. `aiwf check`: 0 errors.

The golden regeneration was verified a pure permutation of its predecessor —
every domain point canonicalized and compared, zero verdict changes — and the
review re-derived that independently rather than accepting it.

## Deferrals

None. The two candidates were dispositioned rather than punted: the AST
completeness policy is recorded above as considered and declined, and the
value-axis question is covered under *Reviewer notes* as backstopped rather than
open. No gap was filed, because neither needs its own branch, review, or an
undecided call.

## Reviewer notes

Two independent fresh-context lenses ran over the full change-set — code-quality
and design-quality — each briefed to measure rather than reason. Both returned
findings that changed the work.

**Fixed, pinned by the check that landed with them:**

- The `Reads` field documented itself as naming the trailers a rule's condition
  *consults*, and the type doc claimed every entry was asserted against
  behavior. Neither was true. First-violation ordering means an earlier rule
  shadowing a later one moves the reported verdict when a trailer toggles,
  without the later rule's condition touching it — so a false input declaration
  is accepted. Measured against `audit-only-non-human`, whose condition never
  reads a force trailer: declaring that it does passes. The docs now state what
  the check establishes, and a characterization test pins the boundary so it is
  met rather than discovered. Given this epic exists to remove surfaces claiming
  enforcement the kernel does not perform, a comment promising a stronger
  verification than the one that runs is exactly the class it targets.
- `RequiresForce` asserted, in its own doc, that it was "what puts a rule in the
  subset `verb.Apply` enforces" — a membership claim keyed on the wrong
  criterion, and one this spec contradicted two files away. Renamed and
  re-documented; see *Decisions*.

**Fixed inline under the cheap-fix test:** the non-human-actor predicate was
spelled four times. Two were production copies in one file with no reason to
differ and are now one helper. One was an in-file re-inlining in the seam-subset
test and now calls the file's own helper. The fourth is kept deliberately: the
design-doc invariants test states the predicate independently, and borrowing the
implementation's notion of "non-human" would let a wrong predicate satisfy both
sides of that test. That reason is now written where the copy lives, so the next
reviewer meets it instead of re-raising it.

**Raised and declined, so the next round does not re-open them:**

- *Move the declaration into a test file, since every consumer is a test.* It is
  a literal instance of the "only its own test reaches it" smell. Kept in
  production anyway: `coherence.go`'s doc comment already enumerates this rule
  set in prose, and splitting the structural declaration from it forks the rules'
  documentation across two files. `internal/workflows/spec` is the house
  precedent for production cells consumed from tests. The design note now argues
  from that rather than from drift, which was the weaker reason.
- *Give the domain a value axis.* Every trailer now carries one uniform value,
  so nothing would catch a rule that read a trailer's **value** rather than its
  presence. No such rule exists, and one would fail safe: it would fire nowhere
  in the domain, and the bijection would name it. A second value per key doubles
  the domain for a rule shape that does not exist. Left as is, knowingly.
- *A literal reading of satisfiability admits more rules to the seam than the
  three enforced there.* Raised by the design lens and out of this milestone's
  scope — it questions D-0060's criterion, not this declaration. Recorded here
  because it is worth someone's attention when the seam's membership is next
  revisited; nothing in this milestone depends on the answer.

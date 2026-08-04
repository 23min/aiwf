---
id: M-0291
title: Wire trailer coherence at the apply seam
status: in_progress
parent: E-0079
tdd: required
acs:
    - id: AC-1
      title: Forced transition by a non-human actor is refused before commit
      status: met
      tdd_phase: done
    - id: AC-2
      title: Coherence verdicts pinned across the full actor-and-trailer domain
      status: open
      tdd_phase: done
    - id: AC-3
      title: No production path commits without passing the coherence guard
      status: open
    - id: AC-4
      title: An ADR records that sovereign acts are prevented at the verb route
      status: open
---

## Goal

Put the trailer-coherence guard at the one seam every verb's commit passes
through, so a forced act by a non-human actor is refused at the moment it is
attempted rather than reported after it has already landed.

## Context

`CheckTrailerCoherence` is reached from two verbs. Every other verb that
constructs a sovereign force trailer commits without consulting it, and the
operator learns of the violation only when the pre-push check walks git history
— by which point the act is in the log and the exits are history rewrites.

The guard was never copyable to the verb layer, which is why it reached one verb
of four rather than drifting there by inattention. A verb's trailer set is
incomplete when the verb returns: the CLI layer appends the principal,
on-behalf-of, authorized-by and scope-ends trailers afterwards, so a guard
placed inside a verb would see no principal and refuse every legitimately
authorized non-human actor. The verbs that do call it assemble a complete set
themselves. `verb.Apply` is the single point downstream of both shapes.

## Acceptance criteria

### AC-1 — Forced transition by a non-human actor is refused before commit

Driven against the real binary, not the library. Every site that constructs a
sovereign force trailer refuses: the shared transition-trailer helper — which
serves promote, cancel, and both AC-granularity transitions — and the inline
sites in `add` and `authorize`.

Refusal is observable rather than inferred: a non-zero exit, a message naming
the rule that refused, and `HEAD` byte-identical before and after. A guard that
refuses but has already written is the failure this milestone exists to prevent,
so the unmoved-`HEAD` assertion is the load-bearing half.

The force-replace verbs stay open to non-human actors. `contract bind`,
`contract recipe` and `update --remove` declare a `--force` that means
force-replace, emit no sovereign trailer, and would break legitimate automation
if swept in — a different word spelled the same.

Evidence: subprocess-level tests per force-trailer construction site asserting
the unmoved `HEAD`, plus a passing case for each force-replace verb under a
non-human actor.

### AC-2 — Coherence verdicts pinned across the full actor-and-trailer domain

The rule set is a predicate over *(actor role × trailer presence-vector)*. The
test generates that cross product rather than enumerating the cases someone
thought of, so coverage is a property of the test's construction and survives a
tenth rule being added.

The guard at the seam enforces all nine rules, not the force rule alone, because
the set is what the function checks. That costs nothing in reach: the
history-walking check already reports eight of the nine at error severity, so a
trailer set that newly fails at the seam is one the push already rejected and
only the timing changes. The ninth, `audit-only-with-force`, has no
history-walking counterpart — the seam is its first enforcement anywhere outside
the audit-only verb's own call. Whether the check side should also grow it is
the epic's open question, and this AC is where it gets answered by measurement.

Evidence: a generated table over the full domain, each combination carrying its
expected verdict.

### AC-3 — No production path commits without passing the coherence guard

A policy under `internal/policies/` fails when a path can reach a commit without
the guard.

The existing apply-callers walker already enumerates every dispatcher reaching
`verb.Apply` — directly or through the `cliutil` finish helpers — and asserts
each takes the repo lock. This is a second predicate over that same population,
so it extends that walker rather than standing up a parallel one.

Placing the guard inside `Apply` is what makes the property structural instead
of policed: the non-dispatcher caller in the cell-coverage fixture is covered
without being named, because it too goes through `Apply`.

Evidence: the policy failing against a fixture that reaches a commit off-seam,
and passing against the tree.

### AC-4 — An ADR records that sovereign acts are prevented at the verb route

The stance has two sides, and recording only the half this milestone builds
would misstate it: sovereign acts are *prevented* at the verb route and
*ratifiable* at the history route, the second being M-0292's subject.

The ADR also records why one seam suffices here where ADR-0038 needed two.

Evidence: the ADR at `accepted` status, with a structural assertion scoped to
its named sections.

## Constraints

- The wiring changes who may wield `--force`, never what it overrides. Tier-1 /
  Tier-2 semantics belong to G-0333 and stay untouched.
- No finding is downgraded to make this pass. `provenance-force-non-human` stays
  at error severity; the epic adds a way to clear it, not a way to ignore it.
- `contract bind`, `contract recipe` and `update --remove` must keep working for
  non-human actors.

## Design notes

- **One seam, not the two ADR-0038 uses.** That ADR needed a claim-side and a
  commit-side guard because a converging verb returns before a plan exists. A
  converging verb also writes no commit, so it emits no trailer and has no
  coherence to violate — the case that forced the second seam there cannot arise
  here.
- **The guard goes in `verb.Apply`, not in `cliutil`.** The self-assembling
  verbs and the cell-coverage fixture reach `Apply` without passing through the
  dispatcher layer, so a `cliutil` placement would leave exactly the paths this
  milestone exists to close.

## Surfaces touched

- `internal/verb/apply.go` — the seam.
- `internal/verb/coherence.go` — the rule set being called.
- `internal/policies/apply_callers_lock.go` — the walker AC-3 extends.
- `docs/adr/` — AC-4's record.

## Out of scope

- Delegated force (G-0023). That changes the provenance model; this makes the
  current model true.
- The ratification path (M-0292) and the surface corrections (M-0293).

## Dependencies

- None. This milestone is the epic's foundation; M-0293 and M-0294 depend on it.

---

## Work log

### AC-1 — Guard at the apply seam

Every force-trailer site now refuses a non-human actor before writing · commit
293dee60d · tests 5/5 subtests, full suite green

Measured before the change: `promote`, `cancel`, the AC phase transition, and
`add` each committed the forced act at exit 0. `authorize` alone refused, via
its own human-actor check rather than coherence — pinned in the same table so
that site stays covered whichever guard holds it.

Two facts the implementation turned up. Enforcing all nine rules at the seam
broke no existing test, so the blast radius the milestone reasoned about is
zero rather than merely small. And a non-human actor cannot reach
`force-non-human` first: getting past the allow-rule requires an active scope,
whose `aiwf-on-behalf-of` trips `force-with-on-behalf-of` earlier in the rule
order. The test asserts a force rule refused rather than naming one, since
naming either would pin the order instead of the behavior.

## Decisions made during implementation

- (none)


---
id: D-0062
title: Derive the coherence rule lists instead of a Pin-and-bijection cell registry
status: proposed
relates_to:
    - M-0294
    - E-0079
    - D-0060
---
## Context

E-0079's scope recorded a cell registry for rule spaces with no FSM coordinate as
decided, mirroring the Pin-and-bijection discipline the branch spec carries from
M-0162. M-0294 was to build it, and reached planning with one blocking question:
is a cell one rule, or one point in the input cross-product?

M-0291 then landed more coverage of that rule space than the scope anticipated.
The coherence domain is generated — every actor role crossed with every subset of
the presence-bearing trailers — so coverage is a property of the generator rather
than of anyone's diligence. A golden pins the verdict at every point, nine
invariants sourced from the design doc each carry a non-vacuity guard, every
declared rule is asserted to fire somewhere, and the seam's subset is pinned in
both directions.

The drift that motivated a registry was closed by a different mechanism than the
one proposed. A guard reaching one call site of four is a call-site failure, not a
rule-coverage failure, and M-0291/AC-3's chokepoint policy holds the seam singular
by resolving calls across every commit-construction primitive.

What remains uncovered is narrower and specific: three hand-maintained lists — the
rule roster, the seam's force-predicated subset, and the domain's own trailer axis.

## Decision

Decline the Pin-and-bijection cell registry. Declare the coherence rules once,
beside the rules themselves, each entry naming the trailers its condition consults
and whether it fires only when a force trailer is present. The domain's trailer
axis, the seam's force-predicated subset, and the reachability roster all derive
from that one declaration.

## Reasoning

Per-rule cells would add a weaker property beside a stronger one that already
holds. A Pin asserts someone wrote a test; the generated domain asserts the rule
demonstrably fires. Registering nine rules to prove the weaker claim buys nothing
and creates a second answer to a question already answered.

Per-combination cells would demand a named pinning test at every point of a domain
the generator covers exhaustively with no pins at all — a large mandate to
re-establish coverage that exists.

Either shape is a mandate, and a mandate costs per subject forever. M-0294's own
design notes required the registry to land with a named owner and a stated
retirement trigger; neither had one.

The remaining hole is real but different in kind from the one a rule-name registry
closes. Because the domain's trailer axis is hand-maintained too, a rule
predicated on a sixth trailer fires at no point in the domain, so the golden never
moves and the reachability assertion never sees it. A roster of rule names leaves
that open. A declaration naming each rule's inputs closes it, because the axis
derives from the same source: a rule reading a new trailer widens the domain by
construction.

The blocking question dissolves rather than being answered. With no cells, there
is nothing to decide about their granularity.

This is not a verdict against the branch spec's registry. That rule space genuinely
is a cross-product of independent coordinates, which is what makes cells the right
key for it; trailer coherence is nine predicates over one small trailer vocabulary,
which wants a list.

## Consequences

E-0079's scope item and its sixth success criterion narrow to the property
actually delivered.

The declaration is verified rather than trusted. Deriving a test's expectation from
a hand-written flag is the vacuity shape this epic has already met twice, so each
entry's claim is asserted against behavior: a rule declared to fire only under a
force trailer must fire at no domain point lacking one.

Table-driven evaluation — dispatching the rules from the declaration, so an
undeclared rule cannot run at all — was considered and rejected here. It refactors
a working guard for a residual case that requires a rule to be both undeclared and
predicated on a new trailer. That residual is stated rather than closed; the
bijection in M-0294/AC-3 catches every undeclared rule that fires within the
current axis.

The membership criterion for the seam's subset is unchanged. Satisfiability, per
D-0060, still decides which rules `verb.Apply` enforces; this decision changes only
where that subset is written down.

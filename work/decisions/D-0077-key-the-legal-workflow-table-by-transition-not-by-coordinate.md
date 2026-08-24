---
id: D-0077
title: Key the legal-workflow table by transition, not by coordinate
status: proposed
relates_to:
    - D-0007
    - G-0631
---
> **Date:** 2026-08-24 · **Decided by:** human/peter

## Question

The legal-workflow spec table is the artifact ADR-0011's three-pass methodology
produced and M-0123 reconciled. It was built to be the place a reader learns what
aiwf permits, and it is guarded against the implementation in both directions.

Enumerating it exhaustively shows it is not total. Over the 99 coordinates formed
by (Kind × FromState × legality verb), 49 carry a cell and 50 do not — and nothing
distinguishes a coordinate deliberately left silent from one nobody considered.
Silence is not a decision, so for half the surface there is no fact of the matter
recorded at all.

So: what makes the table total, and what is the right key for a cell?

## Decision

Three rulings, taken together because each changes what the others must cover.

**The target belongs in the key.** A cell is keyed by
(`Kind`, `FromState`, `Verb`, `ToState`). The table becomes a transition table
rather than a legality list: it says where a verb takes an entity, not merely
whether the verb is refused. A NoOp is expressible as a cell whose target equals
its origin, and an illegal transition from the same origin is a separate cell.

**`authorize` leaves the cell table.** It binds a scope to a branch and has no
`FromState` semantics; its four declared cells all say "wrong kind" and say nothing
about state. The kind restriction moves to `GlobalRules()`, which is where
ADR-0013 put cross-cutting preconditions that carry no cell coordinate, and where
`authorize`'s four substantive rules already live.

**Applicability is a kind-by-verb fact, declared once.** Whether a verb applies to
a kind at all is not a statement about state, so it is not repeated per state. The
applicability table is itself total over every kind-by-verb pair and is policed as
such. Cell-table totality is then defined over applicable rows only.

## Reasoning

Measured 2026-08-24 on `main`, enumerating every coordinate against
`spec.Rules()`:

| Verb | Declared | Undeclared |
|---|---:|---:|
| `promote` | 33 | 0 |
| `cancel` | 12 | 21 |
| `authorize` | 4 | 29 |

The shipped table holds 61 cells, 4 global rules and 12 anti-rules, and 17 of the
61 cells carry any precondition.

**Why the target, and not a third outcome value.** G-0631 records the concrete
failure: all 15 terminal-state `promote` coordinates are declared illegal, while
M-0281/AC-1 makes a promote to the entity's current status a NoOp. One cell covers
both targets from that origin, so no single outcome can be right. A third outcome
value at the same coarse key would record that two outcomes exist without saying
which target produces which — it encodes the ambiguity rather than resolving it.
Adding the target resolves it and simultaneously answers the question a reference
document must answer, which the table cannot answer today: what state the entity is
in afterwards.

The cost is bounded. The target is derivable from `entity.transitions` for every
legal cell, so a mechanical pass fills most of the expansion and leaves the illegal
and NoOp rows to be ruled by hand.

**Why `authorize` leaves rather than being filled in.** Filling its 29 holes would
manufacture questions instead of answering them — eight of them would require
ruling whether authorizing a `done` epic or a `cancelled` milestone is legal, which
are questions the verb's own semantics do not raise. A key that does not fit a verb
is better removed than padded.

**Why applicability is declared once.** With `authorize` gone the cell table carries
two verbs, so applicability is 8 kinds by 2 verbs — small enough to be trivially
total and policed, which is what keeps it from becoming the new place silence
hides. Writing it as per-state cells instead would grow with every future kind-verb
pair while saying nothing about state.

Alternatives considered and rejected:

- **Leave the table as a legality list and add a third outcome for NoOp.** Cheapest,
  and the schema already permits complementary cells at one coordinate. Rejected
  because it completes the grid without making it readable — see above.
- **Treat a NoOp as legal.** Loses the operator-relevant difference between a verb
  that moved something and one that did nothing, and remains wrong in the other
  direction, since the genuinely illegal target at that origin is then mislabelled.
- **Model the table in a formal specification language and check it there.** The
  coordinate space is small enough to enumerate exhaustively against the real
  kernel, and a separate model would be a second hand-maintained description of the
  same rules with nothing re-deriving it.

## Consequences

- `Rule` gains a target field and the enforced uniqueness key changes. Existing
  cells split by target; the table grows from 61 cells before the applicable-row
  totality invariant is applied.
- The positive and negative cell drivers are re-keyed, and their coverage becomes
  exhaustive by construction rather than by declaration — a coordinate with no cell
  becomes a policy failure rather than a silence.
- G-0631 is resolved by this schema change rather than by correcting its 15 cells.
- `spec.go`'s package documentation states that (`Kind`, `FromState`, `Verb`)
  uniquely keys each Rule. The enforced invariant is that tuple plus `Outcome`, with
  complementary cells explicitly permitted. The documentation is corrected as part of
  the key change rather than tracked separately.
- The table becomes renderable as a legality reference that cannot drift, because it
  is generated from the table rather than describing it. Whether to ship that render
  is a separate decision.
- What would reopen this: a verb that is genuinely state-sensitive and has no target
  — it would need a cell shape this key cannot express.

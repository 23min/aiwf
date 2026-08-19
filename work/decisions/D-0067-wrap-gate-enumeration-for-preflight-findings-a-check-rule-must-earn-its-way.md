---
id: D-0067
title: Wrap-gate enumeration for preflight findings; a check rule must earn its way
status: accepted
---
> **The pass this governs was dropped** — see D-0069 and E-0086. The reasoning
> stands; it has no subject. Details in Consequences.

## Question

A preflight finding is recorded in the body of the entity whose lifetime it
shares, and dispositioned at wrap: **pinned** by a check, **recorded** as a
decision or a declined line, **tracked** as a gap, or superseded by another
finding. Repairing the text is a fact a row records alongside its disposition,
not a disposition itself — a defect merely fixed is the silent correction the
project's code-health force forbids.

That model is only worth its words if the disposition actually happens. Measured
on this tree, body sections nobody enforces are filled between 14% and 51% of the
time. So what makes wrap-time disposition happen?

## Decision

The wrap ritual's declared-sequence gate enumerates every undispositioned row
before the human approves it. **No check rule ships.**

The trial records the count of rows still undispositioned at wrap, per subject.
That number is what would earn a check rule later; until it exists, the rule is
not justified.

## Reasoning

**The measurement that frames it.** Adoption of a slot depends almost entirely on
whether something enforces it, not on how clearly it is explained:

| slot | mechanism | adoption |
|---|---|---|
| `addressed_by` on addressed gaps | check-enforced | 397/397 = 100% |
| `## Decisions made during implementation` | prose section | 157/309 = 51% |
| `discovered_in` on gaps | optional reference field | 245/588 = 42% |
| `## Deferrals` | prose section | 109/309 = 35% |
| `## ADRs produced` | prose section | 12/85 = 14% |

`## Deferrals` carries the most explicit instruction of any section — every
deferral that survives the cheap-fix test must be opened as a gap and its id
mirrored back — and scores worse than `## Decisions`, whose instruction is
vaguer. Clearer wording does not move the number.

**Why the 100% is not available here.** `addressed_by` is enforced by
`gap-addressed-has-resolver`, but that is the weaker half of why it is filled
every time. The stronger half is ergonomic: `aiwf promote <id> addressed --by
<milestone-id>` carries the value on the verb, and the flag's own help text names
the rule it satisfies. Filling it is the path of least resistance.

A ledger disposition has no equivalent. It is prose written during triage, with
no flag to hang it on; inventing one means a new verb for a record whose shape is
not yet settled. So a check rule here buys the enforcement half without the
ergonomic half, and its expected adoption sits somewhere between the unenforced
42% and the enforced-and-ergonomic 100%, at a point nothing in this tree
predicts.

**What the rule would cost.** Six check rules measured as introduced run 187 to
353 lines, median 266. This one needs more than the model it copies —
`gapAddressedHasResolver` is sixteen lines because it reads a frontmatter field,
while a ledger rule must parse a table — plus a finding-code constant, a
remediation hint, a row in the `aiwf-check` skill's code table, a firing fixture
for the coverage meta-gate, two template edits, and a referencing structural test
for each. Call it 300 to 350 lines. Per the code-health force that prefers a ban
paid once to a mandate paid per subject, a rule of this shape is a permanent tax
and needs a named retirement trigger before it lands.

**What the gate buys instead.** The wrap ritual already fires a declared-sequence
gate that enumerates every action verbatim and lets the human approve a subset.
Adding undispositioned rows to that enumeration costs a ritual edit and puts the
list in front of the one reader who can act on it. Its weakness is real and
worth naming: it depends on the assistant enumerating honestly, which is not a
guarantee in the sense the kernel uses the word. That weakness is comparable to
the check-only rule's, which depends on a warning being acted on with no
ergonomic path to acting.

**What would reverse this.** The trial's undispositioned-at-wrap rate. If rows
are routinely left undispositioned, the gate is not working and the rule earns
its cost; if they are not, the rule would have been a tax. Either way the number
decides, and it does not exist yet.

## Consequences

The extraction guarantee is a human at a gate, not a rule — stated plainly rather
than implied, because the difference matters when someone later asks what
enforces this.

The preflight epic keeps its constraint that the kernel does not change: no
finding code, no config field, no schema entry. That constraint was load-bearing
for keeping the epic small, and this decision is what preserves it.

The trial acquires one more required measurement. A trial that does not record
the undispositioned rate cannot promote this decision or retire it, which makes
that metric part of what the instrumentation must carry rather than an optional
extra.

The pass this governs was dropped. D-0069 rejects the dispatched reading pass
and E-0086 closes re-scoped to the lab rule alone, so no ledger is produced and
no wrap gate enumerates one. The reasoning above holds on its own terms — a gate
in front of a human beats an unenforced check rule where no verb carries the
value — and has nothing left to govern. The undispositioned-at-wrap rate is not
recorded, so the retirement trigger named above is unreachable and this decision
cannot be promoted or retired by the route it specifies.

The adoption table is about this repository rather than about the pass, so it
outlives the drop: it is evidence for how often an unenforced body section gets
filled here, and bears on any proposal to add another one. Its figures name no
command, so they record a measurement rather than settle a claim — re-derive
before relying on them.

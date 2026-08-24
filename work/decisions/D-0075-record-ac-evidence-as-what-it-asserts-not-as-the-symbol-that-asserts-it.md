---
id: D-0075
title: Record AC evidence as what it asserts, not as the symbol that asserts it
status: accepted
relates_to:
    - D-0038
    - D-0050
    - G-0579
---
> **Date:** 2026-08-23 · **Decided by:** human/peter

## Question

Standing guidance lets an AC reach `met` only on mechanical evidence — a test or
check that fails if the AC's claim breaks. It does not say how that evidence is
written down, and the obvious answer is to name the test function.

Names move. So: what does an entity body owe a reader who wants to know whether a
claim is still guarded, and does the tree need a check that cited symbols resolve?

## Decision

An entity body records evidence as **what the guard asserts**. A symbol name may
sit beside that as a locator; it is never the claim itself. A reader who cannot
find the symbol must still be able to find the guard from what the record says
about it.

The reference is load-bearing only while the entity is live. Once archived, a
milestone spec records what was true at close, and a stale symbol in it is history
rather than a defect — repairing it would make the record false in a new way.

No mechanical check enforces this, and none should on today's evidence.

## Reasoning

Measured on `main@9c1562181`: entity bodies cite 914 distinct Go test-function
names, of which 707 resolve and 207 do not. Of the 244 dead citation sites, 238
sit in archived entities. Reading the five live names settles what a checker would
have had to decide:

- Two are glob patterns naming test families (`TestBulkRevwalk_*` and
  `TestFSMHistoryConsistent_*` in G-0328), and both families are intact — 10 and
  23 functions.
- One is a hypothetical inside an argument: D-0038 rejects `--evidence
  TestSomeUnrelatedThing` as proof of relevance, so the counterexample it invents
  is not a citation.
- One names a test in the same sentence that says the test is deleted, which is
  accurate.
- One is a real defect, filed as G-0579 before this measurement ran.

A corpus-wide check would therefore fire five times on live entities, three of
them false, to surface one already-known defect. A finding stream at that ratio is
noise from its first run, and it consumes the attention that would otherwise
notice the real one. The 207 is a detector artifact for the same reasons: it
cannot tell a claim from an illustration, a glob from a symbol, or an accurate
past-tense sentence from a stale one — and reading is what a count is supposed to
spare you.

The defect this rule prevents is created at rename time. At wrap a citation is at
its freshest and least likely to be wrong, so a gate there sits where the defect
is not; the one person who knows that a test's old name is now its new one is
whoever performs the rename. That is why 188 of the 207 have no recoverable
near-match — the mapping existed in one head, at one moment, and was never
written anywhere.

Alternatives considered and rejected:

- **A check that every cited symbol resolves.** The false-positive ratio above,
  plus 207 existing violations to grandfather. D-0038 already ruled that symbol
  existence is not relevance; this is the same objection one level removed.
- **A wrap-time lint.** Placed where the defect does not yet exist.
- **A rename-time habit** — grep `work/` for the old name. Real, and
  unenforceable; a rule nobody follows is worse than no rule, because it reads as
  coverage.

## Consequences

- The AC-evidence rule in standing guidance carries the recording discipline, so
  the instruction arrives at the moment the citation is written. The guidance line
  budget rose from 144 to 146 to fit it, as that guard's own comment prescribes.
- This completes D-0038 rather than revising it. D-0038 rejected a mechanical
  evidence gate and left enforcement to review without saying what a reviewer
  looks for; this says.
- G-0579 remains a defect under this rule, because D-0015 is live.
- Archived entities are not swept. A stale symbol there is left alone.
- What would reopen this: a live-entity violation count that a sample shows is
  mostly real. The shape then worth building is diff-scoped rather than
  corpus-scoped — does any entity body mention a symbol this commit removed? — so
  it fires only where the information still exists, needs no grandfathering, and
  follows ADR-0033's repair-at-move-time principle for paths.

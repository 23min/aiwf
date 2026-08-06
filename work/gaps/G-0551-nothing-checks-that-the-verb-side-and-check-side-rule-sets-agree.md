---
id: G-0551
title: Nothing checks that the verb-side and check-side rule sets agree
status: open
priority: medium
discovered_in: M-0291
---
## What's missing

The coherence rules exist twice. The verb layer evaluates them on an assembled
trailer set before a commit; the history-walking audit re-implements them over
git log. Nothing checks that the two agree.

The duplication is structural rather than careless: the verb package imports the
check package, so the check package cannot import back, and sharing the
predicate would require extracting it to a third package below both.

## Why it matters

The divergences are a class rather than a list. The audit returns no findings
at all for a commit carrying no `aiwf-actor`, so every coherence rule diverges
on actor-less commits — G-0550 records the force instance; the same silence
covers the rest. Two further divergences are specific. The principal rule is wide on the verb side and
narrow on the check side — recorded in D-0059 and deliberately not filed,
because the two are equivalent for every commit the audit can actually see. And
`audit-only-with-force` exists verb-side with no check-side counterpart, which
M-0291 answered by measurement rather than by adding one.

Neither is a live defect. Both are drift, and drift with no detector is the
condition under which the third divergence is a defect and nobody notices. The
claim that one identifier routes a refusal at the verb and a finding on a landed
commit rests on the two implementations agreeing, and that agreement is
currently maintained by hand.

## Options

1. **Extract the predicate to a package below both** and have each side call it.
   Removes the divergence class entirely. The cost is a new package boundary for
   one predicate, and the two sides genuinely differ in output shape — the verb
   side returns the first violation, the audit reports all of them — so the
   extracted form has to serve both.
2. **A test that runs both implementations over the generated domain** and
   asserts they agree, with an explicit allowlist for divergences that are
   deliberate. Cheaper, keeps both implementations, and turns hand-maintained
   agreement into a checked one. The allowlist is where the two known
   divergences get recorded rather than remembered.
3. **Leave it and rely on review.** The status quo.

Option 2 is the lean: it buys the detection without the boundary, and the
allowlist documents the deliberate differences in the one place that fails when
an undeliberate one appears.

## Scope

Named by the design lens of M-0291's wrap review. The dual-emission pattern
itself is sanctioned; what is missing is the check that the two emissions say
the same thing.

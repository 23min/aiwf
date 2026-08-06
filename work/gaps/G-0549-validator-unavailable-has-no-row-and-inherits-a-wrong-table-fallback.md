---
id: G-0549
title: validator-unavailable has no row and inherits a wrong-table fallback
status: open
priority: medium
---
## What's missing

`contract-config/validator-unavailable` emits warning by default and error
only under the strict-validators setting, so it is a conditional-severity
finding. It has no row of its own in the `aiwf-check` skill, so it falls back
to the bare `contract-config` row — which sits in the errors table. A consumer
who hits it reads that it blocks their push, and by default it does not.

The new placement check cannot see this. It routes an emission to a documented
row by reading the `Code:` field at the construction site, and this one passes
the code in as a local variable rather than naming it, so the site resolves to
nothing and is dropped before any comparison happens. The row is therefore
wrong and unpoliced at the same time, which is the combination the check was
built to eliminate.

## Why it matters

This is the one known-wrong row the placement check is blind to, in the same
commit that added the check. Left unrecorded it reads as a row the check
approved, which is the opposite of true.

The blindness generalizes past this row. Any rule that computes its code
rather than naming it is invisible to the whole family of policies built on
the shared emission enumerator — the hint-completeness check and the
documented-in-skill check as well as this one. G-0068 records the same shape
for a computed `Subcode:`; a computed `Code:` is its sibling and is not
covered by that gap's text.

## Direction

Two halves, and the first is worth doing whether or not the second is:

- Give the subcode its own row, filed by the severity it actually emits. That
  fixes what a consumer reads today.
- Decide what to do about codes the enumerator cannot resolve. Either teach it
  to follow a code passed as a parameter into the helper that emits it, or ban
  the shape so every emission names its code literally — the same fork G-0068
  frames for subcodes, and the two should be settled together rather than one
  rule at a time.

Whichever way that goes, the honest interim is that the unresolvable set is
named somewhere a reader will find it, rather than being silently absent from
what the checks cover.

## Provenance

Found 2026-08-05 by the third independent review of the G-0542 patch, while
checking whether that patch's scope expansion into the contract packages had
caught everything it should have. The condition predates the patch.

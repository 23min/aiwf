---
id: D-0046
title: Diff the shared contract gate by finding identity, not the full struct
status: accepted
relates_to:
    - M-0273
---
# D-0046 — Diff the shared contract gate by finding identity, not the full struct

> **Date:** 2026-07-21 · **Decided by:** human/peter

## Question

`internal/verb/contractgate.go`'s shared diff-based validation gate
(built for M-0273/AC-1) computes a mutation's introduced findings as a
before/after diff of `contractcheck.Run`'s output. The first design
diffed on the full `check.Finding` struct — every field had to match
for two findings across the two runs to be treated as "the same."
Should the diff continue comparing full structs, or key on a narrower
subset of fields? The answer was non-obvious because `check.Finding`
looked like a natural, already-comparable value type, and nothing in
its own package signaled that one of its fields carries positional
rather than identifying information.

## Decision

Key the diff on a `findingIdentity` subset — `Code`, `Severity`,
`EntityID`, `Subcode`, `Path` — instead of the full `check.Finding`
struct. `Message`, `Line`, `Hint`, and `Field` are excluded from the
identity; the diff still returns the real `check.Finding` values (full
`Message` intact) for whatever it reports as introduced, only the
matching/counting key changed.

## Reasoning

The full-struct diff broke on its first real exercise against a
multi-entry `contracts.Entries` list: `contractcheck.Run`'s `Message`
field embeds the finding's positional index within
`contracts.Entries` (`"contracts.entries[1] (id=...): schema path..."`).
Removing or inserting an earlier entry shifts every later entry's
index and its `Message` text, even though nothing about that entry
actually changed. `TestContractUnbind_OnlyRemovesNamedID` — a
pre-existing test migrated unchanged onto the new gate during AC-2 —
caught this immediately: unbinding the middle of three bound contracts
wrongly triggered the gate to report a "regression" and block the
unbind, because the untouched third entry's finding shifted from
`contracts.entries[2]` to `contracts.entries[1]` and no longer matched
its own pre-existing occurrence byte-for-byte.

Two alternatives were considered and rejected:

- **Change `contractcheck.Run`'s `Message` format to reference the
  entity id instead of the positional index.** Rejected: touches a
  shared, more foundational package outside this milestone's declared
  scope (M-0273's own "Out of scope" section rules out changes to
  what the underlying contract check validates) — a bigger blast
  radius for a fix that is local to the gate's own comparison
  semantics.
- **Keep the full-struct diff but special-case the index-embedding
  field.** Rejected as needless complexity; a general identity-subset
  key is the more direct fix and generalizes to any future
  `contractcheck.Run` field that turns out to be similarly derived or
  positional rather than identifying.

A `wf-vacuity` audit on the fix found that `EntityID`'s presence in
`findingIdentity` was not actually pinned by any test that existed at
the time — dropping it from the struct left the whole suite green,
because the two tests that seemed to rely on it happened to pass by
fixture-ordering coincidence, not real discrimination. Closed with an
adversarial test constructed so an `EntityID`-blind diff would report
the *wrong* entity's finding, not just the wrong count.

## Consequences

Any future consumer that diffs two `contractcheck.Run` (or similarly
shaped) result sets for equality should treat `Message` (and any other
`Run()`-computed prose field) as unsafe for an equality/identity key
across two separate invocations whose input differs by more than the
field under test — position-dependent or context-dependent rendering
can silently break a naive full-struct comparison.

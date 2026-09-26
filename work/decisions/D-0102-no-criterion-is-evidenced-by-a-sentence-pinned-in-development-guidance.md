---
id: D-0102
title: No criterion is evidenced by a sentence pinned in development guidance
status: accepted
relates_to:
    - D-0091
    - E-0092
    - M-0333
---
> **Date:** 2026-09-25 · **Decided by:** Peter Bruinsma (human), while wrapping M-0333

## Question

D-0091 bars evidencing an acceptance criterion with a sentence pinned in `CLAUDE.md`, and enforces it with a diff-scoped scan of test source for phrase-presence assertions. Implementing that scan in M-0333 showed that deciding from syntax whether a test *asserts a phrase* keeps growing special cases — polarity, rooting, helpers, positions — and stays partial. E-0092 also covers more than `CLAUDE.md`: `AGENTS.md`, the project router and the documents it routes to. How is the rule held across that set, and by what mechanism?

## Decision

- No acceptance criterion is evidenced by a sentence pinned in this repository's development guidance: a `CLAUDE.md` or `AGENTS.md` at any depth, the project router `.guidance/project.md`, and the documents it links to.
- Every test that reads one of those documents from the repository root is listed, and each entry names it a pin to retire, a relationship check, an absence check, or a test that names a guidance document without reading it. The list is held equal to the tree's readers in both directions over the whole tree.
- The existing pins are list entries; the E-0092 milestone that moves a pinned passage retires its pin and deletes the entry.

## Reasoning

- Whether a test reads a document is decidable from syntax; whether it asserts a phrase in it is not. A scan of assertion shapes grew a case per spelling and still missed shapes, so a green result claimed more than it guaranteed. A read check is total for the forms it sees, fails closed, and puts every reader in front of a reviewer.
- The list changes where the pin-or-not judgment sits. Under a diff-scoped scan a new pin fails automatically; under the list a new reader fails until an entry is written, and the entry's kind is the judgment, held at review. That is weaker for a reviewer who labels a pin a relationship check, and stronger for every reader the scan would not have recognized.
- Checking the whole tree rather than a diff also catches a reader created by a change outside its own test file, which a diff-scoped check cannot see, and needs no base ref.
- The scope follows the guidance set E-0092 defines: the rule's ground — a pin holds one reading and locks the file's size — is the same for every document primed or routed to.

## Consequences

- `guidance-readers` in `internal/policies/` holds the list; `CLAUDE.md`'s AC-evidence section and substring-assertion bullet cite this decision.
- A new test reading development guidance lands with a list entry naming what it is; a test meant to establish "the guidance documents X" states it as a relationship check or records it as an observation.
- The rule does not see a root resolved other than through `repoRoot` helpers or a literal climbing path, nor a path computed at run time; `guidance_readers.go` states these limits.

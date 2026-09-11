---
id: ADR-0048
title: A create must be complete; an edit must not regress
status: accepted
supersedes:
    - ADR-0043
---
## Context

ADR-0043 placed body-section membership at the write seams and nowhere else, and
that placement stands. What it also fixed was the question each seam asks: *"A
scan over the bytes a verb is about to write, called by every body-supplying
verb, refusing the write at error severity for every kind."* Every write must
produce a complete body.

Its case for that rests on a remedy it describes as free — *"An author who does
not yet know a section's content keeps the heading and leaves it empty, which is
the existing `entity-body-empty` finding."* Measured on a gap whose body omits
`## Why it matters`:

    section omitted        aiwf check reports nothing
    heading added, empty   error entity-body-empty

For epic and milestone that finding is a warning, and the remedy is as cheap as
described. For the born-complete kinds it is an error, and the pre-push hook
blocks on it. Those kinds carry 54 of the 55 live entities that omit a required
section — 30 open gaps and 24 accepted decisions — so for almost every entity
holding this debt the remedy converts silence into a blocking finding, and the
real instruction is not "add the heading" but "add the heading and write the
section". `aiwf edit-body` offers no `--force` — and adding one would not
rescue completeness here, because `--force` is sovereign and human-only. An AI
actor would be hard-blocked from editing any of those 54 entities at all, with no
override available to it. That is the argument against completeness at an edit;
the absence of the flag today is only its cheaper form.

Completeness at a create carries no such cost. A new entity has no committed body
to be judged against, and the scaffold writes every heading, so a body missing
one arrives only from an explicit `--body` or `--body-file`.

Separately, "every body-supplying verb" names four. `aiwf import` is the fourth
and is deprecated, though no surface records that (G-0667).

## Decision

Membership is enforced at the write seams and nowhere else. A violation is a
section the kind requires not present as a top-level `## ` heading; a nested
heading counts as absent, sections beyond the required set are legal, and order
is not enforced. Emptiness remains a separate property at a separate seam.
Neither rule joins `check.Run`. All of that is ADR-0043's and is unchanged.

**What the seam asks depends on whether the write creates or edits.**

- **`aiwf add` requires a complete body**, for every kind, at error severity.
  `--force --reason` bypasses it as a sovereign override and stamps its trailer
  only where a refusal was actually overridden.
- **`aiwf edit-body` requires non-regression**: it refuses a write that drops a
  section the committed body carries, and permits one that keeps an omission
  already there. Both of its modes apply the same rule, so the same edit is
  judged alike whether it arrives as supplied bytes or as a working-copy edit.
  There is no `--force`; a deliberate removal is recorded with
  `aiwf acknowledge illegal`.

A create has no history to be held to and an edit does. Holding an edit to
completeness would refuse an author over an omission they did not introduce,
which is the cost measured above; holding a create to non-regression would
demand nothing at all, since a create has no baseline to regress against.

**`aiwf import` is excluded** rather than gated, on its deprecation. Until
G-0667 records that, the exclusion rests on a decision the tree does not carry.

**Seam two — the push — is unchanged and unbuilt.** A gate riding the commit
range the provenance audit resolves, scoped to entities whose body content the
range changed, at error severity. It remains the authority, for the reason
ADR-0043 gives: a body can reach a commit without passing any verb.

## Consequences

- The existing omissions never converge. Non-regression means an entity missing a
  section keeps it missing through every subsequent edit, so the debt is
  permanent rather than merely current until something reads bodies it is not
  writing. Seam two, or a tree-side rule with a baseline, is the only thing that
  changes that.
- No surface can answer "which entities are incomplete?". `aiwf check` is
  unchanged by design, and the write seams see only what they are writing. The
  guarantee this decision buys is *no new entity is born incomplete, and no edit
  makes a body worse* — not a property of the tree.
- An author is never blocked by debt they did not create, which is what makes a
  refusal with no `--force` tolerable at the edit seams.
- `--force` at `aiwf add` now applies to every kind, where under the previous
  born-complete-only gate it was inert on epic and milestone. It is a sovereign
  act whose scope widened.
- **The two decisions above compose into a permanent exemption.** A body forced
  past the absence half lands incomplete; the edit seams then compare against
  that committed body and find the section already gone, so no verb ever asks it
  to converge, and no check rule reports an absent section. `--force` at a create
  is therefore not "create anyway" but "create anyway, and stay exempt". Forcing
  past the *emptiness* half carries no such consequence, because
  `entity-body-empty` still reports the section afterwards — which is the reason
  to leave a heading in and empty rather than drop it.
- A body-writing verb that does not call the scan is unenforced at seam one, and
  nothing forces the call. `aiwf import` is the live instance, deliberately.

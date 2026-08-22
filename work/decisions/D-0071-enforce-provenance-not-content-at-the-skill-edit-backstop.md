---
id: D-0071
title: Enforce provenance, not content, at the skill-edit backstop
status: accepted
relates_to:
    - G-0220
    - M-0196
---
> **Date:** 2026-08-21 · **Decided by:** human/peter

## Question

G-0220 recorded a shippable ritual edit that landed with no gap, no acceptance
criterion, no owning milestone, and no test. The human caught it; the kernel said
nothing. Three of the gap's four complaints are about provenance — nothing owned the
edit, nothing recorded why it happened. One is about a missing test.

The chokepoint that shipped in M-0196 implemented only the fourth complaint, in the
weakest form available: the edited `SKILL.md` path must appear as a string literal
somewhere in the concatenated `internal/policies` test sources. So the question is what
an edit to a shipped surface should actually have to prove.

## Decision

The skill-edit backstop enforces **provenance**: an edit to a shipped surface must ride
a commit whose `aiwf-entity` trailer names an owning entity that exists. It no longer
requires that a policy test reference the edited path.

That trailer is the whole of the requirement. No `aiwf-verb` is asked for, because a
ritual `SKILL.md` is source rather than an entity file and no aiwf verb commits it — the
closed set `trailer-verb-unknown` enforces carries no value meaning "I edited a shipped
surface", so requiring one would mandate exactly the fabricated trailer G-0150 closed. No
`aiwf-actor` either: where no verb ran, "who ran the verb" is undefined, since an edit
directed in conversation has the human as principal and the assistant as a tool.

## Reasoning

The content mandate is the generator of the corpus the companion decision retires.
Policy-test references to embedded-ritual paths stood at 1 in late May and 10 in early
June; the backstop landed at the end of June; they reached 22, then 33, then 44 over the
following weeks. The mandate did not discover a need for those tests — it created the
obligation that produced them.

It also cannot discharge the obligation it creates. D-0050 says so directly: the
backstop "does not say what that test should assert, and its own v1 granularity — the
edited path appears as a string literal in some policy test — is satisfied by a test
that checks nothing in particular." A gate whose passing condition is satisfiable by a
vacuous test is a gate that measures compliance rather than correctness.

The repo's own code-health force names the shape of the error. A ban costs once; a
mandate costs per subject, and lands with a named owner and a retirement trigger or it
is a permanent tax. "Every skill edit needs a referencing test" is a mandate with
neither.

Provenance, by contrast, is mechanically checkable without judgment, costs the same
whether one skill changes or ten, and is what G-0220 asked for in the first place — the
question recorded there is whether there was a gap and how the edit was documented.

Alternatives considered and rejected:

- **Retire the backstop outright.** Returns the repo to the state G-0220 recorded, where
  a shippable edit reaches consumers with nothing noticing. The complaint was real even
  though the remedy was mis-aimed.
- **Narrow the content requirement to the two evidence-backed classes** — citation
  checks and trigger phrases. Still a mandate, and worse, one that fires on edits it
  cannot possibly apply to, since most skill edits touch neither class.
- **Strengthen it to "the test references the changed section"**, which is what G-0220
  itself proposed. D-0050's four measured failure mechanisms argue the opposite
  direction: the stronger the phrase-level claim, the more ways it fails while staying
  green. A stricter version of a check that cannot work is not a fix.

## Consequences

- The replacement is itself a mandate in H3's grammar — every skill edit needs a trailer —
  and it is accepted as permanent rather than given a retirement trigger, because none
  exists to name. Its owner is `CLAUDE.md` §"Ritual content authoring", and this decision
  is its record. The asymmetry with the rule it retires is deliberate: H3's requirement
  exists to bound accretion, and the two mandates accrete differently. The old one minted a
  durable artifact per subject — a test function to maintain, able to rot green, and the
  growth curve above is what that produced. This one mints a line in a commit message,
  paid once, leaving nothing behind to maintain and nothing that can drift. There is no
  population to bound, so what the trigger requirement protects against does not arise.
- Content drift in shipped prose is held at review rather than by a gate. This is the
  same trade the companion decision takes, and the two stand or fall together.
- This decision does not remove the existing assertion corpus; it removes the obligation
  that regrows it.
- CLAUDE.md's ritual-authoring and enforcement sections describe the current mandate and
  need updating in the same work.
- The entity-trailer requirement is already familiar machinery — the `provenance-untrailered-entity-commit`
  finding enforces the analogous property over entity files — so the new predicate reuses
  an established shape rather than inventing one.
- G-0504's separate complaint, that `aiwf doctor` byte-checks only verb skills while
  ritual and guidance drift read as healthy, is untouched by this decision and stays
  open.

## Provenance

Decided alongside the companion decision retiring the prose-assertion corpus, while
scoping G-0596. The archaeology that prompted it: reading G-0220 against the policy that
closed it showed a provenance complaint answered with a content mandate, and the growth
curve of policy-test path references showed the mandate generating obligations rather
than catching defects. E-0048 is the epic that shipped the original backstop.

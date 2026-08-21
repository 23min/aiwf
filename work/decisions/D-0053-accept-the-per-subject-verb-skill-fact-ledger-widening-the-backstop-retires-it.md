---
id: D-0053
title: Accept the per-subject verb-skill fact ledger; widening the backstop retires it
status: proposed
relates_to:
    - G-0520
    - G-0523
---
## Question

`internal/policies/verb_skill_factual_test.go` pins corrected facts in `aiwf-*`
verb skills — one hand-written test per fact, each added when a drift is found by
hand. It now carries five.

Two forces disagree about it. **D5 (findings become checks)** requires a
confirmed defect to leave behind a check that fails without the fix; a
skill-prose correction with no pin is the silent correction D5 forbids. **H3
(additions carry)** prefers an addition that costs once to one that costs per
subject, and holds that a mandate without a named owner and a stated retirement
trigger is a permanent tax. The ledger has neither.

Do further verb-skill fact corrections keep landing entries there?

## Decision

Yes for now — with a named retirement trigger, which is what H3 actually
requires.

Corrections continue to land a per-fact pin in `verb_skill_factual_test.go`,
section-scoped via the file's `sectionUnder` helper. The ledger is accepted as a
deliberate interim, not as the target design.

**What retires it:** D-0070's deletion pass. That decision retires prose-presence
assertions over shipped surfaces, scoped to the surface set the `skill-body-id`
check scans — which includes the verb-skill tree at `internal/skills/embedded/`.
Every entry in this ledger is such an assertion, so the ledger goes with them
rather than being generalised into a rule.

The trigger this decision originally named — widening the skill-edit backstop's
watched set so a verb-skill edit would require a referencing structural test —
is void. D-0071 re-pointed that backstop at provenance, and a provenance gate
cannot retire a content ledger: asking who owns an edit is a different question
from pinning what the prose says. The generalisation this decision anticipated
turned out to be the wrong end state, and the answer is deletion instead.

## Reasoning

D5 and H3 are each right about their own axis, and the resolution is not to pick
one. D5 governs the individual defect: leaving a confirmed skill-prose error
unpinned is exactly the silent correction it exists to prevent. H3 governs the
pattern: entries that are individually justified, unowned, and carried forever
are the accretion it warns about. Satisfying D5 now at the cost H3 objects to,
while discharging H3's actual requirement — the owner-and-retirement clause, not
a ban — settles both.

H3's once-cost form, here, is deletion rather than generalisation. Widening the
backstop to demand a structural test for every verb skill would have spread the
per-subject cost rather than removing it — a structural test for every existing
verb skill on its next edit, or a grandfather ledger of the kind the
firing-fixture gate already maintains. D-0070 measured the corpus these
assertions belong to and found no recorded catch across it, which settles the
question the other way: the entries go, and nothing replaces them.

The interim is cheap per entry and the entries are honest — each is verified to
fail against its own pre-fix text, so none is vacuous. The cost H3 names is real,
but it is bounded and now written down, which is the difference between an
interim and a tax.

## Follow-ups

- The skill-edit backstop's scope constant carries a comment stating the
  verb-skill tree is out of scope. That remains true under the provenance
  predicate, and for the same reason: G-0220 was about rituals.
- The ledger's deletion rides D-0070's pass. A reader arriving here after that
  pass should find the file gone and this decision explaining why it existed.

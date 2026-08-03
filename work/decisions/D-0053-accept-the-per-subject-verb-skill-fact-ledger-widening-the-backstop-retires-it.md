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

**What retires it:** widening `skillRitualsDir` in
`internal/policies/skill_edit_structural_test_backstop.go` to cover the
verb-skill tree (`internal/skills/embedded/`), so every verb-skill edit requires
a referencing structural test the way every ritual edit already does. When that
lands, new corrections stop needing a bespoke entry and the existing ones become
ordinary instances of a general rule.

## Reasoning

D5 and H3 are each right about their own axis, and the resolution is not to pick
one. D5 governs the individual defect: leaving a confirmed skill-prose error
unpinned is exactly the silent correction it exists to prevent. H3 governs the
pattern: entries that are individually justified, unowned, and carried forever
are the accretion it warns about. Satisfying D5 now at the cost H3 objects to,
while discharging H3's actual requirement — the owner-and-retirement clause, not
a ban — settles both.

Widening the backstop is the once-cost form H3 prefers, and it is the right end
state. It is not patch-sized: it would demand a structural test for every
existing verb skill on its next edit, or a grandfather ledger of the kind the
firing-fixture gate already maintains. Sizing that is its own work, and deciding
it here would be deciding it without the measurement.

The interim is cheap per entry and the entries are honest — each is verified to
fail against its own pre-fix text, so none is vacuous. The cost H3 names is real,
but it is bounded and now written down, which is the difference between an
interim and a tax.

## Follow-ups

- The backstop's scope constant carries a comment stating the verb-skill tree is
  out of scope. If the widening lands, that comment and this decision are what a
  reader should find from either direction.

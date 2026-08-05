---
id: G-0553
title: The aiwf-check Meaning cells drift from the rules they describe
status: open
priority: high
---
## What's missing

Every row of the `aiwf-check` skill's findings tables carries a Meaning cell
describing what makes the rule fire. Nothing checks those descriptions against
the rules. Placement is now derived and pinned, and the escalation sentences
are pinned in both directions, but the sentence a reader actually acts on —
*what condition produces this finding* — is unverified prose.

A row-by-row read of all 102 rows against their emitting functions found six
cells that state something the code contradicts:

- `acs-shape/tdd-policy` describes an AC at `tdd_phase: done` under a
  non-`required` milestone. The rule consults no AC at all: it fires when the
  milestone's own `tdd:` value falls outside the allowed set. The in-binary
  hint for the same code already says this correctly, so the two surfaces
  disagree.
- `acs-shape/tdd-phase` states two triggers joined by "or". The first —
  `tdd_phase` set on a milestone that is not `tdd: required` — fires on
  nothing; an absent phase is always legal and a present one is checked only
  against the closed phase set.
- `environment` documents a code with **no emission site anywhere**. Its
  constant is declared and never used. The condition the cell describes, a
  validator binary missing from PATH, emits a different code that resolves to
  `contract-config` with a subcode, at warning severity by default. So an
  errors-table row promises a push-block for something that cannot fire.
  Fix this together with the missing-row half recorded in G-0549 — they are
  two ends of one broken mapping.
- `gap-addressed-has-resolver` says the gap's `addressed_by` is empty. The
  rule requires `addressed_by` **and** `addressed_by_commit` both empty, so a
  gap closed by commit SHA — the path this repo's own guidance prescribes for
  a patch — never fires it.
- `no-cycles` and `no-cycles/supersedes` name the ADR `supersedes` field. The
  cycle graph is built exclusively from `superseded_by`. Both fields are real,
  which is what makes the substitution invisible.
- `provenance-authorization-ended` names a `revoke` verb. No such verb exists;
  the scope-ending trailer is written by a terminal promote or by cancel. The
  same wording sits in the hint table, so the operator-facing message is wrong
  too.

A further group is technically defensible but will send a reader the wrong
way. The pattern is uniform: the cell states the trigger and omits a guard
that decides whether the rule fires at all. Among them — `unexpected-tree-file`
and the pre-commit hook row claim the whole of `work/` when only four
subdirectories are walked; `load-error` claims `work/` when the ADR directory
is also a load root; `terminal-entity-not-archived` omits that milestones are
excluded outright; `entity-body-empty` reads as always-fires next to the word
"unconditionally" when terminal and archived entities are skipped;
`area-overlap` says "directory" where the matcher returns paths;
`fsm-history-consistent/illegal-transition` and
`provenance-untrailered-entity-commit` both omit their merge-commit carve-outs,
which is exactly what a reader debugging "why didn't my merge fire" needs;
`area-unknown` omits the reserved sentinel its sibling row names. Two Fix
cells prescribe a remedy for a case the rule cannot report, and one of those
(`case-paths`) tells the operator to `git mv`, which the shipped guidance
elsewhere forbids and which trips another finding.

## Why it matters

This is the surface a consumer reads to decide what to do about a finding, and
it is the copy furthest from the code. A wrong trigger is worse than a missing
one: it reads as authoritative, and the reader stops looking.

Two of the three blocking defects found while reviewing G-0542 were of exactly
this shape — a cell whose severity claim the code contradicted. That work
pinned the severity half. The trigger half is the larger population and is
still held only by review, which is how six of them survived to now.

The distribution is the useful signal. The rows rewritten during G-0542 all
checked out; the defects sit in rows nobody has touched since they were
written. Prose here does not rot on its own — it goes stale when the rule
moves underneath it, silently, because nothing re-reads the pair.

## Direction

Correcting the six false cells is the obvious half, and each correction needs
verifying against the predicate rather than against the old sentence — the
wrong wording is often plausible, which is why it lasted.

The part worth deciding first is how much of this can stop being prose. Three
sub-properties look mechanizable, in descending order of value:

- **A documented row whose code nothing emits.** `environment` would have been
  caught the moment such a check existed. The existing chokepoint asserts
  every emitted code is documented; the converse is unasserted, and the two
  together would close the set. The obstacle is telling a genuinely dead row
  apart from one whose emission the AST cannot resolve — the same limitation
  recorded in G-0068 and G-0549.
- **A cell naming a frontmatter field the rule never reads.** Both `no-cycles`
  rows would have been caught. Each finding already carries a `Field`, so the
  comparison has something real to check against.
- **A cell naming a verb that does not exist.** The skill-coverage policy
  already resolves backticked `aiwf <verb>` spans; `revoke` escaped because it
  appears as bare prose rather than as a command.

What stays judgment is the omitted-guard class, which is most of the volume.
No check can decide whether an unmentioned carve-out is a defect or an
acceptable simplification — that depends on whether a reader would act
differently knowing it.

## Provenance

Found 2026-08-05 by a review pass dedicated to this one question, run after
the G-0542 patch had been committed, covering all 102 finding-code rows plus
the hook table and the id-rule table. The conditions predate that patch: every
row it touched was verified accurate in the same pass.

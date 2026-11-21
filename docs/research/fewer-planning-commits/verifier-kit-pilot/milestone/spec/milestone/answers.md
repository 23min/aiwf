# Answers to the milestone readings' questions

Decided by human/peter on 2026-11-16, in conversation, from `spine diff` of reading-a against
reading-b and `spine check` of each. The chapter carries each answer at the rule it settles.

- **Q1. Does ADR-0036's same-status convergence cover an AC?** Yes. meet on a `met` AC, defer on
  a `deferred` AC, reopen on an `open` AC, and drop on a `deferred` or `cancelled` AC succeed
  with no commit. (Chapter rule 11.)
- **Q2. May the AC move while the milestone is `draft`?** Yes, every move rules 7 to 9 allow.
  (Chapter rule 10.)
- **Q3. May the AC change once the milestone is `done` or `cancelled`?** No: a finished
  milestone's AC is frozen, so a `met` AC can't be descoped after the milestone closed. One
  exception: an `open` AC on a finished milestone, an error state that only a merge
  (G-0575) or a hand edit produces, may still be met, deferred or cancelled, since closing it
  ends the error without reopening the milestone. A step that asks for the status already there
  still converges under Q1. Evidence weighed: `sources-ac-less.md`. (Chapter rule 10.)
- **Q4. Does a milestone whose only AC is `deferred` or `cancelled` count as having no ACs under
  D-0039?** Yes: start needs an AC that is `open` or `met`. (Chapter rule 6.)

Notation, not a question: where the readings differ only in writing a same-status milestone
step as `stay` with no output or as `ignore`, the spine writes `stay` with `outputs: []`, since
ADR-0036 makes it a success with no commit.

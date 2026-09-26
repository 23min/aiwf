---
title: A subtraction review on demand — does this change need everything it adds?
status: captured
date: 2026-09-25
---

# A subtraction review on demand

## Classifier note

This is a forward-looking initiative: a captured idea for a new skill,
awaiting promotion to an epic. The skill would ship in the generic workflow
plugin, beside `wf-review-code`, `wf-vacuity` and `wf-rethink`. Any repository
could use it, with or without aiwf. The document proposes a complement to the
existing review skills and records no defect in any of them.

## The idea

A skill that asks one question of a diff or a named unit: **does this change
need everything it adds?** It covers consolidation, trimming, shrinking, reuse,
and deletion.

A fresh agent answers by measurement, not by reading:

- a compression trial proven by a differential test;
- a caller named for every guard;
- a search for existing code that already does what the new code does;
- a break-to-tests table;
- every removal settled by breaking what it protected and watching something go red.

The skill proposes. The human approves each behaviour change before anything
is rewritten, and a second agent confirms each removal before it is committed.

## Why a separate skill

The review questions the rituals ask each tend to add code when answered.
Correctness finds an edge case to handle. Test sufficiency finds an assertion
to strengthen. The branch walk asks for a test on every branch. The code review
checklist even guards against removal ("no tests removed without an explicit
reason"). Asking what can go is a different question with the opposite
default. Put inside a checklist whose default is to add, it would be outweighed,
so it earns its own skill.

## How it relates to the skills around it

| Skill | The question it asks | Default direction | Scope | What an uncaught break means |
|---|---|---|---|---|
| `wf-review-code` | Is this change correct and complete? | Add: a test per change, a branch walk | One diff | A missing test |
| `wf-vacuity` | Can the tests fail at all? | Strengthen weak assertions | One unit's tests | Add or tighten a test |
| `wf-rethink` | Would I build this unit the same way from scratch? | Keep, unless a rebuild is simpler and keeps every obligation | One unit, design level | — |
| `wf-structural-sweep` | Where is the whole codebase carrying dead or duplicated weight? | Report, triage, file | Whole codebase, no trial | — |
| `wf-codebase-health` H1–H3 | Reuse, no dead weight, additions carry | Forces to hold while writing | Principles, no procedure | — |
| **this skill** | Does this change need everything it adds? | **Remove, unless something stops it** | One diff or unit, with trials | Is the guard needed at all? |

It borrows `wf-vacuity`'s break mechanism but reads the result the other way.
When a break goes uncaught, vacuity concludes that a test is missing. This
skill asks whether any caller can produce the state the guard catches:

- If none can, the guard goes, and so does the test the branch-coverage rule would demand for it.
- If one can, the guard stays and goes to `wf-vacuity` as "keep, needs a test".

The keep-or-remove table of guards is where the two directions meet instead of
undoing each other.

The skill states each procedure once and points at the others rather than
restating them:

- `wf-vacuity`'s mutation probe, and its "defer to a real tool first", for the break step;
- `wf-structural-sweep`'s "triage before you delete", for ownership;
- `wf-codebase-health` H1–H3 as the forces it applies after the fact.

## Where it runs

- **On demand**, over a diff or a named unit.
- **Milestone wrap**, as a third lens beside code quality (`wf-review-code`)
  and design quality (`wf-rethink`).
  - The wrap's inline "measure the change's shape" block is this skill's
    procedure written into a ritual: recurring obligations, deletions,
    same-outcome clusters, compression, over-guarding. It is replaced by a
    call to the skill, so the procedure has one source.
  - Test sufficiency stays with `wf-vacuity`, which the TDD cycle already
    invokes per criterion. The guard table's "keep, needs a test" list feeds it.
- **Patch wrap**, as a lens beside code quality, design quality and test
  sufficiency, which the patch ritual already runs. Today the patch ritual asks
  no shape question at all. For a change with no logic, the existing carve-out
  applies.
- **Order within a wrap:** the design and subtraction lenses run first. Their
  approved rewrites land. Test sufficiency runs over the guards that stayed.
  The deciding code-quality pass then reads the result. Run after the deciding
  review, every accepted cut reopens a review that has already passed.

## Usable downstream

The skill must work in any consumer repository, not only in the one that
builds aiwf:

- **Generic workflow skill.** No aiwf verb is required. Where aiwf is present,
  it points at the gap and decision recorders the way `wf-rethink` names the
  decision recorder: as an option, not a dependency.
- **Consumer-scoped text.** Imperative instruction only: no aiwf ids, no
  development history, no rationale war stories. The evidence below stays in
  this document.
- **Stack-neutral.** Each step is described by its method. A per-stack tool
  table names the mutation harness, the coverage profile and the clone
  detector for each stack, as `wf-structural-sweep` does.
- **Obligations from whatever the project states.** A specification,
  acceptance criteria, a ticket, or the user's instruction. Nothing assumes
  aiwf's entity model.
- **Minimum requirements:**
  - a runnable test suite;
  - a statement coverage profile;
  - version control that can hold an isolated working copy for trials.

  A mutation harness is optional: it strengthens the break step and is used
  when present.

## Evidence: two runs by hand

### Run 1 — a script patch

Observed 2026-09-25 on the G-0110 patch (`patch/G-0110-mutate-diff-changed-lines`),
in the devcontainer.

- **Size.** `git show <rev>:<path> | wc -l` measured two files on `main`
  (`47a10f9ce`) and at the committed fix (`4573c00af`):
  - `scripts/mutate-diff.sh`: 114 → 268 lines;
  - `internal/policies/mutate_diff_test.go`: 158 → 924 lines.

  The fix had been through three rounds of independent review, each finding
  handled as it came.
- **The trim.** A fresh agent ran `wf-rethink` with the obligation list split
  three ways, then the milestone wrap's compression and over-guarding
  questions. It applied a trial in an isolated worktree, which measured 187
  and 569 lines with the tests and lint green.
- **The confirmation.** Two further fresh agents reviewed the trial. One looked
  at code quality; the other broke the script line by line to see which
  breaks a test caught.

### Run 2 — a milestone's policy code

Observed 2026-09-25 on M-0333
(`milestone/M-0333-fence-project-guidance-while-preserving-managed-updates`),
in the devcontainer, Go 1.25.11. The scope was the milestone's non-frozen
logic: a reader list, a size ceiling, link resolution, a change to an existing
analysis engine, and their tests. One file due to be rewritten elsewhere was
excluded. The run came after the milestone's deciding review had passed.

- **The trim.** A fresh agent followed the sketch below with a three-group
  obligation list. Its verdict was "marginal".
  - Its trial cut the in-scope logic from 490 to 422 code lines (14%) with the
    gates green.
  - It proved the rewrites with a differential test: old and new
    implementations over seeded inputs (3,000 cases for the ceiling, 9,488
    for the engine) plus the real tree gave identical output.
- **What it found:**
  - two duplicates the change itself created, with a third blocked by scope;
  - four checks that could never fire;
  - twelve kept guards with no test;
  - one false positive;
  - one mismatch between the code's model of a host and that host's own documentation.
- **The confirmation.** A second fresh agent re-ran every claim and found two
  defects in the trim:
  - a rewrite introduced a branch no test reached;
  - a test removal the trim judged redundant under two narrower rules failed on its own under a third.
- **Applied, measured with `grep -cvE '^\s*(//|$)'` per file** (code lines,
  comments and blanks excluded):
  - logic went from 1,391 to 1,341 lines across five files;
  - tests went from 1,067 to 1,133 lines;
  - each of the twelve guards now fails its new test when broken;
  - the three behaviour fixes and one false coverage annotation landed with tests;
  - the measured outputs the milestone records were unchanged.

### What worked

- **Obligations in three groups** — stated (by the user, a specification or a
  decision), measured (a failure actually seen), and author-added. Everything
  in the third group was presumed cuttable until it showed a reason to stay,
  which let the trim cut with confidence.
- **Rebuilding before reading** the current code surfaced parts that existed
  only because of the order the code was built in.
- **A differential test for every rewrite.** Old against new, on generated
  inputs and the real tree, proves "same behaviour". A green suite proves only
  what the suite pins.
- **Breaking each kept guard on purpose** showed which guards a test protects
  and which none does.
- **Listing every behaviour change** for the human, before any rewrite, kept
  each cut a decision rather than a side effect.
- **Independent confirmation** caught defects in the trim itself in both runs.

### What went wrong, and what each asks of the skill

- **A guard was cut on a measurement that was wrong** (run 1). The trim
  reported that a git environment variable made no difference; the next
  reviewer re-ran it and it did. A removal's evidence has to be a command and
  its output that a second agent re-runs, not a conclusion.
- **Folding two similar uses onto one shared value created a defect** (run 1).
  The script's own file listing and the diff it hands the mutation tool came to
  share one option string that only the second needed. The listing broke in a
  way the green tests did not show. A merge needs its own check: does every
  user of the shared thing still get what it needs?
- **A consolidation of test helpers left four new copies of an existing
  helper** (run 1). The search for reusable code went by name; it has to go by
  what the code does.
- **After the trim, the test-sufficiency review asked for more assertions** on
  the guards that stayed (run 1). The two reviews pull in opposite directions,
  and nothing weighed them against each other. The guard table's "keep, needs a
  test" handoff is the answer.
- **One rule can be carried by several independent guards** (run 1). A single
  hostile setting in a test pinned only the colour flag among the several flags
  that together implement "ignore the operator's git config". Each of the
  others needed its own exercise.
- **A file allowlist hid a duplicate** (run 2). Its second copy sat in a file
  the change touched but the brief did not list, and the verdict counted only
  in-scope cuts. Scope by the change-set, with any frozen file marked "report,
  do not cut", and count cuts the scope blocked in the verdict.
- **Statement coverage hid twelve untested guards** (run 2). Each was one
  operand of a condition in a line that ran. Break conditions, not only
  statements.
- **A false "unreachable" annotation was found by accident** (run 2). A coverage
  exclusion claimed a branch could not run, and a test ran it. The coverage
  profile settles this mechanically: an exclusion on a line that executed is
  false.
- **A rewrite added a branch nothing tested** (run 2). Re-run the breaks on the
  rewritten code, not only on the original.
- **A test judged redundant was not** (run 2). Two hand-picked narrower rules
  broke it together with its neighbour, while a third broke it alone. Judge
  redundancy from a break-to-tests table: a test is a removal candidate only
  when every break that turns it red also turns another test red.
- **A behaviour fix needed a record edit the trim had not listed** (run 2).
  For each behaviour change, list every record and comment stating the old
  behaviour.
- **The model mismatch surfaced only as a side note** (run 2). For each rule
  the code encodes about an outside system, name the document that defines
  that system and compare the two.
- **Running after the deciding review reopened it** (run 2). Run before.

## Sketch of the skill

1. **Invocation.** On demand, or as a wrap lens. The input is a diff (a branch
   against its base, or what is staged) or a named unit. It reports and
   changes nothing without approval.
2. **Independence.** A fresh agent runs it. The author writes only the
   obligation list; the author re-reading their own code is what the skill
   replaces.
3. **Obligations in three groups** — stated, measured, author-added. Record
   each measured item with the input or test that reproduces it. The agent
   derives its own list from the stated sources before reading the author's,
   and reports the differences.
4. **Scope and shape.** The scope is the change-set, with frozen files marked
   "report, do not cut". Report lines added and removed against the base,
   split into logic, tests and prose, with the command recorded so the next
   reader re-runs it.
5. **Reuse, by what code does.** For each new function or helper, find existing
   code that does the same job, including copies the change itself creates and
   copies across files. Report older copies of a pattern the change joins as
   candidates for a wider sweep.
6. **Compression trial** in an isolated working copy. Aim for half the lines,
   run the gates, and name the constraint that stops you. Prove each rewrite
   with a differential test against the old implementation, then re-run the
   breaks on the rewritten code.
7. **Guards.** For each guard, including each coverage exclusion, name the
   caller that can produce the state it catches, or demonstrate that none can.
   Check each exclusion against the coverage profile. Break conditions as well
   as statements, using the mutation harness when one exists. A removal counts
   only when breaking what it protected turns something red, recorded as a
   command and its output.
8. **Merges.** When two uses are folded onto one shared thing, confirm with a
   check that each still gets what it needs.
9. **Tests.** Build the break-to-tests table. A test is a removal candidate
   only when every break that turns it red also turns another test red. Guards
   that stay without a test go to `wf-vacuity` as a list.
10. **Models.** For each rule about an outside system, compare the code with
    the document that defines that system.
11. **Report.** It holds:
    - a verdict that counts cuts the scope blocked;
    - the behaviour changes for approval, each with the records and comments that state the old behaviour;
    - a keep-or-remove table of guards with the evidence for each;
    - the handoffs.
12. **Independent confirmation** of every removal and rewrite claim, by a
    second fresh agent, before the change is committed.

## Non-goals

- **Not a gate.** It advises; it never blocks a commit or a push.
- **Not whole-codebase discovery** — that is `wf-structural-sweep`, which this
  skill may recommend for a pattern the change joins.
- **Not a design rebuild** — that is `wf-rethink`, which this skill may
  recommend.
- **Not a test-sufficiency review** — that is `wf-vacuity`, which receives the
  guards that stay without a test.
- **Not an auto-applier.** Every rewrite waits for the human.
- **Not a line-count contest.** Half the lines is a probe that finds what
  resists removal, not a target to hit.

## Open questions

| Question | Notes |
|---|---|
| Name and home | Working name `wf-trim`, in the generic workflow plugin beside `wf-review-code`, `wf-vacuity` and `wf-rethink`. |
| The milestone wrap's shape block | Replaced by a call to the skill. Settle which of its five questions move whole and whether any stays in the ritual. |
| Cost in the patch wrap | Two fresh agents per patch is heavy for a small change. Settle a threshold — logic lines changed, or new guards — below which the lens is skipped with the skip stated at the commit gate. |
| Size limit | How large a diff one agent can take before it goes shallow. The milestone wrap slices large reviews by concern. |
| How to test the skill | Shipped prose cannot be pinned by phrase assertions. Decide what relationship check, or recorded trial, shows the skill works. |
| Measuring its value | Re-run it over a few recent patches and milestones and compare what it finds against what landed. |
| Other stacks | The per-stack tool table: the mutation harness, the coverage profile and the clone detector per stack, and what the skill does where one is missing. |

## Provenance

Captured 2026-09-25 from a session patching G-0110, after the hand-run trim
described under Run 1. Extended the same day with Run 2, on M-0333, and with
the placement of the skill among the wrap lenses and its downstream
requirements.

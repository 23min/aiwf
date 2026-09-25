---
title: A subtraction review on demand — does this change need everything it adds?
status: captured
date: 2026-09-25
---

# A subtraction review on demand

## Classifier note

This is an initiative document in the forward-looking tier: a captured idea for a
new skill, awaiting promotion to an epic. It proposes a complement to the existing
review skills and records no defect in any of them.

## The idea

A skill an operator runs on demand, over a diff or a named unit, that asks one
question: **does this change need everything it adds?** A fresh agent answers it by
measurement rather than by reading: a compression trial, a caller named for every
guard, a search for existing code that already does what new code does, and every
proposed removal settled by breaking what it protected and watching something go red.
It proposes; the human approves each behaviour change before anything is rewritten.

## Why a separate skill

The review questions the rituals ask each tend, when answered, to add code:
correctness finds an edge case to handle, test sufficiency finds an assertion to
strengthen, and the branch walk asks for a test on every branch. Asking what can go
is a different question with the opposite default, and it earns its own skill rather
than a clause in another.

The skills around it each do their own job, and this one sits beside them:

| Skill | Its job | What this adds |
|---|---|---|
| `wf-rethink` | Rebuilds one unit's design from its obligations; keeps unless the rebuild is simpler and keeps every obligation | The removal question at any size, with a trial and per-guard evidence rather than a design rebuild |
| `aiwfx-wrap-milestone` step 2 | Measures a milestone's change at close, including compression and over-guarding | The same kind of questions, on demand, outside a milestone |
| `wf-codebase-health` H1–H3 | Forces to hold while writing: reuse, no dead weight, additions carry | A procedure that applies them to a change after the fact |
| `wf-structural-sweep` | Whole-codebase discovery of dead paths, clones and unconsumed data | A focused pass over one change or unit |
| `wf-vacuity` | Whether the tests are strong enough | The other direction: whether the code under test needs to exist |

## Evidence: one run by hand

Observed 2026-09-25, on the G-0110 patch (`patch/G-0110-mutate-diff-changed-lines`),
in the devcontainer.

- **Size.** `git show <rev>:<path> | wc -l` gives `scripts/mutate-diff.sh` at 114
  lines on `main` (`47a10f9ce`) and 268 at the committed fix (`4573c00af`), and
  `internal/policies/mutate_diff_test.go` at 158 and 924. The fix had been through
  three rounds of independent review, each finding handled as it came.
- **The trim.** A fresh agent ran `wf-rethink` with the obligation list split three
  ways (below), then the milestone wrap's compression and over-guarding questions,
  applying a trial in an isolated worktree. The trial measured 187 and 569 lines with
  the tests and lint green.
- **The confirmation.** Two further fresh agents reviewed the trial: one for code
  quality, one breaking the script line by line to see which breaks a test caught.

What worked:

- **Obligations in three groups** — stated by the user or the gap, measured (a
  failure actually seen), and author-added — let the trim cut with confidence:
  everything in the third group was presumed cuttable until it showed a reason to
  stay.
- **Rebuilding before reading** the current code surfaced parts that existed only
  because of the order the fix was built in.
- **Breaking each kept guard on purpose** showed which guards a test protects and
  which none does.
- **Listing every behaviour change** for the human, before any rewrite, kept each cut
  a decision rather than a side effect.

What went wrong, and what each one asks of the skill:

- **A guard was cut on a measurement that was wrong.** The trim reported that a git
  environment variable made no difference; the next reviewer re-ran it and it did.
  A removal's evidence has to be a command and its output that a second agent
  re-runs, not a conclusion.
- **Folding two similar uses onto one shared value created a defect.** The script's
  own file listing and the diff it hands the mutation tool came to share one option
  string that only the second needed, and the listing broke in a way the green tests
  did not show. Merging needs its own check: does every user of the shared thing
  still get what it needs?
- **A consolidation of test helpers left four new copies of an existing helper.** The
  search for reusable code went by name; it has to go by what the code does.
- **After the trim, the test-sufficiency review asked for more assertions** on the
  guards that stayed. The two reviews pull in opposite directions and nothing weighed
  them against each other.
- **One rule can be carried by several independent guards.** A single hostile setting
  in a test pinned only the colour flag among the several that together implement
  "ignore the operator's git config"; each of the others needed its own exercise.

## Sketch of the skill

1. **Invocation.** On demand, over a diff (a branch against its base, or what is
   staged) or a named unit. It reports and changes nothing without approval.
2. **Independence.** A fresh agent runs it. The author writes only the obligation
   list; the author re-reading their own code is what the skill replaces.
3. **Obligations in three groups** — stated, measured, author-added — as above.
4. **Shape.** Lines added and removed against the base, split into logic, tests and
   prose, with the command recorded so the next reader re-runs it.
5. **Reuse, by shape.** For each new function or helper, find existing code that does
   the same job, including copies the change itself creates.
6. **Compression trial** in an isolated worktree: aim for half the lines, run the
   gates, and name the constraint that stops it. A green run settles a rewrite, not a
   removal.
7. **Guards.** For each one: the caller that can produce the state it catches, or a
   demonstration that none can. A removal counts once breaking what it protected turns
   something red, recorded as a command and its output.
8. **Merges.** When two uses were folded onto one shared thing, confirm each still
   gets what it needs.
9. **Tests.** One test per rule. Group tests by the outcome they claim; a test is
   removed only when breaking the code shows another test still catches it. Guards
   that stay without a test go to the test-sufficiency lens as a list.
10. **Report.** A verdict; the behaviour changes for approval; a keep-or-remove table
    of guards with the evidence for each; and the handoffs.
11. **Independent confirmation** of every removal claim before the change is
    committed.

## Non-goals

- **Not a gate.** It advises; it never blocks a commit or a push.
- **Not whole-codebase discovery** — that is `wf-structural-sweep`.
- **Not a design rebuild** — that is `wf-rethink`, which this skill may recommend.
- **Not an auto-applier.** Every rewrite waits for the human.
- **Not a line-count contest.** Half the lines is a probe that finds what resists
  removal, not a target to hit.

## Open questions

| Question | Notes |
|---|---|
| Name and home | Working name `wf-trim`, in the `wf-rituals` plugin. A shipped ritual materializes into consumer repos, so its text is consumer-scoped: no aiwf ids, no history. |
| On demand only, or also invoked by rituals? | A patch or milestone wrap could call it. If the milestone wrap keeps its own compression questions too, which text is the source? |
| Weighing it against test sufficiency | How to stop one review undoing the other: the guard table's "keep, needs a test" handoff is one answer. |
| Size limit | How large a diff one agent can review before it goes shallow; the milestone wrap slices large reviews by concern. |
| How to test the skill | Shipped prose cannot be pinned by phrase assertions (D-0070). What relationship check, or recorded trial, shows the skill works? |
| Measuring its value | Re-run it over a few recent patches and compare what it finds against what landed. |
| Other stacks | The trial and the break-it step assume a runnable test suite and a mutation or manual-break route. |

## Provenance

Captured 2026-09-25 from a session patching G-0110, after the hand-run trim
described under *Evidence*.

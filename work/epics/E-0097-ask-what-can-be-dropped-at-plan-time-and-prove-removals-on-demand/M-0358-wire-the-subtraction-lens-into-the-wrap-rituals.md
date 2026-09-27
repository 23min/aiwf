---
id: M-0358
title: Wire the subtraction lens into the wrap rituals
status: in_progress
parent: E-0097
depends_on:
    - M-0357
tdd: none
acs:
    - id: AC-1
      title: The skill answers the four shape questions and the wrap asks only the obligation
      status: open
    - id: AC-2
      title: The patch ritual runs the lens above its threshold and states the skip below it
      status: open
    - id: AC-3
      title: The tracked gap's measurement is re-run and no longer holds
      status: open
---

## Goal

Give the four line-measuring shape questions one home by having the milestone wrap
call the skill, keep with the wrap the question of what a change obliges every later
change to do, and give the patch ritual the lens behind a stated threshold.

## Closes

- G-0662 — the patch ritual reaches the shape questions through the lens.

## Context

The milestone wrap holds five shape questions inline — the only questions at wrap
that ask whether something is surplus rather than missing. The patch ritual asks
none of them, which G-0662 records as a measurement against that ritual's body, on
the argument that the patch is the higher-traffic surface. Which questions move and
which stays is settled in this epic's constraints.

## Acceptance criteria

### AC-1 — The skill answers the four shape questions and the wrap asks only the obligation

The milestone wrap obtains the four line-measuring answers from the skill and asks,
itself, only what the change obliges every later change to do. **Pass criterion**: a
recorded wrap on a real milestone where the skill was invoked, its report answered
the four, and the ritual's own question produced the obligation record in the
milestone spec, each named obligation carrying its owner and what retires it; the
record holds command, expectation, observation and environment. **Edge cases**: a
milestone below the skip threshold, where the skip is stated and the obligation
question is still asked; a milestone with no logic, where the four are stated
inapplicable and the obligation question is still answered. **Code references**: the
embedded `aiwfx-wrap-milestone` ritual body.

### AC-2 — The patch ritual runs the lens above its threshold and states the skip below it

The patch ritual runs the lens above its threshold, and below it states the skip and
its reason at the commit gate where the human can veto it. **Pass criterion**: two
recorded patches, one each side of the threshold, the first carrying the lens's
report and the second a stated skip naming the threshold it fell under. **Edge
cases**: a patch with no logic at all, which already carries a review carve-out and
does not acquire a second statement; a patch changing few lines but adding a guard,
which the threshold catches on guards rather than on lines. **Code references**: the
embedded `wf-patch` ritual body.

### AC-3 — The tracked gap's measurement is re-run and no longer holds

G-0662's measurement no longer holds. **Pass criterion**: that gap's own measurement
re-run against the patch ritual's body and reported with its command and output,
showing the shape questions reachable from that ritual where each previously scored
zero. **Edge cases**: a question reachable only through the skill, which counts as
reachable and is recorded as such rather than as present inline. **Code
references**: `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md`;
closure rides `aiwf promote G-0662 addressed --by-commit <sha>`.

## Constraints

- **The division is drawn on output.** The skill may report an obligation it finds;
  naming that obligation's owner and what retires it stays the wrap's requirement.
  One observation, two outputs, never two copies.
- No procedure is restated: both rituals call the skill.
- A skipped lens is stated where the human can veto it, with its reason.
- The threshold is set from logic lines changed or guards added, is stated in the
  ritual where a reader meets it, and lands with what retires it.
- Shipped text carries no aiwf id, no path into this tree, and no rationale.

## Design notes

- The epic's constraints hold the argument for which questions move; this milestone
  applies it rather than re-deciding it.
- G-0660's repaired oracle travels with the compression question: a removal is
  never settled by a green run. The skill settles it by something going red, or,
  for a guard, by the demonstration that no caller reaches its state; a cut that
  changes no result is a rewrite, settled by a differential test. The wrap block's
  "Nothing red is a surviving mutant, not a clearance" contradicts the second route
  until the block calls the skill.
- `wf-trim` joins the milestone wrap's independent review as a third lens beside
  code quality and design quality, and the shape block keeps only the obligation
  question. The block's exclusion of mandated comments from half, and its wording
  for the Deletions question, are not carried: where compression cannot reach half,
  the skill names the constraint that stops it, a mandated comment among them, and
  every empty section of its report states why it is empty. The skill is unchanged.
  Calling it also settles a rewrite by a differential test, where the block settles
  one by a green gate run.
- The lens adds no human step. Its report joins the other lenses' findings, and its
  proposals are approved with theirs once every review has returned; the skill's
  second-agent confirmation runs after that approval and before any cut is
  committed. The patch ritual takes the same shape, with its commit gate as the
  approval point.
- The patch ritual's threshold is any change to logic: a patch that changes only
  tests, prose or configuration skips the lens and says so at the commit gate. Over
  the 60 most recent patch merges, 25 changed logic. The threshold is retired —
  widened to test-only patches — if a lens run over patches it skipped finds a cut
  their review missed, and leaves with the lens itself under the skill's retirement
  trigger.

## Surfaces touched

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md`
- `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md`

## Out of scope

- Changing what the skill itself does.
- The commit-gate question naming what a change adds beyond the task.
- Porting the lens to the epic wrap.

## Dependencies

- M-0357 — the skill both rituals call.

## References

- G-0662 — the patch ritual asks none of the wrap's shape questions.
- G-0660 — the repaired oracle for a proposed removal.
- G-0585 — rituals that clear a question by reading it.

## Release note

The milestone wrap and the patch ritual now call `wf-trim` as a review lens instead
of asking the line-measuring shape questions inline. In `aiwfx-wrap-milestone` the
independent review runs three lenses — code quality, design quality and subtraction —
and keeps only one shape question of its own: what the change obliges later changes
to do, each obligation named with its owner and what retires it, recorded under the
spec's `## Reviewer notes`. In `wf-patch` the subtraction lens runs whenever a patch
changes logic; a patch that changes only tests, prose or configuration states the
skip at the commit gate. In both rituals the lens applies nothing: its proposals
reach the human with every other lens's findings once all reviews have returned.

## Decisions made during implementation

- None — the wrap-lens design and the patch threshold are pre-locked under Design notes.

## Validation

Observed 2026-09-27 in the Linux development container: Claude Code 2.1.283,
`claude-opus-5-5`, Go 1.25.11.

**Method.** Each trial replays a real change in a scratch clone of this repository
that keeps its aiwf configuration, since both rituals call aiwf verbs. The rituals
were materialized by a binary built from this branch (`aiwf update --no-prompt` in a
throwaway clone) and copied into each trial clone's `.claude/` before its session
started: `wf-patch` `8b8d2a99…` for the first round, `1e7e3135…` (`19263a6ef`) for
the prose re-run, and `aiwfx-wrap-milestone` `97474cc7…` throughout. Each session
was a fresh interactive session in its clone, confirmed through `/proc/<pid>/cwd`
before its prompt was sent, and told to stop at the first human decision and apply,
commit and promote nothing. A session-start sync outside aiwf left an unstaged
`CLAUDE.md` edit and untracked `AGENTS.md` and `.ai-dotfiles/` in two clones; none
is in any diff under review.

| Run | Input | Prompt | Lens outcome |
|---|---|---|---|
| Wrap | M-0321, `a1f974207..214069a1a` | `aiwfx-wrap-milestone` step 2, stop where findings go to the human | `wf-trim` ran: 2 removals, 1 rewrite, 3 cuts blocked |
| Patch, logic | G-0695, 3 logic lines staged on `d4b4edf27` | `wf-patch` steps 5–8, stop at the commit gate | `wf-trim` ran: 0 removals, 1 rewrite |
| Patch, prose | G-0652, prose only, staged on `d5c243d41` | same | stated skip: "No subtraction lens: no logic changed." |

**M-0358 AC-1.**
- The wrap dispatched `wf-trim` as a third lens beside code quality and design
  quality. Its report answers the four line-measuring questions: what was retired
  (Shape and Removals), whether tests fail for distinct reasons (the break-to-tests
  table settling both removals), compression (a trial on the largest unit, 24 lines
  to 28, with a differential test, reported as not proposed), and each guard's
  caller (the guard table, a break command per row).
- The wrap's own question wrote seven obligations into M-0321's `## Reviewer notes`
  in the working tree, each with its owner and what retires it. The lens reported
  obligations as findings; the wrap's answer named their owners rather than
  repeating them.
- Every lens's findings reached the human once, together; nothing was applied and
  HEAD stayed `214069a1a`.
- Edge case, a milestone below a skip threshold: inapplicable. The milestone wrap
  runs the lens on every milestone and has no threshold.
- Edge case, a milestone with no logic: M-0358's own wrap, recorded under
  `## Reviewer notes`.

**M-0358 AC-2.**
- Above the threshold, G-0695: the lens ran as its own fresh agent with the code
  quality and test sufficiency lenses. Its rewrite reached the commit gate with the
  other lenses' fixes; the staged diff matched its pre-dispatch fingerprint.
- Below the threshold, G-0652: the gate's lens table states "No subtraction lens: no
  logic changed.", quoting the rule's "prose (shipped instructions included)"; code
  review still ran, because the review carve-out did not apply, so the skip is its
  own statement rather than a second copy of the carve-out's.
- Edge case, few lines adding a guard: G-0695's three logic lines are the guard, and
  the lens ran on them.
- An earlier prose run on `8b8d2a99…`, whose rule said "non-test code", ran the lens
  on the same prose-only patch, reading a shipped skill as code. `19263a6ef`
  restates the threshold in the skill's own logic / tests / prose terms, and the
  re-run above observed the skip.

**M-0358 AC-3.** G-0662 records its result — each of the five questions scores
zero in `wf-patch` — and no command. Reconstructed as a count of each question's
label as the wrap writes it:

```bash
P=internal/skills/embedded-rituals/plugins
for q in 'Recurring obligation' 'Deletions' 'Same-outcome clusters' 'Compression' 'Over-guarding'; do
  git show <rev>:$P/wf-rituals/skills/wf-patch/SKILL.md | grep -c "\*\*$q\.\*\*"
done
```

- At `9ce6f8fda`, where G-0662 was filed: 0 for each question in `wf-patch`, 1 for
  each in `aiwfx-wrap-milestone` — the gap's measurement reproduced.
- At `19263a6ef`: 0 inline in both rituals, and `wf-patch` names `wf-trim` once.
  Through the skill each question is reachable: compression at step 4, guards at
  step 5 and report § Guards, removals at step 6 and report § Removals, same-outcome
  tests at step 8's break-to-tests table, and obligations at report § "Obligations
  the change adds for later changes".
- Obligations are reached as findings only. Naming each one's owner and what retires
  it stays with the milestone wrap, whose spec records the answer; `wf-patch` has no
  such record and does not ask it.

**Gates**, on `milestone/M-0358-wire-the-subtraction-lens-into-the-wrap-rituals` at
`19263a6ef`: `make check-fast` → exit 0; `go test -count=1 -run
TestClaudeArtifacts_MatchBaseline ./internal/cli/integration/` → `ok`; `aiwf check` →
0 errors.

## Deferrals

- (none)

## Reviewer notes

- (none)

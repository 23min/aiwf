---
id: M-0358
title: Wire the subtraction lens into the wrap rituals
status: done
parent: E-0097
depends_on:
    - M-0357
tdd: none
acs:
    - id: AC-1
      title: The skill answers the four shape questions and the wrap asks only the obligation
      status: met
    - id: AC-2
      title: The patch ritual runs the lens above its threshold and states the skip below it
      status: met
    - id: AC-3
      title: The tracked gap's measurement is re-run and no longer holds
      status: met
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
  committed. The patch ritual takes the same shape: proposals go to the human once
  every lens has returned, and an approved cut is applied like a blocking fix before
  the commit gate.
- The patch ritual's threshold is any change to code, tests included: a patch that
  changes only prose (shipped instructions included) or configuration skips the lens
  and says so at the commit gate. Tests count because a test-only patch can add a
  check every later change must pass, and the lens reports what such a patch obliges
  later changes to do: of the 15 test-only patches in the sample below (every `.go`
  row a `_test.go` file), 3 added a policy test file (an `A` row under
  `internal/policies/` ending `_test.go`). Of the 60 most recent patch merges on
  `main` at `ad30ac6f0`, 40
  changed a Go file: `git log --merges --first-parent --format='%H
  %s' main | grep -E ' Merge (branch .)?patch/' | head -60`, each merge `m` counted
  when `git diff --numstat m^1 m` has a `.go` row. If a lens run over
  configuration-only patches the threshold skipped finds a cut their review missed,
  the lens is widened to run on configuration-only patches too; the threshold leaves
  with the lens itself under the skill's retirement trigger.

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

The milestone wrap and the patch ritual now call `wf-trim` as a review lens. In
`aiwfx-wrap-milestone` it replaces the line-measuring shape questions the wrap asked
inline; in `wf-patch`, which asked none of them, it is a new lens. In the wrap the
independent review runs three lenses — code quality, design quality and subtraction —
and keeps one shape question of its own: what the change obliges later changes
to do, each obligation named with its owner and what retires it, recorded under the
spec's `## Reviewer notes`. In `wf-patch` the subtraction lens runs on a reviewed
patch that changes code, tests included; a patch that changes only prose or
configuration states the skip at the commit gate. In both rituals the lens applies
nothing: its proposals reach the human once all reviews have returned.

## Decisions made during implementation

- The patch ritual's threshold is any change to code, tests included; Design notes
  carry its reasoning and what retires it. Counting test code departs from this
  spec's constraint that the threshold is set from logic lines or guards added.

## Validation

Observed 2026-09-27 in the Linux development container: Claude Code 2.1.283,
`claude-opus-5-5`, Go 1.25.11.

**Method.** Each trial replays a real change in a scratch clone of this repository
that keeps its aiwf configuration, since both rituals call aiwf verbs. The rituals
were materialized by a binary built from this branch (`aiwf update --no-prompt` in a
throwaway clone) and copied into each trial clone's `.claude/` before its session
started: `wf-patch` `8b8d2a99…` for the logic run and the shipped `0460911f…` for
the prose run, and `aiwfx-wrap-milestone` `97474cc7…` for the wrap run. The logic
run's text differs from what ships only in the threshold sentence, which read
"non-test code"; a patch changing logic runs the lens under that wording and the
shipped one. The wrap run's text differs from what ships in three lines. It
acted on the earlier wording of two: the obligation paragraph, which now also asks
for the counting command and records the answer at step 4, so the run's obligation
answer names no command; and the lens bullet, whose new clause governs an approved
cut, past the point the run stopped at. Step 4's `## Reviewer notes` line, which
now names the obligation answer, lies beyond that point. This milestone's own
obligation answer, under `## Reviewer notes`, names its counting commands. Each session
was a fresh interactive session in its clone, confirmed through `/proc/<pid>/cwd`
before its prompt was sent, and told to stop at the first human decision and apply,
commit and promote nothing. A session-start sync outside aiwf left unstaged edits to
`CLAUDE.md` and `AGENTS.md` (untracked in the prose clone) and an untracked
`.ai-dotfiles/` in two clones; none is in any diff under review.

| Run | Input | Prompt | Lens outcome |
|---|---|---|---|
| Wrap | M-0321, `a1f974207..214069a1a` | `aiwfx-wrap-milestone` step 2, stop where findings go to the human | `wf-trim` ran: 2 removals, 1 rewrite, 3 cuts blocked |
| Patch, logic | G-0695, 3 logic lines staged on `d4b4edf27` | `wf-patch` steps 5–8, stop at the commit gate | `wf-trim` ran: 0 removals, 1 rewrite |
| Patch, prose | G-0652, prose only, staged on `d5c243d41` | same | stated skip: "No subtraction lens: no code changed." |

**M-0358 AC-1.**
- The wrap dispatched `wf-trim` as a third lens beside code quality and design
  quality. Its report answers the four line-measuring questions: what was retired
  (Shape and Removals), whether tests fail for distinct reasons (the break-to-tests
  table settling both removals), compression (a trial on the largest unit, 24 lines
  to 28, with a differential test, reported as not proposed), and each guard's
  caller (the guard table, a break command per row).
- The wrap's own question wrote seven obligations into M-0321's `## Reviewer notes`
  in the working tree, each with what retires it; six name an owner, and the seventh
  states that it has none. The lens reported
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
- Below the threshold, G-0652, on the shipped text: the gate states "No subtraction
  lens: no code changed.", classing the shipped skill edit as prose; code
  review still ran, because the review carve-out did not apply, so the skip is its
  own statement rather than a second copy of the carve-out's.
- Edge case, a patch taking the review carve-out: not observed; no trial patch was
  trivial enough to skip independent review. `wf-patch` step 6 states that the
  carve-out's statement covers the lens skip.
- Edge case, few lines adding a guard: G-0695's three logic lines are the guard, and
  the lens ran on them.
- The threshold names shipped instructions as prose because a rule worded as
  "non-test code" was observed to send a prose-only patch through the lens, reading a
  shipped skill as code.

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
- At this milestone's head: 0 inline in both rituals, and `wf-patch` names `wf-trim`
  once, run on every patch that changes code. Through the skill each question is
  reachable: compression at step 4, guards at
  step 5 and report § Guards, removals at step 6 and report § Removals, same-outcome
  tests at step 8's break-to-tests table, and obligations at report § "Obligations
  the change adds for later changes".
- Obligations are reached as findings only. Naming each one's owner and what retires
  it stays with the milestone wrap, whose spec records the answer; `wf-patch` has no
  such record and does not ask it.

**Gates**, on `milestone/M-0358-wire-the-subtraction-lens-into-the-wrap-rituals`
after the review's corrections: `make check-fast` → exit 0; `go test -count=1 -run
TestClaudeArtifacts_MatchBaseline ./internal/cli/integration/` → `ok`; `aiwf check` →
0 errors.

## Deferrals

- (none)

## Reviewer notes

- Deciding review, over `cd48c204e..HEAD`: the shipped rituals, the test change and
  the inventories hold; its findings were in this spec's evidence records and are
  corrected here. The loop closed on a fresh scoped confirmation of those
  corrections, not on a further full pass. Doc-lint over the change-set: clean.
- **Obligations this milestone adds**, counted with `git diff --name-status
  cd48c204e..HEAD` (every file modified, none added), `git diff cd48c204e..HEAD --
  '*.go' | grep -cE '^\+func|^\+\s*Policy:'` → 0, and a read of the four ritual
  diffs; no mechanical rule is added.
  - Every milestone wrap runs `wf-trim` as a third lens. Owner: `aiwfx-wrap-milestone`
    step 2. Retires with the skill, under its retirement trigger.
  - Every patch that changes code runs `wf-trim`, and every other patch states the
    skip at the commit gate. Owner: `wf-patch` steps 6 and 8. Retires as Design notes
    state.
  - Every milestone records its obligation answer under `## Reviewer notes`. Owner:
    `aiwfx-wrap-milestone` step 2. Retires when a check derives the obligations a
    change adds, so the answer no longer needs writing.
- **M-0358 AC-1, a milestone with no logic.** This wrap's own subtraction lens ran
  over `cd48c204e..d95ed7077` in a clone of the milestone branch, Linux, Go 1.25.11.
  It reported logic +0 −0 and stated compression and guards inapplicable, skipping
  steps 3–5, 7 and 9 with that reason. The change has a tests bucket (the D-0054
  locator and the frozen inventories), so the skill ran steps 6 and 8 on it as
  written, rather than stating all four questions inapplicable as this criterion's
  edge case expects. No change leads the skill to state all four inapplicable: with
  neither logic nor tests it leaves step 6 neither skipped nor run, which G-0723
  records. What the edge case exists to show holds: the obligation question is
  answered where there is nothing to compress and no guard to justify. Each break
  was an exact-string edit reverted with `git
  checkout -- <file>`, judged by `go test -count=1 -run
  TestD0054_FixedAndPinnedDispositionAcrossSurfaces ./internal/policies/` and `go
  test -count=1 -run TestClaudeArtifacts_MatchBaseline ./internal/cli/integration/`,
  both `ok` before any break:
  - the test's heading locator reverted → the D-0054 test red;
  - the wrap's step 2 heading reverted → the D-0054 and baseline tests red;
  - the wrap's subtraction bullet, the wrap's obligation paragraph, or `wf-patch`'s
    lens paragraph deleted → the baseline test red, each alone.

  No test is a removal candidate. The obligation question was still answered, above.
- Declined: dispatching the lens alongside the others rather than before the
  deciding review. Its cuts are approved once every review has returned and land as
  corrective commits before the fresh full pass that decides, which is the ordering
  the skill asks for.
- Declined: cutting the wrap bullet's one-line description of the lens as a restated
  procedure; each sibling lens bullet describes its lens in one line.
- Declined: naming `wf-trim` in the Codex review-dispatch fragment, which lists the
  review lenses by example and names neither the test-sufficiency lens nor this one;
  the rituals name the lens where they dispatch it.
- Known edges of `wf-patch`'s threshold, left as written: test data and golden
  files are neither code nor prose, and usually travel with the tests that read them;
  a configuration-only patch that adds a check skips the lens, which is what the
  threshold's widening trigger watches for.
- A patch with no linked issue gives the lens no ticket to derive stated obligations
  from; the skill's step 1 then derives them from what the user asked for.
- G-0662's closure rests on the M-0358 AC-3 record: the four line-measuring
  questions reach every patch that changes code through the lens, and obligations
  reach it as the lens's findings, with owner and retirement asked only at the
  milestone wrap.

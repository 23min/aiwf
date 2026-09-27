---
name: wf-trim
description: Ask of one diff or one named unit whether the change needs everything it adds — duplicated jobs, logic that compresses, guards no caller reaches, tests no break needs — and settle every proposed removal by a command — breaking what it protected turns something red, or a guard is shown unreachable. Proposes and reports; applies nothing without approval. Use before the deciding review of a patch or milestone, when a change looks bigger than its task, or when the user invokes wf-trim.
---

# wf-trim

One question, asked of one diff or one named unit: **does this change need everything it adds?** The review skills around it each default to adding — a test per branch, a stronger assertion, a kept design. This one defaults the other way: everything the change added is a removal candidate until something stops it. What stops it is always a measurement — a constraint named, a caller shown, a command going red — never a reading.

It proposes. The human approves each behaviour change before anything is rewritten, and a second fresh agent confirms each removal before it is committed.

## When to use

- Before the deciding review of a patch or a milestone, so accepted cuts land before the review that reads the result.
- A change looks bigger than the task it served.
- The user names a diff or a unit and asks what can go.

Not for the whole codebase — that is `wf-structural-sweep`. Not for a design rebuild — that is `wf-rethink`. Not for test sufficiency — that is `wf-vacuity`. This skill recommends each of them where its findings point there.

## Independence

Run it as a **fresh agent** that did not write the change, briefed per `wf-review-code` §"Independence". The author writes only the obligation list (step 1). The agent derives every number and every verdict itself; a figure the author supplies about their own change is what independence replaces.

## Requirements

- A runnable test suite.
- A statement coverage profile.
- Version control that can hold a separate working copy for trials.

A mutation harness is optional; use it when present and runnable. See §"Per-stack tools".

## Workflow

Do every trial in a **separate working copy** — a worktree or a clone — never in the checkout a commit will be made from. Mutate and revert as `wf-vacuity` §"Mutation probe" specifies. Record every measurement as the command and its output, so the next reader re-runs it instead of re-reasoning it.

### 1. Obligations, in three groups

- **Stated** — by the user, a specification, acceptance criteria, a ticket, a recorded decision.
- **Measured** — a failure actually observed. Record the input or test that reproduces it.
- **Author-added** — everything else the change does.

Derive the stated group yourself from the sources before reading the author's list, and report where the two differ. Everything in the author-added group is cuttable until it shows a reason to stay. Before proposing to remove anything, triage it as `wf-structural-sweep` §"Triage before you delete" specifies: an open issue or a coupled change can own code no obligation names.

### 2. Scope and shape

The scope is the change-set: every file the diff touches, not a hand-picked list. Mark any file the change may not alter as **frozen — report, do not cut**.

Count lines added and removed against the base, split into logic, tests and prose, in the project's own terms:

```bash
git diff --numstat <base> HEAD
```

Where there is no logic bucket, say so, skip steps 3–7 and 9, and report the skip with its reason. Run step 8 when there is a tests bucket; stop after step 2 when there is neither.

### 3. Reuse, by what the code does

For each new function or helper, search for existing code doing the same job — by behaviour, not by name. Include copies the change itself creates, and copies across files. List older copies of a pattern the change joins as candidates for `wf-structural-sweep`.

### 4. Compression trial

Take the largest logic bucket. Write a version that does the same job in **half** the lines, apply it in the working copy, and run the gates. If you cannot reach half, report how far you got and name the constraint that stops you — an interface you do not control, a branch each arm needs, an invariant the project enforces.

Prove each rewrite with a **differential test**: old and new implementations over generated inputs and the project's real data, with identical output. A green suite proves only what the suite pins. Then re-run step 5's breaks on the rewritten code.

Half is a probe that finds what resists removal, not a target.

### 5. Guards

For each guard the change adds — each coverage exclusion included — name the caller that can produce the state it catches, or demonstrate that none can.

- **A caller can** → the guard stays. If no test breaks when the guard is broken, list it for `wf-vacuity` as "keep, needs a test".
- **None can** → the guard is a removal candidate, and so is the test written only to cover it. Demonstrate it: show the type, invariant or check that forbids the state. Not thinking of a caller is not a demonstration.
- **An exclusion on a line the coverage profile shows executed** is false. Report it.

Break conditions, not only statements: one operand of a condition can be unguarded on a line that ran. Use the mutation harness where one exists.

Defaults: compression cuts unless a constraint stops it; a guard stays unless proven dead. Where one cut falls under both, the guard's default governs.

### 6. Settling a removal

A removal is settled in one of two ways, recorded as the command and its output:

- **Red** — breaking what the removed thing protected turns something else red: a test, a check, a gate.
- **Dead** — for a guard, step 5's demonstration that no caller can produce the state it catches, beside the break that shows nothing goes red.

A green run after a cut is an absent witness, not a verdict. A break that turns nothing red on a guard a caller *can* reach is a surviving mutant: the guard stays and goes to `wf-vacuity` as "keep, needs a test".

### 7. Merges

Where two uses are folded onto one shared thing, run a check per use confirming each still gets what it needs. A shared value only one of them needed breaks the other where the green suite does not look.

### 8. Tests — the break-to-tests table

Record which tests each break turns red. A test is a removal candidate only when every break that turns it red also turns another test red. One rule can be carried by several independent guards; each needs its own break.

### 9. Rules about outside systems

For each rule the code encodes about a system outside the project — a host, a format, a protocol — name the document that defines that system and compare the two. Report each disagreement as a behaviour change awaiting approval, or hand it to `wf-rethink`.

### 10. Report, then gate

Emit the report (below). Every behaviour change waits for explicit human approval before any rewrite, and lists every record and comment that states the old behaviour. Apply nothing yourself.

### 11. Confirmation

Before any approved removal or rewrite is committed, a **second fresh agent** re-runs every command the report cites and confirms or refutes each claim. A removal the second agent cannot reproduce — red, or dead by demonstration — does not land.

## Per-stack tools

| Stack | Mutation harness | Coverage profile | Clone detector |
|---|---|---|---|
| Go | `gremlins` | `go test -coverprofile=cover.out ./...`, read with `go tool cover -func` | `golangci-lint run --enable-only dupl ./...` — the linter runs without a separate `dupl` binary |
| Python | `mutmut` | `coverage run --branch -m pytest`, then `coverage report` | `pylint --disable=all --enable=duplicate-code` |
| JavaScript / TypeScript | Stryker | `c8` or `nyc` | `jscpd` |
| JVM | PIT | JaCoCo | PMD CPD |

Before relying on a tool, check whether the project switches it off for part of the tree — a clone detector excluded from test files, say — and report that part as uncovered by it. Where a stack lacks a tool: no mutation harness, or one that cannot run here → the manual probe in `wf-vacuity`, and say why in the report; no clone detector → step 3 by reading, and say so; no coverage profile → check exclusions by breaking their lines instead, and say so.

## Output format

```markdown
# Trim — <diff or unit>

**Verdict:** <N removals proposed, M rewrites proposed, K cuts the scope blocked> — <one line>

## Shape
<command> → <logic / tests / prose, added and removed>

## Behaviour changes — awaiting approval
- <change> — records stating the old behaviour: <file:line, …>

## Removals
- <what> — protected: <state> — settled: `<command>` → <red output, or dead: the demonstration>

## Guards — keep or remove
| Guard | Caller, or demonstration none exists | Break command → result | Disposition |
|---|---|---|---|

## Rewrites
- <what> — differential test: `<command>` → <result>

## Blocked by scope
- <cut> — blocked by <frozen file / out-of-scope file>

## Obligations the change adds for later changes
- <rule, check or required artefact any future change must satisfy>

## Handoffs
- wf-vacuity: <guards kept without a test>
- wf-rethink: <unit whose design the findings question>
- wf-structural-sweep: <older copies of a pattern the change joins>

## Skipped
- <step> — <reason>
```

Every section is present. One with nothing in it says so and why.

If the project tracks gaps or decisions — the `aiwfx-record-gap` and `aiwfx-record-decision` rituals, or its own tracker — file a finding nobody will act on now there, rather than leaving it in the report.

## Anti-patterns

- *Settling a removal on a green run.* Break what it protected; watch for red.
- *Clearing a guard because no caller came to mind.* Show the type, invariant or check that forbids the state.
- *Scoping by a file list.* A duplicate in a touched file the list omitted goes uncounted.
- *Counting only cuts in scope.* The verdict counts cuts the scope blocked.
- *Trusting statement coverage for guards.* Break conditions.
- *Running after the deciding review.* Every accepted cut then reopens a review that already passed.
- *Applying a cut in the report's own pass.* Report, gate, confirm, then commit.

## Constraints

- 🛑 Proposes and reports; never rewrites, removes or commits without explicit human approval.
- 🛑 Every removal carries a command and its red output, or the demonstration that no caller can produce the state it guarded.
- 🛑 Trials run in a separate working copy; the checkout under review ends byte-identical.
- Advisory: never a gate on a commit or a push.
- One diff or one unit per run. A large diff is split by concern, one agent per slice.

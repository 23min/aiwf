---
name: aiwfx-record-gap
description: Records a defect found in passing as an aiwf gap — decides whether it is a gap at all, drafts the body from the gap template, reproduces the claim so a later reader can re-run it, and reviews the batch before it is committed. Use when the user says "file a gap", "record this defect", "capture this", "note this for later", "that's a gap", or when a defect surfaces mid-flow and the work in hand is something else. Calls aiwf — requires the aiwf binary.
---

# aiwfx-record-gap

Records a defect as a gap entity, with the claim reproduced rather than asserted.

## When to use

A defect surfaced and the work in hand is something else. Invoke this instead of
reaching straight for `aiwf add gap`, which validates and commits a body you have
already written but never prompts you to open the template.

Gaps usually arrive in batches — one pass through a file, one review, one
debugging session turns up several. Run this ritual once over the batch, not once
per gap. A single gap is a batch of one.

## Is this a gap at all?

Settle this before writing anything. Open `.claude/templates/gap.md` first: it
states what earns a gap — that it names where, and that something breaks — and it
is where those rules are maintained. Two further tests it does not carry:

- **It is one defect.** A body enumerating four concerns cannot close in any
  shape. Split it, or write the container as an epic.
- **It is not a decision.** An argued position on how to fix something is a
  decision, not a defect — hand off to `aiwfx-record-decision` instead.

## Workflow

### 1. Draft each body from the template

Read `.claude/templates/gap.md` and fill it. If the directory is absent the
templates are not materialized — run `aiwf update`. Never reconstruct a body by
copying an existing gap, which drifts from the template and drops its shape.

The template's own sections carry the gap. Everything else the defect suggests
has a home already:

| what you have | where it goes |
| --- | --- |
| a file or a symbol — where the defect shows, and where a fix would land | `## What's missing` |
| the command, the expectation, the observation | `## What's missing` |
| what breaks while this stays open — the consequence | `## Why it matters` |
| where it was found | the `discovered_in` field |
| what it relates to, and anything else that cannot go stale | a section of your own |
| an argued position on the fix, with alternatives | a decision entity |
| a sequence of steps to carry out | an epic or a milestone |

Add a section of your own only for something that cannot go stale. A section
proposing a fix dates the file the moment the plan changes, and pre-empts a
decision that belongs to whoever picks the gap up.

### 2. Reproduce the claim

Run the thing. Write into `## What's missing` the command, what you expected, what
you saw, and where it ran. A defect stated as a measurement can be re-run by the
next reader; one stated as a conclusion has to be taken on trust, and the tree
moves underneath it.

**When there is nothing to run, say so in one line and name what a reader should
look at instead** — an absent detector, a missing seam, and a shape claim about a
tree are all real defects with no command behind them. Never invent a command or
paste output you did not see. A fabricated reproduction reads as evidence, which
makes it worse than none.

### 3. Independent review of the batch — not self-review

Dispatch a *fresh-context* reviewer (a subagent with no authorship attachment)
over the drafted bodies, before anything is committed. Where a fresh reviewer is
already in the loop — a ritual that just dispatched one over its own subject —
extend that brief to cover the drafts rather than opening a second dispatch; the
requirement is that someone who did not write them reads them, not that a
subagent is spawned here. Re-reading your own draft
is not a substitute — it shares the context that produced the claim, which is the
failure this step exists to close.

Brief the reviewer to return a **record, not a verdict**. "Confirmed" is a claim
nothing re-checks, and it goes stale exactly like the prose it was meant to
validate. For each gap it returns:

- **The reproduction, re-run.** The command, what it printed, and where. If the
  draft's reproduction does not reproduce, that is the finding — the claim is
  wrong, or the tree has already moved.
- **Non-reproducible, with the reason**, where step 2 named no command. The
  reviewer confirms there is nothing to run rather than inventing something.
- **Whether the defect names where**, and whether the consequence is a consequence
  rather than a preference.
- **Anything misfiled**, against the table in step 1.

**Reviewer-dispatch contract.** The drafts are uncommitted until step 4, so name
their paths in the brief — the reviewer reads the draft files directly. Reading
committed state instead reaches an earlier tree that excludes the subject, and a
reviewer following that route reports on gaps nobody asked about.

The reviewer must not mutate the tree the ritual is about to commit from — no
`git stash`, no in-place edits, no branch switch. Re-running a reproduction is
not mutating: running the command the draft names is the point of the step. Where
a check genuinely needs to alter something, it copies to a scratch directory.

Fix what the review finds, in the draft, before step 4. A finding that changes a
claim sends that gap back through step 2.

### 4. File each gap

```
aiwf add gap --title "<what is wrong — the defect, not the fix>" --body-file <path>
```

Optional: `--discovered-in <id>` for the milestone or epic it surfaced in,
`--priority urgent|high|medium|low` for triage. In a multi-clone setup pass
`--fetch` so the id is allocated against the freshest published view.

The body lands in the create commit. A gap has no draft phase — it is live and
referenceable the moment the commit lands, so `aiwf add` refuses a body that omits
a required section or leaves one empty.

### 5. Mirror the ids back

Report each `G-NNNN` to whatever was in flight, so the caller can carry on. If a
ritual dispatched here, hand the ids back and continue where it left off.

## Anti-patterns

- **Batching the review away.** A filing pass shares one reviewer dispatch, not
  none — the batch is what makes the review affordable, not what excuses it.

## Constraints

- One gap per defect; one ritual run per filing pass. A filing pass is one
  declared-sequence gate, not one gate per gap.
- Priority is triage, not severity — leave it unset rather than guessing.

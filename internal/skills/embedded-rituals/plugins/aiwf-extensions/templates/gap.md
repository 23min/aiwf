---
# Field vocabulary — the allowed statuses, which fields are optional, and
# what each reference accepts — is printed by `aiwf schema gap`;
# the value set behind a flag is in `aiwf add gap --help`.
id: G-NNNN
title: <what is wrong — the defect, not the fix>
status: open
priority:
discovered_in:
---

<!-- How to use this file. A gap is born complete: `aiwf add` refuses to create one
     that omits a required section or leaves one empty, so the body is written
     first and lands in the create commit, not after it.

       1. Copy this file and delete the `---` block above — it is field reference,
          and `aiwf add` writes the real frontmatter itself. Body content passed with
          its own frontmatter is refused, since the two blocks would concatenate into
          a file the loader cannot parse.
       2. Fill the two sections below.
       3. `aiwf add gap --title "<title>" --body-file <your-file>`

     The `aiwfx-record-gap` ritual walks these steps and reproduces the claim
     before the create commit; reach for it rather than filling this by hand.

     `aiwf edit-body G-NNNN` is for revising the body later. Delete this comment. -->

A gap records **one defect**, not a plan. A body enumerating several concerns cannot
close in any shape: split it, or write the container as an epic. Whether the fix
lands as a patch or a milestone is decided by whoever reads the gap; writing that here
splits one decision across two places and dates the file the moment the plan changes.

The sections below are required. Add one of your own only for something that
cannot go stale — what it relates to, or where the gap came from. Where the
`--discovered-in` flag can name the origin, it records that instead. Do not add one
that proposes a fix: no options, no direction, no resolution shape. An argued
position on how to fix it is a decision, so record it as one; a sequence of steps is
work, so write it as an epic or a milestone.

## What's missing

What is wrong, and **where** — a file, a symbol, or an observable behaviour a
reader can go and look at. Name both ends when they differ: where the defect shows,
and where a fix would land. Naming the file a fix would touch is a location and
belongs here; naming what the fix should do is a plan and does not.

A gap that cannot name where is not a defect, it is a wish. If there is nothing to
point at, what you have is work you want done: write it as an epic or a milestone,
where a plan belongs.

Say what you measured rather than what you infer — the command, what you expected,
what you saw, and where it ran. Paste the command and its output here rather than in
a section of its own. A defect stated as a measurement can be reproduced by the next
reader; one stated as a conclusion has to be taken on trust.

When there is nothing to run — an absent detector, a missing seam, a claim about a
tree's shape — say so in one line and name what a reader should look at instead.
Never invent a command or paste output you did not see: a fabricated reproduction
reads as evidence, which makes it worse than none.

Keep this short — with the one exception that a measurement is worth its space.

## Why it matters

What breaks while this stays open. Name the consequence — what fails, who notices,
what class of error it lets through. Keep this short — with the one exception that
a defect with more than one consequence gives each its own paragraph. A fix, a plan,
or where the gap came from does not belong here; the top of this file says where
each goes.

"It is untidy" is not a consequence. If nothing breaks, this is a preference, and a
preference does not need an entity.

---
id: G-NNNN
title: <what is wrong — the defect, not the fix>
status: open             # aiwf gap statuses: open | addressed | wontfix
priority:                # optional: urgent | high | medium | low
discovered_in:           # optional: the milestone or epic where this surfaced
---

<!-- How to use this file. A gap is born complete: `aiwf add` refuses to create one
     whose sections are empty, so the body is written first and lands in the create
     commit, not after it.

       1. Copy this file and delete the `---` block above — it is field reference,
          and `aiwf add` writes the real frontmatter itself. Body content passed with
          its own frontmatter is refused, since the two blocks would concatenate into
          a file the loader cannot parse.
       2. Fill the two sections below.
       3. `aiwf add gap --title "<title>" --body-file <your-file>`

     `aiwf edit-body G-NNNN` is for revising the body later. Delete this comment. -->

A gap records a **defect**, not a plan. Two sections, both short, and nothing
else — no options, no direction, no resolution shape. Where the fix lands, and
whether it is a patch or a milestone, is decided by whoever reads the gap; writing
it here splits one decision across two places and dates the file the moment the
plan changes.

## What's missing

What is wrong, and **where** — a file, a symbol, or an observable behaviour a
reader can go and look at. One paragraph.

A gap that cannot name where is not a defect, it is a wish. If there is nothing to
point at, what you have is work you want done: write it as an epic or a milestone,
where a plan belongs.

Say what you measured rather than what you infer — the command, what you expected,
what you saw. A defect stated as a measurement can be reproduced by the next
reader; one stated as a conclusion has to be taken on trust.

## Why it matters

What breaks while this stays open. Name the consequence — what fails, who notices,
what class of error it lets through — in one paragraph.

"It is untidy" is not a consequence. If nothing breaks, this is a preference, and a
preference does not need an entity.

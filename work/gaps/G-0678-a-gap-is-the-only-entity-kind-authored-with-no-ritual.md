---
id: G-0678
title: A gap is the only entity kind authored with no ritual
status: open
---
## What's missing

A gap is authored with no ritual. In
`internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/`, rituals cover
four of the six kinds: `aiwfx-plan-epic` and `aiwfx-plan-milestones` for epics and
milestones, `aiwfx-record-decision` for ADRs and decisions. Each reads that kind's
template and walks the author through it. Gap and contract have none. For a gap the
whole path is `aiwf add gap` through the `aiwf-add` skill, which validates and
commits a body you have already written — it never prompts you to open the
template.

Nothing reads the gap's claim afterwards either. `wf-patch` dispatches an
independent reviewer, but §6 of its `SKILL.md` scopes that reviewer to the staged
diff — the fix — so the claim that motivated the work is the one artefact in the
loop no second party checks.

Measured 2026-09-14 against `main`:

```
$ grep -rniE "review.{0,40}\bgap\b|\bgap\b.{0,40}review" internal/skills/ | wc -l
0

$ # active gap bodies carrying a command inside a fenced block
8 of 201
```

The template asks every gap to state the command, what was expected and what was
seen. Eight of 201 do. Separately, 131 of 201 carry a section beyond the two the
template names, which is the same absence seen from the other side: no step
between writing a gap and committing it compares what was written to what the
template asks for.

## Why it matters

A gap's claim is the input to every later decision about it — whether to fix it,
how urgently, and whether it is still live at all. Where the claim cannot be
reproduced, a reader cannot tell a live defect from one the tree has already moved
past, and the only way to find out is to re-derive the whole thing from the prose.
At 8 of 201 that is not the exception, it is how gaps are normally read.

The cost lands on whoever picks the gap up, which is rarely whoever filed it, and
it compounds: the longer a gap sits, the more the tree has moved underneath the
claim, and the less the prose alone can settle.

---
id: G-0723
title: wf-trim step 2 leaves step 6 unstated for a change with neither logic nor tests
status: open
discovered_in: M-0357
---
## What's missing

`wf-trim`'s step 2 does not say whether step 6 runs on a change with neither a
logic nor a tests bucket. The rule, at
`internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-trim/SKILL.md` step 2,
reads:

> Where there is no logic bucket, say so, skip steps 3–5, 7 and 9, and report the
> skip with its reason. Run steps 8 and 6 when there is a tests bucket. Go on to
> step 10 either way.

Step 6 is neither skipped nor run for a change with neither bucket. Its routes are
something going red, a guard shown unreachable, and a differential test; only the
first can reach a document, and only where a check reads it.

Observed 2026-09-27 on the text at `36e5c1494`: a fresh session ran the skill over a
prose-only diff (three documentation files, +51 −5) in a clone carrying no aiwf
configuration. Its report states "steps 3–5, 7, 8 and 9 are skipped …, and step 6 is applied to prose",
supplies its own settling rule for prose ("a removal is held by an obligation that
states it, by a reader that depends on it, or by a check that goes red without it"),
and proposes removing a 46-line document for approval on the ground that no stated
obligation requires it.

## Why it matters

A prose-only change gets whatever settling rule the running agent invents, and a
reader approving a cut the run proposes has no stated rule to judge it against.

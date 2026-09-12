---
id: G-0670
title: Shipped skills name internal iteration labels and a repo filesystem path
status: open
discovered_in: E-0091
---
## What's missing

Five shipped verb skills name labels that belong to this repo's own
development and resolve to nothing in the consumer repo the skill
materializes into. They divide by mechanism, not by appearance:

Labels naming an iteration of this repo's plan:

- `aiwf-history/SKILL.md:8` and `aiwf-promote/SKILL.md:121` — "the I2.5
  provenance trailers"
- `aiwf-history/SKILL.md:56` — "Pre-I2 promote commits don't carry `aiwf-to:`"
- `aiwf-contract/SKILL.md:211` — "Manifest extension lands in I2 of the
  contracts plan."
- `aiwf-list/SKILL.md:14` — "V1 filter axes:"

Real gap ids written without their hyphen, which read as labels of the same
kind:

- `aiwf-history/SKILL.md:57` — "Pre-`G37` reallocates", naming G-0037
- `aiwf-promote/SKILL.md:104` — "the `G24` recovery story", naming G-0024
- `aiwf-authorize/SKILL.md:65` — "`G23` reserves a future `--allow-force`
  … sub-agent delegation is `G22`", naming G-0023 and G-0022

Some of the facts behind these are legitimate and survive a fix. A consumer's
git log really does contain promote commits predating the `aiwf-to:` trailer,
and reallocates predating `prior_ids` — past states a reader can still
encounter, so they read as current truth about the input space. It is the
label naming *when* they arrived that carries no meaning outside this repo.

## Why it matters

`skill-body-id` fires on a real entity id or an off-width placeholder in a
shipped surface, and reaches neither shape here. So the rule that a shipped
surface carries no development history is enforced for one of the things it
names, and the rest accumulate: none of these has ever been reported.

An iteration label also decays differently from an id. An id at least resolves
to something a reader can look up; `I2.5` resolves to nothing at all, and a
consumer meeting it has no way to learn what it means or whether it still
applies.

The unhyphenated ids are a second mechanism wearing the first one's clothes.
They name live entities, so `skill-body-id`'s own polarity says they are
defects — but the rule anchors detection on a literal hyphen after the kind
prefix, so it never classifies them as candidates. That narrowing is G-0369's
subject; these are instances of it in the shipped tree rather than in an
entity body.

## Scope

The path clause belongs to G-0548, which owns the whole population of this
repo's filesystem paths cited in shipped surfaces and has yet to settle which
kinds of citation count. `aiwf-add/SKILL.md:250` is two of those citations,
and is the design-doc shape G-0548 flags as possibly legitimate. It is left
there rather than settled here under a different gap's name.

## Related

- G-0548 — owns the filesystem-path clause of the same shipped-surface rule.
- G-0369 — the hyphen-anchored token pattern that hides the unhyphenated ids.

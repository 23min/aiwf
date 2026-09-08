---
id: G-0670
title: Shipped skills name internal iteration labels and a repo filesystem path
status: open
discovered_in: E-0091
---
## What's missing

Four shipped skill bodies name this repo's internal iteration labels, which mean
nothing in the consumer repo the skill materializes into:

- `internal/skills/embedded/aiwf-history/SKILL.md:8` — "the I2.5 provenance
  trailers"
- `internal/skills/embedded/aiwf-history/SKILL.md:56` — "Pre-I2 promote commits
  don't carry `aiwf-to:`"
- `internal/skills/embedded/aiwf-contract/SKILL.md:211` — "Manifest extension
  lands in I2 of the contracts plan."
- `internal/skills/embedded/aiwf-promote/SKILL.md:121` — "the I2.5 provenance
  trailers"

A fifth instance is a filesystem path rather than a label:
`internal/skills/embedded/aiwf-add/SKILL.md:250` cites
`docs/archive/pocv3/acs-and-tdd-plan.md` as a bare backticked path in prose. That
path exists in this repo and in no consumer's.

The facts behind two of them are legitimate and should survive a fix. A
consumer's git log really does contain promote commits predating the `aiwf-to:`
trailer, which is the past state a reader can still encounter; and the trailer
set the history skill lists is real. It is the label naming *when* they arrived
that carries no meaning outside this repo.

## Why it matters

`skill-body-id` fires on a real entity id or an off-width placeholder in a
shipped surface, and reaches neither of these shapes. So the rule that a shipped
surface carries no development history and no filesystem path is enforced for
exactly one of the three things it names, and the other two accumulate: these
five instances all predate the check and none has ever been reported.

An iteration label also decays differently from an id. An id at least resolves to
something a reader can look up; `I2.5` resolves to nothing at all, and a consumer
meeting it has no way to learn what it means or whether it still applies.

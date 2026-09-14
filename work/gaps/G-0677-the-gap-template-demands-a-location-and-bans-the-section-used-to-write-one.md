---
id: G-0677
title: The gap template demands a location and bans the section used to write one
status: addressed
addressed_by_commit:
    - f952d05fb
---
## What's missing

The gap template demands a location and then bans the section authors reach for
to write one. In
`internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/gap.md`, the
`## What's missing` section asks for "a file, a symbol, or an observable behaviour
a reader can go and look at", while the opening paragraph allows "two sections,
both short, and nothing else". The same section also asks the author to state the
command, the expectation and the observation, and caps itself at one paragraph.

Measured 2026-09-14 over the 201 gap files in `work/gaps/`, by counting `## `
headings and classifying what sits under each:

    131 of 201 carry a third section
     36 `## Resolution shape`   24 `## Scope`   23 `## Related`
     20 `## Problem`            18 `## Where to fix`
      8 of 201 carry a reproduction — a command inside a fenced block

`aiwf check` on that tree reports 0 errors: none of this is a finding at any
severity. Per entity the shape is re-derivable from `aiwf show <id> --format
json`, whose `result.body` is keyed by heading slug.

Reading the content rather than the heading: `## Problem` appears in 20 gaps and
in 0 of them alongside `## What's missing`, so it is that section renamed. The
shortest `## Where to fix` sections are file-and-symbol lists with no prose — the
location the template already demands, written where it was not banned. Every gap
that does carry a reproduction runs four to eight paragraphs, so no gap can obey
the one-paragraph cap and the measurement rule at once.

## Why it matters

The template has no chokepoint; being read is its only power, so a rule an author
cannot follow discredits the rules in the same file that they could. The measured
cost is that the evidence lands where no reader knows to look: locations and
reproductions scatter into ad-hoc sections, and the rule asking for a reproduction
is followed in 8 of 201 gaps. A later reader who cannot reproduce a gap's claim
cannot tell a live defect from one the tree has already moved past, and the only
way to find out is to re-derive the whole thing.

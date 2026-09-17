---
id: G-0665
title: No surface owns what an acceptance criterion body holds
status: addressed
discovered_in: E-0091
addressed_by_commit:
    - db514ab28213dd5bd30f2136d2289c3de9179abd
---
## What's missing

Two shipped surfaces define what an acceptance criterion body carries, and
neither is the owner. `internal/skills/embedded/aiwf-add/SKILL.md` gives three
parts: the assertable claim, the edge cases a test must cover, and the code or
test file the criterion lands against. The milestone-spec template's own
placeholder gives a looser set — examples, edge cases, and references to
decisions and surfaces touched. Eleven shipped markdown files mention acceptance
criteria at all.

Neither states what does *not* belong, which is the half a rule against growth
needs. The two are not in conflict: a claim plus its evidence accommodates all
three of the verb skill's parts. What is absent is any record of which surface a
new constraint on the body should be written into, so such a rule can only be
added as one more unowned statement beside the two that exist.

## Why it matters

G-0659 measures criterion bodies growing after the promote that accepted them,
each increment landing on a review round. What grows is prose defending the
claim rather than stating it, and stopping that needs a rule about what the body
excludes — a rule that has to live where the body is defined.

An attempt to write it lands nowhere. Added to the verb skill it governs a
command rather than the artefact; added to the template it reaches the author
once, at scaffold time; added to a ritual it becomes a third voice on a subject
two surfaces already speak to. The choice is not hard, but nothing records it,
and until something does, the anti-growth half of the routing rule cannot ship.

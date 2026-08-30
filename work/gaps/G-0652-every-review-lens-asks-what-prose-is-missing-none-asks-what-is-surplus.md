---
id: G-0652
title: Every review lens asks what prose is missing; none asks what is surplus
status: open
---
## What's missing

Every review lens in the shipped tree asks whether something is absent or
wrong. None asks whether something is surplus.

Measured 2026-08-30 over `internal/skills/embedded-rituals/` and
`internal/skills/embedded-guidance/`: no surface asks what prose a change could
remove. `wf-doc-lint` enumerates its checks in its own description — broken code
references, removed-feature docs, orphan files, documentation TODOs, broken
links, stale CLI invocations, structural drift — and every one fires on
something missing or unresolvable. It reports only and never rewrites prose, and
nothing downstream of it decides a paragraph should go.

The compression question added to `aiwfx-wrap-milestone` step 2 under G-0650 is
the shape that would work, and deliberately does not apply here: it is scoped to
the largest *logic* bucket, and puts the comments a project mandates, tests
pinning distinct rules, and planning prose out of scope. That exclusion is right
for its own subject — a lens told to shrink a change without the bucket split in
front of it attacks exactly the prose the rules mandate. So the prose case needs
its own scope boundary, not a widening of that question.

What that boundary must protect is already written down: the guidance fragment
holds that a judgment, a rejected alternative, and why the obvious approach
fails are each worth their words, while a restated fact another record owns is
not. A removal lens that gets this backwards deletes the reasoning and keeps the
arithmetic — worse than the accretion it was built to stop.

## Why it matters

Reported from a consumer repo: across successive review rounds prose was added
every round and removed in none, and one claim came to sit in several places,
drifting between them. The mechanism is structural rather than a lapse of
discipline — a round has an add path and no remove path, so volume only
ratchets.

The corpus-level measurement is on file. G-0595 carries it with the command
behind each finding: live planning records widely assert what the kernel
contradicts while `aiwf check` reports the tree clean, because every defect is
semantic. G-0636 is one instance in full — a single fact about milestone-spec
section timing restated across five shipped surfaces, two of which already
disagree, changing what an agent does depending on which it loaded.

Writing rules do not reach this. The rules against copying a fact and against
rewriting from memory are both shipped and both were in force while that drift
accumulated; they govern how a sentence is written, not whether the round that
added it also removed one. D-0070 rules out the mechanical check that would pin
prose content on a shipped surface, so for that tier a review question is the
only available instrument.

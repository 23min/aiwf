---
id: G-0652
title: Every review lens asks what prose is missing; none asks what is surplus
status: open
---
## What's missing

The review lenses in the shipped tree ask whether prose is missing, whether it is
wrong, and whether it narrates. None asks whether a paragraph restates a fact
another record already holds — the prose the guidance fragment says is not worth
its words — so a review round has questions that add prose and none that removes
it.

The narration questions cover one shape. `wf-review-code` §"Documentation" flags
a comment narrating a defect a reader can no longer encounter and a body
paragraph retracting an earlier claim, and `aiwfx-wrap-milestone` step 2 asks
whether each evidence claim belongs, failing an account of how the text or the
work got there. A true, present-tense sentence repeating what a check, a field or
another record carries passes all of them.

Measured 2026-09-16 on `main` at `4bd05ac13`:

```
$ R=internal/skills/embedded-rituals/plugins
$ grep -n -i -E 'restat|second copy|another record|duplicat|could remove|surplus' \
    $R/wf-rituals/skills/wf-review-code/SKILL.md \
    $R/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md \
    $R/aiwf-extensions/agents/reviewer.md \
    $R/wf-rituals/skills/wf-rethink/SKILL.md \
    $R/wf-rituals/skills/wf-doc-lint/SKILL.md | sed "s|$R/||" | cut -c1-90
wf-rituals/skills/wf-review-code/SKILL.md:100:**A blocking defect is fixed *and pinned*.**
```

That one hit is "duplicating what already has an owner", about filing a tracked
issue for a finished fix, not about prose. `wf-doc-lint` enumerates its checks in
its own description — broken code references, removed-feature docs, orphan
files, documentation TODOs, broken links, stale CLI invocations, structural
drift — and every one fires on something missing or unresolvable. It reports only
and never rewrites prose, and nothing downstream of it decides a paragraph should
go.

The compression question in `aiwfx-wrap-milestone` step 2 is the shape that would
work, and deliberately does not apply here: it is scoped to the largest *logic*
bucket, and puts the comments a project mandates, tests pinning distinct rules,
and planning prose out of scope. That exclusion is right for its own subject — a
lens told to shrink a change without the bucket split in front of it attacks
exactly the prose the rules mandate. So the prose case needs its own scope
boundary, not a widening of that question.

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
semantic. G-0636 records one instance in full: a single fact about milestone-spec
section timing, restated across five shipped surfaces, two of which disagreed, so
what an agent did depended on which it loaded.

Writing rules do not reach this. The rules against copying a fact and against
rewriting from memory are both shipped and both were in force while that drift
accumulated; they govern how a sentence is written, not whether the round that
added it also removed one. D-0070 rules out the mechanical check that would pin
prose content on a shipped surface, so for that tier a review question is the
only available instrument.

---
id: G-0594
title: Two authoring rules shipped, and the next spec written breached both
status: open
discovered_in: E-0086
---
## What's missing

Two authoring rules shipped to trunk in one session. G-0592 put a genre
instruction in both spec templates: a spec states what will be built, and the
reasoning behind a choice is referenced rather than reproduced. G-0593 put
attribution-by-extraction in the always-on guidance, in both spec templates,
and in the code-review lens: writing that a record settles something means
locating the exact sentence that carries it.

The next specification written after both shipped breached both. An independent
pass found the breaches; the author did not.

Five attributions claimed more than their record carries. D-0054 was cited as
settling which tier a record lives in, where it settles that records are tiered
and how to price them. D-0038 was cited as settling that a presence assertion
proves existence rather than relevance, where that phrase sits in its Reasoning
scoped to one rejected mechanism. G-0584 was cited as settling what a
transcribed expected value costs, and it is an open gap, which records a defect
and settles nothing. The archive bar for a forward-looking document was stated
as reached when an epic promotes its threads, where the bar is every thread and
the epic records one as not promoted. And a dated observation of a single run
was described as a vocabulary a record must carry.

The mechanism is identifiable rather than mysterious, and it has two halves.

The check was applied to text formatted as a citation and not to claims carried
by ordinary prose. Six citation-shaped references were verified by extraction
before the specification was presented. The five defects were sentences that
name a record in passing, which read as background rather than as attributions.

The check confirmed each phrase was present in the cited record and not what the
record scopes it to. That is the failure G-0593 already names — re-reading
confirms a claim's direction and not its scope — arriving through the rule
written to prevent it.

## Why it matters

A shipped rule that fails at its first opportunity is weaker evidence for the
next rule of the same shape. The remedy that suggests itself is sharpening the
wording again, which is what just failed, so this gap records the measurement
rather than proposing that remedy.

The genre breach has the same character. Argument appeared in acceptance
criteria bodies and design notes in a document written from a template whose
preamble bans it, by an author who had read the preamble that session.

What no count carries: whether the rules would hold for an author who did not
write them, and whether an independent pass is the only thing that finds this
class.

## The class recurred in the next two documents

A later epic under the same initiative wrote two more documents — a milestone
spec and the test file shipped with it — and the class recurred in both. This
is the same author again, so it still says nothing about an author who did not
write the rules; what it does carry is that the rules did not hold on repetition
by an author who had just been shown the failure.

Five more attributions claimed more than their record carries.

ADR-0019 was cited as the advisory `wf-*` genus. It says "The code-health rubric
is the same genus as the existing `wf-*` engineering skills" — placing a skill
in a genus that predates it rather than defining one.

ADR-0007 was cited for a `wf-*` skill's placement. Its Decision covers the
boundary between kernel-embedded verb wrappers and the rituals plugin's
`aiwfx-*`, and states no rule for the `wf-*` genus at all.

G-0548 was cited as naming package directories the clearest violations. It says
a package reference "reads as a name rather than a location", and reserves "is
the clearest violation" for a source-file reference — the opposite ranking.

A path check over one file was described as the only mechanical form its
collision admits, where G-0548 describes a second: "the check is a near-mirror
of `skill-body-id`, over the same corpus".

A shipped skill defined the reading state *ambiguous* as the subject neither
asserting nor denying a premise. The specification's only gloss is "that a
sentence carries a second reading" — a claim with two readings, not one with
none.

More than one observation now bears on the open half of the question. An
independent pass found all but one of these; the author found the first, before
dispatch, while checking that same document for exactly this class.

The third occurrence is the sharpest available instance of the mechanism. It
sits in the comment of the test guarding a skill whose own section instructs its
reader to open the cited record and find the sentence, and states the rule the
comment breaks: the test is scope, not presence. Proximity to the rule, in the
artifact that carries the rule, did not prevent it.

---
id: G-0431
title: Milestone/epic wrap never closes gaps their own prose claims to fix
status: addressed
discovered_in: E-0069
addressed_by_commit:
    - 9cad2af3
---
## What's missing

None of `aiwfx-start-milestone`, `aiwfx-wrap-milestone`, or `aiwfx-wrap-epic`
promote a gap that a milestone's own spec names as the thing it fixes. A
milestone's body prose (Goal/Context/AC sections) routinely cites the gap
id(s) it closes — e.g. M-0269's spec named G-0426/G-0427/G-0428 throughout,
titled itself after exactly those three bugs, and both the milestone and its
parent epic (E-0069) reached `done` with all three gaps still `open`. Nothing
in the wrap path ever runs `aiwf promote G-NNNN addressed --by-commit <sha>`
for a gap the work actually closed.

Contrast with `wf-patch`: its wrap gate explicitly includes tracker closure
("`aiwf promote G-NNNN addressed --by-commit <sha>` when the patch closes a
tracked item") as one of the declared-sequence steps, and it works reliably
(spot-checked: G-0422/G-0423 both closed correctly via their own
`patch/G-NNNN-<slug>` branches). It works there because a patch branch is
scoped 1:1 to the gap it closes — the id rides in the branch name itself, so
"which gap does this close" is unambiguous and mechanical. A milestone has no
equivalent binding: it can reference, discover, and/or close several gaps in
loose prose, and no ritual step ever walks that prose back to a promote.

A cross-reference of all 273 milestone specs in the tree against all 427
gaps' current status (as of this writing) surfaces the same shape — a `done`
milestone whose own text names a gap as fixed, gap still `open` — across
dozens of epics going back to E-0017. That scan is a raw keyword match (it
also catches a milestone correctly citing a gap *it opened* as a deferral,
which is supposed to stay open, so the exact count is noisy), but the pattern
itself is old and systemic, not specific to E-0069.

`aiwf check` has no finding rule for this either — "a gap a done milestone's
prose claims to fix is still open" is a free-text correlation nothing
mechanical audits today.

## Why it matters

The whole point of a gap is that it's the tracked backlog item a fix closes
the loop on — `aiwf list --kind gap` (or the archive-sweep threshold) is how
an operator or a future epic-planning pass sees what's actually still open.
When a fix lands but the gap it fixed never closes, the backlog silently
overstates itself: G-0426/G-0427/G-0428 sat as `open` for as long as this
epic remained active, indistinguishable from genuinely-unaddressed work, even
though the fix was merged, reviewed, and wrapped. At epic-planning scale this
means the audit that originally filed a batch of gaps (like the verb-layer
call-graph audit that produced G-0426 through G-0430) can never be trusted to
reflect "what's actually left" without a human manually reconciling gap
status against milestone history — exactly the kind of drift this repo's own
"framework correctness must not depend on LLM behavior" principle exists to
prevent (a human or an LLM operator has to *remember* to run the promote by
hand; nothing catches its absence).

A fix likely needs two parts, not one: a ritual-level step (the natural point
is `aiwfx-wrap-milestone`, since the milestone spec's own body already names
which gaps it claims to fix — walk the References/AC prose for cited gap ids
and prompt to promote each, with `aiwfx-wrap-epic` carrying a lighter backstop
confirming no wrapped milestone left a self-claimed-fixed gap open); and,
because free-text prose is a fuzzy signal for a step meant to be reliable, a
possible structural fix — e.g. a `closes: [G-NNNN]` frontmatter field on
milestones that `aiwf check` could then cross-reference against gap status
mechanically, rather than relying on grep. The two aren't mutually exclusive;
which one (or both) is the right shape is a design question for whoever picks
this up, not decided here.

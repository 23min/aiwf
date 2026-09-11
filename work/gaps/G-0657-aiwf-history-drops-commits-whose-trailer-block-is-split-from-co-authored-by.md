---
id: G-0657
title: aiwf history drops commits whose trailer block is split from Co-Authored-By
status: open
discovered_in: M-0327
---
## What's missing

Git's trailer parser reads only a commit message's last paragraph. Where an aiwf
trailer block is separated from a trailing `Co-Authored-By:` line by a blank line,
git reports no aiwf trailers at all, and every consumer that asks git for them sees
none — including the `%(trailers:key=...)` extraction in
`internal/entityview/historyevent.go` that `aiwf history` is built on.

Measured 2026-09-11 over 11,054 commits across all refs: 53 carry an `aiwf-entity:`
line in the message while
`git log --pretty='%(trailers:key=aiwf-entity,valueonly=true,unfold=true)'` returns
empty for them. `9644942a5` is one; its message ends

    aiwf-verb: wrap-milestone
    aiwf-entity: M-0209
    aiwf-actor: human/peter

    Co-Authored-By: Claude <noreply@anthropic.com>

`git interpret-trailers --parse` on that message returns the `Co-Authored-By:` line
alone, and `aiwf history M-0209` does not list the commit. The population spans
2026-04-28 to 2026-07-11.

It is closed rather than merely quiet. `aiwf check --commit-msg` refuses the shape
at composition time, exiting 1 and naming each key left as body prose — landed in
`fc5501d9f` under M-0327/AC-3. What stays open is the landed population above,
invisible to `aiwf history` and not rewritten, on the same reasoning the parent
epic applies to the Work logs of terminal milestones.

Probing each `aiwf-verb` value the population carries against the closed set, only
`wrap-milestone` and `wrap-epic` are accepted; every other value in it — `feat`,
`test`, `wrap`, `docs`, `refactor`, `implement`, `patch`, `fix`, `addressed` —
would be refused were git able to read it.

## Why it matters

The commits carrying `wrap-milestone` and `wrap-epic` are the ones that closed a
milestone or an epic, and they are absent from the timeline `aiwf history` renders
for the entity they name — an operator auditing that entity sees the promote and not
the wrap. Nothing reports the omission, because the commit is not untrailered: it
carries a full trailer block that git declines to parse, so a rule keyed on absent
trailers has nothing to fire on.

That blindness was also a hole in the verb chokepoint: `trailer-verb-unknown` exists
to refuse a fabricated `aiwf-verb` value, and a blank line before `Co-Authored-By:`
was enough to carry one past it. The commit-msg refusal closes that route for new
commits. It reaches nothing already landed, so the values listed above sit in the
tree unrefused, and a reader of `aiwf history` still cannot see the commits carrying
them.

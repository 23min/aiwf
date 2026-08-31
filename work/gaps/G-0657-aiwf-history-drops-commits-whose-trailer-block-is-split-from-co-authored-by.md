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

Measured over 10,801 commits across all refs: 50 carry an `aiwf-entity:` line in the
message while
`git log --pretty='%(trailers:key=aiwf-entity,valueonly=true,unfold=true)'` returns
empty for them. `9644942a5` is one; its message ends

    aiwf-verb: wrap-milestone
    aiwf-entity: M-0209
    aiwf-actor: human/peter

    Co-Authored-By: Claude <noreply@anthropic.com>

`git interpret-trailers --parse` on that message returns the `Co-Authored-By:` line
alone, and `aiwf history M-0209` does not list the commit. The population spans
2026-04-28 to 2026-06-29 and has not grown since.

The shape also passes `aiwf check --commit-msg`. Written as one paragraph, a message
carrying `aiwf-verb: feat` is refused with exit 1 naming the value; split by a blank
line before `Co-Authored-By:`, the same message exits 0. Among the 50, the only
`aiwf-verb` values that are not ones the closed set refuses are `wrap-milestone` and
`wrap-epic`.

## Why it matters

The commits carrying `wrap-milestone` and `wrap-epic` are the ones that closed a
milestone or an epic, and they are absent from the timeline `aiwf history` renders
for the entity they name — an operator auditing that entity sees the promote and not
the wrap. Nothing reports the omission, because the commit is not untrailered: it
carries a full trailer block that git declines to parse, so a rule keyed on absent
trailers has nothing to fire on.

The same blindness is a standing hole in the verb chokepoint. `trailer-verb-unknown`
exists to refuse a fabricated `aiwf-verb` value, and a blank line before
`Co-Authored-By:` is enough to carry one past it.

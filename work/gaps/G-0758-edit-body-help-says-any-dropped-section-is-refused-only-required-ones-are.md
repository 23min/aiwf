---
id: G-0758
title: edit-body --help says any dropped section is refused; only required ones are
status: open
---
## What's missing

`aiwf edit-body --help` (`internal/cli/editbody/editbody.go`) says "A write that
would drop a `## <Section>` the committed body carries is refused, naming the
section", and that a deliberate removal is recorded "with aiwf acknowledge
illegal". The rule it describes, `sectionsDroppedSince`
(`internal/verb/editbody.go`), compares only the kind's required sections
(`internal/entity/required_sections.go`), and its own refusal says so: "this
write drops required section(s)". A section the kind does not require is dropped
without a word. The help follows ADR-0049's decision bullet, "`aiwf edit-body`
refuses a write that drops a section the committed body carries", which the same
ADR scopes a few lines later: a violation is "a required section not present as
a top-level `## ` heading, with sections beyond the set legal". The code follows
the scope, the help the bullet.

Reproduced 2026-11-01 with aiwf built from `main` at `c06ffdaf9`
(`vcs.modified=false`), git 2.51.1, Linux, with the hooks `aiwf init` installs, in a fresh repository holding a gap `G-0001` created by `aiwf add gap --title
"A gap"` from the body `## What's missing\n\nM.\n\n## Why it matters\n\nW.\n\n## Notes\n\nN.\n`:

```
$ aiwf edit-body --help 2>&1 | grep -A1 'would drop'
A write that would drop a `## <Section>` the committed body carries is refused,
naming the section. Both modes apply it, so the same edit is judged the same way
$ grep -n '^## ' work/gaps/G-0001-*.md
6:## What's missing
10:## Why it matters
14:## Notes
$ sed -i '/^## Notes$/,$d' work/gaps/G-0001-*.md
exit 0
$ aiwf edit-body G-0001
aiwf edit-body G-0001
exit 0
$ grep -n '^## ' work/gaps/G-0001-*.md
6:## What's missing
10:## Why it matters
$ sed -i '/^## Why it matters$/,$d' work/gaps/G-0001-*.md
exit 0
$ aiwf edit-body G-0001
aiwf edit-body: G-0001: this write drops required section(s) `## Why it matters` the committed body carries — restore the heading before committing
exit 2
```

Dropping `## Notes` committed (exit 0); dropping `## Why it matters` was refused
(exit 2), naming it as a required section.

## Why it matters

An operator who relies on the help to catch an accidental deletion loses a
section the kind does not require — a gap's `## Relates to`, an ADR's
`## Validation` or `## References` — in a commit the verb accepts, and the help
sends them to `aiwf acknowledge illegal` for a removal that needs nothing.

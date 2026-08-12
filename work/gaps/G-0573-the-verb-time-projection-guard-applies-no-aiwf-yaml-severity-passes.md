---
id: G-0573
title: The verb-time projection guard applies no aiwf.yaml severity passes
status: open
priority: high
discovered_in: M-0300
---
## What's missing

The verb-time projection guard decides whether a mutation may land by projecting
the post-mutation tree and looking for newly-introduced error-severity findings.
It calls `check.Run` directly, and applies none of the four `aiwf.yaml` severity
passes the full `aiwf check` applies afterwards.

So a knob that escalates a finding to error severity is invisible to every verb.
Measured in a repo with `tdd.strict: true`:

```
$ aiwf add epic --title "Strict probe epic"
ok — no findings
aiwf add epic E-0001 "Strict probe epic"        exit 0

$ aiwf check
...epic.md:1: error entity-body-empty/epic: E-0001 body section `## Goal` is empty
... (x3)                                         exit 1
```

The verb reports success, commits, and prints `ok — no findings` for a state the
pre-push hook refuses.

## Why it matters

The guard exists so a verb refuses rather than landing a bad state; a
configuration that makes a state bad is exactly what it cannot see. The
operator's own `aiwf.yaml` therefore has the perverse property that raising a
severity makes the *gate* stricter while leaving every *writer* unchanged, and
the gap between them is only discovered at push.

This is the same missing-escalation shape as G-0567, on the write side rather
than a read surface, and it is the more consequential half: a read surface that
under-reports is a nuisance the next full check corrects, whereas a writer that
reports success while landing a gate-refused state has already committed by the
time anything disagrees.

Both belong to one unpinned property: seven call sites reach `check.Run` and
each decides independently which severity passes to compose — four for `check`,
two for `check --fast`, zero for `status`, `show`, `render`, `doctor` and this
guard. Nothing mechanical holds them in any relation to each other, so the
next added pass will reach whichever call sites its author happened to edit.

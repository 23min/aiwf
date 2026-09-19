---
id: G-0684
title: A path git quotes is invisible to the push gate
status: addressed
discovered_in: M-0331
addressed_by_commit:
    - b9dd0ea91f37d210fc078ab20aa37deab6f5551b
---
## What's missing

The push gate in `internal/check/entity_body_section_dropped.go` reads its trees
and its commit range with `-c core.quotePath=false`, which stops git quoting
non-ASCII paths only. A path carrying a double quote, a backslash, a tab or a
control byte is always quoted by `git ls-tree` and `git log --name-status`,
and a quoted path matches no entity shape, so the entity is invisible to the
walk at both ends of the range. Measured against a scratch repo with an
upstream: an entity at `work/gaps/G-0001-has "quote".md` whose pushed commit
drops `## Why it matters` — `aiwf check` expected one
`entity-body-section-dropped`, observed `ok — no findings`, exit 0. Passing
`-z` to both git calls and splitting on NUL removes the quoting; the tab-split
of `--name-status` lines goes with it. So does the assumption the range log's
record parse rests on: without `-z`, git quotes a path carrying a control byte,
which is what keeps a separator byte in a path from splitting a record. With
`-z` the raw bytes reach the stream, and a path carrying `0x1E` or `0x1F` splits
the record unless the parse guards its field count.

## Why it matters

The gate's population is hand-written commits, and a hand-written filename is
what reaches it — aiwf's own slugifier never emits these bytes, so nothing a
verb creates is affected, but a file renamed by hand is judged by nothing and
the push passes. A silent pass in a refusal gate is the failure the gate exists
to prevent.

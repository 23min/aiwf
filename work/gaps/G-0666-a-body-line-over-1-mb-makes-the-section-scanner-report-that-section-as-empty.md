---
id: G-0666
title: A body line over 1 MB makes the section scanner report that section as empty
status: open
discovered_in: M-0329
---
## What's missing

`scanH2Sections` (`internal/check/entity_body.go:268`) reads a body through a
`bufio.Scanner` capped at a 1 MB token and never consults `scanner.Err()`. A line
longer than the cap ends the scan silently: the section it sits under is recorded
carrying no content, and every heading after it is never seen at all.

Measured against a gap whose `## What's missing` holds one 1.5 MB line, followed
by a filled `## Why it matters`:

    aiwf check --format=json
    error  entity-body-empty  body section `## What's missing` is empty

The section holding 1.5 MB of prose is reported empty, and the section after it
is reported as neither empty nor absent.

Four sibling scanners share the shape and the missing `scanner.Err()` consult:
`internal/check/entity_body.go:303` and `:350`, `internal/check/acs.go:679`, and
`internal/check/locate.go:62`.

## Why it matters

The finding is error severity, so the pre-push hook blocks on a body that is not
empty and the operator has nothing to fix. The invisible-suffix half is the
opposite failure: a required section genuinely missing after a long line is
reported by nothing.

`aiwf add` and `aiwf edit-body` now refuse a body omitting a required section, so
the same truncation refuses the write outright, naming a heading the file
carries. On the `edit-body` path that refusal has no `--force`.

How to fix it is a decision the five call sites should take together: raise the
ceiling, which moves the threshold rather than removing it; read lines directly
off the byte slice, which has no ceiling and no extra copy, since the body is
already in memory; or keep the scanner and fail loudly when `scanner.Err()` is
non-nil.

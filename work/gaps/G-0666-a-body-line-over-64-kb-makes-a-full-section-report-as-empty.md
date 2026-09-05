---
id: G-0666
title: A body line over 64 KB makes a full section report as empty
status: open
discovered_in: M-0329
---

## What's missing

`isAllWhitespaceOrHeadings` (`internal/check/entity_body.go:319`) decides whether
a body section carries anything an author wrote. It reads the section through a
`bufio.Scanner` left at the default buffer and never consults `scanner.Err()`. A
line at or above bufio's 65,536-byte default token size ends the scan, and the
function returns true — nothing but whitespace — for content it never read.

Measured, on a gap body whose `## What's missing` holds one line of n bytes:

    n = 60,000   EmptyRequiredSections = []
    n = 65,535   EmptyRequiredSections = []
    n = 65,536   EmptyRequiredSections = [What's missing]
    n = 70,000   EmptyRequiredSections = [What's missing]

and end to end, on the same body through the real binary:

    aiwf check --format=json
    error  entity-body-empty  body section `## What's missing` is empty

Three sibling scanners share the missing `scanner.Err()` consult, each having
raised its ceiling to 1 MB rather than removed it: `scanACBodies`
(`internal/check/entity_body.go:271`), `scanACHeadings`
(`internal/check/acs.go:679`), and `scanFieldLines`
(`internal/check/locate.go:62`).

## Why it matters

The finding is error severity, so the pre-push hook blocks on a section that is
full, and the operator has nothing to fix. Raising a ceiling moves the threshold
without removing it; what the four call sites need to agree on is whether a body
they cannot finish reading is judged silently at all.

The section-heading half of this is already gone. M-0329 retired the capped
scanner that decided which sections a body carries, so a heading after a long
line is no longer invisible — measured on the same 1.5 MB body, both sections are
now found. What remains is the content half, where the answer is not "the section
is missing" but "the section is empty", about a section that is neither.

Reading the whole slice is the alternative to a ceiling: an entity body is
already in memory when these functions are called, so walking it by newline costs
no copy and has no threshold to exceed.

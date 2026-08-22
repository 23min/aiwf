---
id: G-0607
title: Regexp content matching over shipped surfaces bypasses the shipped-prose ban
status: open
discovered_in: M-0313
---
## What's missing

`shipped-prose-assertion` inspects call arguments for a needle. A regexp match
carries its needle inside a compiled pattern instead, so
`regexp.MustCompile("...").MatchString(body)` over shipped content is outside
the predicate.

Measured at the time of writing: twelve sites match a regexp against content
read from a shipped surface. Nearly all are genuinely structural — counting
numbered step headings, extracting id patterns, checking a shell script parses
ahead/behind counts — and would survive D-0070 on their own terms. At least one
is closer to prose.

The unsolved part is not detection but classification. `\d+\.\s` is structure;
`chore\(epic\): wrap` is prose; both are string literals inside a
`regexp.MustCompile` call, and nothing in the syntax distinguishes them. A rule
that fires on every regexp over shipped content would condemn the structural
majority, and one that fires on none leaves the bypass open.

## Why it matters

It is the most plausible way the retired class returns: an author who trips the
ban on `strings.Contains` and reaches for a regexp gets a green suite, and the
reach is a small enough step that it would not read as evasion.

Nothing is broken today — the structural uses are legitimate and the prose-ish
one is a single site. What is missing is a decision on where the line falls, and
that decision needs the classification question answered first.

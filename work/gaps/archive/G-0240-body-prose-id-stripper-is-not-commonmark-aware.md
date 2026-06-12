---
id: G-0240
title: body-prose-id stripper is not CommonMark-aware
status: addressed
addressed_by_commit:
    - 8b18ca7b
---
## What's missing

The `body-prose-id` check rule (G-0184) strips three markdown shapes
before scanning for id-shaped tokens: single-backtick inline code spans
(`...`), triple-backtick fenced blocks (` ``` `...` ``` `), and tilde
fenced blocks (`~~~...~~~`). Four CommonMark shapes are NOT handled:

1. **Multi-backtick inline code spans.** CommonMark allows a code span
   to use any number of opening backticks (typically used when the
   span itself contains a backtick). `` `` `id` `` `` is a valid double-
   backtick span containing `` `id` ``. The current single-tick regex
   matches only the inner span and leaves the outer ticks; tokens
   inside don't fire today because they're stripped, but the inverse
   case (operator uses double-ticks deliberately to embed backticks
   around an id-shape) would silently leak.

2. **Indented code blocks.** CommonMark recognizes a paragraph break
   followed by 4+ space-indented lines as a code block. Indent-style
   code (the convention many style guides recommend over fences for
   short snippets) is not stripped; tokens inside fire as if they were
   prose.

3. **Markdown link URLs.** `[text](url)` syntax is scanned verbatim;
   id-shaped tokens embedded in URLs (`[old gap](work/gaps/G-9999.md)`)
   fire `unresolved` even though the operator intent is a path
   reference, not a prose reference. Real-world impact: zero in the
   current tree (all path-form references resolve), but a body that
   links to a since-deleted entity would surface a confusing finding.

4. **Multi-line inline code spans.** The single-line pattern
   `` `[^`\n]*` `` deliberately refuses to span newlines (matches
   CommonMark semantics), but malformed input (an unclosed backtick
   at end of line) can chew through prose on the next line into the
   next backtick, leading to surprising masking behavior.

## Why it matters

The rule's contract is "tokens inside any markdown code construct are
silent, tokens in prose fire." The current stripper covers the common
case (single-tick spans, triple-tick fences) and is hint-supported
("wrap in backticks if discussing id syntax"). But the rule's
correctness depends on the stripper recognizing every code construct
CommonMark defines as "not prose." Four edge cases either silently
mask tokens that should fire (#1 inverse) or fire false positives on
tokens that are inside code (#2, #3, #4).

Reviewer-pass evidence: surfaced as track-for-later T1, T2, T9 across
both G-0184 reviewer passes. Zero current-tree impact, but the rule's
guarantee is fragile because a future entity body using indented code
or link URLs gets unexplained findings, and the hint ("wrap in
backticks") doesn't apply to those shapes.

## Direction

Replace the three regex-based strippers in `internal/check/body_prose_id.go`
with a CommonMark-aware walker. Two viable shapes:

- **AST walker via `github.com/yuin/goldmark`** (already a dependency
  for HTML render): walk the AST, skip code blocks / inline code /
  link URLs / fenced blocks generically. Robust, single source of
  truth for "what counts as prose," extends to future markdown shapes
  for free. Cost: an extra dependency edge in the check rule.

- **Hand-rolled state machine** that tracks line-level state (open
  fence? open span? indented block?) and feeds the scanner only the
  prose runs. Smaller surface, no new dependency, but reinvents what
  goldmark already implements.

The AST approach is the kernel principle ("the boring solution")
applied to markdown semantics. Recommend goldmark unless the
dependency-edge concern justifies the state-machine cost.

## Test surface

Per-shape regression cases:
- Multi-backtick span: `` `` `M-a` `` `` silent.
- Indented code block (4-space): `M-a` inside silent.
- Markdown link URL: `[label](path/M-9999.md)` silent (the URL is not
  prose; if a reader wants to flag broken links, that's a separate
  rule, not body-prose-id).
- Multi-line corruption: unclosed backtick at EOL does not chew
  through prose on subsequent lines.

Plus all current edge-case tests in `TestBodyProseID_EdgeCases` stay
green to confirm no regression in the common-path behavior.

## Source

G-0184 reviewer passes (initial + verb-time follow-up): track-for-later
T1 (indented blocks), T2 (link URLs), T9 (multi-line spans), plus the
inverse multi-backtick case noted in the verb-time pass.

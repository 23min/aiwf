---
id: G-0503
title: The coverage:ignore escape opens on lines that are not the directive
status: addressed
priority: medium
addressed_by_commit:
    - 187f69437
---
## What's missing

`blockHasCoverageIgnore` decides whether a coverage block is exempt by testing `strings.Contains(src[line-1], "//coverage:ignore")` against each raw source line in the block's span. Any line carrying those characters suppresses the finding, whatever the characters are doing there.

Measured against the function directly, one line per block:

| source line | suppressed |
| --- | --- |
| `return nil, err //coverage:ignore genuinely unreachable` | yes — the intended case |
| `return nil, err //coverage:ignore` | yes — no reason given |
| `/* //coverage:ignore inside a block */` | yes |
| `s := "//coverage:ignore"` | yes — a string literal |
| `// see the //coverage:ignore escape described in CLAUDE.md` | yes — prose about the escape |
| `return nil, err //coverage:ignoreable` | yes — a longer word |
| `x := n // coverage:ignore with a space` | no |

This is the defect class G-0496 closed for `//history:ok` and `//exec:ok`, in the third escape of the same family. Those two now share one matcher that requires the marker to open a whole comment and be separated from its reason by whitespace; this one has no such rule and does not look at comments at all.

## Why it matters

The failure is silent in the expensive direction. An untested branch stays untested, the gate reports clean, and nothing says a rule was skipped — the same shape as the push-gate hole, against the audit that is supposed to catch missing tests.

Two of the reachable forms arrive by accident rather than intent. Prose about the escape is exactly the kind of comment someone writes while documenting it, and a bare marker with no reason reads as an annotation whose author simply did not explain themselves — neither looks like a suppression when read.

## Resolution shape

The engine differs from the two siblings and that is the substance of the work. `blockHasCoverageIgnore` scans raw source lines within a coverage block's line span; the siblings parse the file and match whole comments. Making this one directive-accurate means parsing for comments and mapping them onto block spans, then delegating to `hasDirectiveComment` so all three escapes obey one rule.

The migration is smaller than the call-site count suggests. An AST census across the tree finds 635 occurrences in comments, of which **590 are already directive-shaped** — the marker opens the comment, whitespace separates a reason — and would be unaffected. The remaining 43 are prose mentions and bare markers; 2 more sit in string literals. Most of the 43 sit in doc comments above a function, outside any block span, so they suppress nothing today and only need correcting where they fall inside one.

Two calls to take deliberately:

- **Whether a reason becomes mandatory.** It is for the two siblings, and it is what makes a bare marker stop being an escape. Some of the 43 are bare markers in live code, so this is the part that requires edits rather than only a matcher change.
- **Whether the block-span mapping keeps today's generosity.** A comment anywhere in the span currently exempts the whole block. Narrowing that to the annotated statement is a separate tightening with its own migration, and is not required to close the hole above.

## Where to fix

- `internal/policies/branch_coverage_audit.go` — `blockHasCoverageIgnore`, the raw-line scan; and its caller, which supplies the block spans.
- `internal/policies/directive_comment.go` — `hasDirectiveComment` is the shared matcher the two siblings already route through.
- The 43 non-conforming sites, once a census run with the new matcher names them.

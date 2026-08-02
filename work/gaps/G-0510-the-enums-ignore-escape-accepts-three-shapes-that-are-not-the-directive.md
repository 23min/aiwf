---
id: G-0510
title: The enums:ignore escape accepts three shapes that are not the directive
status: open
---
## What's missing

`collectIgnoredLines` accepts as the `//enums:ignore` directive three comment shapes that are not it, including one its own documentation says it rejects.

The matcher strips a leading `//`, trims surrounding space, and asks whether what remains opens with the marker. Measured against it directly:

| comment | suppressed |
| --- | --- |
| `//enums:ignore a real reason` | yes — the intended case |
| `// enums:ignore spaced, not directive-shaped` | yes |
| `//enums:ignore` | yes — no reason given |
| `//enums:ignoreable` | yes — a longer word |
| `// see the enums:ignore escape in CLAUDE.md` | no |
| `/* enums:ignore in a block */` | no |
| `//ENUMS:IGNORE shouty` | no |

The bare-marker row contradicts the policy's own doc comment, which states that the reason "is required prose so the suppression carries audit context". Nothing verifies it, so a suppression can carry no context at all.

This is the last of four escapes in the same family still matching by hand. The other three — `//history:ok`, `//exec:ok`, `//coverage:ignore` — share `hasDirectiveComment`, which requires the marker to open the comment and be separated from a mandatory reason by whitespace.

## Why it matters

The escape hatch of a suppression rule fails silently in the direction that costs something: the literal stays unconverted, the policy reports clean, and nothing announces that a rule was skipped.

Two of the three accepted shapes arrive by accident rather than intent. The spaced form is what an author writes who has not noticed the convention is directive-shaped, and gofumpt rewrites a non-directive comment into it — so a correctly-spelled directive that loses its shape keeps working, and the convention erodes without a signal. A bare marker is what an author leaves who means to explain and does not.

This one is the mildest of the four: it already rejects prose mentions and block comments, which is what made the sibling defects reachable by accident. What remains is the reason requirement and the two spelling holes.

## Resolution shape

Route `collectIgnoredLines` through `hasDirectiveComment(c.Text, enumsIgnoreMarker)`, as the three siblings do, and introduce the marker as a named constant rather than an inline literal. That closes all three holes at once and makes the family's contract uniform — the point of sharing the matcher is that a convention cannot hold for one escape and not another.

The migration is bounded: 39 occurrences of the marker across the tree, most of them in `internal/stresstest` classify tests that already spell it directive-shaped with a reason. A census run with the new matcher names the exceptions, which then need a reason added or the spacing corrected.

One call to take deliberately: whether the spaced form stays accepted during a transition. Accepting it indefinitely keeps the erosion path open; rejecting it outright is a one-commit sweep over whatever the census turns up.

The policy's doc comment also needs correcting either way — as written it describes a reason requirement the code does not implement.

## Where to fix

- `internal/policies/enum_literal_adoption.go` — `collectIgnoredLines`, the matcher; and the doc comment on `PolicyEnumLiteralAdoption` asserting the reason is required.
- `internal/policies/directive_comment.go` — `hasDirectiveComment` is the shared matcher the other three route through.
- The occurrences a census names, once the matcher is tightened.

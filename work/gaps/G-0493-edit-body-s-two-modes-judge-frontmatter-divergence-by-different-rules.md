---
id: G-0493
title: edit-body's two modes judge frontmatter divergence by different rules
status: open
discovered_in: M-0283
---
## What's missing

Both `aiwf edit-body` modes refuse when the working copy's frontmatter has
diverged from HEAD's, and since M-0283 both raise the same message. They do not
apply the same test.

Bless mode compares raw bytes. Explicit mode compares decoded fields, so it
tolerates a key reordering or a quoting change that declares the same state.

Measured on one identical working-copy state — frontmatter keys reordered, no
field changed, body edited:

- `aiwf edit-body <id>` refuses with 'frontmatter changed in the working copy'
- `aiwf edit-body <id> --body-file <path>` commits

## Why it matters

Nothing about the entity changed; only its spelling. The refusal names four
structured-state verbs as the fix, and none of them addresses a key order — the
actual escape is the other mode of the same verb, which the message does not
mention.

A shared message now asserts a shared rule. A reader who trusts it will conclude
the modes agree, and the one case where they disagree is the one where the
message's advice cannot help.

## Scope

The predicate each mode uses, and what the shared message may claim. Out of
scope: whether frontmatter divergence should refuse at all — it should, that is
ADR-0038 and G-0463.

## Resolution options

1. Route bless mode through the same field comparison, so a pure reordering is
   accepted by both and the shared message is true. Closest to intent: the rule
   is 'no field changed', and byte equality is a stricter proxy for it.
2. Route explicit mode through byte comparison, so both refuse a reordering.
   Consistent, but it would refuse the re-canonicalization explicit mode
   performs deliberately.
3. Keep both predicates and split the message again, naming the reordering case
   and the other mode as its remedy.
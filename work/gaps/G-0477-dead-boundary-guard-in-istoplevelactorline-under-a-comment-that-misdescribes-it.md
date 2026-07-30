---
id: G-0477
title: Dead boundary guard in isTopLevelActorLine under a comment that misdescribes it
status: open
priority: low
---
## What's missing

`config.isTopLevelActorLine` carries a second guard that can never be false, under a
comment describing a rejection the first guard already performs.

```go
if !strings.HasPrefix(trimmed, "actor:") {
        return false
}
// Reject "actorxxx:" — only a colon-or-whitespace boundary counts.
rest := trimmed[len("actor"):]
return strings.HasPrefix(rest, ":")
```

The first check requires a colon at index 5, so `rest` always begins with one and
the return is unconditionally true. `"actorxxx:"` never reaches the second check —
it fails the first, because its first six bytes are `actorx`, not `actor:`. The
comment names a boundary distinction the function does not make and does not need.

The function's own doc comment compounds it: "Indented lines and lines where
`actor` is a key inside another mapping are left alone." The first clause is true —
`HasPrefix` anchors at column 0. The second describes nesting detection the function
does not perform; a nested key is excluded incidentally, by its indentation, not
because the function inspects mapping structure.

## Why it matters

Small on its own. It matters because it was read as a real distinction and became
the basis of a claimed defect elsewhere: the sibling analysis of duplication in this
area asserted that `initrepo` re-implements this predicate as a "mere prefix test"
while `config`'s was "deliberately top-level-only", and that a divergence between
them could make `initrepo` report removing a field it did not remove. Both
predicates are the same column-0 prefix test. The divergence is not constructible —
17 adversarial inputs and a brute-force sweep produced none — and the comment is
why the two looked different.

A comment that describes a check the code does not make is a trap for exactly this
kind of reasoning. It reads as a considered boundary rule, so a reader building on
it inherits a distinction that isn't there.

## Options

1. **Delete the dead guard and correct both comments.** The function becomes its one
   real check, the doc comment says column-0-anchored rather than
   mapping-aware, and the misleading boundary comment goes. Smallest change, and it
   removes the trap rather than annotating it.
2. **Make the guard real** — accept a colon *or* whitespace boundary, as its comment
   claims, so `actor :` with a space before the colon is also stripped. This changes
   behavior on input the strip currently leaves alone, so it needs a call on whether
   that YAML shape should be stripped at all.
3. **Leave it and rely on the sibling gap's correction.** Cheapest, and it leaves the
   comment in place for the next reader to build on.

Option 1 is the lean. Option 2 is a behavior question worth asking separately if
anyone has seen `actor :` in a real `aiwf.yaml`; nothing in this repo's history has.

## Scope

Surfaced while verifying a duplication claim about `initrepo` and `config`. The
predicate pair is otherwise genuinely duplicated — that part of the analysis stands
— but the wrong-message consequence attributed to it does not, and this is what was
actually wrong in those lines.

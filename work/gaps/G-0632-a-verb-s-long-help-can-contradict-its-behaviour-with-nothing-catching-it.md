---
id: G-0632
title: A verb's long help can contradict its behaviour with nothing catching it
status: open
discovered_in: M-0316
---
## What's missing

A Cobra command's `Long` string describes what the verb does. Nothing asserts it
still agrees with what the verb does, so the two drift silently and the reader
who trusts `--help` is the one who pays.

Measured 2026-08-24. `internal/cli/move/move.go:42` read *"The moved milestone's
own outbound links are not rewritten."* `internal/verb/move.go:85` calls
`RewriteLinkDestinationsForMove` on the moved entity's own body, recomputing its
relative destinations against the destination directory — behaviour M-0315 added
under ADR-0046, and behaviour the archive-and-move outbound test suite already
pins end to end. The sentence was false from the moment M-0315 landed and stayed
that way until a reader happened to run `aiwf move --help` while working on
something else. Corrected in `61abf68c5`.

Nothing caught it, and nothing would have: no test in the tree reads any
command's `Long`, and the ones that read `--help` output check flag and verb
*presence* for completion coverage, never the prose.

## Why it matters

`--help` is not a convenience here. This repo's kernel rules make it a load-
bearing discoverability channel — every verb, flag, JSON field and body-section
name is required to be reachable through `aiwf <verb> --help`, embedded skills,
or the design docs, on the stated grounds that an AI which must grep source to
learn a capability means the capability is undocumented. A `Long` that
confidently states the opposite of the behaviour is worse than an absent one: it
answers the question wrongly and closes it.

The failure is also invisible in the direction that matters. A stale `Long` still
renders, still reads fluently, and still passes every gate — there is no crash,
no finding, no red run. It surfaces only when a human reads it against the code,
which is what happened here and is not a mechanism.

The exposure grows with the surface. This is one sentence in one verb, found by
accident; the tree carries a `Long` for every top-level verb, each describing
behaviour that milestones keep changing.

## Resolution shape

The honest options differ in what they can actually catch, and the cheap one
catches less than it appears to.

**A behavioural relationship check** is the only shape that catches this class:
for a claim a test already pins, assert the `Long` and the test agree — e.g. a
`Long` asserting links are *not* rewritten, beside a test asserting they are, is
a contradiction a check can see. This needs the claim to be expressed in a form
both can read, which in practice means a small structured table of
claim-to-test bindings rather than prose matching. Real, and not free.

**A prose assertion over the `Long`** (grep for the sentence) is the tempting
cheap version and is ruled out by D-0070's reasoning: pinning a phrase pins one
reading, rewording breaks it, and the check tests the words rather than the
claim. It would have caught this specific sentence and nothing else.

**Recording the boundary** — that `Long` prose is held at review, not
mechanically — is the third option and is defensible if the first is judged too
costly. What is not defensible is the current state, where the boundary is
neither guarded nor written down, so its absence reads as an oversight rather
than a decision.

Worth deciding alongside: whether `Long` should describe behaviour at this level
of detail at all. The sentence that went stale was a fine-grained claim about
link rewriting; a `Long` that stayed at the level of *what the verb is for* would
have had nothing to go stale.

## Where to fix

- `internal/cli/*/` — every command's `Long`; `move.go` is where the drift was
  found, not the only place it can occur.
- `internal/policies/` — where a relationship check would live if one is built.
- `CLAUDE.md` §"Kernel functionality must be AI-discoverable" — the rule that
  makes `--help` load-bearing, and so the rule this gap is about.

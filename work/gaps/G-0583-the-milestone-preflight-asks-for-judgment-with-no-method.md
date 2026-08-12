---
id: G-0583
title: The milestone preflight asks for judgment with no method
status: open
---
## What's missing

`aiwfx-start-milestone`'s first step is called "Preflight" and asks the reader to
confirm every acceptance criterion is concrete and testable, stopping to refine any
that is vague. It supplies no method for deciding that. The remaining bullets check
the baseline builds and the tests pass — real, but orthogonal to whether the spec is
right.

So the one step positioned to catch a wrong specification before any code is written
is a request for judgment with nothing behind it, and what it actually verifies is
that the tree compiles.

## Why it matters

Measured on the three most recent milestones, the specification was wrong more often
than the implementation. Three of M-0306's five criteria were rewritten mid-flight
because measurement contradicted their premise, and two more were cancelled after
their content had landed. M-0307 was cancelled outright: a preflight run against it
found the defect real but worse than stated, the surface count three times what the
spec named, and one criterion unwritable without being tautological.

Review does not catch this class. M-0306 had six independent review passes and every
one returned request-changes on the implementation; none questioned the spec, because
a reviewer checks work against a specification rather than a specification against
reality.

The method that did catch it has three parts, and they are not equal. Measuring every
factual claim in the spec by running commands rather than reasoning is cheap, and
caught all three of M-0307's defects on its own. Challenging each criterion — for the
consumer-visible failure it prevents, and for whether a builder could satisfy its
letter while leaving a consumer no better off — is equally cheap, and is what found
the one criterion that could not be written non-vacuously. Sweeping the docs,
entities, shipped surfaces and code comments the milestone's subject touches, for
prose that contradicts the spec or that the milestone would falsify, is the expensive
part and the part that went wrong: run against a stale branch it missed the single
most consequential finding of the day and produced one confident false one. So the
cheap two run first and always; the sweep reads current trunk, and its findings are
hypotheses until a command settles one — the sweep's headline claim was wrong, and one
measurement settled it.

What the sweep produces is edits, not a report, or it joins the documents nothing acts
on. Where it finds one fact stated across several surfaces and some of them wrong, the
edit is an owner and its derivations rather than a correction per copy — E-0081
answered eleven such surfaces with a single `entity.RequiredSections`, and M-0307's own
surviving commit was a ban with both sides derived rather than an assertion. Prose
found contradicting reality this way is a defect in its own right, not only a hazard to
the milestone reading it.

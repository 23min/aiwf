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

The method that did catch it has three parts, and each earned its place on that
milestone: measuring every factual claim in the spec by running commands rather than
reasoning; sweeping the docs, entities, shipped surfaces and code comments that the
milestone's subject touches, for prose that contradicts it or that it would falsify;
and challenging each criterion for the consumer-visible failure it prevents and for
whether a builder could satisfy its letter while leaving a consumer no better off.

Three lessons from running it, which any shipped form has to encode. The sweep must
read current trunk — run against a stale branch it missed the single most
consequential finding of the day and produced one confident false one. Its output has
to be spec edits rather than a report, or it joins the documents nothing acts on. And
subagent findings are hypotheses: the sweep's headline claim was wrong, and one
measurement settled it.

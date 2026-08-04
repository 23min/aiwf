---
id: D-0055
title: Accept the !testpins lint residue over a second lint pass
status: accepted
---
## Question

`golangci-lint` lints the union of the declared build tags in a single pass
rather than each tag configuration separately. So declaring `testpins` brought
the tagged sources into the lint surface and pushed the `//go:build !testpins`
arm out of it, in the same edit. G-0470 reported the first half and left the
second explicitly open: accept the residue, or add a second invocation that
lints without the tag.

What makes it non-obvious is that both sides are cheap to state and priced very
differently. The residue is real lost coverage, and the fix is one more command
— but that command costs a full additional pass over the module, on every push,
against a subject that does not grow.

## Decision

Accept the residue. Do not add a second `golangci-lint` invocation for the
negated arm.

## Reasoning

The cost is not proportional to the subject. `golangci-lint` has no per-tag
mode, so reaching the negated arm means a second whole-module pass in `make
lint`, in the pre-push hook, and in the `lint` CI job alike. That is paid on
every push forever, to reach a fixed and tiny surface.

The subject is scaffolding whose job is to be referenced by the tagged arm — no
branches, no callers outside its own pin. The linters that would newly reach it
police style and misuse in code that has neither.

The asymmetry is the point rather than a side effect: the tagged arm carries the
registry and the assertions, so it is the arm where a finding would mean
something, and declaring the tag is what puts the linter there.

The rejected alternative is not the second invocation but the status quo ante —
leaving `testpins` undeclared, which is what G-0470 reported. It trades a large
unlinted surface for a small one and loses on its own terms.

## Follow-ups

Revisit when a third `//go:build !testpins` file appears, or when the negated
arm grows past scaffolding into logic with branches of its own. Either signals
that the unlinted surface has stopped being a fixed, trivial constant — which is
the only property that makes an extra whole-module pass a bad trade.

No check holds that trigger. A policy counting negated-arm files could, and by
D-0054's rule a check would be the better record — but a chokepoint whose whole
subject is a two-file constant is the per-subject mandate this project declines
by default. The condition stays prose, and the cost of missing it is one lint
pass nobody scheduled.

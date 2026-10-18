---
id: D-0105
title: Restate the depended-on Go pack rules in go-conventions.md
status: proposed
relates_to:
    - E-0092
    - M-0361
---
> **Date:** 2026-10-18 · **Decided by:** the maintainer

## Question

E-0092 gives every guidance rule one home, and its success criteria ask that this repository's development guidance duplicate neither the operating rules nor the selected language packs. M-0361 moved the Go-specific development rules into `.guidance/project/go-conventions.md`, which the router sends a Go task to after the Go pack, and removed the clauses the pack already states. Its observation then showed Claude reading `go-conventions.md` in most Go tasks and the Go pack in fewer than half (M-0361's Validation). A rule stated only in the pack therefore often goes unread. Should this repository's guidance repeat the pack's rules it depends on?

## Decision

- `.guidance/project/go-conventions.md` states the Go pack rules this repository depends on, beside its own additions to and overrides of the pack.
- `.guidance/project/testing.md` states the table rule for tests in any language.
- The rest of the Go pack stays read on demand, through the router's language-pack line.

## Reasoning

- Leaving a rule to the pack alone measurably lost it. With the table rule's multi-case half stated only in the pack, Claude broke it in two of the three T1 runs that wrote code, against none of six at the baseline; with the rule stated in `testing.md`, it broke it in none of the eleven runs that wrote code after (M-0361's Validation).
- Copying the rules into `CLAUDE.md` was ruled out: every task would pay for them in its primed load, and `go-conventions.md` is already read in most Go tasks.
- Pointing at the pack first is what the router and `go-conventions.md` already did while Claude skipped it, so a stronger pointer alone was not expected to change the outcome.

## Consequences

- The restated rules can drift from the pack when `aiwf update` refreshes it. Whoever refreshes the packs compares them; no check does.
- This is the one place E-0092's one-home rule does not hold. The epic records its success criterion on duplicating the language packs as not met for the Go pack.
- G-0738 tracks Claude's reach to the Go pack. If it is resolved, the restated rules can be removed again.

---
id: G-0627
title: AC mechanical-evidence rule has no shape for an observational claim
status: open
discovered_in: M-0317
---
## What's missing

The AC mechanical-evidence rule assumes every acceptance criterion asserts a
standing property of the repository. Some assert an observation instead, and for
those the rule has no shape — it demands a tripwire on something that cannot
subsequently break.

The rule is stated twice: `CLAUDE.md` §"AC promotion requires mechanical
evidence" ("there must be a mechanical assertion that fails if the AC's claim
breaks") and, for consumers, the *Promote an AC to met only with mechanical
evidence* bullet in `internal/skills/embedded-guidance/aiwf-guidance.md`. Neither
distinguishes the two kinds of claim.

Measured instance: M-0317/AC-1 asked for a measurement — break a `docs/`-to-
`work/` link, run the check ADR-0033 delegates to, and record the command, the
expected result, the observed output, and the environment together. That claim
cannot break. The command either was run and reported what it reported, or it was
not. There is nothing for an assertion to watch.

## Why it matters

The rule's escape hatch makes this worse rather than better. `CLAUDE.md` says an
AC whose only available evidence would be a phrase assertion "is an AC stated at
the wrong level; restate what it guarantees so something mechanical can carry
it." Restating has two outcomes and the rule does not distinguish them: narrowing
the claim honestly to what is checkable, or inventing a *proxy* — a different,
checkable claim standing in for the one the AC made.

M-0317/AC-1 took the second. The proxy was "every `docs/` file linking into
`work/` is a file lychee reads", which is checkable only by re-implementing
lychee's file selection in Go, since lychee is not available to the test suite.
That model ran to roughly 580 lines. Four review rounds found four defects in it
and every one was the same shape — the copy disagreeing with the original. It was
removed rather than repaired a fourth time, leaving the milestone at 109 test
lines against 5 lines of production change.

A proxy is worse than an absent check, because it reads as evidence. Nothing in
the rule, and no reviewer prompt derived from it, asks whether the assertion
being offered is the AC's claim or a substitute for it.

## Resolution shape

The discriminator that appears to work is whether the claim can be **re-derived
from artefacts the test can reach**. Three shapes can be:

- **Behavioral** — code produces X for input Y. Test it.
- **Relational** — one artefact agrees with, or references, another. Compare
  them. M-0317/AC-2 is this shape: ADR-0033 cites the gaps owning its residual,
  checked against the tree. It was written once and survived four adversarial
  review rounds without a defect, at twelve lines.
- **Structural** — the tree has a shape. Assert it.

A fourth shape cannot: **observational** — someone ran a command against
something outside the repository and recorded what happened. Re-derivation would
mean re-running it, which the test cannot do. Its honest evidence is the record
the observation itself specifies: command, expectation, observation, environment.
That is what makes it re-runnable by a human, which is the only re-checking
available.

Two consequences worth deciding on, not just one. First, whether the rule should
name the observational shape and prescribe the four-part record for it rather
than a test. Second, whether it should carry a guard against proxies — the
usable form of which is a question an author can answer honestly: *does
satisfying this rule require asserting something the AC did not claim?* If yes,
the AC is observational and the record is the evidence.

Note the feasibility question is not the same as the desirability one, and this
gap is mostly about the first. The lychee model was *feasible* — the final
version matched lychee's real read-set exactly on two full comparisons. It was
still the wrong thing to build. A rule that asks only "can you pin it?" gets a
yes in cases where the honest answer to "should you?" is no.

## Where to fix

- `CLAUDE.md` §"AC promotion requires mechanical evidence" — the repo-development
  statement.
- `internal/skills/embedded-guidance/aiwf-guidance.md` — the shipped bullet, which
  reaches consumer repos and so carries the same defect outward.
- `internal/policies/m0211_guidance_operating_anchors.go` — if the shipped bullet
  is reworded, the curated anchor set is what holds it in place.

---
id: G-0465
title: No chokepoint catches shipped surfaces drifting from verb behavior
status: wontfix
priority: high
discovered_in: M-0281
---
## What's missing

Nothing mechanically catches a shipped surface that describes verb behavior the
code no longer has. `skill-edit-structural-test-backstop` covers only
`internal/skills/embedded-rituals/**`, so `internal/skills/embedded/**` — the
per-verb `aiwf-*` skills — and every Cobra `Long`/`Example` string are
unguarded, as are the Normative-tier docs under `docs/design/`.

M-0281 is the evidence. It changed the outcome of thirteen verb paths, and
successive independent review rounds kept finding surfaces still asserting the
old behavior — each round fixing the ones it found and missing others:

- Round 1 found `aiwf-set-priority` and `aiwf-promote` claiming a no-op is
  refused.
- Round 2 found `aiwf-rename` and `aiwf-retitle` doing the same, for the very
  criterion that changed them.
- Round 3 found `aiwf-add` (three claims, covering both verbs that milestone
  treated as correctness fixes), the same two sentences duplicated verbatim in
  `aiwfx-plan-milestones`, three Cobra `Long` strings listing the no-op among a
  verb's refusals, and `CLAUDE.md`'s own kernel-commitment list.

Every one was found by a human or an agent reading prose, which is exactly the
kind of guarantee this repo declines to rely on elsewhere: *"framework
correctness must not depend on LLM behavior."*

## Why it matters

These surfaces materialize into consumer repos via `aiwf init` / `aiwf update`,
and the per-verb skills are the primary channel an AI assistant reads before
running a verb. A skill that says a verb refuses what it now converges will
produce agents that route around a NoOp, or report a failure that did not
happen. The Cobra `Long` strings are what `--help` prints, and `--help` is the
first AI-discoverable channel this repo's own engineering principles name.

The failure is silent and durable: nothing errors, no test fails, and the drift
only surfaces when someone reads the sentence next to the behavior.

## Shape of a fix

The hard part is that "does this prose describe current behavior" is not
mechanically decidable in general. Two tractable approximations:

1. **Diff-scoped, like the ritual backstop.** When a commit changes a verb's
   entry point under `internal/verb/`, require it to also touch that verb's
   `aiwf-<verb>` skill and its Cobra command file — or carry an explicit
   annotation saying the surfaces are unaffected. Cheap, mechanical, and it
   fires exactly when drift is introduced rather than long afterwards. It cannot
   tell a real update from a whitespace change, which is the honest limit.
2. **A claim vocabulary.** Grep shipped surfaces for a closed set of
   behavioral assertions — "refuses", "always commits", "one commit per
   invocation", "rejected" — near a verb name, and require each match to carry a
   marker tying it to a test. More precise, more machinery, and it needs the
   vocabulary curated as prose evolves.

Option 1 is the closer analogue of what already works here, and it is the one
that would have caught every instance above.

Worth deciding alongside: whether `--help` text belongs in the same net. It
drifted in this milestone too, and it is generated from the same source of truth
the skills describe.

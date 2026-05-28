---
id: G-0175
title: rituals plugin emits aiwf-verb trailer for non-aiwf-verb ritual operations
status: open
---
## Problem

The aiwf-extensions plugin's wrap rituals (`aiwfx-wrap-epic`,
`aiwfx-wrap-milestone`, and the prep step inside `aiwfx-wrap-epic`)
emit commits whose messages carry `aiwf-verb: wrap-epic`,
`aiwf-verb: wrap`, and `aiwf-verb: wrap-prep` trailers respectively.
None of those values are registered top-level verbs or sub-commands
in aiwf's Cobra command tree — they are plugin-side ritual
operations, not aiwf verbs.

G-0150 added `aiwf check`'s `trailer-verb-unknown` finding which
flags `aiwf-verb:` trailer values that aren't in the registered
Cobra verb set. Under the new rule, a fresh wrap commit sitting in
`@{u}..HEAD` (i.e., between the wrap operation and the next push)
will surface a `trailer-verb-unknown` warning for the wrap-side
trailer. The push isn't blocked (warning severity), but the noise
is real.

## Why it matters

The `aiwf-verb:` trailer is the kernel's "this is an aiwf CLI verb
invocation" marker. `aiwf history <entity>` projections render
commits carrying that trailer as verb events. A plugin-side ritual
that adopts the same trailer key misrepresents itself as a kernel
verb invocation — the same projection-correctness concern G-0150
named, but propagated by intent rather than by LLM fabrication.

The cleanest separation: kernel-emitted `aiwf-verb:` for the
binary's Cobra verbs; plugin-emitted `aiwfx-ritual:` (or similar
distinct key) for plugin rituals. `aiwf history` can then render
both surfaces with their correct provenance — kernel-verb events
versus ritual events — without conflation.

## Proposed fix

Change the aiwf-extensions plugin's wrap skills
(`aiwfx-wrap-epic`, `aiwfx-wrap-milestone`) to emit a distinct
trailer key on the commits they produce. Candidates:

- `aiwfx-ritual: wrap-epic` (parallels the plugin's namespace)
- `aiwfx-verb: wrap-epic` (parallels aiwf-verb, sibling key)

Either works; the plugin maintainer picks. Kernel-side: extend
`aiwf history` rendering (separate work, lower priority) so it
recognizes the new key as a plugin-ritual event class. Until that
landscape changes, the trailer-verb-unknown rule's existing warning
severity is the right behavior — surfaces the trailer-key drift
without blocking.

## Discovered

G-0150 implementation pass on 2026-05-28: live `aiwf check` against
the kernel repo showed 21 historical warnings on `wrap-epic`,
`wrap`, `wrap-prep` values (alongside the ~44 genuinely-fabricated
LLM values G-0150 was filed against). G-0150's `@{u}..HEAD` scope
hides these historical warnings, but a fresh wrap commit will
trigger one until the rituals plugin migrates.

## Out of scope

- The aiwf-side `aiwf history` rendering update for the new key —
  separate concern; the trailer-key migration can land first.
- Retroactive fixup of the 21 historical wrap commits on trunk —
  out-of-scope per G-0150's "stop the bleed, not retroactively
  rewrite history" framing.

## Related

- G-0150 — the kernel rule whose live run surfaced this gap.
- aiwf-extensions plugin's `aiwfx-wrap-epic` and `aiwfx-wrap-milestone`
  skills — the emit sites that need to change.

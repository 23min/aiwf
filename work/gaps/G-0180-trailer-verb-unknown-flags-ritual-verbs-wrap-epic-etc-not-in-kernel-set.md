---
id: G-0180
title: trailer-verb-unknown flags ritual verbs (wrap-epic etc.) not in kernel set
status: addressed
discovered_in: E-0038
addressed_by_commit:
    - 089adb25
---
## What

The kernel's `trailer-verb-unknown` check validates each commit's `aiwf-verb:`
trailer against the running binary's Cobra command tree (the closed set of
top-level verbs and subverbs). Ritual skills stamp **ritual verbs** that are
not kernel verbs — `wrap-epic`, `wrap-milestone`, `start-epic`,
`start-milestone`, `record-decision`, etc. — so every commit a ritual emits
with one of those verbs trips a `trailer-verb-unknown` warning.

## Evidence

Wrapping E-0038 produced two commits carrying `aiwf-verb: wrap-epic` (the
trailered integration merge and the wrap-artefact commit, exactly as
`aiwfx-wrap-epic` instructs). `aiwf check` / the pre-push hook flag both:

    trailer-verb-unknown (warning) × 2 — commit 19829699 carries
    aiwf-verb: "wrap-epic" which is not a registered top-level verb or subverb

Advisory only (push succeeds), but it recurs on **every** epic wrap and on any
ritual that stamps a non-kernel verb, so the warning count grows with normal
use and dilutes the signal of a genuinely-unknown verb.

## Tension

The ritual deliberately uses `wrap-epic` as the provenance label — it is
meaningful in `aiwf history E-NNNN` and distinguishes the wrap merge from an
ordinary `promote`. The kernel's closed-set check, sourced from the binary's
own command tree, has no knowledge of ritual verbs (the rituals live upstream
and are advisory). Both behaviors are individually correct; they collide at the
trailer-verb-unknown rule.

## Candidate resolutions

1. **Kernel learns a ritual-verb allowlist** — a small known set
   (`wrap-epic`, `wrap-milestone`, `start-epic`, `start-milestone`,
   `record-decision`, …) recognized by `trailer-verb-unknown` in addition to
   the Cobra command tree. Keeps the ritual provenance labels; the allowlist is
   the chokepoint.
2. **Namespaced ritual verbs** — e.g. `aiwf-verb: ritual/wrap-epic`, and the
   check skips (or separately validates) the `ritual/` namespace.
3. **Rituals reuse a kernel verb** — stamp `aiwf-verb: promote` (or a generic
   `wrap`) on wrap commits. Loses the wrap-epic/wrap-milestone distinction in
   history; weakest option.

Lean: option 1 (allowlist) — smallest change, preserves the existing ritual
labels and `aiwf history` rendering, single chokepoint.

## References

- **ADR-0014** / **E-0038** — the epic whose wrap surfaced this.
- `aiwfx-wrap-epic` / `aiwfx-wrap-milestone` skills — stamp `wrap-epic` / `promote`.
- Kernel `trailer-verb-unknown` finding rule.

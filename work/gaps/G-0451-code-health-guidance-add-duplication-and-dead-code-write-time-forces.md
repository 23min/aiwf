---
id: G-0451
title: 'Code-health guidance: add duplication and dead-code write-time forces'
status: open
---
## What's missing

The code-health guidance names YAGNI, single-source-of-truth, and "no half-finished implementations," yet an assistant can be fully compliant with all of them and still introduce the exact defects a structural sweep just surfaced: a routine block copy-pasted across dozens of call sites, a shared helper re-inlined instead of reused, and speculative code kept alive by a linter-silencing guard. Three specific forces are absent:

1. **Reuse-or-extract (search before you write).** No principle tells the assistant to look for an existing helper before authoring a routine-looking block, nor to treat *duplicated logic* as a defect in its own right. The rubric's single-source-of-truth force is about *data facts* ("why does the UI show 5 when the DB says 4"); low-coupling only gestures at shared code structurally. Neither counters the assistant's default of writing fresh code rather than reaching for an existing seam — the failure mode behind a trailer-helper that already existed being re-implemented inline at several sites.
2. **No code kept alive by a reference-only guard.** Nothing names the anti-pattern of adding a blank-identifier reference (`var _ = fn`) solely to silence the unused-symbol linter on reserved-for-future code — which defeats the very check whose job is to find it.
3. **A refactor deletes its own leftovers.** "No half-finished implementations" and "no shim without a named removal trigger" exist, but nothing states that when a change supersedes a code path, the old one is removed in the *same* change (or, if it must stay, gets a tracked owner and a named removal trigger — not silence). "Superseded but left behind" reads as live.

## Why it matters

These are the highest-frequency, lowest-visibility structural defects, and they land one plausible line at a time — no single diff looks wrong, so per-diff review does not catch them and neither does an exact-clone linter (the duplication is textually divergent). Priming is where they are cheapest to prevent: catching convergent duplication at review or sweep time is far more expensive than not writing the second copy. This is the write-time (priming) complement to the discovery-time sweep ritual — fewer defects introduced, and the ones that leak are found periodically rather than never.

Force #1 is the dominant lever: it targets both copy-paste *and* helper-bypass, and it addresses an assistant-shaped habit (author-fresh over search-first) that no current principle speaks to.

## Resolution shape

Add the three forces to the shipped code-health guidance, placed by frequency so the always-on subset stays lean:

- **Force #1 → a named force in the code-health rubric AND the always-on priming subset.** High-frequency; the one most worth priming every turn.
- **Force #2 → the code-health rubric (general form) and the repo's Go-conventions guidance (the concrete `var _ =` form).** The blank-identifier mechanism is Go-specific; the principle ("no reference added solely to keep unused code alive") is not.
- **Force #3 → the code-health rubric**, extending the existing named-removal-trigger principle rather than a new standalone.

Each addition is imperative, consumer-scoped, and shipped-surface-clean (no real ids, paths, or history) where it lands on a materialized surface. Keep the set small — three forces, not a checklist — to avoid diluting the guidance.

The parallel edits to the operator's personal language-agnostic and Go dotfiles are out of this repo's scope and tracked separately; this gap covers only the aiwf-shipped surfaces.

## Where to fix

- The code-health rubric skill (`wf-codebase-health`, and the language-agnostic `code-health` skill) — add force #1 as a named principle; add #2/#3 as principle lines.
- The always-on embedded guidance fragment — add force #1's one-line priming form to the code-health priming subset.
- The repo's Go-conventions guidance — add force #2's concrete `var _ =` form.
- Each shipped-ritual edit lands alongside a referencing structural test per the skill-edit-structural-test-backstop.

## Related

- G-0450 — `wf-structural-sweep`; the discovery-side complement to this priming-side gap.
- G-0447 — the convergent-duplication tax these forces target at write time.
- G-0449 — the dead-code (reserved-guard) class force #2 targets.
- `wf-codebase-health` / `code-health` — the rubric the new forces extend.

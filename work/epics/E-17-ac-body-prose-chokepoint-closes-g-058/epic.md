---
id: E-17
title: AC body prose chokepoint (closes G-058)
status: active
---

## Goal

Make non-empty AC body prose a kernel-enforced property. The design has always specified that each AC's body section carries prose detail (description, examples, edge cases, references — see [`docs/pocv3/plans/acs-and-tdd-plan.md:22`](../../../docs/pocv3/plans/acs-and-tdd-plan.md), [`docs/pocv3/design/design-decisions.md:139`](../../../docs/pocv3/design/design-decisions.md)) but no chokepoint enforces it: `aiwf add ac` scaffolds a bare heading, `acs-body-coherence` only checks heading↔frontmatter pairing, and the `aiwf-add` skill never prompts the operator to fill the body in. Result is repo-wide skimping (every milestone M-049..M-061 ships with empty AC bodies). See [G-058](../../gaps/G-058-ac-body-sections-ship-empty-no-chokepoint-enforces-prose-intent.md) for the full evidence.

End state: `aiwf check` reports `acs-body-empty` for any AC whose body section is empty (warning by default; error under `aiwf.yaml: tdd.strict: true`); `aiwf add ac` accepts `--body-file` per AC so the body lands in the same atomic commit; the `aiwf-add` skill names "fill in the body" as a required follow-up step. Together these make the design intent mechanically enforceable rather than aspirational.

## Scope

- `aiwf check` finding `acs-body-empty` (warning by default; error under `tdd.strict: true`). Shares the `tdd.strict` config field with [E-16](../E-16-tdd-policy-declaration-chokepoint-closes-g-055/epic.md)'s `milestone-tdd-undeclared` (single source of truth for the project's TDD strictness posture).
- `aiwf add ac` accepts `--body-file <path>` per AC for in-verb body scaffolding. Multi-AC form pairs `--body-file` positionally with `--title` (or accepts a directory of `AC-N.md` files).
- `aiwf-add` skill update: name "fill in the AC body" as the required next step after `aiwf add ac` when `--body-file` was not used; document the body shape (one paragraph: pass criteria, edge cases, code references).

## Out of scope

- **Retroactive backfill of historical AC bodies.** Existing milestones (M-049..M-061) surface as warnings under the new check finding; this epic does not write the prose for them. Authors who care can backfill milestone-by-milestone; that work is not blocking.
- **Schema for body content.** The body is markdown prose, and the kernel principle "prose is not parsed" applies (per `acs-and-tdd-plan.md:197`). The check rule asserts presence, not structure. No grammar, no required keywords, no required sub-headings.
- **Promote-time guard.** Refusing AC `open -> met` (or milestone `draft -> in_progress`) when AC bodies are empty is overkill once the check finding fires — same reasoning as G-055's deferred promote-time guard. YAGNI; revisit if drift shows up.
- **Title-length validator changes.** The existing 80-char limit on AC titles is fine; this epic puts the detail in the body, not the title.
- **Render-side enforcement.** The I3 governance render reads what's written; if the body is empty, the page renders empty. Surfacing "this AC has an empty body" in the render is downstream of the check finding and not in this epic's scope.
